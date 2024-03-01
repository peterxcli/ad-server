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

func (r *Runner) handleCreateAdRequest(req CreateAdRequest) {
	// create ad
	// create conditions
	// create ad response

}

func (r *Runner) handleGetAdRequest(req GetAdRequest) {
	// get ad
	// get conditions
	// create get ad response
}

func (r *Runner) Start() {
	for {
		select {
		case req := <-r.RequestChan:
			switch req.(type) {
			case CreateAdRequest:
				// the create ad request is from the rabbitmq
				r.handleCreateAdRequest(req.(CreateAdRequest))
			case GetAdRequest:
				r.handleGetAdRequest(req.(GetAdRequest))
			}
		}
	}
}
