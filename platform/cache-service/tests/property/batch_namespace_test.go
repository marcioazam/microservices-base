// Package property contains property-based tests for batch and namespace operations.
package property

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Property 5: Batch Operations Equivalence
// Validates: Requirements 1.6, 1.7
func TestProperty5_BatchOperationsEquivalence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("Batch SET/GET equivalent to individual operations", prop.ForAll(
		func(entries map[string][]byte) bool {
			if len(entries) == 0 {
				return true
			}

			filtered := make(map[string][]byte)
			for k, v := range entries {
				if k != "" && len(v) > 0 {
					filtered[k] = v
				}
			}
			if len(filtered) == 0 {
				return true
			}

			client := NewMockRedisClient()
			ctx := context.Background()

			if err := client.SetWithExpire(ctx, filtered, time.Hour); err != nil {
				return false
			}

			keys := make([]string, 0, len(filtered))
			for k := range filtered {
				keys = append(keys, k)
			}

			results, err := client.MGet(ctx, keys...)
			if err != nil {
				return false
			}

			for i, key := range keys {
				expected := filtered[key]
				result := results[i]
				if result == nil {
					return false
				}

				var actual []byte
				switch v := result.(type) {
				case string:
					actual = []byte(v)
				case []byte:
					actual = v
				default:
					return false
				}

				if len(actual) != len(expected) {
					return false
				}
				for j := range expected {
					if actual[j] != expected[j] {
						return false
					}
				}
			}

			return true
		},
		gen.MapOf(
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 20 }),
			gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 100 }),
		).SuchThat(func(m map[string][]byte) bool { return len(m) > 0 && len(m) <= 10 }),
	))

	properties.TestingRun(t)
}

// Property 12: Namespace Isolation
// Validates: Requirements 5.5
func TestProperty12_NamespaceIsolation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("Different namespaces are isolated", prop.ForAll(
		func(nsBase, key string, value1, value2 []byte) bool {
			// Ensure different namespaces by appending suffixes
			ns1 := nsBase + "_ns1"
			ns2 := nsBase + "_ns2"

			client := NewMockRedisClient()
			ctx := context.Background()

			key1 := ns1 + ":" + key
			key2 := ns2 + ":" + key

			if err := client.Set(ctx, key1, value1, time.Hour); err != nil {
				return false
			}

			if err := client.Set(ctx, key2, value2, time.Hour); err != nil {
				return false
			}

			result1, err := client.Get(ctx, key1)
			if err != nil {
				return false
			}

			result2, err := client.Get(ctx, key2)
			if err != nil {
				return false
			}

			if len(result1) != len(value1) || len(result2) != len(value2) {
				return false
			}

			for i := range value1 {
				if result1[i] != value1[i] {
					return false
				}
			}
			for i := range value2 {
				if result2[i] != value2[i] {
					return false
				}
			}

			return true
		},
		gen.Identifier(),                                  // Non-empty alphanumeric string for namespace
		gen.Identifier(),                                  // Non-empty alphanumeric string for key
		gen.SliceOfN(10, gen.UInt8()).Map(func(b []byte) []byte { // Ensure non-empty value
			if len(b) == 0 {
				return []byte("default1")
			}
			return b
		}),
		gen.SliceOfN(10, gen.UInt8()).Map(func(b []byte) []byte { // Ensure non-empty value
			if len(b) == 0 {
				return []byte("default2")
			}
			return b
		}),
	))

	properties.TestingRun(t)
}
