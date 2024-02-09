package worker

import (
	"bikefest/pkg/line_utils"
	"bikefest/pkg/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/redis/go-redis/v9"
	"log"
)

const (
	TypeEventReminder = "reminder"
)

// Task payload for any event notification related tasks.
type eventNotificationPayload struct {
	UserID  string
	EventID string
}

type EventTaskHandler struct {
	cache    *redis.Client
	eventSvc model.EventService
	bot      *linebot.Client
}

func (eth *EventTaskHandler) HandleEventTask(ctx context.Context, t *asynq.Task) error {
	var p eventNotificationPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// store and read event in a ttlmap and as L0 cache
	event, err := eth.eventSvc.FindByID(ctx, p.EventID)

	// convert event.EventDetail to model.EventDetails, the event.EventDetail is stringed json
	eventDetails := model.EventDetails{}
	err = json.Unmarshal([]byte(*event.EventDetail), &eventDetails)
	if err != nil {
		log.Println(err)
		return err
	}

	var message linebot.SendingMessage
	flexContainer, err := line_utils.CreateFlexMessage(&eventDetails)
	if err != nil {
		message = linebot.NewTextMessage(fmt.Sprintf("Event: %s, 即將開始", eventDetails.Name))
	} else {
		message = linebot.NewFlexMessage(fmt.Sprintf("Event: %s, 即將開始", eventDetails.Name), *flexContainer)
	}

	//message := linebot.NewTextMessage(fmt.Sprintf("Hello, Event %s is going to start within 30 minutes!!!", p.EventID))

	_, err = eth.bot.PushMessage(p.UserID, message).Do()
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func NewEventTaskHandler(cache *redis.Client, eventSvc model.EventService, bot *linebot.Client) *EventTaskHandler {
	return &EventTaskHandler{cache: cache, eventSvc: eventSvc, bot: bot}
}
