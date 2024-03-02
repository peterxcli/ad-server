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
	// adStream is the redis stream name for the ad
	adStream string
	stopCh   chan struct{}
}

// Restore restores the latest version of an ad from the database.
// It returns the version number of the restored ad and any error encountered.
// The error could be ErrRecordNotFound if no ad is found or a DB connection error.
func (a *AdService) Restore() (version int, err error) {
	err = a.db.Model(&model.Ad{}).Select("MAX(version)").Scan(&version).Error
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Subscribe implements model.AdService.
// Subscribe implements model.AdService.
func (a *AdService) Subscribe(offset int) error {
	ctx := context.Background()
	lastID := fmt.Sprintf("%d-0", offset) // Assuming offset can be mapped directly to Redis Stream IDs

	for {
		select {
		case <-a.stopCh:
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
