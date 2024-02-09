package router

import (
	"bikefest/pkg/bootstrap"
	"bikefest/pkg/controller"
)

func RegisterPsychoTestRouter(app *bootstrap.Application, controller *controller.PsychoTestController) {
	r := app.Engine.Group("/psycho-test")
	r.GET("/type-create", controller.CreateType)
	r.GET("/type-addcount", controller.TypeAddCount)
	r.GET("/type-percentage", controller.CountTypePercentage)
}
