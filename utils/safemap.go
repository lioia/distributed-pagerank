package utils

import (
	"sync"
)

type Number interface {
	int | int32 | int64 | float32 | float64
}

type SafeMap[K comparable, V Number] struct {
	mutex sync.Mutex
	data  map[K]V
}

func NewSafeMap[K comparable, V Number]() *SafeMap[K, V] {
	return &SafeMap[K, V]{data: make(map[K]V)}
}

func (m *SafeMap[K, V]) Get(key K) V {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.data[key]
}

func (m *SafeMap[K, V]) Put(key K, value V) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[key] = value
}

func (m *SafeMap[K, V]) Increment(key K, increment V) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[key] += increment
}

func (m *SafeMap[K, V]) Keys() []K {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

func (m *SafeMap[K, V]) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data = make(map[K]V)
}

func (m *SafeMap[K, V]) Clone() map[K]V {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	data := make(map[K]V)
	for key, v := range m.data {
		data[key] = v
	}
	return data
}
