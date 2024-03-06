package router

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/controller"
)

func RegisterAdRouter(app *bootstrap.Application, controller *controller.AdController) {
	r := app.Engine.Group("/api/v1/ad")

	r.POST("", controller.CreateAd)
	r.GET("", controller.GetAd)
}
