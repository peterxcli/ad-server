package runner

import (
	"dcard-backend-2024/pkg/model"
	"encoding/json"
)

type Request struct {
	RequestID string `json:"request_id"`
}

type Response struct {
	RequestID string `json:"request_id"`
}

type CreateAdRequest struct {
	Request
	*model.Ad
}

type CreateBatchAdRequest struct {
	Request
	Ads []*model.Ad
}

func (r *CreateAdRequest) ToMap() (map[string]interface{}, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *CreateAdRequest) FromMap(m map[string]interface{}) error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsonData, r); err != nil {
		return err
	}
	return nil
}

type CreateAdResponse struct {
	Response
	AdID string
	Err  error
}

type GetAdRequest struct {
	Request
	*model.GetAdRequest
}

type GetAdResponse struct {
	Response
	Ads   []*model.Ad
	Total int
	Err   error
}
