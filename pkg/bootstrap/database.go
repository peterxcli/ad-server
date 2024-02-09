package bootstrap

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type DBEnv struct {
	Kind     string `env:"KIND"`
	Host     string `env:"HOST"`
	Port     uint   `env:"PORT"`
	Username string `env:"USERNAME"`
	Password string `env:"PASSWORD"`
	Database string `env:"DATABASE"`
}

func (env *DBEnv) Dialect(kind string) gorm.Dialector {
	switch kind {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", env.Username, env.Password, env.Host, env.Port, env.Database)
		return mysql.Open(dsn)
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s TimeZone=Asia/Taipei", env.Host, env.Port, env.Username, env.Database, env.Password)
		return postgres.Open(dsn)
	case "mssql", "sqlserver":
		dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", env.Username, env.Password, env.Host, env.Port, env.Database)
		return sqlserver.Open(dsn)
	default:
		panic("Unsupported database kind")
	}
}

func NewDB(env *Env) *gorm.DB {
	db, err := gorm.Open(env.DB.Dialect(env.DB.Kind), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database handle: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	return db
}
