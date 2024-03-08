package runner

import (
	"dcard-backend-2024/pkg/model"
	"reflect"
	"testing"
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
			want: false,
		},
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
