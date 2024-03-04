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
		app.Runner,
		app.Conn,
		app.Cache,
		app.RedisLock,
	)

	services := &bootstrap.Services{
		AdService: adService,
	}

	// Init routes
	router.RegisterRoutes(app, services)

	// setup swagger
	// @securityDefinitions.apikey ApiKeyAuth
	// @in header
	// @name Authorization
	SetUpSwagger(docs.SwaggerInfo, app)

	app.Engine.GET("/swagger/*any",
		ginSwagger.WrapHandler(
			swaggerfiles.Handler,
			ginSwagger.DeepLinking(true),
		),
	)
	app.Engine.GET("/docs/*any", ReverseProxy())

	app.Run(services)
}
