package runner

import (
	"context"
	"dcard-backend-2024/pkg/model"
	"testing"
	"time"
)

func TestRunner_IsRunning(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan map[string]chan interface{}
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
				ResponseChan: make(map[string]chan interface{}),
				Store:        nil,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRunner(tt.fields.Store)
			go r.Start()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			for {
				if r.IsRunning() {
					break
				}
				select {
				case <-ctx.Done():
					t.Errorf("Runner.IsRunning() = %v, want %v", r.IsRunning(), tt.want)
				default:
				}
			}
		})
	}
}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				RequestChan:  tt.fields.RequestChan,
				ResponseChan: tt.fields.ResponseChan,
				Store:        tt.fields.Store,
			}
			go r.Start()
			if got := r.IsRunning(); got != tt.want {
				t.Errorf("Runner.IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}
