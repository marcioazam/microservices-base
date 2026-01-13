// Package cache provides decision caching for IAM Policy Service.
package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// LocalFallback provides an in-memory cache for local fallback.
type LocalFallback struct {
	mu      sync.RWMutex
	data    map[string]*cacheEntry
	maxSize int
	hits    atomic.Int64
	misses  atomic.Int64
}

type cacheEntry struct {
	decision  *Decision
	expiresAt time.Time
}

// NewLocalFallback creates a new local fallback cache.
func NewLocalFallback(maxSize int) *LocalFallback {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &LocalFallback{
		data:    make(map[string]*cacheEntry),
		maxSize: maxSize,
	}
}

// Get retrieves a decision from local cache.
func (lf *LocalFallback) Get(key string) (*Decision, bool) {
	lf.mu.RLock()
	entry, ok := lf.data[key]
	lf.mu.RUnlock()

	if !ok {
		lf.misses.Add(1)
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		lf.mu.Lock()
		delete(lf.data, key)
		lf.mu.Unlock()
		lf.misses.Add(1)
		return nil, false
	}

	lf.hits.Add(1)
	return entry.decision, true
}

// Set stores a decision in local cache.
func (lf *LocalFallback) Set(key string, decision *Decision, ttl time.Duration) {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	// Evict if at capacity
	if len(lf.data) >= lf.maxSize {
		lf.evictOldest()
	}

	lf.data[key] = &cacheEntry{
		decision:  decision,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a decision from local cache.
func (lf *LocalFallback) Delete(key string) {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	delete(lf.data, key)
}

// Clear removes all entries from local cache.
func (lf *LocalFallback) Clear() {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	lf.data = make(map[string]*cacheEntry)
}

// Stats returns cache statistics.
func (lf *LocalFallback) Stats() CacheStats {
	lf.mu.RLock()
	size := len(lf.data)
	lf.mu.RUnlock()

	return CacheStats{
		Hits:   lf.hits.Load(),
		Misses: lf.misses.Load(),
		Size:   size,
	}
}

// evictOldest removes the oldest entry (simple eviction strategy).
func (lf *LocalFallback) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range lf.data {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(lf.data, oldestKey)
	}
}
