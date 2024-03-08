package runner

import (
	"reflect"
	"sync"
	"testing"
)

func TestMap_LoadOrStore(t *testing.T) {
	type fields struct {
		// syncMap sync.Map
	}
	type args struct {
		key   string
		value chan interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   chan interface{}
		want1  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				syncMap: sync.Map{},
			}
			got, got1 := m.LoadOrStore(tt.args.key, tt.args.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Map.LoadOrStore() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Map.LoadOrStore() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMap_Load(t *testing.T) {
	type fields struct {
		// syncMap sync.Map
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   chan interface{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				syncMap: sync.Map{},
			}
			if got := m.Load(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Map.Load() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMap_Exists(t *testing.T) {
	type fields struct {
		// syncMap sync.Map
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				syncMap: sync.Map{},
			}
			if got := m.Exists(tt.args.key); got != tt.want {
				t.Errorf("Map.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMap_Store(t *testing.T) {
	type fields struct {
		// syncMap sync.Map
	}
	type args struct {
		key   string
		value chan interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				syncMap: sync.Map{},
			}
			m.Store(tt.args.key, tt.args.value)
		})
	}
}

func TestMap_Delete(t *testing.T) {
	type fields struct {
		syncMap sync.Map
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				syncMap: sync.Map{},
			}
			m.Delete(tt.args.key)
		})
	}
}
