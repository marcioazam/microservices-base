package authplatform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JWKSCache caches JWKS for token validation.
type JWKSCache struct {
	uri       string
	ttl       time.Duration
	cache     jwk.Set
	cacheTime time.Time
	mu        sync.RWMutex
}

// NewJWKSCache creates a new JWKS cache.
func NewJWKSCache(uri string, ttl time.Duration) *JWKSCache {
	return &JWKSCache{
		uri: uri,
		ttl: ttl,
	}
}

// ValidateToken validates a JWT using cached JWKS.
func (c *JWKSCache) ValidateToken(ctx context.Context, token string, audience string) (*Claims, error) {
	keySet, err := c.getKeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	parsed, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(keySet),
		jwt.WithAudience(audience),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	// Extract claims
	claims := &Claims{
		Subject:   parsed.Subject(),
		Issuer:    parsed.Issuer(),
		ExpiresAt: parsed.Expiration().Unix(),
		IssuedAt:  parsed.IssuedAt().Unix(),
	}

	// Handle audience (can be string or array)
	if aud := parsed.Audience(); len(aud) > 0 {
		claims.Audience = aud
	}

	// Extract custom claims
	if scope, ok := parsed.Get("scope"); ok {
		if s, ok := scope.(string); ok {
			claims.Scope = s
		}
	}
	if clientID, ok := parsed.Get("client_id"); ok {
		if s, ok := clientID.(string); ok {
			claims.ClientID = s
		}
	}

	return claims, nil
}

func (c *JWKSCache) getKeySet(ctx context.Context) (jwk.Set, error) {
	c.mu.RLock()
	if c.cache != nil && time.Since(c.cacheTime) < c.ttl {
		defer c.mu.RUnlock()
		return c.cache, nil
	}
	c.mu.RUnlock()

	// Need to refresh
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.cache != nil && time.Since(c.cacheTime) < c.ttl {
		return c.cache, nil
	}

	keySet, err := jwk.Fetch(ctx, c.uri)
	if err != nil {
		return nil, err
	}

	c.cache = keySet
	c.cacheTime = time.Now()

	return keySet, nil
}

// Invalidate clears the cache.
func (c *JWKSCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = nil
	c.cacheTime = time.Time{}
}
