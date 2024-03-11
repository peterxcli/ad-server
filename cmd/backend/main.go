package main

import (
	"dcard-backend-2024/docs"
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/router"
	"dcard-backend-2024/pkg/service"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

func SetUpSwagger(spec *swag.Spec, app *bootstrap.Application) {
	spec.BasePath = "/"
	spec.Host = fmt.Sprintf("%s:%d", "localhost", app.Env.Server.Port)
	spec.Schemes = []string{"http", "https"}
	spec.Title = "Dcard Internship Assignment 2024 - Advertisement Backend API"
	spec.Description = "This is the API document for the Dcard Internship Assignment 2024 - Advertisement Backend"
}

func SetUpAsynqMon(app *bootstrap.Application) {
	readonly := false
	h := asynqmon.New(asynqmon.Options{
		RootPath:     "/monitoring", // RootPath specifies the root for asynqmon app
		RedisConnOpt: asynq.RedisClientOpt{Addr: app.Cache.Options().Addr},
		ReadOnly:     readonly,
	})

	// Use Gin's Group function to create a route group with the specified prefix
	monitoringGroup := app.Engine.Group(h.RootPath())

	// Use the Gin.WrapH function to convert Asynqmon's http.Handler to a Gin-compatible handler
	// and register it to handle all routes under "/monitoring/"
	monitoringGroup.Any("/*action", gin.WrapH(h))
}

func ReverseProxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		director := func(req *http.Request) {
			// Copy the original request to retain headers and other attributes
			originalURL := *c.Request.URL

			// Set the scheme and host
			// Set req.URL.Scheme as "http" or "https" based on the protocol your target is using
			req.URL.Scheme = "http"
			req.URL.Host = c.Request.Host

			// If the suffix is '/docs', then we need to change it to '/swagger/index.html'
			// Otherwise, we need to substitute '/docs' with '/swagger'
			if originalURL.Path == "/docs/" {
				req.URL.Path = "/swagger/index.html"
			} else {
				req.URL.Path = "/swagger" + originalURL.Path[len("/docs"):]
			}
			fmt.Println(req.URL.Path)
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	// Init config
	app := bootstrap.App()

	// Init services
	// eventService := service.NewEventService(app.Conn, app.Cache)
	adService := service.NewAdService(
		app.Dispatcher,
		app.Conn,
		app.Cache,
		app.RedisLock,
		app.AsynqClient,
	)

	taskService := service.NewTaskService(adService)

	services := &bootstrap.Services{
		AdService:   adService,
		TaskService: taskService,
	}

	// Init routes
	router.RegisterRoutes(app, services)

	// Init asynq
	router.RegisterAsynqMux(app, services)

	// setup swagger
	// @securityDefinitions.apikey ApiKeyAuth
	// @in header
	// @name Authorization
	SetUpSwagger(docs.SwaggerInfo, app)

	// setup asynqmon
	SetUpAsynqMon(app)

	app.Engine.GET("/swagger/*any",
		ginSwagger.WrapHandler(
			swaggerfiles.Handler,
			ginSwagger.DeepLinking(true),
		),
	)
	app.Engine.GET("/docs/*any", ReverseProxy())

	app.Run(services)
}
