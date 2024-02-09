package router

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/controller"
)

func RegisterOAuthRouter(app *bootstrap.Application, controller *controller.OAuthController) {
	lineRouter := app.Engine.Group("/line-login")
	lineRouter.GET("/auth", controller.LineLogin)
	lineRouter.GET("/callback", controller.LineLoginCallback)
}
