package router

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/controller"
)

func RegisterPsychoTestRouter(app *bootstrap.Application, controller *controller.PsychoTestController) {
	r := app.Engine.Group("/psycho-test")
	r.GET("/type-create", controller.CreateType)
	r.GET("/type-addcount", controller.TypeAddCount)
	r.GET("/type-percentage", controller.CountTypePercentage)
}
