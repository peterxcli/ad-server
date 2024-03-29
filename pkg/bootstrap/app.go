package bootstrap

import (
	"context"
	"dcard-backend-2024/pkg/dispatcher"
	"dcard-backend-2024/pkg/inmem"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bsm/redislock"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AppOpts func(app *Application)

type Application struct {
	Env            *Env
	Conn           *gorm.DB
	Cache          *redis.Client
	AsynqClient    *asynq.Client
	AsynqServer    *asynq.Server
	Engine         *gin.Engine
	RedisLock      *redislock.Client
	Dispatcher     *dispatcher.Dispatcher
	AsyncServerMux *asynq.ServeMux
}

func App(opts ...AppOpts) *Application {
	env := NewEnv()
	db := NewDB(env)
	cache := NewCache(env)
	asynqClient := NewAsynqClient(env)
	asynqServer := NewAsynqServer(env)
	redisLock := NewRdLock(cache)
	engine := gin.New()
	adInMemStore := inmem.NewInMemoryStore()
	dispatcher := dispatcher.NewDispatcher(adInMemStore)
	asynqServerMux := asynq.NewServeMux()

	// Set timezone
	tz, err := time.LoadLocation(env.Server.TimeZone)
	if err != nil {
		log.Fatal(err)
	}
	time.Local = tz

	app := &Application{
		Env:            env,
		Conn:           db,
		Cache:          cache,
		Engine:         engine,
		RedisLock:      redisLock,
		Dispatcher:     dispatcher,
		AsynqClient:    asynqClient,
		AsynqServer:    asynqServer,
		AsyncServerMux: asynqServerMux,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

type Mocks struct {
	CacheMock redismock.ClientMock
	DBMock    sqlmock.Sqlmock
}

func NewTestApp(opts ...AppOpts) (*Application, *Mocks) {
	env := NewEnv()
	db, dbMock := NewMockDB()
	cache, cacheMock := NewMockCache()
	redisLock := NewRdLock(cache)
	asynqClient := NewAsynqClient(env)
	asynqServer := NewAsynqServer(env)
	engine := gin.Default()
	gin.SetMode(gin.TestMode)
	adInMemStore := inmem.NewInMemoryStore()
	dispatcher := dispatcher.NewDispatcher(adInMemStore)
	asynqServerMux := asynq.NewServeMux()

	// Set timezone
	tz, err := time.LoadLocation(env.Server.TimeZone)
	if err != nil {
		log.Fatal(err)
	}
	time.Local = tz

	app := &Application{
		Env:            env,
		Conn:           db,
		Cache:          cache,
		Engine:         engine,
		RedisLock:      redisLock,
		Dispatcher:     dispatcher,
		AsynqClient:    asynqClient,
		AsynqServer:    asynqServer,
		AsyncServerMux: asynqServerMux,
	}

	mocks := &Mocks{
		CacheMock: cacheMock,
		DBMock:    dbMock,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app, mocks
}

// Run the application
func (app *Application) Run(services *Services) {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Env.Server.Port),
		Handler: app.Engine,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Println("Starting Asynq Worker...")
		if err := app.AsynqServer.Run(app.AsyncServerMux); err != nil {
			log.Println(err)
			serverErrors <- err
		}
	}()

	go func() {
		log.Printf("Background Services are running...")
		for err := range services.Run() {
			log.Printf("Error from background service: %v\n", err)
			serverErrors <- err
		}
	}()

	go func() {
		log.Printf("Server is running on port %d", app.Env.Server.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error from server: %v\n", err)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		log.Println("Shutting down the background services and server...")
		err = services.Shutdown(ctx)
		if err != nil {
			log.Printf("Could not stop services: %v\n", err)
		}
		os.Exit(1)
	case <-shutdown:
		log.Println("Shutting down the server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Could not stop server gracefully: %v", err)
			err = srv.Close()
			if err != nil {
				log.Fatalf("Could not stop http server: %v", err)
			}
		}

		err = services.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Could not stop ad service: %v", err)
		}
	}
}
