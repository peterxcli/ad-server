package main

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/model"
	"log"
)

func main() {
	env := bootstrap.NewEnv()
	db := bootstrap.NewDB(env)
	err := db.AutoMigrate(
		&model.Event{},
	)
	if err != nil {
		log.Fatal(err)
	}
}
