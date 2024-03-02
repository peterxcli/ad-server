package model

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Ad struct {
	ID       uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Title    string         `gorm:"type:text" json:"title"`
	Content  string         `gorm:"type:text" json:"content"`
	StartAt  time.Time      `gorm:"type:timestamp" json:"start_at"`
	EndAt    time.Time      `gorm:"type:timestamp" json:"end_at"`
	AgeStart int            `json:"age_start"`
	AgeEnd   int            `json:"age_end"`
	Gender   pq.StringArray `gorm:"type:text[]" json:"gender"`
	Country  []string       `gorm:"type:text[]" json:"country"`
	Platform []string       `gorm:"type:text[]" json:"platform"`
	Version  int            `gorm:"type:integer" json:"version"` // Version log index(offset) in the redis stream
}

func (a *Ad) BeforeCreate(*gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
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
	CreateAd(ctx context.Context, ad *Ad) (adID string, er error)
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
