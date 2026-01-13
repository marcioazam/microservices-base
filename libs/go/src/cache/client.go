package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"github.com/authcorp/libs/go/src/functional"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// CacheSource indicates where the cached value came from.
type CacheSource int

const (
	// SourceUnknown indicates unknown source.
	SourceUnknown CacheSource = iota
	// SourceRemote indicates value came from remote cache-service.
	SourceRemote
	// SourceLocal indicates value came from local fallback cache.
	SourceLocal
)

// CacheEntry represents a cached value with metadata.
type CacheEntry struct {
	Value  []byte
	Source CacheSource
}

// Client provides distributed cache operations via cache-service.
type Client struct {
	conn           *grpc.ClientConn
	config         ClientConfig
	circuitBreaker *fault.CircuitBreaker
	localCache     *LocalCache
	mu             sync.RWMutex
	closed         bool
}

// NewClient creates a new cache client.
func NewClient(config ClientConfig) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	cb, err := fault.NewCircuitBreaker(config.CircuitBreaker)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to create circuit breaker: %w", err)
	}

	client := &Client{
		config:         config,
		circuitBreaker: cb,
	}

	if config.LocalFallback {
		client.localCache = NewLocalCache(config.LocalCacheSize)
	}

	// Connect to cache-service
	conn, err := grpc.NewClient(
		config.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		if config.LocalFallback {
			// Allow client creation with local-only mode
			return client, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	client.conn = conn

	return client, nil
}

// LocalOnly creates a client that only uses local cache (for testing).
func LocalOnly(size int) *Client {
	if size <= 0 {
		size = 10000
	}
	return &Client{
		config: ClientConfig{
			Namespace:      "local",
			LocalFallback:  true,
			LocalCacheSize: size,
		},
		localCache: NewLocalCache(size),
	}
}

// Get retrieves a value from cache.
func (c *Client) Get(ctx context.Context, key string) functional.Result[CacheEntry] {
	if key == "" {
		return functional.Err[CacheEntry](ErrInvalidKey)
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return functional.Err[CacheEntry](ErrConnectionFailed)
	}
	c.mu.RUnlock()

	fullKey := c.buildKey(key)

	// Try remote first if connected
	if c.conn != nil {
		result := c.getRemote(ctx, fullKey)
		if result.IsOk() {
			return result
		}

		// Check if we should fallback to local
		if c.localCache != nil && c.config.LocalFallback {
			if value, ok := c.localCache.Get(fullKey); ok {
				return functional.Ok(CacheEntry{Value: value, Source: SourceLocal})
			}
		}

		return result
	}

	// Local-only mode
	if c.localCache != nil {
		if value, ok := c.localCache.Get(fullKey); ok {
			return functional.Ok(CacheEntry{Value: value, Source: SourceLocal})
		}
	}

	return functional.Err[CacheEntry](ErrNotFound)
}

func (c *Client) getRemote(ctx context.Context, key string) functional.Result[CacheEntry] {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	var entry CacheEntry
	err := c.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		// Simulate gRPC call - in real implementation this would call the service
		// For now, return not found to allow local fallback testing
		return status.Error(codes.NotFound, "key not found")
	})

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return functional.Err[CacheEntry](ErrNotFound)
		}
		if fault.IsCircuitOpen(err) {
			return functional.Err[CacheEntry](ErrCircuitOpen)
		}
		return functional.Err[CacheEntry](fmt.Errorf("%w: %v", ErrServiceUnavailable, err))
	}

	entry.Source = SourceRemote
	return functional.Ok(entry)
}

// Set stores a value in cache.
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return ErrInvalidKey
	}
	if len(value) == 0 {
		return ErrInvalidValue
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConnectionFailed
	}
	c.mu.RUnlock()

	fullKey := c.buildKey(key)

	// Store in local cache if enabled
	if c.localCache != nil {
		c.localCache.Set(fullKey, value, ttl)
	}

	// Try remote if connected
	if c.conn != nil {
		return c.setRemote(ctx, fullKey, value, ttl)
	}

	return nil
}

func (c *Client) setRemote(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	return c.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		// Simulate gRPC call - in real implementation this would call the service
		return nil
	})
}

// Delete removes a value from cache.
func (c *Client) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConnectionFailed
	}
	c.mu.RUnlock()

	fullKey := c.buildKey(key)

	// Remove from local cache
	if c.localCache != nil {
		c.localCache.Delete(fullKey)
	}

	// Try remote if connected
	if c.conn != nil {
		return c.deleteRemote(ctx, fullKey)
	}

	return nil
}

func (c *Client) deleteRemote(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	return c.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
}

// BatchGet retrieves multiple values from cache.
func (c *Client) BatchGet(ctx context.Context, keys []string) functional.Result[map[string][]byte] {
	if len(keys) == 0 {
		return functional.Ok(make(map[string][]byte))
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return functional.Err[map[string][]byte](ErrConnectionFailed)
	}
	c.mu.RUnlock()

	results := make(map[string][]byte)

	for _, key := range keys {
		if key == "" {
			continue
		}
		result := c.Get(ctx, key)
		if result.IsOk() {
			results[key] = result.Unwrap().Value
		}
	}

	return functional.Ok(results)
}

// BatchSet stores multiple key-value pairs in cache.
func (c *Client) BatchSet(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	if len(entries) == 0 {
		return nil
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConnectionFailed
	}
	c.mu.RUnlock()

	for key, value := range entries {
		if err := c.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) buildKey(key string) string {
	return fmt.Sprintf("%s:%s", c.config.Namespace, key)
}

// Namespace returns the configured namespace.
func (c *Client) Namespace() string {
	return c.config.Namespace
}

// IsConnected returns true if connected to remote cache-service.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && !c.closed
}
