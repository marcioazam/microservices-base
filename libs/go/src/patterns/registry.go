package patterns

import (
	"sync"

	"github.com/authcorp/libs/go/src/functional"
)

// Registry provides a thread-safe registry pattern.
type Registry[K comparable, V any] struct {
	items map[K]V
	mu    sync.RWMutex
}

// NewRegistry creates a new registry.
func NewRegistry[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{
		items: make(map[K]V),
	}
}

// Register adds an item to the registry.
func (r *Registry[K, V]) Register(key K, value V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = value
}

// Unregister removes an item from the registry.
func (r *Registry[K, V]) Unregister(key K) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[key]; ok {
		delete(r.items, key)
		return true
	}
	return false
}

// Get retrieves an item from the registry.
func (r *Registry[K, V]) Get(key K) functional.Option[V] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if v, ok := r.items[key]; ok {
		return functional.Some(v)
	}
	return functional.None[V]()
}

// Has checks if a key exists.
func (r *Registry[K, V]) Has(key K) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.items[key]
	return ok
}

// Keys returns all registered keys.
func (r *Registry[K, V]) Keys() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]K, 0, len(r.items))
	for k := range r.items {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all registered values.
func (r *Registry[K, V]) Values() []V {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]V, 0, len(r.items))
	for _, v := range r.items {
		values = append(values, v)
	}
	return values
}

// Size returns the number of registered items.
func (r *Registry[K, V]) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}

// Clear removes all items.
func (r *Registry[K, V]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[K]V)
}

// ForEach iterates over all items.
func (r *Registry[K, V]) ForEach(fn func(K, V)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.items {
		fn(k, v)
	}
}

// GetOrRegister gets existing or registers new value.
func (r *Registry[K, V]) GetOrRegister(key K, factory func() V) V {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.items[key]; ok {
		return v
	}
	v := factory()
	r.items[key] = v
	return v
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

// Update updates a value by key using the provided function.
func (r *Registry[K, V]) Update(key K, fn func(V, bool) V) V {
	r.mu.Lock()
	defer r.mu.Unlock()
	oldValue, exists := r.items[key]
	newValue := fn(oldValue, exists)
	r.items[key] = newValue
	return newValue
}

// FilterRegistry returns a new registry with entries that satisfy the predicate.
func (r *Registry[K, V]) FilterRegistry(predicate func(K, V) bool) *Registry[K, V] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := NewRegistry[K, V]()
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
	result := NewRegistry[K, V]()
	for k, v := range r.items {
		result.items[k] = v
	}
	return result
}
