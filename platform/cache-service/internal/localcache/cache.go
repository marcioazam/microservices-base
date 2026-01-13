// Package localcache provides in-memory caching with TTL and LRU eviction.
package localcache

import (
	"container/list"
	"sync"
	"time"
)

// entry represents a cache entry with metadata.
type entry struct {
	key        string
	value      []byte
	expiresAt  time.Time
	accessedAt time.Time
	element    *list.Element
}

// Cache implements an in-memory cache with LRU eviction and TTL support.
type Cache struct {
	mu          sync.RWMutex
	data        map[string]*entry
	lruList     *list.List
	maxSize     int
	defaultTTL  time.Duration
	cleanupTick time.Duration
	stopCleanup chan struct{}
}

// Config holds local cache configuration.
type Config struct {
	MaxSize     int
	DefaultTTL  time.Duration
	CleanupTick time.Duration
}

// DefaultConfig returns the default local cache configuration.
func DefaultConfig() Config {
	return Config{
		MaxSize:     10000,
		DefaultTTL:  time.Hour,
		CleanupTick: time.Minute,
	}
}

// New creates a new local cache with the given configuration.
func New(config Config) *Cache {
	// Apply defaults for zero values
	if config.MaxSize <= 0 {
		config.MaxSize = 10000
	}
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = time.Hour
	}
	if config.CleanupTick <= 0 {
		config.CleanupTick = time.Minute
	}

	c := &Cache{
		data:        make(map[string]*entry),
		lruList:     list.New(),
		maxSize:     config.MaxSize,
		defaultTTL:  config.DefaultTTL,
		cleanupTick: config.CleanupTick,
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Get retrieves a value from the cache.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.data[key]
	if !ok {
		return nil, false
	}

	// Check expiration
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		c.removeEntry(e)
		return nil, false
	}

	// Update LRU
	e.accessedAt = time.Now()
	c.lruList.MoveToFront(e.element)

	// Return a copy to prevent mutation
	result := make([]byte, len(e.value))
	copy(result, e.value)

	return result, true
}

// Set stores a value in the cache with TTL.
func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Check if key exists
	if e, ok := c.data[key]; ok {
		// Update existing entry
		e.value = make([]byte, len(value))
		copy(e.value, value)
		e.expiresAt = expiresAt
		e.accessedAt = time.Now()
		c.lruList.MoveToFront(e.element)
		return
	}

	// Evict if at capacity
	for c.lruList.Len() >= c.maxSize {
		c.evictOldest()
	}

	// Create new entry
	e := &entry{
		key:        key,
		value:      make([]byte, len(value)),
		expiresAt:  expiresAt,
		accessedAt: time.Now(),
	}
	copy(e.value, value)

	e.element = c.lruList.PushFront(e)
	c.data[key] = e
}

// Delete removes a value from the cache.
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.data[key]
	if !ok {
		return false
	}

	c.removeEntry(e)
	return true
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*entry)
	c.lruList = list.New()
}

// Size returns the number of entries in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Close stops the cleanup goroutine.
func (c *Cache) Close() {
	close(c.stopCleanup)
}

func (c *Cache) removeEntry(e *entry) {
	c.lruList.Remove(e.element)
	delete(c.data, e.key)
}

func (c *Cache) evictOldest() {
	oldest := c.lruList.Back()
	if oldest == nil {
		return
	}

	e := oldest.Value.(*entry)
	c.removeEntry(e)
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *Cache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var toRemove []*entry

	for _, e := range c.data {
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			toRemove = append(toRemove, e)
		}
	}

	for _, e := range toRemove {
		c.removeEntry(e)
	}
}

// Keys returns all keys in the cache (for debugging/testing).
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}
