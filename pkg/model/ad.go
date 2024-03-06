package model

import (
	"context"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Ad struct {
	ID       uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Title    string         `gorm:"type:text" json:"title"`
	Content  string         `gorm:"type:text" json:"content"`
	StartAt  CustomTime     `gorm:"type:timestamp" json:"start_at" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
	EndAt    CustomTime     `gorm:"type:timestamp" json:"end_at" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
	AgeStart int            `gorm:"type:integer" json:"age_start"`
	AgeEnd   int            `gorm:"type:integer" json:"age_end"`
	Gender   pq.StringArray `gorm:"type:text[]" json:"gender"`
	Country  pq.StringArray `gorm:"type:text[]" json:"country"`
	Platform pq.StringArray `gorm:"type:text[]" json:"platform"`
	// Version, cant use sequence number, because the version is not continuous if we want to support update and delete
	Version   int        `gorm:"index" json:"version"`
	CreatedAt CustomTime `gorm:"type:timestamp" json:"created_at"`
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
	Title    string     `json:"title" binding:"required"`
	Content  string     `json:"content" binding:"required"`
	StartAt  CustomTime `json:"start_at" binding:"required" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
	EndAt    CustomTime `json:"end_at" binding:"required" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
	AgeStart int        `json:"age_start" binding:"required" example:"18"`
	AgeEnd   int        `json:"age_end" binding:"required" example:"65"`
	Gender   []string   `json:"gender" binding:"required" example:"F"`
	Country  []string   `json:"country" binding:"required" example:"US"`
	Platform []string   `json:"platform" binding:"required" example:"ios"`
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
	Subscribe() error
	Restore() error
	Run() error
	Shutdown(ctx context.Context) error
}

type InMemoryStore interface {
	CreateAd(ad *Ad) (string, error)
	GetAds(req *GetAdRequest) ([]*Ad, int, error)
	// Restore the ads from the db, and return the highest version in the store
	CreateBatchAds(ads []*Ad) (err error)
}
