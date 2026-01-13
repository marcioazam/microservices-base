// Package property contains property-based tests.
package property

import (
	"sync"
	"testing"
	"time"

	"github.com/auth-platform/cache-service/internal/localcache"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestConcurrentOperationsSafety tests Property 8: Concurrent Operations Safety.
// Validates: Requirements 3.3
func TestConcurrentOperationsSafety(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property: Concurrent writes to same key don't cause data corruption
	properties.Property("concurrent writes are safe", prop.ForAll(
		func(key string, values []string) bool {
			cache := localcache.New(localcache.Config{
				MaxSize:    1000,
				DefaultTTL: time.Minute,
			})

			var wg sync.WaitGroup
			for _, v := range values {
				wg.Add(1)
				go func(val string) {
					defer wg.Done()
					cache.Set(key, []byte(val), time.Minute)
				}(v)
			}
			wg.Wait()

			// After all writes, key should exist with one of the values
			result, found := cache.Get(key)
			if !found {
				return false
			}

			// Result should be one of the written values
			resultStr := string(result)
			for _, v := range values {
				if resultStr == v {
					return true
				}
			}
			return false
		},
		gen.Identifier(),                   // Non-empty alphanumeric key
		gen.SliceOfN(10, gen.Identifier()), // Non-empty alphanumeric values
	))

	// Property: Concurrent reads and writes don't panic
	properties.Property("concurrent read-write is safe", prop.ForAll(
		func(keys []string) bool {
			cache := localcache.New(localcache.Config{
				MaxSize:    1000,
				DefaultTTL: time.Minute,
			})

			var wg sync.WaitGroup
			for i, key := range keys {
				wg.Add(2)
				go func(k string, idx int) {
					defer wg.Done()
					cache.Set(k, []byte("value"), time.Minute)
				}(key, i)
				go func(k string) {
					defer wg.Done()
					cache.Get(k)
				}(key)
			}
			wg.Wait()
			return true // No panic means success
		},
		gen.SliceOfN(20, gen.Identifier()), // Non-empty alphanumeric keys
	))

	// Property: Concurrent deletes are safe
	properties.Property("concurrent deletes are safe", prop.ForAll(
		func(key string) bool {
			cache := localcache.New(localcache.Config{
				MaxSize:    1000,
				DefaultTTL: time.Minute,
			})

			cache.Set(key, []byte("value"), time.Minute)

			var wg sync.WaitGroup
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					cache.Delete(key)
				}()
			}
			wg.Wait()

			_, found := cache.Get(key)
			return !found // Key should be deleted
		},
		gen.Identifier(), // Non-empty alphanumeric key
	))

	properties.TestingRun(t)
}
