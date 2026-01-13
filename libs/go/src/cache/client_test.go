package cache

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property 1: Cache Round-Trip Consistency
// For any valid key and value, setting a value in the cache and then getting it
// should return the same value.
// Validates: Requirements 1.1, 5.10, 8.1
func TestProperty_CacheRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random key and value
		key := rapid.StringMatching(`[a-zA-Z0-9_-]{1,100}`).Draw(t, "key")
		value := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "value")

		// Create local-only client for testing
		client := LocalOnly(10000)
		defer client.Close()

		ctx := context.Background()

		// Set the value
		err := client.Set(ctx, key, value, 5*time.Minute)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Get the value back
		result := client.Get(ctx, key)
		if !result.IsOk() {
			t.Fatalf("Get failed: %v", result.UnwrapErr())
		}

		entry := result.Unwrap()

		// Verify round-trip consistency
		if len(entry.Value) != len(value) {
			t.Fatalf("Value length mismatch: got %d, want %d", len(entry.Value), len(value))
		}

		for i := range value {
			if entry.Value[i] != value[i] {
				t.Fatalf("Value mismatch at index %d: got %d, want %d", i, entry.Value[i], value[i])
			}
		}
	})
}

// Property 2: Namespace Isolation
// For any two different namespaces and the same key, setting a value in one
// namespace should not affect the value in another namespace.
// Validates: Requirements 1.2
func TestProperty_NamespaceIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random namespaces and key
		ns1 := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "ns1")
		ns2 := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "ns2")

		// Ensure namespaces are different
		if ns1 == ns2 {
			ns2 = ns2 + "_different"
		}

		key := rapid.StringMatching(`[a-zA-Z0-9_-]{1,50}`).Draw(t, "key")
		value1 := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value1")
		value2 := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value2")

		// Create two clients with different namespaces
		config1 := DefaultConfig()
		config1.Namespace = ns1
		config1.LocalFallback = true
		client1 := &Client{
			config:     config1,
			localCache: NewLocalCache(10000),
		}
		defer client1.Close()

		config2 := DefaultConfig()
		config2.Namespace = ns2
		config2.LocalFallback = true
		client2 := &Client{
			config:     config2,
			localCache: NewLocalCache(10000),
		}
		defer client2.Close()

		ctx := context.Background()

		// Set value1 in namespace1
		err := client1.Set(ctx, key, value1, 5*time.Minute)
		if err != nil {
			t.Fatalf("Set in ns1 failed: %v", err)
		}

		// Set value2 in namespace2
		err = client2.Set(ctx, key, value2, 5*time.Minute)
		if err != nil {
			t.Fatalf("Set in ns2 failed: %v", err)
		}

		// Get from namespace1 should return value1
		result1 := client1.Get(ctx, key)
		if !result1.IsOk() {
			t.Fatalf("Get from ns1 failed: %v", result1.UnwrapErr())
		}

		entry1 := result1.Unwrap()
		for i := range value1 {
			if entry1.Value[i] != value1[i] {
				t.Fatalf("Namespace isolation violated: ns1 value changed")
			}
		}

		// Get from namespace2 should return value2
		result2 := client2.Get(ctx, key)
		if !result2.IsOk() {
			t.Fatalf("Get from ns2 failed: %v", result2.UnwrapErr())
		}

		entry2 := result2.Unwrap()
		for i := range value2 {
			if entry2.Value[i] != value2[i] {
				t.Fatalf("Namespace isolation violated: ns2 value changed")
			}
		}
	})
}

// Unit test for empty key validation
func TestGet_EmptyKey(t *testing.T) {
	client := LocalOnly(100)
	defer client.Close()

	result := client.Get(context.Background(), "")
	if result.IsOk() {
		t.Error("Expected error for empty key")
	}
	if !IsNotFound(result.UnwrapErr()) && result.UnwrapErr() != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey, got %v", result.UnwrapErr())
	}
}

// Unit test for empty value validation
func TestSet_EmptyValue(t *testing.T) {
	client := LocalOnly(100)
	defer client.Close()

	err := client.Set(context.Background(), "key", []byte{}, time.Minute)
	if err != ErrInvalidValue {
		t.Errorf("Expected ErrInvalidValue, got %v", err)
	}
}

// Unit test for delete operation
func TestDelete(t *testing.T) {
	client := LocalOnly(100)
	defer client.Close()

	ctx := context.Background()

	// Set a value
	err := client.Set(ctx, "key", []byte("value"), time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete it
	err = client.Delete(ctx, "key")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get should return not found
	result := client.Get(ctx, "key")
	if result.IsOk() {
		t.Error("Expected not found after delete")
	}
}

// Unit test for batch operations
func TestBatchOperations(t *testing.T) {
	client := LocalOnly(100)
	defer client.Close()

	ctx := context.Background()

	entries := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// Batch set
	err := client.BatchSet(ctx, entries, time.Minute)
	if err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	// Batch get
	keys := []string{"key1", "key2", "key3", "key4"}
	result := client.BatchGet(ctx, keys)
	if !result.IsOk() {
		t.Fatalf("BatchGet failed: %v", result.UnwrapErr())
	}

	values := result.Unwrap()
	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	for key, expected := range entries {
		if got, ok := values[key]; !ok {
			t.Errorf("Missing key %s", key)
		} else if string(got) != string(expected) {
			t.Errorf("Value mismatch for %s: got %s, want %s", key, got, expected)
		}
	}
}
