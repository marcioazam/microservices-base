// Package registry provides a generic thread-safe Registry[K, V] type.
package registry

import "sync"

// Registry is a thread-safe key-value store.
type Registry[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

// New creates a new empty Registry.
func New[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{items: make(map[K]V)}
}

// Register stores a value with the given key.
func (r *Registry[K, V]) Register(key K, value V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = value
}

// Get retrieves a value by key. Returns the value and true if found.
func (r *Registry[K, V]) Get(key K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.items[key]
	return value, ok
}

// GetOrDefault retrieves a value by key, or returns the default if not found.
func (r *Registry[K, V]) GetOrDefault(key K, defaultVal V) V {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if value, ok := r.items[key]; ok {
		return value
	}
	return defaultVal
}

// Has returns true if the key exists in the registry.
func (r *Registry[K, V]) Has(key K) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.items[key]
	return ok
}

// Unregister removes a value by key. Returns true if the key was found.
func (r *Registry[K, V]) Unregister(key K) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[key]; !ok {
		return false
	}
	delete(r.items, key)
	return true
}

// Keys returns all keys in the registry.
func (r *Registry[K, V]) Keys() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]K, 0, len(r.items))
	for k := range r.items {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values in the registry.
func (r *Registry[K, V]) Values() []V {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]V, 0, len(r.items))
	for _, v := range r.items {
		values = append(values, v)
	}
	return values
}

// ForEach applies fn to each key-value pair.
func (r *Registry[K, V]) ForEach(fn func(K, V)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.items {
		fn(k, v)
	}
}

// Clear removes all entries from the registry.
func (r *Registry[K, V]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[K]V)
}

// Len returns the number of entries in the registry.
func (r *Registry[K, V]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}

// GetOrRegister retrieves a value by key, or registers and returns the default if not found.
func (r *Registry[K, V]) GetOrRegister(key K, defaultVal V) V {
	r.mu.Lock()
	defer r.mu.Unlock()
	if value, ok := r.items[key]; ok {
		return value
	}
	r.items[key] = defaultVal
	return defaultVal
}

// ComputeIfAbsent retrieves a value by key, or computes and registers it if not found.
func (r *Registry[K, V]) ComputeIfAbsent(key K, compute func() V) V {
	r.mu.Lock()
	defer r.mu.Unlock()
	if value, ok := r.items[key]; ok {
		return value
	}
	value := compute()
	r.items[key] = value
	return value
}

// Update updates a value by key using the provided function.
func (r *Registry[K, V]) Update(key K, fn func(V, bool) V) V {
	r.mu.Lock()
	defer r.mu.Unlock()
	oldValue, exists := r.items[key]
	newValue := fn(oldValue, exists)
	r.items[key] = newValue
	return newValue
}

// Filter returns a new registry with entries that satisfy the predicate.
func (r *Registry[K, V]) Filter(predicate func(K, V) bool) *Registry[K, V] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := New[K, V]()
	for k, v := range r.items {
		if predicate(k, v) {
			result.items[k] = v
		}
	}
	return result
}

// Clone returns a copy of the registry.
func (r *Registry[K, V]) Clone() *Registry[K, V] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := New[K, V]()
	for k, v := range r.items {
		result.items[k] = v
	}
	return result
}
