package main

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/router"
	"dcard-backend-2024/pkg/service"
	"dcard-backend-2024/pkg/worker"
	"log"

	"github.com/hibiken/asynq"
	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {
	// init config
	app := bootstrap.App()

	// init services
	eventService := service.NewEventService(app.Conn, app.Cache)

	services := &router.Services{
		EventService: eventService,
	}

	// Create a new LINE SDK client.
	bot, err := linebot.New(
		app.Env.Line.ChannelSecret,
		app.Env.Line.ChannelToken,
	)
	if err != nil {
		panic(err)
	}

	// Create an asynq server.
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: app.Cache.Options().Addr},
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()

	handler := worker.NewEventTaskHandler(
		app.Cache,
		services.EventService,
		bot,
	)

	worker.RegisterTaskHandler(mux, handler)

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
