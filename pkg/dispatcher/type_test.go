package dispatcher

import (
	"dcard-backend-2024/pkg/model"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestCreateAdRequest_ToMap(t *testing.T) {
	id := uuid.New()
	ad := &model.Ad{ID: id, Title: "Test Ad", Content: "Test Description"}
	request := CreateAdRequest{
		Request: Request{RequestID: "req-456"},
		Ad:      ad,
	}

	_, err := request.ToMap()
	if err != nil {
		t.Errorf("ToMap() error = %v", err)
		return
	}
}

func TestCreateAdRequest_FromMap(t *testing.T) {
	id := uuid.New()
	m := map[string]interface{}{
		"request_id": "req-789",
		"ID":         id.String(),
		"Title":      "Test Ad From Map",
		"Content":    "Description From Map",
	}

	expectedAd := &model.Ad{ID: id, Title: "Test Ad From Map", Content: "Description From Map"}
	expectedRequest := CreateAdRequest{
		Request: Request{RequestID: "req-789"},
		Ad:      expectedAd,
	}

	var request CreateAdRequest
	if err := request.FromMap(m); err != nil {
		t.Errorf("FromMap() error = %v", err)
		return
	}

	if request.RequestID != expectedRequest.RequestID || !reflect.DeepEqual(request.Ad, expectedRequest.Ad) {
		t.Errorf("FromMap() got = %v, want %v", request, expectedRequest)
	}
}

func TestCreateAdRequest_FromMap_Error(t *testing.T) {
	// Providing incorrect type for fields
	m := map[string]interface{}{
		"request_id":  789,
		"id":          "ad-321",
		"title":       123,
		"description": "Description From Map",
	}

	var request CreateAdRequest
	err := request.FromMap(m)
	if err == nil {
		t.Errorf("FromMap() did not produce an error with incorrect types")
	}
}
