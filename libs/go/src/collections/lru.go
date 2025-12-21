// Package collections provides generic data structures including LRU cache.
package collections

import (
	"container/list"
	"iter"
	"sync"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// Stats holds cache statistics for monitoring.
type Stats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Evictions   int64   `json:"evictions"`
	Expirations int64   `json:"expirations"`
	Size        int     `json:"size"`
	Capacity    int     `json:"capacity"`
	HitRate     float64 `json:"hit_rate"`
}

// lruEntry represents a cache entry with TTL support.
type lruEntry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// LRUCache is a thread-safe LRU cache with TTL, eviction callbacks, and stats.
// This is the single authoritative LRU implementation (consolidated from cache/).
type LRUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	order    *list.List
	mu       sync.RWMutex
	onEvict  func(K, V)
	ttl      time.Duration
	stats    Stats
}

// NewLRUCache creates a new LRU cache with the specified capacity.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		order:    list.New(),
		stats:    Stats{Capacity: capacity},
	}
}

// WithTTL sets the default TTL for cache entries.
func (c *LRUCache[K, V]) WithTTL(ttl time.Duration) *LRUCache[K, V] {
	c.ttl = ttl
	return c
}

// WithEvictCallback sets the eviction callback function.
func (c *LRUCache[K, V]) WithEvictCallback(fn func(K, V)) *LRUCache[K, V] {
	c.onEvict = fn
	return c
}

// Get retrieves a value from the cache, returning Option[V] for type-safe access.
func (c *LRUCache[K, V]) Get(key K) functional.Option[V] {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return functional.None[V]()
	}

	entry := elem.Value.(*lruEntry[K, V])

	// Check TTL expiration
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		c.removeElement(elem)
		c.stats.Expirations++
		c.stats.Misses++
		return functional.None[V]()
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	c.stats.Hits++
	return functional.Some(entry.value)
}

// Put adds or updates a value in the cache using default TTL.
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.PutWithTTL(key, value, c.ttl)
}

// PutWithTTL adds or updates a value with a specific TTL.
func (c *LRUCache[K, V]) PutWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Update existing entry
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		entry := elem.Value.(*lruEntry[K, V])
		entry.value = value
		entry.expiresAt = expiresAt
		return
	}

	// Evict if at capacity
	if c.order.Len() >= c.capacity {
		c.evictOldest()
	}

	// Add new entry
	entry := &lruEntry[K, V]{key: key, value: value, expiresAt: expiresAt}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
	c.stats.Size = c.order.Len()
}

// GetOrCompute returns cached value or computes, caches, and returns it.
func (c *LRUCache[K, V]) GetOrCompute(key K, compute func() V) V {
	if opt := c.Get(key); opt.IsSome() {
		return opt.Unwrap()
	}
	value := compute()
	c.Put(key, value)
	return value
}

// Remove removes a key from the cache.
func (c *LRUCache[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
		return true
	}
	return false
}

// Contains checks if key exists in cache (without updating LRU order).
func (c *LRUCache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Size returns current number of entries in the cache.
func (c *LRUCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

// IsEmpty returns true if cache has no entries.
func (c *LRUCache[K, V]) IsEmpty() bool {
	return c.Size() == 0
}

// Clear removes all entries from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*list.Element)
	c.order.Init()
	c.stats.Size = 0
}

// Keys returns all keys in the cache.
func (c *LRUCache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]K, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

// Stats returns cache statistics.
func (c *LRUCache[K, V]) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stats := c.stats
	stats.Size = c.order.Len()
	if total := stats.Hits + stats.Misses; total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}
	return stats
}

// Cleanup removes expired entries and returns count of removed entries.
func (c *LRUCache[K, V]) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for elem := c.order.Back(); elem != nil; {
		prev := elem.Prev()
		entry := elem.Value.(*lruEntry[K, V])
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			c.removeElement(elem)
			c.stats.Expirations++
			removed++
		}
		elem = prev
	}
	c.stats.Size = c.order.Len()
	return removed
}

// All returns a Go 1.23+ iterator over all key-value pairs.
func (c *LRUCache[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		for elem := c.order.Front(); elem != nil; elem = elem.Next() {
			entry := elem.Value.(*lruEntry[K, V])
			// Skip expired entries
			if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
				continue
			}
			if !yield(entry.key, entry.value) {
				return
			}
		}
	}
}

// Collect returns all non-expired entries as a slice of key-value pairs.
func (c *LRUCache[K, V]) Collect() []functional.Pair[K, V] {
	var result []functional.Pair[K, V]
	for k, v := range c.All() {
		result = append(result, functional.NewPair(k, v))
	}
	return result
}

func (c *LRUCache[K, V]) evictOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
		c.stats.Evictions++
	}
}

func (c *LRUCache[K, V]) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry[K, V])
	delete(c.items, entry.key)
	c.order.Remove(elem)
	c.stats.Size = c.order.Len()
	if c.onEvict != nil {
		c.onEvict(entry.key, entry.value)
	}
}

// Peek retrieves a value without marking it as recently used.
func (c *LRUCache[K, V]) Peek(key K) functional.Option[V] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	elem, ok := c.items[key]
	if !ok {
		return functional.None[V]()
	}

	entry := elem.Value.(*lruEntry[K, V])
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return functional.None[V]()
	}
	return functional.Some(entry.value)
}

// Values returns all values in order from most to least recently used.
func (c *LRUCache[K, V]) Values() []V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	values := make([]V, 0, c.order.Len())
	for elem := c.order.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*lruEntry[K, V])
		if entry.expiresAt.IsZero() || time.Now().Before(entry.expiresAt) {
			values = append(values, entry.value)
		}
	}
	return values
}

// Resize changes the capacity of the cache.
func (c *LRUCache[K, V]) Resize(capacity int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if capacity <= 0 {
		capacity = 1
	}
	c.capacity = capacity
	c.stats.Capacity = capacity
	for c.order.Len() > capacity {
		c.evictOldest()
	}
}
