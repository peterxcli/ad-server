package main

import (
	"bikefest/pkg/bootstrap"
	"bikefest/pkg/model"
	"log"
)

func main() {
	env := bootstrap.NewEnv()
	db := bootstrap.NewDB(env)
	err := db.AutoMigrate(
		&model.Event{},
		&model.User{},
		&model.PsychoTest{},
	)
	if err != nil {
		log.Fatal(err)
	}
}
