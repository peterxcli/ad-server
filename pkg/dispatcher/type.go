package dispatcher

import (
	"dcard-backend-2024/pkg/model"
	"encoding/json"
)

type IRequest interface {
	RequestUID() string
}

type Request struct {
	IRequest
	RequestID string `json:"request_id"`
}

func (r *Request) RequestUID() string {
	return r.RequestID
}

type IResult interface {
	Error() error
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
	IResult
	Response
	AdID string
	Err  error
}

func (r *CreateAdResponse) Error() error {
	return r.Err
}

type GetAdRequest struct {
	Request
	*model.GetAdRequest
}

type GetAdResponse struct {
	IResult
	Response
	Ads   []*model.Ad
	Total int
	Err   error
}

func (r *GetAdResponse) Error() error {
	return r.Err
}

type DeleteAdRequest struct {
	Request
	AdID string
}

type DeleteAdResponse struct {
	IResult
	Response
	Err error
}

func (r *DeleteAdResponse) Error() error {
	return r.Err
}
