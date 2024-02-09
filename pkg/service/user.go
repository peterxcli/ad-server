package service

import (
	"context"
	"errors"
	"time"

	tokensvc "bikefest/internal/token"
	"bikefest/pkg/model"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrEventExceedsMaxSubscriptions = errors.New("exceeds the maximum number of subscriptions")
	ErrEventAlreadySubscribed       = errors.New("already subscribed to this event")
	ErrUserNotFound                 = errors.New("user not found")
)

func NewUserService(db *gorm.DB, cache *redis.Client) model.UserService {
	return &UserServiceImpl{
		db:    db,
		cache: cache,
	}
}

type UserServiceImpl struct {
	db    *gorm.DB
	cache *redis.Client
}

func (us *UserServiceImpl) SubscribeEvent(ctx context.Context, userID string, eventID string) error {
	event := &model.Event{}
	err := us.db.WithContext(ctx).Model(&model.User{ID: userID}).Association("Events").Find(event, &model.Event{ID: &eventID})
	if err != nil {
		return err
	}
	if event.ID != nil {
		return ErrEventAlreadySubscribed
	}
	// Check if events exceed 10
	// TODO: meow
	if count := us.db.WithContext(ctx).Model(&model.User{ID: userID}).Association("Events").Count(); count >= 10 {
		return ErrEventExceedsMaxSubscriptions
	}
	return us.db.WithContext(ctx).Model(&model.User{ID: userID}).Association("Events").Append(&model.Event{ID: &eventID})
}

func (us *UserServiceImpl) UnsubscribeEvent(ctx context.Context, userID string, eventID string) error {
	return us.db.WithContext(ctx).Model(&model.User{ID: userID}).Association("Events").Delete(&model.Event{ID: &eventID})
}

func (us *UserServiceImpl) GetUserSubscribeEvents(ctx context.Context, userID string) ([]*model.Event, error) {
	var events []*model.Event
	err := us.db.WithContext(ctx).Model(&model.User{ID: userID}).Association("Events").Find(&events)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (us *UserServiceImpl) ListUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	err := us.db.WithContext(ctx).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (us *UserServiceImpl) CreateFakeUser(ctx context.Context, user *model.User) error {
	return us.db.WithContext(ctx).Create(user).Error
}

// GetUserByID implements model.UserService.
func (us *UserServiceImpl) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	err := us.db.WithContext(ctx).Where(&model.User{ID: id}).First(user).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, ErrUserNotFound
	case err != nil:
		return nil, err
	}
	return user, nil
}

func (us *UserServiceImpl) CreateAccessToken(_ context.Context, user *model.User, secret string, expiry int64) (accessToken string, err error) {
	return tokensvc.CreateAccessToken(user, secret, expiry)
}

func (us *UserServiceImpl) CreateRefreshToken(_ context.Context, user *model.User, secret string, expiry int64) (refreshToken string, err error) {
	return tokensvc.CreateRefreshToken(user, secret, expiry)
}

func (*UserServiceImpl) VerifyRefreshToken(_ context.Context, refreshToken string, secret string) (user *model.User, err error) {
	return tokensvc.VerifyRefreshToken(refreshToken, secret)
}

// Logout implements model.UserService.
func (us *UserServiceImpl) Logout(ctx context.Context, token *string, secret string) error {
	claims, err := tokensvc.ExtractCustomClaimsFromToken(token, secret)
	if err != nil {
		return err
	}
	ttl := claims.ExpiresAt.Unix() - time.Now().Unix()
	if ttl < 0 {
		return nil
	}
	return us.cache.Set(ctx, claims.ID, *token, time.Duration(ttl)*time.Second).Err()
}
