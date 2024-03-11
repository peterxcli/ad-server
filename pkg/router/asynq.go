package router

import (
	"dcard-backend-2024/pkg/bootstrap"
)

func RegisterAsynqMux(app *bootstrap.Application, services *bootstrap.Services) {
	services.TaskService.RegisterTaskHandler(app.AsyncServerMux)
}
