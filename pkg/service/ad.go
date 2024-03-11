package service

import (
	"context"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/runner"
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
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrTimeout = fmt.Errorf("timeout")
	ErrUnknown = fmt.Errorf("unknown error")
)

type AdService struct {
	shutdown    atomic.Bool
	runner      *runner.Runner
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
	go a.runner.Start() // Start the runner
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
	txn := a.db.Begin()
	err = txn.Raw("SELECT COALESCE(MAX(version), 0) FROM ads").Scan(&a.Version).Error
	if err != nil {
		return err
	}
	var ads []*model.Ad
	err = txn.Find(&ads).Error
	if err != nil {
		return err
	}
	err = txn.Commit().Error
	if err != nil {
		return err
	}
	requestID := uuid.New().String()
	a.runner.ResponseChan.Store(requestID, make(chan interface{}, 1))
	defer a.runner.ResponseChan.Delete(requestID)
	a.runner.RequestChan <- &runner.CreateBatchAdRequest{
		Request: runner.Request{RequestID: requestID},
		Ads:     ads,
	}

	select {
	case resp := <-a.runner.ResponseChan.Load(requestID):
		if resp, ok := resp.(*runner.CreateAdResponse); ok {
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
					ad := &runner.CreateAdRequest{}
					streamVersion, _ := strconv.ParseInt(strings.Split(m.ID, "-")[1], 10, 64)
					if a.Version < int(streamVersion) {
						a.Version = int(streamVersion)
						lastID = m.ID
					} else {
						// our version is the same or higher than the stream version
						continue
					}
					json.Unmarshal([]byte(m.Values["ad"].(string)), ad)
					a.runner.RequestChan <- ad
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
	adReq := &runner.CreateAdRequest{
		Request: runner.Request{RequestID: requestID},
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
		Values:     []interface{}{"ad", string(adReqMapStr)},
		ID:         fmt.Sprintf("0-%d", ad.Version),
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
	a.runner.ResponseChan.Store(requestID, make(chan interface{}, 1))
	defer a.runner.ResponseChan.Delete(requestID)
	err = a.storeAndPublishWithLock(ctx, ad, requestID)
	if err != nil {
		return "", err
	}

	select {
	case resp := <-a.runner.ResponseChan.Load(requestID):
		if resp, ok := resp.(*runner.CreateAdResponse); ok {
			return resp.AdID, resp.Err
		}
	case <-time.After(3 * time.Second):
		return "", ErrTimeout
	}

	return "", ErrUnknown
}

// GetAds implements model.AdService.
func (a *AdService) GetAds(ctx context.Context, req *model.GetAdRequest) ([]*model.Ad, int, error) {
	a.wg.Add(1)
	defer a.wg.Done()

	requestID := uuid.New().String()

	a.runner.ResponseChan.Store(requestID, make(chan interface{}, 1))
	defer a.runner.ResponseChan.Delete(requestID)

	a.runner.RequestChan <- &runner.GetAdRequest{
		Request:      runner.Request{RequestID: requestID},
		GetAdRequest: req,
	}

	select {
	case resp := <-a.runner.ResponseChan.Load(requestID):
		if resp, ok := resp.(*runner.GetAdResponse); ok {
			return resp.Ads, resp.Total, resp.Err
		}
	case <-time.After(3 * time.Second):
		return nil, 0, ErrTimeout
	}

	return nil, 0, ErrUnknown
}

func NewAdService(runner *runner.Runner, db *gorm.DB, redis *redis.Client, locker *redislock.Client, asynqClient *asynq.Client) model.AdService {
	return &AdService{
		runner:      runner,
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
