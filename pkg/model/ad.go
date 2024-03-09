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
	AgeStart uint8          `gorm:"type:integer" json:"age_start"`
	AgeEnd   uint8          `gorm:"type:integer" json:"age_end"`
	Gender   pq.StringArray `gorm:"type:text[]" json:"gender"`
	Country  pq.StringArray `gorm:"type:text[]" json:"country"`
	Platform pq.StringArray `gorm:"type:text[]" json:"platform"`
	// Version, cant use sequence number, because the version is not continuous if we want to support update and delete
	Version   int        `gorm:"index" json:"version"`
	IsActive  bool       `gorm:"type:boolean; default:true" json:"-" default:"true"`
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
	// AgeStart <= Age <= AgeEnd
	Age      uint8  `form:"age" binding:"omitempty,gt=0"`
	Country  string `form:"country" binding:"omitempty,iso3166_1_alpha2"`
	Gender   string `form:"gender" binding:"omitempty,oneof=M F"`
	Platform string `form:"platform" binding:"omitempty,oneof=android ios web"`

	Offset int `form:"offset,default=0" binding:"min=0"`
	Limit  int `form:"limit,default=10" binding:"min=1,max=100"`
}

type GetAdsPageResponse struct {
	Ads   []*Ad `json:"ads"`
	Total int   `json:"total"`
}

type CreateAdRequest struct {
	Title    string     `json:"title" binding:"required,min=5,max=100"`
	Content  string     `json:"content" binding:"required"`
	StartAt  CustomTime `json:"start_at" binding:"required" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
	EndAt    CustomTime `json:"end_at" binding:"required,gtfield=StartAt" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
	AgeStart uint8      `json:"age_start" binding:"gtefield=AgeStart,lte=100" example:"18"`
	AgeEnd   uint8      `json:"age_end" binding:"required" example:"65"`
	Gender   []string   `json:"gender" binding:"required,dive,oneof=M F" example:"F"`
	Country  []string   `json:"country" binding:"required,dive,iso3166_1_alpha2" example:"TW"`
	Platform []string   `json:"platform" binding:"required,dive,oneof=android ios web" example:"ios"`
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
