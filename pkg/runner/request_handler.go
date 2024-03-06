package runner

import (
	"dcard-backend-2024/pkg/model"
)

type Runner struct {
	RequestChan  chan interface{}
	ResponseChan map[string]chan interface{}
	Store        model.InMemoryStore
}

func NewRunner(store model.InMemoryStore) *Runner {
	return &Runner{
		RequestChan:  make(chan interface{}),
		ResponseChan: make(map[string]chan interface{}),
		Store:        store,
	}
}

func (r *Runner) handleCreateBatchAdRequest(req *CreateBatchAdRequest) {
	err := r.Store.CreateBatchAds(req.Ads)

	if r.ResponseChan[req.RequestID] != nil {
		r.ResponseChan[req.RequestID] <- &CreateAdResponse{
			Response: Response{RequestID: req.RequestID},
			Err:      err,
		}
	}
}

func (r *Runner) handleCreateAdRequest(req *CreateAdRequest) {
	adIDr, err := r.Store.CreateAd(req.Ad)

	if r.ResponseChan[req.RequestID] != nil {
		r.ResponseChan[req.RequestID] <- &CreateAdResponse{
			Response: Response{RequestID: req.RequestID},
			AdID:     adIDr,
			Err:      err,
		}
	}
}

func (r *Runner) handleGetAdRequest(req *GetAdRequest) {
	ads, total, err := r.Store.GetAds(req.GetAdRequest)

	if r.ResponseChan[req.RequestID] != nil {
		r.ResponseChan[req.RequestID] <- &GetAdResponse{
			Response: Response{RequestID: req.RequestID},
			Ads:      ads,
			Total:    total,
			Err:      err,
		}
	}
}

func (r *Runner) Start() {
	for {
		select {
		case req := <-r.RequestChan:
			switch req.(type) {
			case *CreateBatchAdRequest:
				r.handleCreateBatchAdRequest(req.(*CreateBatchAdRequest))
			case *CreateAdRequest:
				// the create ad request is from the redis stream
				r.handleCreateAdRequest(req.(*CreateAdRequest))
			case *GetAdRequest:
				r.handleGetAdRequest(req.(*GetAdRequest))
			}
		}
	}
}
