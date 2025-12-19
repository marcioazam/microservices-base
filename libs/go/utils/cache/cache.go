// Package cache provides a generic TTL cache.
package cache

import (
	"sync"
	"time"
)

// Cache is a generic TTL cache.
type Cache[K comparable, V any] struct {
	defaultTTL time.Duration
	items      map[K]*item[V]
	mu         sync.RWMutex
	stopClean  chan struct{}
}

type item[V any] struct {
	value     V
	expiresAt time.Time
}

// Stats contains cache statistics.
type Stats struct {
	Hits   int64
	Misses int64
	Size   int
}

// New creates a new Cache with the given default TTL.
func New[K comparable, V any](defaultTTL time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		defaultTTL: defaultTTL,
		items:      make(map[K]*item[V]),
		stopClean:  make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// Set adds or updates a value with the default TTL.
func (c *Cache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL adds or updates a value with a custom TTL.
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &item[V]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	if time.Now().After(item.expiresAt) {
		var zero V
		return zero, false
	}

	return item.value, true
}

// GetOrCompute gets a value or computes and stores it if not present.
func (c *Cache[K, V]) GetOrCompute(key K, compute func() V) V {
	// Try read first
	c.mu.RLock()
	item, ok := c.items[key]
	if ok && time.Now().Before(item.expiresAt) {
		c.mu.RUnlock()
		return item.value
	}
	c.mu.RUnlock()

	// Compute and store
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	item, ok = c.items[key]
	if ok && time.Now().Before(item.expiresAt) {
		return item.value
	}

	value := compute()
	c.items[key] = &item[V]{
		value:     value,
		expiresAt: time.Now().Add(c.defaultTTL),
	}
	return value
}

// GetOrComputeWithTTL gets a value or computes and stores it with custom TTL.
func (c *Cache[K, V]) GetOrComputeWithTTL(key K, compute func() V, ttl time.Duration) V {
	c.mu.RLock()
	item, ok := c.items[key]
	if ok && time.Now().Before(item.expiresAt) {
		c.mu.RUnlock()
		return item.value
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok = c.items[key]
	if ok && time.Now().Before(item.expiresAt) {
		return item.value
	}

	value := compute()
	c.items[key] = &item[V]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return value
}

// Delete removes a key from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*item[V])
}

// Len returns the number of items (including expired).
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Keys returns all non-expired keys.
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	keys := make([]K, 0, len(c.items))
	for k, item := range c.items {
		if now.Before(item.expiresAt) {
			keys = append(keys, k)
		}
	}
	return keys
}

// Contains checks if a key exists and is not expired.
func (c *Cache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return false
	}
	return time.Now().Before(item.expiresAt)
}

// Refresh updates the expiration time for a key.
func (c *Cache[K, V]) Refresh(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		return false
	}
	item.expiresAt = time.Now().Add(c.defaultTTL)
	return true
}

// RefreshWithTTL updates the expiration time with a custom TTL.
func (c *Cache[K, V]) RefreshWithTTL(key K, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		return false
	}
	item.expiresAt = time.Now().Add(ttl)
	return true
}

// Close stops the cleanup goroutine.
func (c *Cache[K, V]) Close() {
	close(c.stopClean)
}

func (c *Cache[K, V]) cleanupLoop() {
	ticker := time.NewTicker(c.defaultTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopClean:
			return
		}
	}
}

func (c *Cache[K, V]) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, k)
		}
	}
}
