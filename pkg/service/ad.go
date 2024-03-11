package service

import (
	"context"
	"database/sql"
	"dcard-backend-2024/pkg/dispatcher"
	"dcard-backend-2024/pkg/model"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bsm/redislock"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrTimeout = fmt.Errorf("timeout")
	ErrUnknown = fmt.Errorf("unknown error")
)

type AdService struct {
	shutdown    atomic.Bool
	dispatcher  *dispatcher.Dispatcher
	db          *gorm.DB
	redis       *redis.Client
	locker      *redislock.Client
	asynqClient *asynq.Client
	lockKey     string
	// adStream is the redis stream name for the ad
	adStream   string
	mu         sync.Mutex
	wg         sync.WaitGroup
	onShutdown []func()
	Version    int // Version is the latest version of the ad
}

// DeleteAd implements model.AdService.
func (a *AdService) DeleteAd(ctx context.Context, adID string) error {
	ctx = context.Background()
	// RedisLock Lock Key: lock:ad
	lock, err := a.locker.Obtain(ctx, a.lockKey, 100*time.Millisecond, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.ExponentialBackoff(1*time.Millisecond, 5*time.Millisecond), 10),
	})
	if err != nil {
		log.Printf("error obtaining lock: %v", err)
		return err
	}
	defer func() {
		// Release Lock
		err := lock.Release(ctx)
		if err != nil {
			log.Printf("error releasing lock: %v", err)
		}
	}()
	// Begin Transaction
	// UPDATE ads SET is_active = false AND version = `SELECT MAX(version) FROM ads` + 1 WHERE id = adID
	// DELETE FROM ads WHERE version < `SELECT MAX(version) FROM ads` AND is_active = false
	// Commit Transaction
	txn := a.db.Begin()
	if err = txn.Error; err != nil {
		return err
	}
	var maxVersion int
	if err = txn.Raw("SELECT COALESCE(MAX(version), 0) FROM ads").Scan(&maxVersion).Error; err != nil {
		txn.Rollback()
		return err
	}
	maxVersion++
	if err = txn.Model(&model.Ad{}).Where("id = ?", adID).
		Update("is_active", false).
		Update("version", maxVersion).Error; err != nil {
		txn.Rollback()
		return err
	}
	if err = txn.Delete(&model.Ad{}, "version < ? AND is_active = false", maxVersion).Error; err != nil {
		txn.Rollback()
		return err
	}
	err = txn.Commit().Error
	if err != nil {
		return err
	}
	adReqMapStr, err := json.Marshal(dispatcher.DeleteAdRequest{AdID: adID})
	if err != nil {
		log.Printf("error marshalling ad request: %v", err)
		return err
	}
	// Publish to Redis Stream
	// XADD ad 0-`SELECT MAX(version) FROM ads` {"ad": "adReqJsonStr"}
	_, err = a.redis.XAdd(ctx, &redis.XAddArgs{
		Stream:     a.adStream,
		NoMkStream: false,
		Approx:     false,
		MaxLen:     100000,
		Values: []interface{}{
			"ad", string(adReqMapStr),
			"type", "delete",
		},
		ID: fmt.Sprintf("0-%d", maxVersion),
	}).Result()
	if err != nil {
		log.Printf("error publishing to redis: %v", err)
		return err
	}
	return nil
}

