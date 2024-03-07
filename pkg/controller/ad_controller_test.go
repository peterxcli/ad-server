package controller

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/service"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func boot() (app *bootstrap.Application, services *bootstrap.Services) {
	gin.SetMode(gin.TestMode)
	app = bootstrap.NewTestApp()
	app.Conn.AutoMigrate(&model.Ad{})
	adService := service.NewAdService(
		app.Runner,
		app.Conn,
		app.Cache,
		app.RedisLock,
	)
	services = &bootstrap.Services{
		AdService: adService,
	}
	go app.Run(services)
	return
}

func TestAdController_GetAd(t *testing.T) {
	_, services := boot()
	type fields struct {
		adService model.AdService
	}
	type args struct {
		c            *gin.Context
		requestQuery url.Values
	}
	tests := []struct {
		name   string
		fields fields
		args   args
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/ad"+"?"+tt.args.requestQuery.Encode(), nil)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			ac := &AdController{
				adService: tt.fields.adService,
			}
			ac.GetAd(c)
		})
	}
}
