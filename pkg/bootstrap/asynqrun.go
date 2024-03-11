package bootstrap

import (
	"log"

	"github.com/hibiken/asynq"
)

func RunAsynq(app *Application, services *Services, errorChan chan error) {
	mux := asynq.NewServeMux()

	services.TaskService.RegisterTaskHandler(mux)

	if err := app.AsynqServer.Run(mux); err != nil {
		log.Println(err)
		errorChan <- err
	}
}
