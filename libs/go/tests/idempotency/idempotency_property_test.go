package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/idempotency"
	"pgregory.net/rapid"
)

// TestStoreSetGetRoundtrip verifies set/get roundtrip preserves data.
func TestStoreSetGetRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "key")
		response := rapid.SliceOf(rapid.Byte()).Draw(t, "response")
		statusCode := rapid.IntRange(200, 599).Draw(t, "statusCode")

		store := idempotency.NewMemoryStore(time.Hour)
		ctx := context.Background()

		entry := &idempotency.Entry{
			Key:        key,
			Response:   response,
			StatusCode: statusCode,
		}

		err := store.Set(ctx, entry)
		if err != nil {
			t.Fatalf("set failed: %v", err)
		}

		retrieved, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if retrieved == nil {
			t.Fatal("retrieved entry is nil")
		}

		if retrieved.Key != key {
			t.Errorf("key mismatch: %s != %s", retrieved.Key, key)
		}

		if retrieved.StatusCode != statusCode {
			t.Errorf("status code mismatch: %d != %d", retrieved.StatusCode, statusCode)
		}

		if len(retrieved.Response) != len(response) {
			t.Errorf("response length mismatch: %d != %d", len(retrieved.Response), len(response))
		}
	})
}

// TestStoreDeleteRemovesEntry verifies delete removes entries.
func TestStoreDeleteRemovesEntry(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "key")

		store := idempotency.NewMemoryStore(time.Hour)
		ctx := context.Background()

		entry := &idempotency.Entry{
			Key:        key,
			Response:   []byte("test"),
			StatusCode: 200,
		}

		store.Set(ctx, entry)
		store.Delete(ctx, key)

		retrieved, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if retrieved != nil {
			t.Error("entry should be nil after delete")
		}
	})
}

// TestStoreLockUnlockBehavior verifies lock/unlock behavior.
func TestStoreLockUnlockBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "key")

		store := idempotency.NewMemoryStore(time.Hour)
		ctx := context.Background()

		// First lock should succeed
		acquired, err := store.Lock(ctx, key)
		if err != nil {
			t.Fatalf("lock failed: %v", err)
		}
		if !acquired {
			t.Error("first lock should succeed")
		}

		// Second lock should fail
		acquired2, err := store.Lock(ctx, key)
		if err != nil {
			t.Fatalf("second lock failed: %v", err)
		}
		if acquired2 {
			t.Error("second lock should fail")
		}

		// Unlock
		err = store.Unlock(ctx, key)
		if err != nil {
			t.Fatalf("unlock failed: %v", err)
		}

		// Third lock should succeed after unlock
		acquired3, err := store.Lock(ctx, key)
		if err != nil {
			t.Fatalf("third lock failed: %v", err)
		}
		if !acquired3 {
			t.Error("third lock should succeed after unlock")
		}
	})
}

// TestStoreExpiration verifies entries expire correctly.
func TestStoreExpiration(t *testing.T) {
	store := idempotency.NewMemoryStore(50 * time.Millisecond)
	ctx := context.Background()

	entry := &idempotency.Entry{
		Key:        "expire-test",
		Response:   []byte("test"),
		StatusCode: 200,
	}

	store.Set(ctx, entry)

	// Should exist immediately
	retrieved, _ := store.Get(ctx, "expire-test")
	if retrieved == nil {
		t.Error("entry should exist immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	retrieved, _ = store.Get(ctx, "expire-test")
	if retrieved != nil {
		t.Error("entry should be expired")
	}
}

// TestStoreCleanupRemovesExpired verifies cleanup removes expired entries.
func TestStoreCleanupRemovesExpired(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		entryCount := rapid.IntRange(1, 10).Draw(t, "entryCount")

		store := idempotency.NewMemoryStore(10 * time.Millisecond)
		ctx := context.Background()

		for i := 0; i < entryCount; i++ {
			key := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "key")
			store.Set(ctx, &idempotency.Entry{
				Key:        key,
				Response:   []byte("test"),
				StatusCode: 200,
			})
		}

		// Wait for expiration
		time.Sleep(50 * time.Millisecond)

		removed := store.Cleanup()
		if removed != entryCount {
			t.Errorf("expected %d removed, got %d", entryCount, removed)
		}
	})
}

// TestStoreGetNonExistent verifies get returns nil for non-existent keys.
func TestStoreGetNonExistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "key")

		store := idempotency.NewMemoryStore(time.Hour)
		ctx := context.Background()

		retrieved, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if retrieved != nil {
			t.Error("non-existent key should return nil")
		}
	})
}
