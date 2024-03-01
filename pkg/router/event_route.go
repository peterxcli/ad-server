package router

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/controller"
)

func RegisterEventRouter(app *bootstrap.Application, controller *controller.EventController) {
	r := app.Engine.Group("/events")

	r.GET("", controller.GetAllEvent)
	r.GET("/:id", controller.GetEventByID)
	r.GET("/test-store-all", controller.StoreAllEvent)
}
