package crypto_test

import (
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/crypto"
)

func TestKeyMetadataCache_SetAndGet(t *testing.T) {
	cache := crypto.NewKeyMetadataCache(5 * time.Minute)

	keyID := crypto.KeyID{Namespace: "test", ID: "key1", Version: 1}
	metadata := &crypto.KeyMetadata{
		ID:        keyID,
		Algorithm: "AES-256-GCM",
		State:     "ACTIVE",
	}

	cache.Set(keyID, metadata)

	got, ok := cache.Get(keyID)
	if !ok {
		t.Fatal("expected to find cached metadata")
	}

	if got.Algorithm != metadata.Algorithm {
		t.Errorf("expected algorithm %s, got %s", metadata.Algorithm, got.Algorithm)
	}
}

func TestKeyMetadataCache_Expiration(t *testing.T) {
	// Use very short TTL for testing
	cache := crypto.NewKeyMetadataCache(10 * time.Millisecond)

	keyID := crypto.KeyID{Namespace: "test", ID: "key1", Version: 1}
	metadata := &crypto.KeyMetadata{
		ID:        keyID,
		Algorithm: "AES-256-GCM",
	}

	cache.Set(keyID, metadata)

	// Should be found immediately
	_, ok := cache.Get(keyID)
	if !ok {
		t.Fatal("expected to find cached metadata immediately after set")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should not be found after expiration
	_, ok = cache.Get(keyID)
	if ok {
		t.Error("expected cache entry to be expired")
	}
}

func TestKeyMetadataCache_Invalidate(t *testing.T) {
	cache := crypto.NewKeyMetadataCache(5 * time.Minute)

	keyID := crypto.KeyID{Namespace: "test", ID: "key1", Version: 1}
	metadata := &crypto.KeyMetadata{ID: keyID}

	cache.Set(keyID, metadata)

	// Verify it's cached
	_, ok := cache.Get(keyID)
	if !ok {
		t.Fatal("expected to find cached metadata")
	}

	// Invalidate
	cache.Invalidate(keyID)

	// Should not be found
	_, ok = cache.Get(keyID)
	if ok {
		t.Error("expected cache entry to be invalidated")
	}
}

func TestKeyMetadataCache_InvalidateAll(t *testing.T) {
	cache := crypto.NewKeyMetadataCache(5 * time.Minute)

	// Add multiple entries
	for i := 1; i <= 5; i++ {
		keyID := crypto.KeyID{Namespace: "test", ID: "key", Version: uint32(i)}
		cache.Set(keyID, &crypto.KeyMetadata{ID: keyID})
	}

	if cache.Size() != 5 {
		t.Errorf("expected 5 entries, got %d", cache.Size())
	}

	cache.InvalidateAll()

	if cache.Size() != 0 {
		t.Errorf("expected 0 entries after invalidate all, got %d", cache.Size())
	}
}

func TestKeyMetadataCache_Cleanup(t *testing.T) {
	cache := crypto.NewKeyMetadataCache(10 * time.Millisecond)

	// Add entries
	for i := 1; i <= 3; i++ {
		keyID := crypto.KeyID{Namespace: "test", ID: "key", Version: uint32(i)}
		cache.Set(keyID, &crypto.KeyMetadata{ID: keyID})
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Cleanup should remove expired entries
	removed := cache.Cleanup()
	if removed != 3 {
		t.Errorf("expected 3 entries removed, got %d", removed)
	}

	if cache.Size() != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", cache.Size())
	}
}

func TestKeyMetadataCache_NotFound(t *testing.T) {
	cache := crypto.NewKeyMetadataCache(5 * time.Minute)

	keyID := crypto.KeyID{Namespace: "test", ID: "nonexistent", Version: 1}

	_, ok := cache.Get(keyID)
	if ok {
		t.Error("expected not to find non-existent key")
	}
}