// Shutdown implements model.AdService.
func (a *AdService) Shutdown(ctx context.Context) error {
	a.shutdown.Store(true)
	done := make(chan struct{})
	a.mu.Lock()
	for _, f := range a.onShutdown {
		go f()
	}
	a.mu.Unlock()
	go func() {
		a.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run implements model.AdService.
func (a *AdService) Run() error {
	go a.dispatcher.Start() // Start the dispatcher
	stopCh := make(chan struct{}, 1)

	a.wg.Add(1)
	defer a.wg.Done()

	a.registerOnShutdown(func() {
		close(stopCh)
	})

	operation := func() error {
		err := a.Restore()
		if err != nil {
			log.Printf("error restoring: %v", err)
			return err
		}
		err = a.Subscribe()
		if err != nil {
			log.Printf("error subscribing: %v", err)
			return err
		}
		return nil
	}

	maxRetry := 5
	operationBackoff := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(maxRetry))

	for a.shutdown.Load() == false {
		select {
		case <-stopCh:
			return nil
		default:
			err := backoff.Retry(operation, operationBackoff)
			if err != nil {
				log.Printf("error running: %v", err)
				return err
			}
		}
	}
	return nil
}

// Restore restores the latest version of an ad from the database.
// It returns the version number of the restored ad and any error encountered.
// The error could be ErrRecordNotFound if no ad is found or a DB connection error.
func (a *AdService) Restore() (err error) {
	txn := a.db.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	err = txn.Raw("SELECT COALESCE(MAX(version), 0) FROM ads").Scan(&a.Version).Error
	if err != nil {
		return err
	}
	var ads []*model.Ad
	err = txn.Where("is_active = ?", true).Find(&ads).Error
	if err != nil {
		return err
	}
	err = txn.Commit().Error
	if err != nil {
		return err
	}
	requestID := uuid.New().String()
	a.dispatcher.ResponseChan.Store(requestID, make(chan interface{}, 1))
	defer a.dispatcher.ResponseChan.Delete(requestID)
	a.dispatcher.RequestChan <- &dispatcher.CreateBatchAdRequest{
		Request: dispatcher.Request{RequestID: requestID},
		Ads:     ads,
	}

	select {
	case resp := <-a.dispatcher.ResponseChan.Load(requestID):
		if resp, ok := resp.(*dispatcher.CreateAdResponse); ok {
			if resp.Err == nil {
				log.Printf("Restored version: %d successfully\n", a.Version)
			}
			return resp.Err
		}
	case <-time.After(3 * time.Second):
		return ErrTimeout
	}
	return ErrUnknown
}

// Subscribe implements model.AdService.
func (a *AdService) Subscribe() error {
	log.Printf("subscribing to redis stream with offset: %d", a.Version)
	ctx := context.Background()
	lastID := fmt.Sprintf("0-%d", a.Version) // Assuming offset can be mapped directly to Redis Stream IDs
	stopCh := make(chan struct{}, 1)

	a.wg.Add(1)
	defer a.wg.Done()

	a.registerOnShutdown(func() {
		close(stopCh)
	})

	for a.shutdown.Load() == false {
		select {
		case <-stopCh:
			return nil
		default:
			// Reading from the stream
			xReadArgs := &redis.XReadArgs{
				Streams: []string{a.adStream, lastID},
				Block:   3 * time.Second,
				Count:   10,
			}
			msgs, err := a.redis.XRead(ctx, xReadArgs).Result()
			if err != nil {
				// log.Printf("error reading from redis: %v", err)
				continue
			}
			for _, msg := range msgs {
				for _, m := range msg.Messages {
					log.Printf("received message: %v\n", m)
					streamVersion, _ := strconv.ParseInt(strings.Split(m.ID, "-")[1], 10, 64)
					if a.Version < int(streamVersion) {
						a.Version = int(streamVersion)
						lastID = m.ID
					} else {
						// our version is the same or higher than the stream version
						continue
					}
					switch m.Values["type"].(string) {
					case "create":
						payload := &dispatcher.CreateAdRequest{}
						json.Unmarshal([]byte(m.Values["ad"].(string)), payload)
						a.dispatcher.RequestChan <- payload
					case "delete":
						payload := &dispatcher.DeleteAdRequest{}
						json.Unmarshal([]byte(m.Values["ad"].(string)), payload)
						a.dispatcher.RequestChan <- payload
					default:
						log.Printf("unknown message type: %s", m.Values["type"].(string))
					}
				}
			}
		}
	}
	return nil
}

func (a *AdService) registerOnShutdown(f func()) {
	a.mu.Lock()
	a.onShutdown = append(a.onShutdown, f)
	a.mu.Unlock()
}

func (a *AdService) onShutdownNum() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.onShutdown)
}

