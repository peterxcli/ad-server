package service

import (
	"context"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/runner"
	"fmt"
	"time"

	"github.com/bsm/redislock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrTimeout = fmt.Errorf("timeout")
	ErrUnknown = fmt.Errorf("unknown error")
)

type AdService struct {
	runner  *runner.Runner
	db      *gorm.DB
	redis   *redis.Client
	locker  *redislock.Client
	lockKey string
	stopCh  chan struct{}
}

// Restore implements model.AdService.
func (a *AdService) Restore() (version int, err error) {
	panic("unimplemented")
}

// Subscribe implements model.AdService.
func (a *AdService) Subscribe(offset int) error {
	panic("unimplemented")
}

// lockAndStoreAndPublish
//
// 1. locks the lockKey
//
// 2. stores the ad in the database, and set the version of the new ad to `SELECT MAX(version) FROM ad“ + 1
//
// 3. publishes the ad into redis stream. (ensure the message sequence number is the same as the ad's version)
//
// 4. releases the lock
func (a *AdService) lockAndStoreAndPublish(ctx context.Context, ad *model.Ad) error {
	lock, err := a.locker.Obtain(ctx, a.lockKey, 0, nil)
	if err != nil {
		return err
	}
	defer lock.Release(ctx)
	txn := a.db.Begin()
	if err := txn.Error; err != nil {
		return err
	}
	var maxVersion int
	if err := txn.Raw("SELECT MAX(version) FROM ad").Scan(&maxVersion).Error; err != nil {
		txn.Rollback()
		return err
	}
	ad.Version = maxVersion + 1
	if err := txn.Create(ad).Error; err != nil {
		txn.Rollback()
		return err
	}
	err = txn.Commit().Error
	if err != nil {
		return err
	}
	_, err = a.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: "ad",
		Values: ad,
	}).Result()
	if err != nil {
		return err
	}
	return nil
}

// CreateAd implements model.AdService.
func (a *AdService) CreateAd(ctx context.Context, ad *model.Ad) (string, error) {
	err := a.lockAndStoreAndPublish(ctx, ad)
	if err != nil {
		return "", err
	}

	requestID := uuid.New().String()
	a.runner.RequestChan <- &runner.CreateAdRequest{
		Request: runner.Request{RequestID: requestID},
		Ad:      ad,
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
		runner:  runner,
		db:      db,
		redis:   redis,
		locker:  locker,
		lockKey: "lock:ad",
		stopCh:  make(chan struct{}),
	}
}
