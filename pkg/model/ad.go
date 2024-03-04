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
	Country  pq.StringArray `gorm:"type:text[]" json:"country"`
	Platform pq.StringArray `gorm:"type:text[]" json:"platform"`
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
	Age      int    `form:"age" binding:"omitempty"`
	Country  string `form:"country" binding:"omitempty"`
	Gender   string `form:"gender" binding:"omitempty"`
	Platform string `form:"platform" binding:"omitempty"`

	Offset int `form:"offset" binding:"omitempty"`
	Limit  int `form:"limit" binding:"omitempty"`
}

type GetAdsPageResponse struct {
	Ads   []*Ad `json:"ads"`
	Total int   `json:"total"`
}

type CreateAdRequest struct {
	Title    string    `json:"title" binding:"required"`
	Content  string    `json:"content" binding:"required"`
	StartAt  time.Time `json:"start_at" binding:"required"`
	EndAt    time.Time `json:"end_at" binding:"required"`
	AgeStart int       `json:"age_start" binding:"required"`
	AgeEnd   int       `json:"age_end" binding:"required"`
	Gender   []string  `json:"gender" binding:"required"`
	Country  []string  `json:"country" binding:"required"`
	Platform []string  `json:"platform" binding:"required"`
}

type CreateAdResponse struct {
	Response
	// Data id of the created ad
	Data string `json:"data"`
}

type AdService interface {
	CreateAd(ctx context.Context, ad *Ad) (adID string, er error)
	GetAds(ctx context.Context, req *GetAdRequest) ([]*Ad, int, error)
	// Subscribe to the redis stream
	Subscribe(offset int) error
	Restore() (version int, err error)
	Run() error
	Shutdown(ctx context.Context) error
}

type InMemoryStore interface {
	CreateAd(ad *Ad) (string, error)
	GetAds(req *GetAdRequest) ([]*Ad, int, error)
	// Restore the ads from the db, and return the highest version in the store
	CreateBatchAds(ads []*Ad) (version int, err error)
}
