// Package lru provides a generic LRU cache.
package lru

import (
	"container/list"
	"sync"
)

// Cache is a generic LRU cache.
type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	order    *list.List
	mu       sync.RWMutex
	onEvict  func(K, V)
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// New creates a new LRU cache with the given capacity.
func New[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		order:    list.New(),
	}
}

// WithEvictCallback sets a callback for evicted items.
func (c *Cache[K, V]) WithEvictCallback(fn func(K, V)) *Cache[K, V] {
	c.onEvict = fn
	return c
}

// Set adds or updates a value.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*entry[K, V]).value = value
		return
	}

	if c.order.Len() >= c.capacity {
		c.evictOldest()
	}

	e := &entry[K, V]{key: key, value: value}
	elem := c.order.PushFront(e)
	c.items[key] = elem
}

// Get retrieves a value and marks it as recently used.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		return elem.Value.(*entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Peek retrieves a value without marking it as recently used.
func (c *Cache[K, V]) Peek(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if elem, ok := c.items[key]; ok {
		return elem.Value.(*entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Contains checks if a key exists without marking it as recently used.
func (c *Cache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Remove removes a key from the cache.
func (c *Cache[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
		return true
	}
	return false
}

// Keys returns all keys in order from most to least recently used.
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, c.order.Len())
	for elem := c.order.Front(); elem != nil; elem = elem.Next() {
		keys = append(keys, elem.Value.(*entry[K, V]).key)
	}
	return keys
}

// Values returns all values in order from most to least recently used.
func (c *Cache[K, V]) Values() []V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	values := make([]V, 0, c.order.Len())
	for elem := c.order.Front(); elem != nil; elem = elem.Next() {
		values = append(values, elem.Value.(*entry[K, V]).value)
	}
	return values
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

// Clear removes all items from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.onEvict != nil {
		for elem := c.order.Front(); elem != nil; elem = elem.Next() {
			e := elem.Value.(*entry[K, V])
			c.onEvict(e.key, e.value)
		}
	}

	c.items = make(map[K]*list.Element)
	c.order.Init()
}

// Resize changes the capacity of the cache.
func (c *Cache[K, V]) Resize(capacity int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = capacity
	for c.order.Len() > capacity {
		c.evictOldest()
	}
}

// GetOrSet gets a value or sets it if not present.
func (c *Cache[K, V]) GetOrSet(key K, value V) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		return elem.Value.(*entry[K, V]).value, true
	}

	if c.order.Len() >= c.capacity {
		c.evictOldest()
	}

	e := &entry[K, V]{key: key, value: value}
	elem := c.order.PushFront(e)
	c.items[key] = elem
	return value, false
}

func (c *Cache[K, V]) evictOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *Cache[K, V]) removeElement(elem *list.Element) {
	c.order.Remove(elem)
	e := elem.Value.(*entry[K, V])
	delete(c.items, e.key)
	if c.onEvict != nil {
		c.onEvict(e.key, e.value)
	}
}
