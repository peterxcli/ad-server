package model

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

var (
	EventTimeLayout = "2006/01/02 15:04"
	EventCacheKey   = "Event:"
)

type Event struct {
	gorm.Model
	// the event id is defne at the frontend, if frontend don't have event id, the event id would be calculated by the hash of event detail and event time
	ID             *string    `gorm:"type:varchar(36);primary_key" json:"id" redis:"id"`
	EventTimeStart *time.Time `gorm:"type:timestamp" json:"event_time_start" redis:"event_time_start"`
	EventTimeEnd   *time.Time `gorm:"type:timestamp" json:"event_time_end" redis:"event_time_end"`
	// the `EventDetail` field store the event detail in json format, this would be parsed when send to line message API
	EventDetail *string `gorm:"type:varchar(1024)" json:"event_detail" redis:"event_detail"`
}

type EventCache struct {
	ID             string    `json:"id" redis:"id"`
	EventTimeStart time.Time `json:"event_time_start" redis:"event_time_start"`
	EventTimeEnd   time.Time `json:"event_time_end" redis:"event_time_end"`
	EventDetail    string    `json:"event_detail" redis:"event_detail"`
	CreatedAt      time.Time `json:"created_at" redis:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" redis:"updated_at"`
}

func (e *Event) BeforeCreate(*gorm.DB) error {
	if e.ID == nil {
		uuidStr := uuid.New().String()
		e.ID = &uuidStr
	}
	return nil
}

//	{
//	   "id": "1",
//	   "name": "企管系-企鵝管理員",
//	   "activity": "科系博覽",
//	   "project": "科系體驗坊",
//	   "description": "好冷好冷，住在冰屋的小企鵝要沒食物吃了，整天在家就是吃飯睡覺打咚咚，想說去買個彩券發家致富結果輸到脫褲，口袋空空的企鵝透過SWOT分析發現自己很適合管理，便應徵到位於北極的南極股份公司的管理員，透過成為企鵝管理員發大財！你也想了解發大財的秘訣嗎？那就快來參加企管的科系體驗坊吧！",
//	   "date": "3/2",
//	   "startTime": "09:00",
//	   "endTime": "10:20",
//	   "location": "小西門砲台側",
//	   "host": "單車節學術部",
//	   "link": "https://kktix.com/events/17th-2/registrations/new"
//	 }
type EventDetails struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Activity    string `json:"activity"`
	Project     string `json:"project"`
	Description string `json:"description"`
	Date        string `json:"date"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Location    string `json:"location"`
	Host        string `json:"host"`
	Link        string `json:"link"`
}

func CaculateEventID(event *Event) (string, error) {
	eventMap := make(map[string]interface{})
	eventMap["event_time_start"] = event.EventTimeStart
	eventMap["event_time_end"] = event.EventTimeEnd
	eventMap["event_detail"] = event.EventDetail

	// stringfy the event map and calculate the hash
	eventJson, err := json.Marshal(eventMap)
	if err != nil {
		return "", err
	}
	return uuid.NewSHA1(uuid.Nil, eventJson).String(), nil
}

type CreateEventRequest struct {
	ID             *string `json:"id"`
	EventTimeStart string  `json:"event_time_start" example:"2021/01/01 00:00"`
	EventTimeEnd   string  `json:"event_time_end" example:"2021/01/01 00:00"`
	EventDetail    *string `json:"event_detail" example:"{\"title\":\"test event\",\"description\":\"test event description\"}"`
}

type EventResponse struct {
	Msg  string `json:"msg"`
	Data *Event `json:"data"`
}

type EventListResponse struct {
	Msg  string   `json:"msg"`
	Data []*Event `json:"data"`
}

type EventService interface {
	FindAll(ctx context.Context, page, limit int64) ([]*Event, error)
	FindByID(ctx context.Context, id string) (*Event, error)
	Store(ctx context.Context, event *Event) error
	Update(ctx context.Context, event *Event) (rowAffected int64, err error)
	Delete(ctx context.Context, event *Event) (rowAffected int64, err error)
	StoreAll(ctx context.Context, events []*Event) error
}

type AsynqNotificationService interface {
	EnqueueEventNotification(ctx context.Context, userID, eventID, eventStartTime string) error
	DeleteEventNotification(ctx context.Context, TaskID string) error
	// TODO: delete event notification by event id
}
