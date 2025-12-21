package authplatform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
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
	return JWKSCacheConfig{
		URI: uri,
		TTL: time.Hour,
	}
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

	// Register the JWKS URL with auto-refresh
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

// ValidateToken validates a JWT using cached JWKS.
func (c *JWKSCache) ValidateToken(ctx context.Context, token string, audience string) (*Claims, error) {
	keySet, err := c.getKeySet(ctx)
	if err != nil {
		c.recordError()
		return nil, &SDKError{
			Code:    ErrCodeValidation,
			Message: "failed to fetch JWKS",
			Cause:   err,
		}
	}

	parseOpts := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	}

	if audience != "" {
		parseOpts = append(parseOpts, jwt.WithAudience(audience))
	}

	parsed, err := jwt.Parse([]byte(token), parseOpts...)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeTokenInvalid,
			Message: "token validation failed",
			Cause:   err,
		}
	}

	return extractClaims(parsed)
}

func extractClaims(parsed jwt.Token) (*Claims, error) {
	claims := &Claims{}

	if sub, ok := parsed.Subject(); ok {
		claims.Subject = sub
	}
	if iss, ok := parsed.Issuer(); ok {
		claims.Issuer = iss
	}
	if exp, ok := parsed.Expiration(); ok && !exp.IsZero() {
		claims.ExpiresAt = exp.Unix()
	}
	if iat, ok := parsed.IssuedAt(); ok && !iat.IsZero() {
		claims.IssuedAt = iat.Unix()
	}
	if aud, ok := parsed.Audience(); ok && len(aud) > 0 {
		claims.Audience = aud
	}

	var scope string
	if err := parsed.Get("scope", &scope); err == nil {
		claims.Scope = scope
	}
	var clientID string
	if err := parsed.Get("client_id", &clientID); err == nil {
		claims.ClientID = clientID
	}

	return claims, nil
}

func (c *JWKSCache) getKeySet(ctx context.Context) (jwk.Set, error) {
	keySet, err := c.cache.Lookup(ctx, c.uri)
	if err != nil {
		// Try fallback keys if available
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

	// Store as fallback for future failures
	c.mu.Lock()
	c.fallbackKeys = keySet
	c.mu.Unlock()

	c.recordHit()
	return keySet, nil
}


// Invalidate clears the cache and forces a refresh on next access.
func (c *JWKSCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallbackKeys = nil
	c.recordRefresh()
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

// ValidationOptions holds custom validation options.
type ValidationOptions struct {
	Audience       string
	Issuer         string
	RequiredClaims []string
	SkipExpiry     bool
}

// ValidateTokenWithOpts validates a JWT with custom validation options.
func (c *JWKSCache) ValidateTokenWithOpts(ctx context.Context, token string, opts ValidationOptions) (*Claims, error) {
	keySet, err := c.getKeySet(ctx)
	if err != nil {
		c.recordError()
		return nil, &SDKError{
			Code:    ErrCodeValidation,
			Message: "failed to fetch JWKS",
			Cause:   err,
		}
	}

	parseOpts := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
	}

	if !opts.SkipExpiry {
		parseOpts = append(parseOpts, jwt.WithValidate(true))
	}

	if opts.Audience != "" {
		parseOpts = append(parseOpts, jwt.WithAudience(opts.Audience))
	}

	if opts.Issuer != "" {
		parseOpts = append(parseOpts, jwt.WithIssuer(opts.Issuer))
	}

	parsed, err := jwt.Parse([]byte(token), parseOpts...)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeTokenInvalid,
			Message: "token validation failed",
			Cause:   err,
		}
	}

	// Check required claims
	for _, claim := range opts.RequiredClaims {
		var v interface{}
		if err := parsed.Get(claim, &v); err != nil {
			return nil, &SDKError{
				Code:    ErrCodeTokenInvalid,
				Message: fmt.Sprintf("missing required claim: %s", claim),
			}
		}
	}

	return extractClaims(parsed)
}

// AddJWKSEndpoint adds an additional JWKS endpoint for key rotation support.
func (c *JWKSCache) AddJWKSEndpoint(uri string) error {
	return c.cache.Register(c.ctx, uri)
}