// storeAndPublishWithLock
//
// 1. locks the lockKey
//
// 2. stores the ad in the database, and set the version of the new ad to `SELECT MAX(version) FROM ad“ + 1
//
// 3. publishes the ad into redis stream. (ensure the message sequence number is the same as the ad's version)
//
// 4. releases the lock
func (a *AdService) storeAndPublishWithLock(ctx context.Context, ad *model.Ad, requestID string) (err error) {
	ctx = context.Background()
	lock, err := a.locker.Obtain(ctx, a.lockKey, 100*time.Millisecond, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.ExponentialBackoff(1*time.Millisecond, 5*time.Millisecond), 10),
	})
	if err != nil {
		log.Printf("error obtaining lock: %v", err)
		return
	}
	defer func() {
		err := lock.Release(ctx)
		if err != nil {
			log.Printf("error releasing lock: %v", err)
		}
	}()
	txn := a.db.Begin()
	if err = txn.Error; err != nil {
		return
	}
	var maxVersion int
	if err = txn.Raw("SELECT COALESCE(MAX(version), 0) FROM ads").Scan(&maxVersion).Error; err != nil {
		txn.Rollback()
		return
	}
	ad.Version = maxVersion + 1
	if err = txn.Create(ad).Error; err != nil {
		txn.Rollback()
		return
	}
	err = txn.Commit().Error
	if err != nil {
		return
	}
	adReq := &dispatcher.CreateAdRequest{
		Request: dispatcher.Request{RequestID: requestID},
		Ad:      ad,
	}
	// adReqMap, err := adReq.ToMap()
	adReqMapStr, err := json.Marshal(adReq)
	// return requestID, nil
	// log.Printf("adReqJsonStr: %s", adReqJsonStr)
	if err != nil {
		log.Printf("error marshalling ad request: %v", err)
		return
	}
	_, err = a.redis.XAdd(ctx, &redis.XAddArgs{
		Stream:     a.adStream,
		NoMkStream: false,
		Approx:     false,
		MaxLen:     100000,
		Values: []interface{}{
			"ad", string(adReqMapStr),
			"type", "create",
		},
		ID: fmt.Sprintf("0-%d", ad.Version),
	}).Result()
	if err != nil {
		log.Printf("error publishing to redis: %v", err)
		return
	}
	return nil
}

// CreateAd implements model.AdService.
func (a *AdService) CreateAd(ctx context.Context, ad *model.Ad) (adID string, err error) {
	a.wg.Add(1)
	defer a.wg.Done()
	requestID := uuid.New().String()
	a.dispatcher.ResponseChan.Store(requestID, make(chan interface{}, 1))
	defer a.dispatcher.ResponseChan.Delete(requestID)
	err = a.storeAndPublishWithLock(ctx, ad, requestID)
	if err != nil {
		return "", err
	}

	err = a.registerAdDeleteTask(ad)
	if err != nil {
		return "", err
	}

	select {
	case resp := <-a.dispatcher.ResponseChan.Load(requestID):
		if resp, ok := resp.(*dispatcher.CreateAdResponse); ok {
			return resp.AdID, resp.Err
		}
	case <-time.After(3 * time.Second):
		return "", ErrTimeout
	}

	return "", ErrUnknown
}

func (a *AdService) registerAdDeleteTask(ad *model.Ad) error {
	payload := &model.AsynqDeletePayload{AdID: ad.ID.String()}
	task, err := payload.ToTask()
	if err != nil {
		return err
	}
	taskID := fmt.Sprintf("%s-%s", payload.TypeName(), ad.ID.String())
	processTime := ad.EndAt.T().In(time.Local)
	// FIXME: the timezone is not correct, the scheduled time in the asynq UI is utc+8, but the internal scheduler is utc+0
	_, err = a.asynqClient.Enqueue(
		task,
		asynq.ProcessAt(processTime),
		asynq.TaskID(taskID),
	)
	return err
}

// GetAds implements model.AdService.
func (a *AdService) GetAds(ctx context.Context, req *model.GetAdRequest) ([]*model.Ad, int, error) {
	a.wg.Add(1)
	defer a.wg.Done()

	requestID := uuid.New().String()

	a.dispatcher.ResponseChan.Store(requestID, make(chan interface{}, 1))
	defer a.dispatcher.ResponseChan.Delete(requestID)

	a.dispatcher.RequestChan <- &dispatcher.GetAdRequest{
		Request:      dispatcher.Request{RequestID: requestID},
		GetAdRequest: req,
	}

	select {
	case resp := <-a.dispatcher.ResponseChan.Load(requestID):
		if resp, ok := resp.(*dispatcher.GetAdResponse); ok {
			return resp.Ads, resp.Total, resp.Err
		}
	case <-time.After(3 * time.Second):
		return nil, 0, ErrTimeout
	}

	return nil, 0, ErrUnknown
}

func NewAdService(dispatcher *dispatcher.Dispatcher, db *gorm.DB, redis *redis.Client, locker *redislock.Client, asynqClient *asynq.Client) model.AdService {
	return &AdService{
		dispatcher:  dispatcher,
		db:          db,
		redis:       redis,
		locker:      locker,
		lockKey:     "lock:ad",
		onShutdown:  make([]func(), 0),
		adStream:    "ad",
		asynqClient: asynqClient,
		shutdown:    atomic.Bool{},
		Version:     0,
	}
}
