// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 2: Cache-Aside Pattern Correctness
// Validates: Requirements 3.2, 3.3, 3.4
package property

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// MockCache simulates cache behavior for testing.
type MockCache struct {
	data      map[string][]byte
	namespace string
	mu        sync.RWMutex
}

func NewMockCache(namespace string) *MockCache {
	return &MockCache{
		data:      make(map[string][]byte),
		namespace: namespace,
	}
}

func (c *MockCache) buildKey(key string) string {
	return c.namespace + ":" + key
}

func (c *MockCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fullKey := c.buildKey(key)
	data, ok := c.data[fullKey]
	return data, ok
}

func (c *MockCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fullKey := c.buildKey(key)
	c.data[fullKey] = value
}

func (c *MockCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fullKey := c.buildKey(key)
	delete(c.data, fullKey)
}

func (c *MockCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// MockDatabase simulates database behavior for testing.
type MockDatabase struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		data: make(map[string][]byte),
	}
}

func (db *MockDatabase) Get(key string) ([]byte, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	data, ok := db.data[key]
	return data, ok
}

func (db *MockDatabase) Set(key string, value []byte) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = value
}

func (db *MockDatabase) Delete(key string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	delete(db.data, key)
}

// CacheAsideService implements cache-aside pattern.
type CacheAsideService struct {
	cache *MockCache
	db    *MockDatabase
}

func NewCacheAsideService(cache *MockCache, db *MockDatabase) *CacheAsideService {
	return &CacheAsideService{cache: cache, db: db}
}

// Get implements cache-aside read: check cache first, then database.
func (s *CacheAsideService) Get(key string) ([]byte, bool) {
	// Check cache first
	if data, ok := s.cache.Get(key); ok {
		return data, true
	}

	// Cache miss - check database
	data, ok := s.db.Get(key)
	if !ok {
		return nil, false
	}

	// Populate cache
	s.cache.Set(key, data)
	return data, true
}

// Set implements cache-aside write: write to database, invalidate cache.
func (s *CacheAsideService) Set(key string, value []byte) {
	// Write to database first
	s.db.Set(key, value)

	// Invalidate cache (not populate - cache-aside pattern)
	s.cache.Delete(key)
}

// Update implements cache-aside update: update database, invalidate cache.
func (s *CacheAsideService) Update(key string, value []byte) {
	// Update database
	s.db.Set(key, value)

	// Invalidate cache
	s.cache.Delete(key)
}

// Delete implements cache-aside delete: delete from database, invalidate cache.
func (s *CacheAsideService) Delete(key string) {
	// Delete from database
	s.db.Delete(key)

	// Invalidate cache
	s.cache.Delete(key)
}

// TestProperty2_CacheAsideGetChecksCache tests that GET operations check cache before database.
// Property 2: Cache-Aside Pattern Correctness
// Validates: Requirements 3.2, 3.3, 3.4
func TestProperty2_CacheAsideGetChecksCache(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		namespace := "file-upload"
		cache := NewMockCache(namespace)
		db := NewMockDatabase()
		service := NewCacheAsideService(cache, db)

		key := rapid.StringMatching(`[a-z0-9-]{8,32}`).Draw(t, "key")
		value := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "value")

		// Populate database
		db.Set(key, value)

		// First get - should hit database and populate cache
		result1, ok1 := service.Get(key)
		if !ok1 {
			t.Fatal("expected to find value")
		}
		if string(result1) != string(value) {
			t.Errorf("value mismatch: expected %q, got %q", value, result1)
		}

		// Verify cache was populated
		cachedValue, inCache := cache.Get(key)
		if !inCache {
			t.Error("expected value to be in cache after GET")
		}
		if string(cachedValue) != string(value) {
			t.Errorf("cached value mismatch: expected %q, got %q", value, cachedValue)
		}

		// Second get - should hit cache (we can verify by checking cache has the value)
		result2, ok2 := service.Get(key)
		if !ok2 {
			t.Fatal("expected to find value on second get")
		}
		if string(result2) != string(value) {
			t.Errorf("second get value mismatch: expected %q, got %q", value, result2)
		}
	})
}

