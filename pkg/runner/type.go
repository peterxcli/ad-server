package runner

import (
	"dcard-backend-2024/pkg/model"
)

type Request struct {
	RequestID string
}

type Response struct {
	RequestID string
}

type CreateAdRequest struct {
	Request
	model.Ad
}

type CreateAdResponse struct {
	Response
	AdID string
	Err  error
}

type GetAdRequest struct {
	Request
	model.GetAdRequest
}

type GetAdResponse struct {
	Response
	Ads   []*model.Ad
	total int
	Err   error
}
