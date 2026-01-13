package cache

import (
	"container/list"
	"sync"
	"time"
)

// localEntry represents a cache entry with TTL.
type localEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// LocalCache provides a simple LRU cache for local fallback.
type LocalCache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

// NewLocalCache creates a new local cache with the specified capacity.
func NewLocalCache(capacity int) *LocalCache {
	if capacity <= 0 {
		capacity = 10000
	}
	return &LocalCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves a value from the local cache.
func (c *LocalCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}

	entry := elem.Value.(*localEntry)

	// Check expiration
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		c.removeElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)

	// Return a copy
	result := make([]byte, len(entry.value))
	copy(result, entry.value)
	return result, true
}

// Set stores a value in the local cache.
func (c *LocalCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Update existing entry
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		entry := elem.Value.(*localEntry)
		entry.value = make([]byte, len(value))
		copy(entry.value, value)
		entry.expiresAt = expiresAt
		return
	}

	// Evict if at capacity
	for c.order.Len() >= c.capacity {
		c.evictOldest()
	}

	// Add new entry
	entry := &localEntry{
		key:       key,
		value:     make([]byte, len(value)),
		expiresAt: expiresAt,
	}
	copy(entry.value, value)

	elem := c.order.PushFront(entry)
	c.items[key] = elem
}

// Delete removes a value from the local cache.
func (c *LocalCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return false
	}

	c.removeElement(elem)
	return true
}

// Clear removes all entries from the cache.
func (c *LocalCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
}

// Size returns the number of entries in the cache.
func (c *LocalCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

func (c *LocalCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*localEntry)
	delete(c.items, entry.key)
	c.order.Remove(elem)
}

func (c *LocalCache) evictOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}
