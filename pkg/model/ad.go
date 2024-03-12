package model

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

var (
	TypeEventDelete = "event:delete"
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

// GetValueByKey returns the value of the field with the given key.
// If the field is a slice, it returns a slice of interfaces.
// If the field is a single value, it returns a slice of length 1.
// You can also add custom logic for specific fields, but still return a slice of interfaces.
func (a *Ad) GetValueByKey(key string) ([]interface{}, error) {
	v := reflect.ValueOf(*a)
	fieldVal := v.FieldByName(key)

	if key == "Age" {
		slice := make([]interface{}, a.AgeEnd-a.AgeStart+1)
		for i := a.AgeStart; i <= a.AgeEnd; i++ {
			slice[i-a.AgeStart] = i
		}
		defaultVal := reflect.Zero(reflect.TypeOf(a.AgeStart)).Interface()
		slice = append(slice, defaultVal)
		slice = UniqueSlice(slice)
		return slice, nil
	} else if fieldVal.Kind() == reflect.Slice {
		length := fieldVal.Len()
		slice := make([]interface{}, length)
		for i := 0; i < length; i++ {
			slice[i] = fieldVal.Index(i).Interface()
		}
		defaultVal := reflect.Zero(fieldVal.Type().Elem()).Interface()
		slice = append(slice, defaultVal)
		slice = UniqueSlice(slice)
		return slice, nil
	} else {
		// If it's not a slice, wrap the value in a slice of interfaces.
		slice := make([]interface{}, 1)
		slice[0] = fieldVal.Interface()
		defaultVal := reflect.Zero(fieldVal.Type()).Interface()
		slice = append(slice, defaultVal)
		slice = UniqueSlice(slice)
		return slice, nil
	}
}

func UniqueSlice(slice []interface{}) []interface{} {
	uniqueElements := make(map[interface{}]struct{})
	var result []interface{}
	for _, element := range slice {
		if _, exists := uniqueElements[element]; !exists {
			uniqueElements[element] = struct{}{}
			result = append(result, element)
		}
	}
	return result
}

func (a Ad) GetNextIndexKey(currentKey string) string {
	switch currentKey {
	case "":
		return "Age"
	case "Age":
		return "Country"
	case "Country":
		return "Platform"
	case "Platform":
		return "Gender"
	default:
		return ""
	}
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

func (r *GetAdRequest) GetValueByKey(key string) (interface{}, error) {
	v := reflect.ValueOf(*r)
	fieldVal := v.FieldByName(key)

	if !fieldVal.IsValid() {
		return nil, fmt.Errorf("no such field: %s in obj", key)
	}

	if !fieldVal.CanInterface() {
		return nil, fmt.Errorf("cannot access unexported field: %s", key)
	}

	return fieldVal.Interface(), nil
}

type GetAdsPageResponse struct {
	Ads   []*Ad `json:"ads"`
	Total int   `json:"total"`
}

type AsynqDeletePayload struct {
	AdID string `json:"ad_id"`
}

func (a *AsynqDeletePayload) ToTask() (*asynq.Task, error) {
	payload, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEventDelete, payload), nil
}

func (a *AsynqDeletePayload) FromTask(task *asynq.Task) error {
	return json.Unmarshal(task.Payload(), a)
}

func (AsynqDeletePayload) TypeName() string {
	return TypeEventDelete
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
	DeleteAd(ctx context.Context, adID string) error
	// Subscribe to the redis stream
	Subscribe() error
	Restore() error
	Run() error
	Shutdown(ctx context.Context) error
}

type TaskService interface {
	HandleDeleteAd(ctx context.Context, t *asynq.Task) error
	RegisterTaskHandler(mux *asynq.ServeMux)
}

type InMemoryStore interface {
	CreateAd(ad *Ad) (string, error)
	GetAds(req *GetAdRequest) ([]*Ad, int, error)
	DeleteAd(adID string) error
	// Restore the ads from the db, and return the highest version in the store
	CreateBatchAds(ads []*Ad) (err error)
}
