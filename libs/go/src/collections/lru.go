package collections

import (
	"container/list"
	"sync"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// LRUCache is a thread-safe LRU cache.
type LRUCache[K comparable, V any] struct {
	capacity   int
	items      map[K]*list.Element
	order      *list.List
	mu         sync.RWMutex
	onEvict    func(K, V)
	ttl        time.Duration
}

type lruEntry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// NewLRUCache creates a new LRU cache.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		order:    list.New(),
	}
}

// WithTTL sets TTL for cache entries.
func (c *LRUCache[K, V]) WithTTL(ttl time.Duration) *LRUCache[K, V] {
	c.ttl = ttl
	return c
}

// WithEvictCallback sets eviction callback.
func (c *LRUCache[K, V]) WithEvictCallback(fn func(K, V)) *LRUCache[K, V] {
	c.onEvict = fn
	return c
}

// Get retrieves a value from the cache.
func (c *LRUCache[K, V]) Get(key K) functional.Option[V] {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return functional.None[V]()
	}

	entry := elem.Value.(*lruEntry[K, V])
	if c.ttl > 0 && time.Now().After(entry.expiresAt) {
		c.removeElement(elem)
		return functional.None[V]()
	}

	c.order.MoveToFront(elem)
	return functional.Some(entry.value)
}

// Put adds or updates a value in the cache.
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		entry := elem.Value.(*lruEntry[K, V])
		entry.value = value
		if c.ttl > 0 {
			entry.expiresAt = time.Now().Add(c.ttl)
		}
		return
	}

	if c.order.Len() >= c.capacity {
		c.evictOldest()
	}

	entry := &lruEntry[K, V]{
		key:   key,
		value: value,
	}
	if c.ttl > 0 {
		entry.expiresAt = time.Now().Add(c.ttl)
	}

	elem := c.order.PushFront(entry)
	c.items[key] = elem
}

// GetOrCompute gets value or computes and caches it.
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

// Contains checks if key exists.
func (c *LRUCache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Size returns current cache size.
func (c *LRUCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

// Clear removes all entries.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*list.Element)
	c.order.Init()
}

func (c *LRUCache[K, V]) evictOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *LRUCache[K, V]) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry[K, V])
	delete(c.items, entry.key)
	c.order.Remove(elem)
	if c.onEvict != nil {
		c.onEvict(entry.key, entry.value)
	}
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
