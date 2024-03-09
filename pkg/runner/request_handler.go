package runner

import (
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/syncmap"
	"log"
	"sync/atomic"
)

type Runner struct {
	Running      atomic.Bool
	RequestChan  chan interface{}
	ResponseChan *syncmap.Map
	Store        model.InMemoryStore
}

func (r *Runner) IsRunning() bool {
	return r.Running.Load()
}

func NewRunner(store model.InMemoryStore) *Runner {
	return &Runner{
		RequestChan:  make(chan interface{}),
		ResponseChan: &syncmap.Map{},
		Store:        store,
	}
}

func (r *Runner) handleCreateBatchAdRequest(req *CreateBatchAdRequest) {
	err := r.Store.CreateBatchAds(req.Ads)

	// use sync map to store the response channel
	if r.ResponseChan.Exists(req.RequestID) {
		r.ResponseChan.Load(req.RequestID) <- &CreateAdResponse{
			Response: Response{RequestID: req.RequestID},
			Err:      err,
		}
	}
}

func (r *Runner) handleCreateAdRequest(req *CreateAdRequest) {
	adIDr, err := r.Store.CreateAd(req.Ad)

	if r.ResponseChan.Exists(req.RequestID) {
		r.ResponseChan.Load(req.RequestID) <- &CreateAdResponse{
			Response: Response{RequestID: req.RequestID},
			AdID:     adIDr,
			Err:      err,
		}
	}
}

func (r *Runner) handleGetAdRequest(req *GetAdRequest) {
	ads, total, err := r.Store.GetAds(req.GetAdRequest)

	if r.ResponseChan.Exists(req.RequestID) {
		r.ResponseChan.Load(req.RequestID) <- &GetAdResponse{
			Response: Response{RequestID: req.RequestID},
			Ads:      ads,
			Total:    total,
			Err:      err,
		}
	}
}

func (r *Runner) Start() {
	r.Running.Store(true)
	log.Println("Runner started")
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
				go r.handleGetAdRequest(req.(*GetAdRequest))
			}
		}
	}
}
