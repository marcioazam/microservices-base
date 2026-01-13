// Package property contains property-based tests for the cache service.
// Feature: cache-microservice
package property

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Property 1: Cache Round-Trip Consistency
// Validates: Requirements 1.1, 1.2, 2.1
func TestProperty1_CacheRoundTripConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("SET then GET returns same value", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" || len(value) == 0 {
				return true
			}

			client := NewMockRedisClient()
			ctx := context.Background()

			if err := client.Set(ctx, key, value, time.Hour); err != nil {
				return false
			}

			result, err := client.Get(ctx, key)
			if err != nil {
				return false
			}

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

	properties.TestingRun(t)
}

// Property 2: Cache Miss for Non-Existent Keys
// Validates: Requirements 1.3
func TestProperty2_CacheMissForNonExistentKeys(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("GET on non-existent key returns not found", prop.ForAll(
		func(key string) bool {
			if key == "" {
				return true
			}

			client := NewMockRedisClient()
			ctx := context.Background()

			_, err := client.Get(ctx, key)
			return isNotFound(err)
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
	))

	properties.TestingRun(t)
}

// Property 3: Delete Removes Entries
// Validates: Requirements 1.4
func TestProperty3_DeleteRemovesEntries(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("DELETE removes entry", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" || len(value) == 0 {
				return true
			}

			client := NewMockRedisClient()
			ctx := context.Background()

			if err := client.Set(ctx, key, value, time.Hour); err != nil {
				return false
			}

			if _, err := client.Get(ctx, key); err != nil {
				return false
			}

			count, err := client.Del(ctx, key)
			if err != nil || count != 1 {
				return false
			}

			_, err = client.Get(ctx, key)
			return isNotFound(err)
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 1000 }),
	))

	properties.TestingRun(t)
}

// Property 4: TTL Expiration
// Validates: Requirements 1.5, 2.2
func TestProperty4_TTLExpiration(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("Entry expires after TTL", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" || len(value) == 0 {
				return true
			}

			client := NewMockRedisClient()
			ctx := context.Background()
			ttl := 10 * time.Millisecond

			if err := client.Set(ctx, key, value, ttl); err != nil {
				return false
			}

			if _, err := client.Get(ctx, key); err != nil {
				return false
			}

			time.Sleep(20 * time.Millisecond)

			_, err := client.Get(ctx, key)
			return isNotFound(err)
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 100 }),
	))

	properties.TestingRun(t)
}
