// Package syncmap provides a generic thread-safe ConcurrentMap[K, V] type.
package syncmap

import "sync"

// ConcurrentMap is a thread-safe map with generic key and value types.
type ConcurrentMap[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

// New creates a new empty ConcurrentMap.
func New[K comparable, V any]() *ConcurrentMap[K, V] {
	return &ConcurrentMap[K, V]{items: make(map[K]V)}
}

// Set stores a value with the given key.
func (m *ConcurrentMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = value
}

// Get retrieves a value by key. Returns the value and true if found.
func (m *ConcurrentMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.items[key]
	return value, ok
}

// GetOrSet retrieves a value by key, or sets and returns the default if not found.
func (m *ConcurrentMap[K, V]) GetOrSet(key K, defaultVal V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if value, ok := m.items[key]; ok {
		return value, true
	}
	m.items[key] = defaultVal
	return defaultVal, false
}

// ComputeIfAbsent retrieves a value by key, or computes and stores it if not found.
func (m *ConcurrentMap[K, V]) ComputeIfAbsent(key K, compute func() V) V {
	m.mu.Lock()
	defer m.mu.Unlock()
	if value, ok := m.items[key]; ok {
		return value
	}
	value := compute()
	m.items[key] = value
	return value
}

// Delete removes a value by key. Returns the old value and true if found.
func (m *ConcurrentMap[K, V]) Delete(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.items[key]
	if ok {
		delete(m.items, key)
	}
	return value, ok
}

// Has returns true if the key exists.
func (m *ConcurrentMap[K, V]) Has(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.items[key]
	return ok
}

// Len returns the number of entries.
func (m *ConcurrentMap[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// Clear removes all entries.
func (m *ConcurrentMap[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[K]V)
}

// Keys returns all keys.
func (m *ConcurrentMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values.
func (m *ConcurrentMap[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]V, 0, len(m.items))
	for _, v := range m.items {
		values = append(values, v)
	}
	return values
}

// ForEach applies fn to each key-value pair.
func (m *ConcurrentMap[K, V]) ForEach(fn func(K, V)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.items {
		fn(k, v)
	}
}

// Update atomically updates a value using the provided function.
func (m *ConcurrentMap[K, V]) Update(key K, fn func(V, bool) V) V {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldValue, exists := m.items[key]
	newValue := fn(oldValue, exists)
	m.items[key] = newValue
	return newValue
}

// Compute atomically computes a new value for the key.
func (m *ConcurrentMap[K, V]) Compute(key K, fn func(K, V, bool) (V, bool)) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldValue, exists := m.items[key]
	newValue, keep := fn(key, oldValue, exists)
	if keep {
		m.items[key] = newValue
		return newValue, true
	}
	delete(m.items, key)
	var zero V
	return zero, false
}

// Clone returns a copy of the map.
func (m *ConcurrentMap[K, V]) Clone() *ConcurrentMap[K, V] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := New[K, V]()
	for k, v := range m.items {
		result.items[k] = v
	}
	return result
}
