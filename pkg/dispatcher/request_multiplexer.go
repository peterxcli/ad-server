package dispatcher

import (
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/syncmap"
	"log"
	"sync/atomic"
	"time"
)

type Dispatcher struct {
	Running      atomic.Bool
	RequestChan  chan interface{}
	ResponseChan *syncmap.Map
	Store        model.InMemoryStore
}

func (r *Dispatcher) IsRunning() bool {
	return r.Running.Load()
}

func NewDispatcher(store model.InMemoryStore) *Dispatcher {
	return &Dispatcher{
		RequestChan:  make(chan interface{}),
		ResponseChan: &syncmap.Map{},
		Store:        store,
	}
}

func (r *Dispatcher) handleCreateBatchAdRequest(req *CreateBatchAdRequest) {
	// err := r.Store.CreateBatchAds(req.Ads)
	for _, ad := range req.Ads {
		if time.Now().After(ad.StartAt.T()) {
			_, err := r.Store.CreateAd(ad)
			if err != nil {
				log.Printf("failed to create ad %s: %v", ad.ID, err)
			}
		} else {
			log.Printf("ad %s is scheduled to start at %s", ad.ID, ad.StartAt.T())
			time.AfterFunc(time.Until(ad.StartAt.T()), func() {
				_, err := r.Store.CreateAd(ad)
				if err != nil {
					log.Printf("failed to create ad %s: %v", ad.ID, err)
				} else {
					log.Printf("scheduled ad %s is created", ad.ID)
				}
			})
		}
	}

	// use sync map to store the response channel
	if r.ResponseChan.Exists(req.RequestID) {
		r.ResponseChan.Load(req.RequestID) <- &CreateAdResponse{
			Response: Response{RequestID: req.RequestID},
			Err:      nil,
		}
	}
}

func (r *Dispatcher) handleCreateAdRequest(req *CreateAdRequest) {
	if time.Now().After(req.Ad.StartAt.T()) {
		_, err := r.Store.CreateAd(req.Ad)
		if err != nil {
			log.Printf("failed to create ad %s: %v", req.Ad.ID, err)
		}
	} else {
		log.Printf("ad %s is scheduled to start at %s", req.Ad.ID, req.Ad.StartAt.T())
		time.AfterFunc(time.Until(req.Ad.StartAt.T()), func() {
			_, err := r.Store.CreateAd(req.Ad)
			if err != nil {
				log.Printf("failed to create ad %s: %v", req.Ad.ID, err)
			} else {
				log.Printf("scheduled ad %s is created", req.Ad.ID)
			}
		})
	}

	if r.ResponseChan.Exists(req.RequestID) {
		r.ResponseChan.Load(req.RequestID) <- &CreateAdResponse{
			Response: Response{RequestID: req.RequestID},
			AdID:     req.Ad.ID.String(),
			Err:      nil,
		}
	}
}

func (r *Dispatcher) handleGetAdRequest(req *GetAdRequest) {
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

func (r *Dispatcher) handleDeleteAdRequest(req *DeleteAdRequest) {
	_ = r.Store.DeleteAd(req.AdID)

	// if r.ResponseChan.Exists(req.RequestID) {
	// 	r.ResponseChan.Load(req.RequestID) <- &DeleteAdResponse{
	// 		Response: Response{RequestID: req.RequestID},
	// 		Err:      err,
	// 	}
	// }
}

func (r *Dispatcher) Start() {
	r.Running.Store(true)
	log.Println("Dispatcher started")
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
			case *DeleteAdRequest:
				r.handleDeleteAdRequest(req.(*DeleteAdRequest))
			}
		}
	}
}
