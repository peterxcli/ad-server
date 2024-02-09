package service

import (
	"bikefest/pkg/bootstrap"
	"bikefest/pkg/model"
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// A list of task types.
const (
	TypeEventReminder = "reminder"
)

// Task payload for any event notification related tasks.
type eventNotificationPayload struct {
	UserID  string
	EventID string
}

type AsynqServiceImpl struct {
	client    *asynq.Client
	inspector *asynq.Inspector
	env       *bootstrap.Env
}

func newEventNotification(userId, eventId string) (*asynq.Task, error) {
	payload, err := json.Marshal(eventNotificationPayload{UserID: userId, EventID: eventId})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeEventReminder, payload), nil
}

// DeleteEventNotification deletes the task from the queue.
// the taskID is the userID + eventID
func (as *AsynqServiceImpl) DeleteEventNotification(ctx context.Context, taskID string) error {
	err := as.inspector.DeleteTask("default", taskID)
	switch {
	case errors.Is(err, asynq.ErrTaskNotFound):
		log.Printf("Task with ID %q not found", taskID)
		return nil
	case err != nil:
		return err
	default:
		return nil
	}
}

func (as *AsynqServiceImpl) EnqueueEventNotification(ctx context.Context, userID, eventID, eventStartTime string) error {
	t, err := newEventNotification(userID, eventID)
	if err != nil {
		return err
	}

	location, _ := time.LoadLocation(as.env.Server.TimeZone)
	//TODO: currently we only set the process time 30 minutes before the event start time
	processTime, _ := time.ParseInLocation(model.EventTimeLayout, eventStartTime, location)
	processTime = processTime.Add(-time.Minute * 30)

	info, err := as.client.Enqueue(t, asynq.ProcessAt(processTime), asynq.TaskID(userID+eventID))
	switch {
	case errors.Is(err, asynq.ErrTaskIDConflict):
		log.Printf("Task with ID %q already exists", userID+eventID)
		return nil
	case err != nil:
		return err
	}
	log.Printf(" [*] Successfully enqueued task: %+v\nThe task should be executed at %s", info, processTime.String())
	return nil
}

func NewAsynqService(client *asynq.Client, inspector *asynq.Inspector, env *bootstrap.Env) model.AsynqNotificationService {
	return &AsynqServiceImpl{
		client:    client,
		inspector: inspector,
		env:       env,
	}
}
