package router

import (
	"bikefest/pkg/bootstrap"
	"bikefest/pkg/controller"
)

func RegisterOAuthRouter(app *bootstrap.Application, controller *controller.OAuthController) {
	lineRouter := app.Engine.Group("/line-login")
	lineRouter.GET("/auth", controller.LineLogin)
	lineRouter.GET("/callback", controller.LineLoginCallback)
}
