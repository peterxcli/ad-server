package worker

import (
	"github.com/hibiken/asynq"
)

func RegisterTaskHandler(mux *asynq.ServeMux, handler *EventTaskHandler) {
	mux.HandleFunc(TypeEventReminder, handler.HandleEventTask)
}
