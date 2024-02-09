package config

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
)

type ConfigurationOpts func(app *Configuration)

type Configuration struct {
	Setting *Setting
	Conn    *gorm.DB
	Engine  *gin.Engine
}

func App(filename string, opts ...ConfigurationOpts) *Configuration {
	setting, err := NewSetting(filename)
	if err != nil {
		log.Fatal(err)
	}
	db := NewDatabaseConnection(setting)
	engine := gin.Default()

	// Set timezone
	//tz, err := time.LoadLocation(env.Server.TimeZone)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//time.Local = tz

	app := &Configuration{
		Setting: setting,
		Conn:    db,
		Engine:  engine,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}
