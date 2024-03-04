package router

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/controller"
	"dcard-backend-2024/pkg/middleware"
)


func RegisterRoutes(app *bootstrap.Application, services *bootstrap.Services) {
	// Register Global Middleware
	cors := middleware.CORSMiddleware()
	app.Engine.Use(cors)

	// // Register Event Routes
	// eventController := controller.NewEventController(services.EventService, services.AsynqService)
	// RegisterEventRouter(app, eventController)

	// Register Ad Routes
	adController := controller.NewAdController(services.AdService)
	RegisterAdRouter(app, adController)
}
