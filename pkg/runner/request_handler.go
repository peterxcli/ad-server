package runner

type Runner struct {
	RequestChan  chan interface{}
	ResponseChan map[string]chan interface{}
	Store        *InMemoryStore
}

func NewRunner() *Runner {
	return &Runner{
		RequestChan:  make(chan interface{}),
		ResponseChan: make(map[string]chan interface{}),
		Store:        NewInMemoryStore(),
	}
}

func (r *Runner) handleCreateAdRequest(req *CreateAdRequest) {
	adIDr, err := r.Store.CreateAd(&req.Ad)

	if r.ResponseChan[req.RequestID] != nil {
		r.ResponseChan[req.RequestID] <- &CreateAdResponse{
			Response: Response{RequestID: req.RequestID},
			AdID:     adIDr,
			Err:      err,
		}
	}
}

func (r *Runner) handleGetAdRequest(req *GetAdRequest) {
	ads, total, err := r.Store.GetAds(req)

	if r.ResponseChan[req.RequestID] != nil {
		r.ResponseChan[req.RequestID] <- &GetAdResponse{
			Response: Response{RequestID: req.RequestID},
			Ads:      ads,
			total:    total,
			Err:      err,
		}
	}
}

func (r *Runner) Start() {
	for {
		select {
		case req := <-r.RequestChan:
			switch req.(type) {
			case *CreateAdRequest:
				// the create ad request is from the rabbitmq
				r.handleCreateAdRequest(req.(*CreateAdRequest))
			case *GetAdRequest:
				r.handleGetAdRequest(req.(*GetAdRequest))
			}
		}
	}
}
