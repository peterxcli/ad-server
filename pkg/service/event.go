package service

import (
	"bikefest/pkg/model"
	"context"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"time"
)

type EventServiceImpl struct {
	db    *gorm.DB
	cache *redis.Client
}

func (es *EventServiceImpl) StoreAll(ctx context.Context, events []*model.Event) error {
	txn := es.db.WithContext(ctx).Begin()
	for _, event := range events {
		err := txn.WithContext(ctx).Create(event).Error
		if err != nil {
			txn.Rollback()
			return err
		}
	}
	err := txn.Commit().Error
	if err != nil {
		return err
	}
	for _, event := range events {
		go es.cache.Del(ctx, model.EventCacheKey+*event.ID)
	}
	return err
}

func (es *EventServiceImpl) FindAll(ctx context.Context, page, limit int64) (events []*model.Event, err error) {
	err = es.db.WithContext(ctx).Limit(int(limit)).Offset(int((page - 1) * limit)).Find(&events).Error
	if err != nil {
		return nil, err
	}
	return
}

func (es *EventServiceImpl) FindByID(ctx context.Context, id string) (event *model.Event, err error) {
	eventCache := &model.EventCache{}
	err = es.cache.HGetAll(ctx, model.EventCacheKey+id).Scan(eventCache)
	if err == nil && eventCache.ID != "" {
		event = &model.Event{
			ID:             &eventCache.ID,
			EventTimeStart: &eventCache.EventTimeStart,
			EventTimeEnd:   &eventCache.EventTimeEnd,
			EventDetail:    &eventCache.EventDetail,
			Model: gorm.Model{
				CreatedAt: eventCache.CreatedAt,
				UpdatedAt: eventCache.UpdatedAt,
			},
		}
		return
	}
	err = es.db.WithContext(ctx).Where(&model.Event{ID: &id}).First(&event).Error
	if err != nil {
		return nil, err
	}
	eventCache = &model.EventCache{
		ID:             *event.ID,
		EventTimeStart: *event.EventTimeStart,
		EventTimeEnd:   *event.EventTimeEnd,
		EventDetail:    *event.EventDetail,
		CreatedAt:      event.CreatedAt,
		UpdatedAt:      event.UpdatedAt,
	}
	err = es.cache.HSet(ctx, model.EventCacheKey+id, eventCache).Err()
	es.cache.Expire(ctx, model.EventCacheKey+id, 2*time.Hour)
	return
}

func (es *EventServiceImpl) Store(ctx context.Context, event *model.Event) error {
	err := es.db.WithContext(ctx).Create(event).Error
	if err != nil {
		return err
	}
	go es.cache.Del(ctx, model.EventCacheKey+*event.ID)
	return nil
}

func (es *EventServiceImpl) Update(ctx context.Context, event *model.Event) (rowAffected int64, err error) {
	res := es.db.WithContext(ctx).Model(event).Updates(event)
	rowAffected, err = res.RowsAffected, res.Error
	if err != nil {
		return 0, err
	}
	go es.cache.Del(ctx, model.EventCacheKey+*event.ID)
	return
}

func (es *EventServiceImpl) Delete(ctx context.Context, event *model.Event) (rowAffected int64, err error) {
	res := es.db.WithContext(ctx).Delete(event)
	rowAffected, err = res.RowsAffected, res.Error
	if err != nil {
		return 0, err
	}
	go es.cache.Del(ctx, model.EventCacheKey+*event.ID)
	return
}

func NewEventService(db *gorm.DB, cache *redis.Client) model.EventService {
	return &EventServiceImpl{
		db:    db,
		cache: cache,
	}
}
