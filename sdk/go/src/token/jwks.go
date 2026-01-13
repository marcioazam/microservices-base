package token

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

// JWKSCache caches JWKS for token validation with auto-refresh.
type JWKSCache struct {
	uri          string
	ttl          time.Duration
	cache        *jwk.Cache
	fallbackKeys jwk.Set
	mu           sync.RWMutex
	metrics      *JWKSMetrics
	ctx          context.Context
	cancel       context.CancelFunc
}

// JWKSMetrics tracks cache performance metrics.
type JWKSMetrics struct {
	Hits      int64
	Misses    int64
	Refreshes int64
	Errors    int64
	mu        sync.Mutex
}

// JWKSCacheConfig holds configuration for the JWKS cache.
type JWKSCacheConfig struct {
	URI string
	TTL time.Duration
}

// DefaultJWKSCacheConfig returns default configuration.
func DefaultJWKSCacheConfig(uri string) JWKSCacheConfig {
	return JWKSCacheConfig{URI: uri, TTL: time.Hour}
}

// NewJWKSCache creates a new JWKS cache with default settings.
func NewJWKSCache(uri string, ttl time.Duration) *JWKSCache {
	config := DefaultJWKSCacheConfig(uri)
	config.TTL = ttl
	return NewJWKSCacheWithConfig(config)
}

// NewJWKSCacheWithConfig creates a new JWKS cache with custom configuration.
func NewJWKSCacheWithConfig(config JWKSCacheConfig) *JWKSCache {
	ctx, cancel := context.WithCancel(context.Background())

	cache, err := jwk.NewCache(ctx, httprc.NewClient())
	if err != nil {
		cancel()
		return nil
	}

	if err := cache.Register(ctx, config.URI); err != nil {
		cancel()
		return nil
	}

	return &JWKSCache{
		uri:     config.URI,
		ttl:     config.TTL,
		cache:   cache,
		metrics: &JWKSMetrics{},
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Close stops the background refresh goroutine.
func (c *JWKSCache) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

// GetMetrics returns cache performance metrics.
func (c *JWKSCache) GetMetrics() JWKSMetrics {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	return JWKSMetrics{
		Hits:      c.metrics.Hits,
		Misses:    c.metrics.Misses,
		Refreshes: c.metrics.Refreshes,
		Errors:    c.metrics.Errors,
	}
}

// Invalidate clears the cache and forces a refresh on next access.
func (c *JWKSCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallbackKeys = nil
	c.recordRefresh()
}

// AddJWKSEndpoint adds an additional JWKS endpoint for key rotation support.
func (c *JWKSCache) AddJWKSEndpoint(uri string) error {
	return c.cache.Register(c.ctx, uri)
}

func (c *JWKSCache) getKeySet(ctx context.Context) (jwk.Set, error) {
	keySet, err := c.cache.Lookup(ctx, c.uri)
	if err != nil {
		c.mu.RLock()
		fallback := c.fallbackKeys
		c.mu.RUnlock()

		if fallback != nil {
			c.recordHit()
			return fallback, nil
		}

		c.recordMiss()
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	c.mu.Lock()
	c.fallbackKeys = keySet
	c.mu.Unlock()

	c.recordHit()
	return keySet, nil
}

func (c *JWKSCache) recordHit() {
	c.metrics.mu.Lock()
	c.metrics.Hits++
	c.metrics.mu.Unlock()
}

func (c *JWKSCache) recordMiss() {
	c.metrics.mu.Lock()
	c.metrics.Misses++
	c.metrics.mu.Unlock()
}

func (c *JWKSCache) recordRefresh() {
	c.metrics.mu.Lock()
	c.metrics.Refreshes++
	c.metrics.mu.Unlock()
}

func (c *JWKSCache) recordError() {
	c.metrics.mu.Lock()
	c.metrics.Errors++
	c.metrics.mu.Unlock()
}
