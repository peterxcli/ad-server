package model

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID   string `json:"id" gorm:"type:varchar(36);primary_key"`
	Name string `json:"name" gorm:"type:varchar(255);index"`
	// TODO: add more user info for line login and line message API identity
	Events []Event `gorm:"many2many:user_events;"`
}

func (u *User) BeforeCreate(*gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

type CreateFakeUserRequest struct {
	Name string `json:"name,default=fake_user" binding:"required"`
}

type UserResponse struct {
	Msg  string `json:"msg"`
	Data *User  `json:"data"`
}

type UserListResponse struct {
	Msg  string  `json:"msg"`
	Data []*User `json:"data"`
}

type UserService interface {
	GetUserByID(ctx context.Context, id string) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)
	CreateFakeUser(ctx context.Context, user *User) error
	CreateAccessToken(ctx context.Context, user *User, secret string, expiry int64) (accessToken string, err error)
	CreateRefreshToken(ctx context.Context, user *User, secret string, expiry int64) (refreshToken string, err error)
	VerifyRefreshToken(ctx context.Context, refreshToken string, secret string) (user *User, err error)
	Logout(ctx context.Context, token *string, secret string) error
	SubscribeEvent(ctx context.Context, userID string, eventID string) error
	UnsubscribeEvent(ctx context.Context, userID string, eventID string) error
	GetUserSubscribeEvents(ctx context.Context, userID string) ([]*Event, error)
}
