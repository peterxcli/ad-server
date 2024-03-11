package bootstrap

import (
	"context"
	"dcard-backend-2024/pkg/model"
)

type Services struct {
	AdService   model.AdService
	TaskService model.TaskService
}

func (s *Services) Run() chan error {
	errCh := make(chan error)
	go func() {
		if err := s.AdService.Run(); err != nil {
			errCh <- err
		}
	}()
	return errCh
}

func (s *Services) Shutdown(ctx context.Context) error {
	if err := s.AdService.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
