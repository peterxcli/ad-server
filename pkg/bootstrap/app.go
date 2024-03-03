package bootstrap

import (
	"context"
	"dcard-backend-2024/pkg/inmem"
	"dcard-backend-2024/pkg/runner"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/bsm/redislock"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AppOpts func(app *Application)

type Application struct {
	Env       *Env
	Conn      *gorm.DB
	Cache     *redis.Client
	Engine    *gin.Engine
	RedisLock *redislock.Client
	Runner    *runner.Runner
}

func App(opts ...AppOpts) *Application {
	env := NewEnv()
	db := NewDB(env)
	cache := NewCache(env)
	redisLock := NewRdLock(cache)
	engine := gin.Default()
	adInMemStore := inmem.NewInMemoryStore()
	runner := runner.NewRunner(adInMemStore)

	// Set timezone
	tz, err := time.LoadLocation(env.Server.TimeZone)
	if err != nil {
		log.Fatal(err)
	}
	time.Local = tz

	app := &Application{
		Env:       env,
		Conn:      db,
		Cache:     cache,
		Engine:    engine,
		RedisLock: redisLock,
		Runner:    runner,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Run the application
func (app *Application) Run() {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Env.Server.Port),
		Handler: app.Engine,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server is running on port %d", app.Env.Server.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Error starting server: %v", err)

	case <-shutdown:
		log.Println("Shutting down the server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Could not stop server gracefully: %v", err)
			err = srv.Close()
			if err != nil {
				log.Fatalf("Could not stop http server: %v", err)
			}
		}
	}
}
