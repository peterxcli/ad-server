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

func TestAdService_Shutdown(t *testing.T) {
	type fields struct {
		runner     *runner.Runner
		db         *gorm.DB
		redis      *redis.Client
		locker     *redislock.Client
		lockKey    string
		adStream   string
		onShutdown []func()
		Version    int
	}
	type args struct {
		timeout time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test shutdown",
			fields: fields{
				runner:     app.Runner,
				db:         app.Conn,
				redis:      app.Cache,
				locker:     app.RedisLock,
				lockKey:    lockKey + uuid.New().String(),
				adStream:   adStream + uuid.New().String(),
				onShutdown: []func(){},
				Version:    0,
			},
			args: args{
				timeout: 5 * time.Second,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AdService{
				shutdown:   atomic.Bool{},
				runner:     tt.fields.runner,
				db:         tt.fields.db,
				redis:      tt.fields.redis,
				locker:     tt.fields.locker,
				lockKey:    tt.fields.lockKey,
				adStream:   tt.fields.adStream,
				onShutdown: tt.fields.onShutdown,
				Version:    tt.fields.Version,
			}
			go a.Run()
			ctx, cancel := context.WithTimeout(context.Background(), tt.args.timeout)
			for {
				if a.runner.IsRunning() && a.onShutdownNum() == 2 {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatalf("runner did not start within %v", tt.args.timeout)
				case <-time.After(time.Millisecond * 100):
				}
			}
			cancel()
			ctx, cancel = context.WithTimeout(context.Background(), tt.args.timeout)
			defer cancel()
			if err := a.Shutdown(ctx); (err != nil) != tt.wantErr {
				t.Errorf("AdService.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
			// print the context pass time
			deadline, ok := ctx.Deadline()
			assert.True(t, ok)
			t.Log(time.Duration(time.Second*5 - (deadline.Sub(time.Now()))))
		})
	}
}
