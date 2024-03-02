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
	// Version log index(offset) in the redis stream
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
	Restore() (version int, err error)
}

type InMemoryStore interface {
	CreateAd(ad *Ad) (string, error)
	GetAds(req *GetAdRequest) ([]*Ad, int, error)
	// Restore the ads from the db, and return the highest version in the store
	CreateBatchAds(ads []*Ad) (version int, err error)
}
