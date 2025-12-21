// Feature: go-libs-state-of-art-2025, Property 1: LRU Cache Correctness
// Validates: Requirements 1.3, 1.4, 1.5
package collections_test

import (
	"sync"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/collections"
	"pgregory.net/rapid"
)

// TestLRUCacheCapacityInvariant verifies cache size never exceeds capacity.
func TestLRUCacheCapacityInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, 100).Draw(t, "capacity")
		cache := collections.NewLRUCache[int, string](capacity)

		numOps := rapid.IntRange(1, 500).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			key := rapid.IntRange(0, 200).Draw(t, "key")
			value := rapid.String().Draw(t, "value")
			cache.Put(key, value)

			if cache.Size() > capacity {
				t.Fatalf("cache size %d exceeds capacity %d", cache.Size(), capacity)
			}
		}
	})
}

// TestLRUCacheGetReturnsCorrectValue verifies Get returns Some(v) iff key was Put.
func TestLRUCacheGetReturnsCorrectValue(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(10, 50).Draw(t, "capacity")
		cache := collections.NewLRUCache[string, int](capacity)

		// Track what we put in (limited to capacity to avoid eviction)
		stored := make(map[string]int)
		numPuts := rapid.IntRange(1, capacity).Draw(t, "numPuts")

		for i := 0; i < numPuts; i++ {
			key := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "key")
			value := rapid.Int().Draw(t, "value")
			cache.Put(key, value)
			stored[key] = value
		}

		// Verify all stored values are retrievable
		for key, expectedValue := range stored {
			opt := cache.Get(key)
			if opt.IsNone() {
				t.Fatalf("expected Some for key %q, got None", key)
			}
			if opt.Unwrap() != expectedValue {
				t.Fatalf("expected value %d for key %q, got %d", expectedValue, key, opt.Unwrap())
			}
		}

		// Verify non-existent key returns None
		opt := cache.Get("nonexistent_key_xyz")
		if opt.IsSome() {
			t.Fatal("expected None for non-existent key, got Some")
		}
	})
}

// TestLRUCacheGetOrComputeAlwaysReturnsValue verifies GetOrCompute always returns.
func TestLRUCacheGetOrComputeAlwaysReturnsValue(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 20).Draw(t, "capacity")
		cache := collections.NewLRUCache[int, int](capacity)

		numOps := rapid.IntRange(10, 100).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			key := rapid.IntRange(0, 30).Draw(t, "key")
			computeValue := rapid.Int().Draw(t, "computeValue")

			result := cache.GetOrCompute(key, func() int {
				return computeValue
			})

			// GetOrCompute must always return a value
			// Either the cached value or the computed value
			opt := cache.Get(key)
			if opt.IsNone() {
				t.Fatalf("GetOrCompute should have cached the value for key %d", key)
			}
			if opt.Unwrap() != result {
				t.Fatalf("GetOrCompute returned %d but cache has %d", result, opt.Unwrap())
			}
		}
	})
}

// TestLRUCacheTTLExpiration verifies TTL expiration removes entries correctly.
func TestLRUCacheTTLExpiration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 20).Draw(t, "capacity")
		cache := collections.NewLRUCache[int, string](capacity).WithTTL(10 * time.Millisecond)

		// Use numPuts <= capacity to avoid eviction before TTL check
		numPuts := rapid.IntRange(1, capacity).Draw(t, "numPuts")
		for i := 0; i < numPuts; i++ {
			cache.Put(i, "value")
		}

		// Verify entries exist before expiration
		for i := 0; i < numPuts; i++ {
			if cache.Get(i).IsNone() {
				t.Fatalf("entry %d should exist before TTL expiration", i)
			}
		}

		// Wait for TTL to expire
		time.Sleep(15 * time.Millisecond)

		// Verify entries are expired
		for i := 0; i < numPuts; i++ {
			if cache.Get(i).IsSome() {
				t.Fatalf("entry %d should be expired after TTL", i)
			}
		}
	})
}

