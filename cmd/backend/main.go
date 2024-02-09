package main

import (
	"bikefest/docs"
	"bikefest/pkg/bootstrap"
	"bikefest/pkg/router"
	"bikefest/pkg/service"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

func SetUpSwagger(spec *swag.Spec, app *bootstrap.Application) {
	spec.BasePath = "/"
	spec.Host = fmt.Sprintf("%s:%d", "localhost", app.Env.Server.Port)
	spec.Schemes = []string{"http", "https"}
	spec.Title = "NCKU Bike Festival 2024 Official Website Backend API"
	spec.Description = "This is the official backend API for Bike Festival 2024 Official Website"
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

func SetUpAsynqMon(app *bootstrap.Application) {
	h := asynqmon.New(asynqmon.Options{
		RootPath:     "/monitoring", // RootPath specifies the root for asynqmon app
		RedisConnOpt: asynq.RedisClientOpt{Addr: app.Cache.Options().Addr},
	})

	// Use Gin's Group function to create a route group with the specified prefix
	monitoringGroup := app.Engine.Group(h.RootPath())

	// Use the Gin.WrapH function to convert Asynqmon's http.Handler to a Gin-compatible handler
	// and register it to handle all routes under "/monitoring/"
	monitoringGroup.Any("/*action", gin.WrapH(h))
}

func main() {
	// Init config
	app := bootstrap.App()

	// Init services
	userService := service.NewUserService(app.Conn, app.Cache)
	eventService := service.NewEventService(app.Conn, app.Cache)
	asynqService := service.NewAsynqService(app.AsynqClient, app.AsyncqInspector, app.Env)

	services := &router.Services{
		UserService:  userService,
		EventService: eventService,
		AsynqService: asynqService,
	}

	// Init routes
	router.RegisterRoutes(app, services)

	// setup swagger
	// @securityDefinitions.apikey ApiKeyAuth
	// @in header
	// @name Authorization
	SetUpSwagger(docs.SwaggerInfo, app)
	SetUpAsynqMon(app)
	app.Engine.GET("/swagger/*any",
		ginSwagger.WrapHandler(
			swaggerfiles.Handler,
			ginSwagger.DeepLinking(true),
		),
	)
	app.Engine.GET("/docs/*any", ReverseProxy())

	app.Run()
}
