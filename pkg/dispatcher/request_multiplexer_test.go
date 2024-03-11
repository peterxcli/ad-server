package dispatcher

import (
	"context"
	"dcard-backend-2024/pkg/inmem"
	"dcard-backend-2024/pkg/model"
	"dcard-backend-2024/pkg/syncmap"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDispatcher_IsRunning(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan *syncmap.Map
		Store        model.InMemoryStore
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Test IsRunning",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: &syncmap.Map{},
				Store:        nil,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewDispatcher(tt.fields.Store)
			go r.Start()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			for {
				if r.IsRunning() {
					break
				}
				select {
				case <-ctx.Done():
					t.Errorf("Dispatcher.IsRunning() = %v, want %v", r.IsRunning(), tt.want)
				default:
				}
			}
		})
	}
}

func TestDispatcher_handleCreateBatchAdRequest(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan *syncmap.Map
		Store        model.InMemoryStore
	}
	type args struct {
		req *CreateBatchAdRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test handleCreateBatchAdRequest",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: &syncmap.Map{},
				Store:        inmem.NewInMemoryStore(),
			},
			args: args{
				req: &CreateBatchAdRequest{
					Request: Request{RequestID: "test"},
					Ads: []*model.Ad{
						{
							ID:       uuid.New(),
							Title:    "test",
							Content:  "test",
							StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
							EndAt:    model.CustomTime(time.Now().Add(1 * time.Hour * 24)),
							AgeStart: 18,
							AgeEnd:   65,
							Gender:   []string{"F", "M"},
							Country:  []string{"TW"},
							Platform: []string{"ios"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Dispatcher{
				RequestChan:  tt.fields.RequestChan,
				ResponseChan: tt.fields.ResponseChan,
				Store:        tt.fields.Store,
			}
			tt.fields.ResponseChan.Store(tt.args.req.RequestID, make(chan interface{}))
			go r.handleCreateBatchAdRequest(tt.args.req)
			select {
			case resp := <-tt.fields.ResponseChan.Load(tt.args.req.RequestID):
				if resp, ok := resp.(*CreateAdResponse); ok {
					if resp.Err != nil {
						t.Errorf("Dispatcher.handleCreateBatchAdRequest() = %v, want %v", resp.Err, nil)
					}
					assert.Equal(t, resp.AdID, "")
				}
			case <-time.After(3 * time.Second):
				t.Errorf("Dispatcher.handleCreateBatchAdRequest() = %v, want %v", nil, nil)
			}
		})
	}
}

func TestDispatcher_handleCreateAdRequest(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan *syncmap.Map
		Store        model.InMemoryStore
	}
	type args struct {
		req *CreateAdRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test handleCreateAdRequest",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: &syncmap.Map{},
				Store:        inmem.NewInMemoryStore(),
			},
			args: args{
				req: &CreateAdRequest{
					Request: Request{RequestID: "test"},
					Ad: &model.Ad{
						ID:       uuid.New(),
						Title:    "test",
						Content:  "test",
						StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
						EndAt:    model.CustomTime(time.Now().Add(1 * time.Hour * 24)),
						AgeStart: 18,
						AgeEnd:   65,
						Gender:   []string{"F", "M"},
						Country:  []string{"TW"},
						Platform: []string{"ios"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Dispatcher{
				RequestChan:  tt.fields.RequestChan,
				ResponseChan: tt.fields.ResponseChan,
				Store:        tt.fields.Store,
			}
			tt.fields.ResponseChan.Store(tt.args.req.RequestID, make(chan interface{}))
			go r.handleCreateAdRequest(tt.args.req)
			select {
			case resp := <-tt.fields.ResponseChan.Load(tt.args.req.RequestID):
				if resp, ok := resp.(*CreateAdResponse); ok {
					if resp.Err != nil {
						t.Errorf("Dispatcher.handleCreateAdRequest() = %v, want %v", resp.Err, nil)
					}
					assert.Equal(t, resp.AdID, tt.args.req.Ad.ID.String())
				}
			case <-time.After(3 * time.Second):
				t.Errorf("Dispatcher.handleCreateAdRequest() = %v, want %v", nil, nil)
			}
		})
	}
}

func TestDispatcher_handleGetAdRequest(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan *syncmap.Map
		Store        model.InMemoryStore
	}
	type args struct {
		req *GetAdRequest
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		expectErr error
	}{
		{
			name: "Test handleGetAdRequest",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: &syncmap.Map{},
				Store:        inmem.NewInMemoryStore(),
			},
			args: args{
				req: &GetAdRequest{
					Request: Request{RequestID: "test"},
					GetAdRequest: &model.GetAdRequest{
						Age:     18,
						Country: "TW",
						Limit:   10,
					},
				},
			},
			expectErr: inmem.ErrNoAdsFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Dispatcher{
				RequestChan:  tt.fields.RequestChan,
				ResponseChan: tt.fields.ResponseChan,
				Store:        tt.fields.Store,
			}

			tt.fields.ResponseChan.Store(tt.args.req.RequestID, make(chan interface{}))
			go r.handleGetAdRequest(tt.args.req)
			select {
			case resp := <-tt.fields.ResponseChan.Load(tt.args.req.RequestID):
				if resp, ok := resp.(*GetAdResponse); ok {
					assert.ErrorIs(t, resp.Err, tt.expectErr)
				}
			case <-time.After(3 * time.Second):
				t.Errorf("Dispatcher.handleGetAdRequest() = %v, want %v", nil, nil)
			}
		})
	}
}

