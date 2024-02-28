package runner

type Runner struct {
	State *RunnerState
	Store *InMemoryStore
}

func NewRunner() *Runner {
	return &Runner{
		State: &RunnerState{
			RequestChan: make(chan interface{}),
		},
		Store: NewInMemoryStore(),
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
		case req := <-r.State.RequestChan:
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
