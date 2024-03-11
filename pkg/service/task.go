package service

import (
	"context"
	"dcard-backend-2024/pkg/model"
	"encoding/json"

	"github.com/hibiken/asynq"
)

type TaskService struct {
	adService model.AdService
}

// HandleDeleteAd implements model.TaskService.
func (svc *TaskService) HandleDeleteAd(ctx context.Context, t *asynq.Task) error {
	var deletePayload model.AsynqDeletePayload
	if err := json.Unmarshal(t.Payload(), &deletePayload); err != nil {
		return err
	}
	return svc.adService.DeleteAd(ctx, deletePayload.AdID)
}

func (svc *TaskService) RegisterTaskHandler(mux *asynq.ServeMux) {
	mux.HandleFunc(model.AsynqDeletePayload{}.TypeName(), svc.HandleDeleteAd)
}

func NewTaskService(adService model.AdService) model.TaskService {
	return &TaskService{
		adService: adService,
	}
}
