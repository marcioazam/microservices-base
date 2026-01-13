// Package cache provides decision caching for IAM Policy Service.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/config"
	"github.com/authcorp/libs/go/src/cache"
)

// Decision represents a cached authorization decision.
type Decision struct {
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DecisionCache provides caching for authorization decisions.
type DecisionCache struct {
	client     *cache.Client
	localCache *LocalFallback
	ttl        time.Duration
	mu         sync.RWMutex
	useLocal   bool
}

// NewDecisionCache creates a new decision cache from configuration.
func NewDecisionCache(cfg config.CacheConfig) (*DecisionCache, error) {
	clientCfg := cache.ClientConfig{
		Address:        cfg.Address,
		Namespace:      cfg.Namespace,
		Timeout:        cfg.Timeout,
		LocalFallback:  cfg.LocalFallback,
		LocalCacheSize: cfg.LocalCacheSize,
	}

	client, err := cache.NewClient(clientCfg)
	if err != nil {
		// Fallback to local-only cache
		return &DecisionCache{
			localCache: NewLocalFallback(cfg.LocalCacheSize),
			ttl:        cfg.TTL,
			useLocal:   true,
		}, nil
	}

	dc := &DecisionCache{
		client: client,
		ttl:    cfg.TTL,
	}

	if cfg.LocalFallback {
		dc.localCache = NewLocalFallback(cfg.LocalCacheSize)
	}

	return dc, nil
}

// NewLocalOnlyCache creates a local-only cache for testing.
func NewLocalOnlyCache(size int, ttl time.Duration) *DecisionCache {
	return &DecisionCache{
		localCache: NewLocalFallback(size),
		ttl:        ttl,
		useLocal:   true,
	}
}

// Get retrieves a cached decision.
func (dc *DecisionCache) Get(ctx context.Context, input map[string]interface{}) (*Decision, bool) {
	key := dc.generateKey(input)

	// Try remote cache first
	if dc.client != nil && !dc.useLocal {
		result := dc.client.Get(ctx, key)
		if result.IsOk() {
			entry := result.Unwrap()
			var decision Decision
			if err := json.Unmarshal(entry.Value, &decision); err == nil {
				if time.Now().Before(decision.ExpiresAt) {
					return &decision, true
				}
			}
		}
	}

	// Fallback to local cache
	if dc.localCache != nil {
		return dc.localCache.Get(key)
	}

	return nil, false
}

// Set stores a decision in cache.
func (dc *DecisionCache) Set(ctx context.Context, input map[string]interface{}, decision *Decision) error {
	key := dc.generateKey(input)

	decision.CachedAt = time.Now()
	decision.ExpiresAt = time.Now().Add(dc.ttl)

	data, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("cache: failed to marshal decision: %w", err)
	}

	// Store in remote cache
	if dc.client != nil && !dc.useLocal {
		if err := dc.client.Set(ctx, key, data, dc.ttl); err != nil {
			// Log error but continue with local cache
		}
	}

	// Store in local cache
	if dc.localCache != nil {
		dc.localCache.Set(key, decision, dc.ttl)
	}

	return nil
}

// Delete removes a decision from cache.
func (dc *DecisionCache) Delete(ctx context.Context, input map[string]interface{}) error {
	key := dc.generateKey(input)

	if dc.client != nil && !dc.useLocal {
		if err := dc.client.Delete(ctx, key); err != nil {
			// Log error but continue
		}
	}

	if dc.localCache != nil {
		dc.localCache.Delete(key)
	}

	return nil
}

// Invalidate clears all cached decisions.
func (dc *DecisionCache) Invalidate(ctx context.Context) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.localCache != nil {
		dc.localCache.Clear()
	}

	return nil
}

// Close closes the cache client.
func (dc *DecisionCache) Close() error {
	if dc.client != nil {
		return dc.client.Close()
	}
	return nil
}

// generateKey creates a deterministic cache key from authorization input.
func (dc *DecisionCache) generateKey(input map[string]interface{}) string {
	data, _ := json.Marshal(input)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Stats returns cache statistics.
func (dc *DecisionCache) Stats() CacheStats {
	if dc.localCache != nil {
		return dc.localCache.Stats()
	}
	return CacheStats{}
}

// CacheStats holds cache statistics.
type CacheStats struct {
	Hits   int64
	Misses int64
	Size   int
}
