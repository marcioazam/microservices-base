// Package property contains property-based tests for the cache service.
// Feature: cache-microservice
package property

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/auth-platform/cache-service/internal/localcache"
)

// Property 17: Local Cache Consistency
// For any cache operation when local cache is enabled, the local cache SHALL behave
// identically to Redis cache with respect to TTL and eviction policies.
// Validates: Requirements 10.1, 10.2, 10.4
func TestProperty17_LocalCacheConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("Local cache SET/GET round-trip", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" || len(value) == 0 {
				return true
			}

			cache := localcache.New(localcache.Config{
				MaxSize:     1000,
				DefaultTTL:  time.Hour,
				CleanupTick: time.Minute,
			})
			defer cache.Close()

			// SET
			cache.Set(key, value, time.Hour)

			// GET
			result, ok := cache.Get(key)
			if !ok {
				return false
			}

			// Verify round-trip
			if len(result) != len(value) {
				return false
			}
			for i := range value {
				if result[i] != value[i] {
					return false
				}
			}
			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 1000 }),
	))

	properties.Property("Local cache TTL expiration", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" || len(value) == 0 {
				return true
			}

			cache := localcache.New(localcache.Config{
				MaxSize:     1000,
				DefaultTTL:  time.Hour,
				CleanupTick: time.Millisecond,
			})
			defer cache.Close()

			// SET with short TTL
			cache.Set(key, value, 10*time.Millisecond)

			// Verify exists
			_, ok := cache.Get(key)
			if !ok {
				return false
			}

			// Wait for expiration
			time.Sleep(20 * time.Millisecond)

			// Should be expired
			_, ok = cache.Get(key)
			return !ok
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 100 }),
	))

	properties.Property("Local cache LRU eviction", prop.ForAll(
		func(numEntries int) bool {
			if numEntries < 5 || numEntries > 20 {
				return true
			}

			maxSize := 5
			cache := localcache.New(localcache.Config{
				MaxSize:     maxSize,
				DefaultTTL:  time.Hour,
				CleanupTick: time.Minute,
			})
			defer cache.Close()

			// Add more entries than max size
			for i := 0; i < numEntries; i++ {
				key := string(rune('a' + i))
				cache.Set(key, []byte{byte(i)}, time.Hour)
			}

			// Size should not exceed max
			return cache.Size() <= maxSize
		},
		gen.IntRange(5, 20),
	))

	properties.Property("Local cache delete removes entry", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" || len(value) == 0 {
				return true
			}

			cache := localcache.New(localcache.Config{
				MaxSize:     1000,
				DefaultTTL:  time.Hour,
				CleanupTick: time.Minute,
			})
			defer cache.Close()

			// SET
			cache.Set(key, value, time.Hour)

			// Verify exists
			_, ok := cache.Get(key)
			if !ok {
				return false
			}

			// DELETE
			deleted := cache.Delete(key)
			if !deleted {
				return false
			}

			// Should not exist
			_, ok = cache.Get(key)
			return !ok
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 1000 }),
	))

	properties.TestingRun(t)
}
