package syncmap

import "sync"

type Map struct {
	syncMap sync.Map
}

func (m *Map) LoadOrStore(key string, value chan interface{}) (chan interface{}, bool) {
	val, loaded := m.syncMap.LoadOrStore(key, value)
	return val.(chan interface{}), loaded
}

func (m *Map) Load(key string) chan interface{} {
	val, ok := m.syncMap.Load(key)
	if ok {
		return val.(chan interface{})
	} else {
		return nil
	}
}

func (m *Map) Exists(key string) bool {
	_, ok := m.syncMap.Load(key)
	return ok
}

func (m *Map) Store(key string, value chan interface{}) {
	m.syncMap.Store(key, value)
}

func (m *Map) Delete(key string) {
	m.syncMap.Delete(key)
}
