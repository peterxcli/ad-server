package controller

import (
	"bytes"
	"database/sql/driver"
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/service"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type Any struct{}

func (a Any) Match(v driver.Value) bool {
	return true
}

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

type AnyInt struct{}

func (a AnyInt) Match(v driver.Value) bool {
	_, ok := v.(int)
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
	adService := service.NewAdService(
		app.Runner,
		app.Conn,
		app.Cache,
		app.RedisLock,
		app.AsynqClient,
	)
	taskService := service.NewTaskService(adService)
	services = &bootstrap.Services{
		AdService:   adService,
		TaskService: taskService,
	}
	mocks.DBMock.ExpectBegin()
	mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
		WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
	mocks.DBMock.ExpectQuery("SELECT (.+) FROM \"ads\"").
		WillReturnRows(mocks.DBMock.NewRows([]string{"id", "title", "content", "start_at", "end_at", "age_start", "age_end"}))
	mocks.DBMock.ExpectCommit()

	go app.Run(services)
	return
}

var (
	app      *bootstrap.Application
	services *bootstrap.Services
	mocks    *bootstrap.Mocks
)

func init() {
	gin.SetMode(gin.TestMode)
	app, services, mocks = boot()
	mocks.DBMock.MatchExpectationsInOrder(false)
	mocks.CacheMock.MatchExpectationsInOrder(false)
}

func TestAdController_GetAd(t *testing.T) {
	type fields struct {
		adService model.AdService
	}
	type args struct {
		c            *gin.Context
		requestQuery url.Values
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		expectStatus int
	}{
		{
			name: "Test GetAd",
			fields: fields{
				adService: services.AdService,
			},
			args: args{
				c: nil,
				requestQuery: url.Values{
					"country":  {"TW"},
					"gender":   {"M"},
					"platform": {"ios"},
					"age":      {"20"},
					"offset":   {"0"},
					"limit":    {"10"},
				},
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name: "Test GetAd BadRequest: invalid age",
			fields: fields{
				adService: services.AdService,
			},
			args: args{
				c: nil,
				requestQuery: url.Values{
					"country":  {"TW"},
					"gender":   {"M"},
					"platform": {"ios"},
					"age":      {"-1"},
					"offset":   {"0"},
					"limit":    {"10"},
				},
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Test GetAd BadRequest: invalid enum",
			fields: fields{
				adService: services.AdService,
			},
			args: args{
				c: nil,
				requestQuery: url.Values{
					"country":  {"TW"},
					"gender":   {"???"},
					"platform": {"meow"},
					"age":      {"18"},
					"offset":   {"0"},
					"limit":    {"10"},
				},
			},
			expectStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks.DBMock.ExpectExec("SELECT query")
			req, err := http.NewRequest("GET", "/api/v1/ad"+"?"+tt.args.requestQuery.Encode(), nil)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			ac := &AdController{
				adService: tt.fields.adService,
			}
			ac.GetAd(c)
			assert.Equal(t, tt.expectStatus, w.Code)
			t.Logf("Response: %s", w.Body.String())
		})
	}
}
func TestAdController_CreateAd(t *testing.T) {
	// FIXME: redis mock 好像在 xread & redis lock 會壞掉 qq.
	// 1. xread 讀不到 xadd 發出的訊息
	// 2. redis lock release 的時候會一直說 lock not held
	// 可是換成真的 redis 就沒問題了
	type fields struct {
		adService model.AdService
	}
	type args struct {
		c             *gin.Context
		request       model.CreateAdRequest
		expectVersion int
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		expectStatus int
	}{
		{
			name: "Test CreateAd",
			fields: fields{
				adService: services.AdService,
			},
			args: args{
				c: nil,
				request: model.CreateAdRequest{
					Title:    "test test",
					Content:  "test",
					StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
					EndAt:    model.CustomTime(time.Now().Add(1 * time.Hour * 24)),
					AgeStart: 18,
					AgeEnd:   65,
					Gender:   []string{"F", "M"},
					Country:  []string{"TW"},
					Platform: []string{"ios"},
				},
				expectVersion: 1,
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Test CreateAd BadRequest",
			fields: fields{
				adService: services.AdService,
			},
			args: args{
				c: nil,
				request: model.CreateAdRequest{
					Title:    "test bad request",
					Content:  "test bad request",
					StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
					EndAt:    model.CustomTime(time.Now().Add(-10 * time.Hour * 24)),
					AgeStart: 18,
					AgeEnd:   65,
					Gender:   []string{"F", "M"},
					Country:  []string{"TW"},
					Platform: []string{"ios"},
				},
				expectVersion: 2,
			},
			expectStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBytes, err := json.Marshal(tt.args.request)
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest("POST", "/api/v1/ad", bytes.NewBufferString(string(requestBytes)))
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			ac := &AdController{
				adService: tt.fields.adService,
			}
			mocks.CacheMock.Regexp().ExpectEvalSha(".", []string{"lock:ad"}, ".", ".", ".").SetVal(".")
			mocks.DBMock.ExpectBegin()
			mocks.DBMock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM ads").
				WillReturnRows(mocks.DBMock.NewRows([]string{"COALESCE"}))
			mocks.DBMock.ExpectExec("^INSERT INTO \"ads\".+$").
				WithArgs(
					AnyString{},
					tt.args.request.Title,
					tt.args.request.Content,
					AnyTime{},
					AnyTime{},
					tt.args.request.AgeStart,
					tt.args.request.AgeEnd,
					pq.StringArray(tt.args.request.Gender),
					pq.StringArray(tt.args.request.Country),
					pq.StringArray(tt.args.request.Platform),
					1,
					AnyTime{},
				).WillReturnResult(sqlmock.NewResult(1, 1))
			// WillReturnResult(mocks))
			// WillReturnRows(mocks.DBMock.NewRows([]string{"id"}))
			mocks.DBMock.ExpectCommit()
			requestBytes, err = json.Marshal(tt.args.request)
			mocks.CacheMock.CustomMatch(func(expected, actual []interface{}) error {
				return nil
			}).ExpectXAdd(&redis.XAddArgs{
				Stream:     "ad",
				NoMkStream: false,
				Approx:     false,
				MaxLen:     100000,
				Values:     []interface{}{"ad", string(requestBytes)},
				ID:         fmt.Sprintf("0-%d", tt.args.expectVersion),
			}).SetVal(fmt.Sprintf("0-%d", tt.args.expectVersion))
			mocks.CacheMock.CustomMatch(func(expected, actual []interface{}) error {
				return nil
			}).ExpectEvalSha(".", []string{"lock:ad"}, ".").SetVal(".")

			ac.CreateAd(c)
			assert.Equal(t, tt.expectStatus, w.Code)
			// t.Logf("Response: %s", w.Body.String())
		})
	}
}

func TestNewAdController(t *testing.T) {
	type args struct {
		adService model.AdService
	}
	tests := []struct {
		name string
		args args
		want *AdController
	}{
		{
			name: "Test NewAdController",
			args: args{
				adService: services.AdService,
			},
			want: &AdController{
				adService: services.AdService,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAdController(tt.args.adService); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAdController() = %v, want %v", got, tt.want)
			}
		})
	}
}