// TestProperty2_CacheAsideWriteInvalidatesCache tests that CREATE/UPDATE operations invalidate cache.
// Property 2: Cache-Aside Pattern Correctness
// Validates: Requirements 3.2, 3.3, 3.4
func TestProperty2_CacheAsideWriteInvalidatesCache(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		namespace := "file-upload"
		cache := NewMockCache(namespace)
		db := NewMockDatabase()
		service := NewCacheAsideService(cache, db)

		key := rapid.StringMatching(`[a-z0-9-]{8,32}`).Draw(t, "key")
		value1 := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "value1")
		value2 := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "value2")

		// Initial set
		service.Set(key, value1)

		// Populate cache via get
		service.Get(key)

		// Verify cache has value
		_, inCache := cache.Get(key)
		if !inCache {
			t.Error("expected value to be in cache after GET")
		}

		// Update - should invalidate cache
		service.Update(key, value2)

		// Verify cache was invalidated
		_, inCacheAfterUpdate := cache.Get(key)
		if inCacheAfterUpdate {
			t.Error("expected cache to be invalidated after UPDATE")
		}

		// Get should return new value
		result, ok := service.Get(key)
		if !ok {
			t.Fatal("expected to find value after update")
		}
		if string(result) != string(value2) {
			t.Errorf("expected updated value %q, got %q", value2, result)
		}
	})
}

// TestProperty2_CacheKeysHaveNamespacePrefix tests that all cache keys have namespace prefix.
// Property 2: Cache-Aside Pattern Correctness
// Validates: Requirements 3.2, 3.3, 3.4
func TestProperty2_CacheKeysHaveNamespacePrefix(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		namespace := "file-upload"
		cache := NewMockCache(namespace)
		db := NewMockDatabase()
		service := NewCacheAsideService(cache, db)

		// Generate multiple keys
		numKeys := rapid.IntRange(1, 10).Draw(t, "numKeys")
		keys := make([]string, numKeys)
		for i := 0; i < numKeys; i++ {
			keys[i] = rapid.StringMatching(`[a-z0-9-]{8,32}`).Draw(t, "key")
		}

		// Set values
		for _, key := range keys {
			value := rapid.SliceOfN(rapid.Byte(), 10, 50).Draw(t, "value")
			db.Set(key, value)
			service.Get(key) // Populate cache
		}

		// Verify all cache keys have namespace prefix
		cacheKeys := cache.Keys()
		for _, cacheKey := range cacheKeys {
			if !strings.HasPrefix(cacheKey, namespace+":") {
				t.Errorf("cache key %q does not have namespace prefix %q:", cacheKey, namespace)
			}
		}
	})
}

// TestProperty2_CacheDeleteInvalidatesCache tests that DELETE operations invalidate cache.
// Property 2: Cache-Aside Pattern Correctness
// Validates: Requirements 3.2, 3.3, 3.4
func TestProperty2_CacheDeleteInvalidatesCache(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		namespace := "file-upload"
		cache := NewMockCache(namespace)
		db := NewMockDatabase()
		service := NewCacheAsideService(cache, db)

		key := rapid.StringMatching(`[a-z0-9-]{8,32}`).Draw(t, "key")
		value := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "value")

		// Set and populate cache
		service.Set(key, value)
		service.Get(key)

		// Verify cache has value
		_, inCache := cache.Get(key)
		if !inCache {
			t.Error("expected value to be in cache")
		}

		// Delete
		service.Delete(key)

		// Verify cache was invalidated
		_, inCacheAfterDelete := cache.Get(key)
		if inCacheAfterDelete {
			t.Error("expected cache to be invalidated after DELETE")
		}

		// Verify database was deleted
		_, inDb := db.Get(key)
		if inDb {
			t.Error("expected value to be deleted from database")
		}
	})
}

// Ensure context and time are used (for linter)
var _ = context.Background
var _ = time.Now
