package crypto

import (
	"sync"
	"time"
)

// KeyMetadata holds metadata about a cryptographic key.
type KeyMetadata struct {
	ID                KeyID
	Algorithm         string
	State             string
	CreatedAt         time.Time
	ExpiresAt         time.Time
	RotatedAt         time.Time
	PreviousVersion   KeyID
	OwnerService      string
	AllowedOperations []string
	UsageCount        uint64
}

// CachedKeyMetadata holds cached key information with expiration.
type CachedKeyMetadata struct {
	Metadata  *KeyMetadata
	CachedAt  time.Time
	ExpiresAt time.Time
}

// KeyMetadataCache caches key metadata locally with TTL.
type KeyMetadataCache struct {
	cache map[string]*CachedKeyMetadata
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewKeyMetadataCache creates a new key metadata cache.
func NewKeyMetadataCache(ttl time.Duration) *KeyMetadataCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &KeyMetadataCache{
		cache: make(map[string]*CachedKeyMetadata),
		ttl:   ttl,
	}
}

// Get retrieves cached key metadata if not expired.
func (c *KeyMetadataCache) Get(keyID KeyID) (*KeyMetadata, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := keyID.String()
	cached, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	return cached.Metadata, true
}

// Set stores key metadata in cache with TTL.
func (c *KeyMetadataCache) Set(keyID KeyID, metadata *KeyMetadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.cache[keyID.String()] = &CachedKeyMetadata{
		Metadata:  metadata,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
}

// Invalidate removes a key from cache.
func (c *KeyMetadataCache) Invalidate(keyID KeyID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, keyID.String())
}

// InvalidateAll clears all cached entries.
func (c *KeyMetadataCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CachedKeyMetadata)
}

// Size returns the number of cached entries.
func (c *KeyMetadataCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// Cleanup removes expired entries from cache.
func (c *KeyMetadataCache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			delete(c.cache, key)
			removed++
		}
	}

	return removed
}

// GetTTL returns the cache TTL.
func (c *KeyMetadataCache) GetTTL() time.Duration {
	return c.ttl
}
