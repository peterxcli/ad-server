package model

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Ad struct {
	ID       string
	Title    string
	Content  string
	StartAt  time.Time
	EndAt    time.Time
	AgeStart int
	AgeEnd   int
	Gender   []string
	Country  []string
	Platform []string
	// Version is used to handle optimistic lock
	Version int
}

func (a *Ad) BeforeCreate(*gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return
}

// StartAt < Now() < EndAt
type GetAdRequest struct {
	// AgeStart < Age < AgeEnd
	Age      int
	Country  string
	Gender   string
	Platform string

	Offset int
	Limit  int
}

type AdService interface {
	CreateAd(ctx context.Context, ad *Ad) (string, error)
	GetAds(ctx context.Context, req *GetAdRequest) ([]*Ad, int, error)
	// Subscribe to the redis stream
	Subscribe(offset int) error
}

type InMemoryStore interface {
	CreateAd(ad *Ad) (string, error)
	GetAds(req *GetAdRequest) ([]*Ad, int, error)
}