func TestDispatcher_Start(t *testing.T) {
	type fields struct {
	}
	sharedStore := inmem.NewInMemoryStore()
	sharedRequestChan := make(chan interface{})
	sharedResponseChan := &syncmap.Map{}
	sharedDispatcher := &Dispatcher{
		RequestChan:  sharedRequestChan,
		ResponseChan: sharedResponseChan,
		Store:        sharedStore,
	}
	go sharedDispatcher.Start()
	tests := []struct {
		name      string
		fields    fields
		payload   any
		expectErr error
	}{
		{
			name:   "Test Start CreateAdRequest",
			fields: fields{},
			payload: &CreateAdRequest{
				Request: Request{RequestID: uuid.NewString()},
				Ad: &model.Ad{
					ID:       uuid.New(),
					Title:    "test",
					Content:  "test",
					StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
					EndAt:    model.CustomTime(time.Now().Add(1 * time.Hour * 24)),
					AgeStart: 18,
					AgeEnd:   65,
					Gender:   []string{"F", "M"},
					Country:  []string{"TW"},
					Platform: []string{"ios"},
				},
			},
			expectErr: nil,
		},
		{
			name:   "Test Start GetAdRequest",
			fields: fields{},
			payload: &GetAdRequest{
				Request: Request{RequestID: uuid.NewString()},
				GetAdRequest: &model.GetAdRequest{
					Age:     18,
					Country: "TW",
					Limit:   10,
				},
			},
			expectErr: nil,
		},
		{
			name:   "Test Start CreateBatchAdRequest",
			fields: fields{},
			payload: &CreateBatchAdRequest{
				Request: Request{RequestID: uuid.NewString()},
				Ads: []*model.Ad{
					{
						ID:       uuid.New(),
						Title:    "test",
						Content:  "test",
						StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
						EndAt:    model.CustomTime(time.Now().Add(1 * time.Hour * 24)),
						AgeStart: 18,
						AgeEnd:   65,
						Gender:   []string{"F", "M"},
						Country:  []string{"TW"},
						Platform: []string{"ios"},
					},
				},
			},
			expectErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestID := tt.payload.(IRequest).RequestUID()
			sharedResponseChan.Store(requestID, make(chan interface{}))
			sharedRequestChan <- tt.payload
			select {
			case resp := <-sharedResponseChan.Load(requestID):
				if resp, ok := resp.(IResult); ok {
					assert.ErrorIs(t, resp.Error(), tt.expectErr)
				}
			case <-time.After(3 * time.Second):
				t.Errorf("Dispatcher.Start() = %v, want %v", nil, nil)
			}
		})
	}
}

func TestNewDispatcher(t *testing.T) {
	type args struct {
		store model.InMemoryStore
	}
	tests := []struct {
		name string
		args args
		want *Dispatcher
	}{
		{
			name: "Test NewDispatcher",
			args: args{
				store: inmem.NewInMemoryStore(),
			},
			want: &Dispatcher{
				RequestChan:  make(chan interface{}),
				ResponseChan: &syncmap.Map{},
				Store:        inmem.NewInMemoryStore(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want.Store = tt.args.store
			if got := NewDispatcher(tt.args.store); !reflect.DeepEqual(got.Store, tt.want.Store) {
				t.Errorf("NewDispatcher() = %v, want %v", got, tt.want)
			}
		})
	}
}
