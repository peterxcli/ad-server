package service

import (
	"context"
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/runner"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bsm/redislock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func boot() (app *bootstrap.Application, services *bootstrap.Services, mocks *bootstrap.Mocks) {
	gin.SetMode(gin.TestMode)
	app, mocks = bootstrap.NewTestApp()
	// mocks.DBMock.ExpectExec("SELECT count(*) FROM information_schema.tables")
	mocks.DBMock.ExpectQuery("SELECT count\\(\\*\\) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA\\(\\) AND table_name = \\$1 AND table_type = \\$2").
		WithArgs("ads", "BASE TABLE").WillReturnRows(mocks.DBMock.NewRows([]string{"count"}).AddRow(1))
	mocks.DBMock.ExpectQuery("SELECT CURRENT_DATABASE()").WithoutArgs().WillReturnRows(mocks.DBMock.NewRows([]string{"current_database"}).AddRow("dcard"))
	// mocks.DBMock.ExpectBegin()
	// mocks.DBMock.ExpectExec("CREATE TABLE")
	app.Conn.AutoMigrate(&model.Ad{})
	adService := NewAdService(
		app.Runner,
		app.Conn,
		app.Cache,
		app.RedisLock,
	)
	services = &bootstrap.Services{
		AdService: adService,
	}
	mocks.DBMock.ExpectBegin()
	mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
		WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
	mocks.DBMock.ExpectQuery("SELECT (.+) FROM \"ads\"").
		WillReturnRows(mocks.DBMock.NewRows([]string{"id", "title", "content", "start_at", "end_at", "age_start", "age_end"}))
	mocks.DBMock.ExpectCommit()
	return
}

var (
	app        *bootstrap.Application
	adServices *AdService
	mocks      *bootstrap.Mocks
	lockKey    = "test"
	adStream   = "test"
)

func init() {
	gin.SetMode(gin.TestMode)
	app, _, mocks = boot()
	// adServices = &AdService{
	// 	shutdown: atomic.Bool{},
	// 	runner:   app.Runner,
	// 	db:       app.Conn,
	// 	redis:    app.Cache,
	// 	locker:   app.RedisLock,
	// 	lockKey:  lockKey,
	// 	adStream: adStream,
	// 	Version:  0,
	// }

	mocks.DBMock.MatchExpectationsInOrder(false)
	mocks.CacheMock.MatchExpectationsInOrder(false)
	// adServices.Run()
}