// TestLRUCacheEvictionCallback verifies callback is called for every eviction.
func TestLRUCacheEvictionCallback(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(3, 10).Draw(t, "capacity")

		var mu sync.Mutex
		evicted := make(map[int]string)

		cache := collections.NewLRUCache[int, string](capacity).
			WithEvictCallback(func(k int, v string) {
				mu.Lock()
				evicted[k] = v
				mu.Unlock()
			})

		// Fill cache beyond capacity to trigger evictions
		numPuts := capacity + rapid.IntRange(1, 20).Draw(t, "extraPuts")
		for i := 0; i < numPuts; i++ {
			cache.Put(i, "value")
		}

		// Verify eviction count matches expected
		expectedEvictions := numPuts - capacity
		mu.Lock()
		actualEvictions := len(evicted)
		mu.Unlock()

		if actualEvictions != expectedEvictions {
			t.Fatalf("expected %d evictions, got %d", expectedEvictions, actualEvictions)
		}

		// Verify evicted keys are no longer in cache
		mu.Lock()
		for k := range evicted {
			if cache.Contains(k) {
				t.Fatalf("evicted key %d should not be in cache", k)
			}
		}
		mu.Unlock()
	})
}

// TestLRUCacheLRUOrdering verifies least recently used items are evicted first.
func TestLRUCacheLRUOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := 5
		cache := collections.NewLRUCache[int, string](capacity)

		// Fill cache
		for i := 0; i < capacity; i++ {
			cache.Put(i, "value")
		}

		// Access key 0 to make it recently used
		cache.Get(0)

		// Add new item, should evict key 1 (least recently used)
		cache.Put(100, "new")

		// Key 0 should still exist (was accessed)
		if cache.Get(0).IsNone() {
			t.Fatal("key 0 should exist (was recently accessed)")
		}

		// Key 1 should be evicted (least recently used)
		if cache.Get(1).IsSome() {
			t.Fatal("key 1 should be evicted (least recently used)")
		}
	})
}

// TestLRUCacheIteratorCorrectness verifies All() iterator yields correct entries.
func TestLRUCacheIteratorCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 20).Draw(t, "capacity")
		cache := collections.NewLRUCache[int, int](capacity)

		numPuts := rapid.IntRange(1, capacity).Draw(t, "numPuts")
		expected := make(map[int]int)
		for i := 0; i < numPuts; i++ {
			cache.Put(i, i*10)
			expected[i] = i * 10
		}

		// Collect via iterator
		collected := make(map[int]int)
		for k, v := range cache.All() {
			collected[k] = v
		}

		// Verify iterator yields all entries
		if len(collected) != len(expected) {
			t.Fatalf("iterator yielded %d entries, expected %d", len(collected), len(expected))
		}

		for k, v := range expected {
			if collected[k] != v {
				t.Fatalf("iterator missing or wrong value for key %d", k)
			}
		}
	})
}

// TestLRUCacheStatsAccuracy verifies statistics are accurate.
func TestLRUCacheStatsAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 15).Draw(t, "capacity")
		cache := collections.NewLRUCache[int, int](capacity)

		// Perform operations
		numPuts := rapid.IntRange(1, capacity).Draw(t, "numPuts")
		for i := 0; i < numPuts; i++ {
			cache.Put(i, i)
		}

		// Perform gets (some hits, some misses)
		hits := 0
		misses := 0
		numGets := rapid.IntRange(10, 50).Draw(t, "numGets")
		for i := 0; i < numGets; i++ {
			key := rapid.IntRange(0, numPuts*2).Draw(t, "getKey")
			if cache.Get(key).IsSome() {
				hits++
			} else {
				misses++
			}
		}

		stats := cache.Stats()
		if stats.Hits != int64(hits) {
			t.Fatalf("expected %d hits, got %d", hits, stats.Hits)
		}
		if stats.Misses != int64(misses) {
			t.Fatalf("expected %d misses, got %d", misses, stats.Misses)
		}
	})
}

// TestLRUCacheConcurrentAccess verifies thread-safety.
func TestLRUCacheConcurrentAccess(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(10, 50).Draw(t, "capacity")
		cache := collections.NewLRUCache[int, int](capacity)

		var wg sync.WaitGroup
		numGoroutines := rapid.IntRange(5, 20).Draw(t, "numGoroutines")
		opsPerGoroutine := rapid.IntRange(50, 200).Draw(t, "opsPerGoroutine")

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < opsPerGoroutine; i++ {
					key := (id*100 + i) % (capacity * 2)
					cache.Put(key, i)
					cache.Get(key)
					cache.Contains(key)
				}
			}(g)
		}

		wg.Wait()

		// Cache should still be valid
		if cache.Size() > capacity {
			t.Fatalf("cache size %d exceeds capacity %d after concurrent access", cache.Size(), capacity)
		}
	})
}
