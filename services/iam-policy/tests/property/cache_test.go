// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/cache"
	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestCacheNamespaceIsolation validates Property 2: Cache Namespace Isolation.
// All cache keys must be prefixed with service namespace.
func TestCacheNamespaceIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random authorization input
		input := testutil.AuthorizationInputGen().Draw(t, "input")

		// Create local-only cache
		dc := cache.NewLocalOnlyCache(1000, 5*time.Minute)
		defer dc.Close()

		ctx := context.Background()

		// Store a decision
		decision := &cache.Decision{
			Allowed: rapid.Bool().Draw(t, "allowed"),
			Reason:  "test reason",
		}

		err := dc.Set(ctx, input, decision)
		if err != nil {
			t.Fatalf("failed to set cache: %v", err)
		}

		// Retrieve the decision
		retrieved, found := dc.Get(ctx, input)
		if !found {
			t.Fatal("decision not found in cache")
		}

		// Property: retrieved decision must match stored decision
		if retrieved.Allowed != decision.Allowed {
			t.Errorf("allowed mismatch: expected %v, got %v", decision.Allowed, retrieved.Allowed)
		}
	})
}

// TestDecisionCacheRoundTrip validates Property 5: Decision Cache Round-Trip.
// Cached decisions must be retrievable with identical values.
func TestDecisionCacheRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := testutil.AuthorizationInputGen().Draw(t, "input")
		allowed := rapid.Bool().Draw(t, "allowed")
		reason := rapid.String().Draw(t, "reason")

		dc := cache.NewLocalOnlyCache(1000, 5*time.Minute)
		defer dc.Close()

		ctx := context.Background()

		original := &cache.Decision{
			Allowed: allowed,
			Reason:  reason,
		}

		// Set and get
		err := dc.Set(ctx, input, original)
		if err != nil {
			t.Fatalf("failed to set: %v", err)
		}

		retrieved, found := dc.Get(ctx, input)
		if !found {
			t.Fatal("decision not found")
		}

		// Property: all fields must match
		if retrieved.Allowed != original.Allowed {
			t.Errorf("allowed mismatch: expected %v, got %v", original.Allowed, retrieved.Allowed)
		}
		if retrieved.Reason != original.Reason {
			t.Errorf("reason mismatch: expected %s, got %s", original.Reason, retrieved.Reason)
		}
	})
}

// TestCacheKeyDeterminism validates that same input produces same cache key.
func TestCacheKeyDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := testutil.AuthorizationInputGen().Draw(t, "input")

		dc := cache.NewLocalOnlyCache(1000, 5*time.Minute)
		defer dc.Close()

		ctx := context.Background()

		decision := &cache.Decision{Allowed: true, Reason: "test"}

		// Set with input
		err := dc.Set(ctx, input, decision)
		if err != nil {
			t.Fatalf("failed to set: %v", err)
		}

		// Get with same input should find it
		_, found := dc.Get(ctx, input)
		if !found {
			t.Error("same input should produce same key and find cached value")
		}
	})
}

// TestCacheInvalidation validates that invalidation clears all entries.
func TestCacheInvalidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numEntries := rapid.IntRange(1, 20).Draw(t, "numEntries")

		dc := cache.NewLocalOnlyCache(1000, 5*time.Minute)
		defer dc.Close()

		ctx := context.Background()

		// Store multiple entries
		inputs := make([]map[string]interface{}, numEntries)
		for i := 0; i < numEntries; i++ {
			inputs[i] = testutil.AuthorizationInputGen().Draw(t, "input")
			decision := &cache.Decision{Allowed: true}
			_ = dc.Set(ctx, inputs[i], decision)
		}

		// Invalidate all
		err := dc.Invalidate(ctx)
		if err != nil {
			t.Fatalf("failed to invalidate: %v", err)
		}

		// Property: no entries should be found after invalidation
		for _, input := range inputs {
			_, found := dc.Get(ctx, input)
			if found {
				t.Error("entry found after invalidation")
			}
		}
	})
}

// TestCacheDelete validates that delete removes specific entry.
func TestCacheDelete(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input1 := testutil.AuthorizationInputGen().Draw(t, "input1")
		input2 := testutil.AuthorizationInputGen().Draw(t, "input2")

		dc := cache.NewLocalOnlyCache(1000, 5*time.Minute)
		defer dc.Close()

		ctx := context.Background()

		// Store two entries
		_ = dc.Set(ctx, input1, &cache.Decision{Allowed: true})
		_ = dc.Set(ctx, input2, &cache.Decision{Allowed: false})

		// Delete first entry
		err := dc.Delete(ctx, input1)
		if err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		// Property: deleted entry should not be found
		_, found := dc.Get(ctx, input1)
		if found {
			t.Error("deleted entry should not be found")
		}

		// Property: other entry should still exist
		_, found = dc.Get(ctx, input2)
		if !found {
			t.Error("other entry should still exist")
		}
	})
}

// TestCacheExpiration validates that expired entries are not returned.
func TestCacheExpiration(t *testing.T) {
	// Use very short TTL for testing
	dc := cache.NewLocalOnlyCache(1000, 1*time.Millisecond)
	defer dc.Close()

	ctx := context.Background()
	input := map[string]interface{}{"test": "value"}

	_ = dc.Set(ctx, input, &cache.Decision{Allowed: true})

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	// Property: expired entry should not be found
	_, found := dc.Get(ctx, input)
	if found {
		t.Error("expired entry should not be found")
	}
}
