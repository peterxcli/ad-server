package service

import (
	"context"
	"database/sql/driver"
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/inmem"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/runner"
	"encoding/json"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bsm/redislock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

type AnyString struct{}

func (a AnyString) Match(v driver.Value) bool {
	_, ok := v.(string)
	return ok
}

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
		app.AsynqClient,
	)
	services = &bootstrap.Services{
		AdService: adService,
	}
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

	mocks.DBMock.MatchExpectationsInOrder(false)
	mocks.CacheMock.MatchExpectationsInOrder(false)
}

func TestAdService_storeAndPublishWithLock(t *testing.T) {
	type fields struct {
		runner   *runner.Runner
		db       *gorm.DB
		redis    *redis.Client
		locker   *redislock.Client
		lockKey  string
		adStream string
	}
	type args struct {
		ctx       context.Context
		ad        *model.Ad
		requestID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test store and publish with lock",
			fields: fields{
				runner:   app.Runner,
				db:       app.Conn,
				redis:    app.Cache,
				locker:   app.RedisLock,
				lockKey:  lockKey + uuid.New().String(),
				adStream: adStream + uuid.New().String(),
			},
			args: args{
				ctx: context.Background(),
				ad: &model.Ad{
					ID:       uuid.New(),
					Title:    "test",
					Content:  "test",
					StartAt:  model.CustomTime(time.Now()),
					EndAt:    model.CustomTime(time.Now()),
					AgeStart: 0,
					AgeEnd:   100,
					Gender:   []string{"M"},
					Country:  []string{"TW"},
					Platform: []string{"ios"},
					Version:  1,
				},
				requestID: uuid.New().String(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AdService{
				runner:   tt.fields.runner,
				db:       tt.fields.db,
				redis:    tt.fields.redis,
				locker:   tt.fields.locker,
				lockKey:  tt.fields.lockKey,
				adStream: tt.fields.adStream,
			}
			mocks.CacheMock.Regexp().ExpectEvalSha(".", []string{tt.fields.lockKey}, ".", ".", ".").SetVal(".")
			mocks.DBMock.ExpectBegin()
			mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
				WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
			mocks.DBMock.ExpectExec("^INSERT INTO \"ads\".+$").
				WithArgs(
					AnyString{},
					tt.args.ad.Title,
					tt.args.ad.Content,
					AnyTime{},
					AnyTime{},
					tt.args.ad.AgeStart,
					tt.args.ad.AgeEnd,
					pq.StringArray(tt.args.ad.Gender),
					pq.StringArray(tt.args.ad.Country),
					pq.StringArray(tt.args.ad.Platform),
					tt.args.ad.Version,
					true,
					AnyTime{},
				).WillReturnResult(sqlmock.NewResult(1, 1))
			mocks.DBMock.ExpectCommit()
			requestBytes, err := json.Marshal(runner.CreateAdRequest{
				Request: runner.Request{RequestID: tt.args.requestID},
				Ad:      tt.args.ad,
			})
			assert.Nil(t, err)
			mocks.CacheMock.CustomMatch(func(expected, actual []interface{}) error {
				return nil
			}).ExpectXAdd(&redis.XAddArgs{
				Stream:     tt.fields.adStream,
				NoMkStream: false,
				Approx:     false,
				MaxLen:     100000,
				Values:     []interface{}{"ad", string(requestBytes)},
				ID:         fmt.Sprintf("0-%d", tt.args.ad.Version),
			}).SetVal(fmt.Sprintf("0-%d", tt.args.ad.Version))
			mocks.CacheMock.CustomMatch(func(expected, actual []interface{}) error {
				return nil
			}).ExpectEvalSha(".", []string{tt.fields.lockKey}, ".").SetVal(".")
			if err := a.storeAndPublishWithLock(tt.args.ctx, tt.args.ad, tt.args.requestID); (err != nil) != tt.wantErr {
				t.Errorf("AdService.storeAndPublishWithLock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAdService_CreateAd(t *testing.T) {
	type fields struct {
		runner   *runner.Runner
		db       *gorm.DB
		redis    *redis.Client
		locker   *redislock.Client
		lockKey  string
		adStream string
	}
	type args struct {
		ctx context.Context
		ad  *model.Ad
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantAdID string
		wantErr  bool
	}{
		{
			name: "test store and publish with lock",
			fields: fields{
				runner:   app.Runner,
				db:       app.Conn,
				redis:    app.Cache,
				locker:   app.RedisLock,
				lockKey:  lockKey + uuid.New().String(),
				adStream: adStream + uuid.New().String(),
			},
			args: args{
				ctx: context.Background(),
				ad: &model.Ad{
					ID:       uuid.New(),
					Title:    "test",
					Content:  "test",
					StartAt:  model.CustomTime(time.Now()),
					EndAt:    model.CustomTime(time.Now()),
					AgeStart: 0,
					AgeEnd:   100,
					Gender:   []string{"M"},
					Country:  []string{"TW"},
					Platform: []string{"ios"},
					Version:  1,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AdService{
				runner:   tt.fields.runner,
				db:       tt.fields.db,
				redis:    tt.fields.redis,
				locker:   tt.fields.locker,
				lockKey:  tt.fields.lockKey,
				adStream: tt.fields.adStream,
			}
			mocks.DBMock.ExpectBegin()
			mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
				WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
			mocks.DBMock.ExpectQuery("SELECT (.+) FROM \"ads\"").
				WillReturnRows(mocks.DBMock.NewRows([]string{"id", "title", "content", "start_at", "end_at", "age_start", "age_end"}))
			mocks.DBMock.ExpectCommit()
			go a.Run()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			for {
				if a.runner.IsRunning() && a.onShutdownNum() == 2 {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatalf("service did not start within %v", 5*time.Second)
				case <-time.After(time.Millisecond * 100):
				}
			}
			cancel()
			mocks.CacheMock.Regexp().ExpectEvalSha(".", []string{tt.fields.lockKey}, ".", ".", ".").SetVal(".")
			mocks.DBMock.ExpectBegin()
			mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
				WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
			mocks.DBMock.ExpectExec("^INSERT INTO \"ads\".+$").
				WithArgs(
					AnyString{},
					tt.args.ad.Title,
					tt.args.ad.Content,
					AnyTime{},
					AnyTime{},
					tt.args.ad.AgeStart,
					tt.args.ad.AgeEnd,
					pq.StringArray(tt.args.ad.Gender),
					pq.StringArray(tt.args.ad.Country),
					pq.StringArray(tt.args.ad.Platform),
					tt.args.ad.Version,
					true,
					AnyTime{},
				).WillReturnResult(sqlmock.NewResult(1, 1))
			mocks.DBMock.ExpectCommit()
			requestBytes, err := json.Marshal(runner.CreateAdRequest{
				Request: runner.Request{RequestID: "???????????"},
				Ad:      tt.args.ad,
			})
			assert.Nil(t, err)
			mocks.CacheMock.CustomMatch(func(expected, actual []interface{}) error {
				return nil
			}).ExpectXAdd(&redis.XAddArgs{
				Stream:     tt.fields.adStream,
				NoMkStream: false,
				Approx:     false,
				MaxLen:     100000,
				Values:     []interface{}{"ad", string(requestBytes)},
				ID:         fmt.Sprintf("0-%d", tt.args.ad.Version),
			}).SetVal(fmt.Sprintf("0-%d", tt.args.ad.Version))
			mocks.CacheMock.CustomMatch(func(expected, actual []interface{}) error {
				return nil
			}).ExpectEvalSha(".", []string{tt.fields.lockKey}, ".").SetVal(".")
			gotAdID, err := a.CreateAd(tt.args.ctx, tt.args.ad)
			if (err != nil) != tt.wantErr {
				t.Errorf("AdService.CreateAd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotAdID != tt.wantAdID {
				t.Errorf("AdService.CreateAd() = %v, want %v", gotAdID, tt.wantAdID)
			}
		})
	}
}

func TestAdService_GetAds(t *testing.T) {
	type fields struct {
		runner   *runner.Runner
		db       *gorm.DB
		redis    *redis.Client
		locker   *redislock.Client
		lockKey  string
		adStream string
	}
	type args struct {
		ctx context.Context
		req *model.GetAdRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*model.Ad
		want1   int
		wantErr bool
	}{
		{
			name: "test get ads",
			fields: fields{
				runner:   runner.NewRunner(inmem.NewInMemoryStore()),
				db:       app.Conn,
				redis:    app.Cache,
				locker:   app.RedisLock,
				lockKey:  lockKey + uuid.New().String(),
				adStream: adStream + uuid.New().String(),
			},
			args: args{
				ctx: context.Background(),
				req: &model.GetAdRequest{
					Age:      20,
					Country:  "TW",
					Platform: "ios",
					Offset:   0,
					Limit:    10,
				},
			},
			want:    nil,
			want1:   0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AdService{
				shutdown: atomic.Bool{},
				runner:   tt.fields.runner,
				db:       tt.fields.db,
				redis:    tt.fields.redis,
				locker:   tt.fields.locker,
				lockKey:  tt.fields.lockKey,
				adStream: tt.fields.adStream,
			}
			mocks.DBMock.ExpectBegin()
			mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
				WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
			mocks.DBMock.ExpectQuery("SELECT (.+) FROM \"ads\"").
				WillReturnRows(mocks.DBMock.NewRows([]string{"id", "title", "content", "start_at", "end_at", "age_start", "age_end", "gender", "country", "platform"}))
			mocks.DBMock.ExpectCommit()
			go a.Run()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			for {
				if a.runner.IsRunning() && a.onShutdownNum() == 2 {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatalf("service did not start within %v", 5*time.Second)
				case <-time.After(time.Millisecond * 100):
				}
			}
			cancel()
			got, got1, err := a.GetAds(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AdService.GetAds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AdService.GetAds() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("AdService.GetAds() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestAdService_Shutdown(t *testing.T) {
	type fields struct {
		runner   *runner.Runner
		db       *gorm.DB
		redis    *redis.Client
		locker   *redislock.Client
		lockKey  string
		adStream string
		Version  int
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
				runner:   runner.NewRunner(inmem.NewInMemoryStore()),
				db:       app.Conn,
				redis:    app.Cache,
				locker:   app.RedisLock,
				lockKey:  lockKey + uuid.New().String(),
				adStream: adStream + uuid.New().String(),
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
				shutdown: atomic.Bool{},
				runner:   tt.fields.runner,
				db:       tt.fields.db,
				redis:    tt.fields.redis,
				locker:   tt.fields.locker,
				lockKey:  tt.fields.lockKey,
				adStream: tt.fields.adStream,
			}
			mocks.DBMock.ExpectBegin()
			mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
				WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
			mocks.DBMock.ExpectQuery("SELECT (.+) FROM \"ads\"").
				WillReturnRows(mocks.DBMock.NewRows([]string{"id", "title", "content", "start_at", "end_at", "age_start", "age_end", "gender", "country", "platform"}))
			mocks.DBMock.ExpectCommit()
			go a.Run()
			ctx, cancel := context.WithTimeout(context.Background(), tt.args.timeout)
			for {
				if a.runner.IsRunning() && a.onShutdownNum() == 2 {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatalf("service did not start within %v", tt.args.timeout)
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
