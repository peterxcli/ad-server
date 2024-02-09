package router

import (
	"bikefest/pkg/bootstrap"
	"bikefest/pkg/controller"
	"bikefest/pkg/middleware"
	"bikefest/pkg/model"
)

type Services struct {
	UserService  model.UserService
	EventService model.EventService
	AsynqService model.AsynqNotificationService
}

func RegisterRoutes(app *bootstrap.Application, services *Services) {
	// Register Global Middleware
	cors := middleware.CORSMiddleware()
	app.Engine.Use(cors)

	// Register User Routes
	userController := controller.NewUserController(services.UserService, services.EventService, services.AsynqService, app.Env)
	RegisterUserRoutes(app, userController)

	// Register Event Routes
	eventController := controller.NewEventController(services.EventService, services.AsynqService)
	RegisterEventRouter(app, eventController)

	// Register PsychoTest Routes
	psychoTestController := controller.NewPsychoTestController(app.Conn)
	RegisterPsychoTestRouter(app, psychoTestController)

	// Register OAuth Routes
	oauthController := controller.NewOAuthController(app.LineSocialClient, app.Env, services.UserService)
	RegisterOAuthRouter(app, oauthController)
}
