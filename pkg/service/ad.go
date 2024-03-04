package service

import (
	"context"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/runner"
	"errors"
	"fmt"
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
	shutdown atomic.Bool
	runner   *runner.Runner
	db       *gorm.DB
	redis    *redis.Client
	locker   *redislock.Client
	lockKey  string
	// adStream is the redis stream name for the ad
	adStream   string
	mu         sync.Mutex
	wg         sync.WaitGroup
	onShutdown []func()
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
	stopCh := make(chan struct{}, 1)

	a.wg.Add(1)
	defer a.wg.Done()

	a.registerOnShutdown(func() {
		close(stopCh)
	})

	operation := func() error {
		version, err := a.Restore()
		if err != nil {
			return err
		}
		err = a.Subscribe(version)
		if err != nil {
			return err
		}
		return nil
	}

	operationBackoff := backoff.NewExponentialBackOff()
	currentRetry := 0
	maxRetry := 5
	for a.shutdown.Load() == false {
		select {
		case <-stopCh:
			return nil
		default:
			err := backoff.Retry(operation, operationBackoff)
			if err != nil {
				currentRetry++
				if currentRetry > maxRetry {
					return fmt.Errorf("max retry reached: %w", err)
				}
			} else {
				currentRetry = 0
				return nil
			}
		}
	}
	return nil
}

// Restore restores the latest version of an ad from the database.
// It returns the version number of the restored ad and any error encountered.
// The error could be ErrRecordNotFound if no ad is found or a DB connection error.
func (a *AdService) Restore() (version int, err error) {
	err = a.db.Model(&model.Ad{}).Select("MAX(version)").Scan(&version).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return 0, nil
	case err != nil:
		return 0, err
	}
	return version, nil
}

// Subscribe implements model.AdService.
func (a *AdService) Subscribe(offset int) error {
	ctx := context.Background()
	lastID := fmt.Sprintf("%d-0", offset) // Assuming offset can be mapped directly to Redis Stream IDs
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
				Block:   0,
				Count:   10,
			}
			msgs, err := a.redis.XRead(ctx, xReadArgs).Result()
			if err != nil {
				// Handle error, for example, log or wait before retrying
				time.Sleep(1 * time.Second)
				continue
			}
			for _, msg := range msgs {
				for _, m := range msg.Messages {
					ad := &runner.CreateAdRequest{}
					ad.FromMap(m.Values) // assume data are valid
					a.runner.RequestChan <- ad
					lastID = m.ID // Update lastID to the latest message ID
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

// storeAndPublishWithLock
//
// 1. locks the lockKey
//
// 2. stores the ad in the database, and set the version of the new ad to `SELECT MAX(version) FROM ad“ + 1
//
// 3. publishes the ad into redis stream. (ensure the message sequence number is the same as the ad's version)
//
// 4. releases the lock
func (a *AdService) storeAndPublishWithLock(ctx context.Context, ad *model.Ad) (requestID string, err error) {
	lock, err := a.locker.Obtain(ctx, a.lockKey, 0, nil)
	if err != nil {
		return
	}
	defer lock.Release(ctx)
	txn := a.db.Begin()
	if err = txn.Error; err != nil {
		return
	}
	var maxVersion int
	if err = txn.Raw("SELECT MAX(version) FROM ad").Scan(&maxVersion).Error; err != nil {
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
	requestID = uuid.New().String()
	adReq := &runner.CreateAdRequest{
		Request: runner.Request{RequestID: requestID},
		Ad:      ad,
	}
	adReqMap, err := adReq.ToMap()
	if err != nil {
		return
	}
	_, err = a.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: a.adStream,
		Values: adReqMap,
	}).Result()
	if err != nil {
		return
	}
	return requestID, nil
}

// CreateAd implements model.AdService.
func (a *AdService) CreateAd(ctx context.Context, ad *model.Ad) (adID string, err error) {
	a.wg.Add(1)
	defer a.wg.Done()
	requestID, err := a.storeAndPublishWithLock(ctx, ad)
	if err != nil {
		return "", err
	}

	select {
	case resp := <-a.runner.ResponseChan[requestID]:
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
	a.runner.RequestChan <- &runner.GetAdRequest{
		Request:      runner.Request{RequestID: requestID},
		GetAdRequest: req,
	}

	select {
	case resp := <-a.runner.ResponseChan[requestID]:
		if resp, ok := resp.(*runner.GetAdResponse); ok {
			return resp.Ads, resp.Total, resp.Err
		}
	case <-time.After(3 * time.Second):
		return nil, 0, ErrTimeout
	}

	return nil, 0, ErrUnknown
}

func NewAdService(runner *runner.Runner, db *gorm.DB, redis *redis.Client, locker *redislock.Client) model.AdService {
	return &AdService{
		runner:     runner,
		db:         db,
		redis:      redis,
		locker:     locker,
		lockKey:    "lock:ad",
		onShutdown: make([]func(), 0),
		adStream:   "ad",
		shutdown:   atomic.Bool{},
	}
}
