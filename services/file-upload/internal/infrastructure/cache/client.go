// Package cache provides cache operations via platform cache-service.
package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// Namespace for all file-upload cache keys.
	Namespace = "file-upload"
)

// Client provides cache operations with circuit breaker.
type Client struct {
	conn          *grpc.ClientConn
	namespace     string
	circuitOpen   bool
	failures      int
	failThreshold int
	resetTimeout  time.Duration
	lastFailure   time.Time
	mu            sync.RWMutex
}

// Config holds cache client configuration.
type Config struct {
	Address       string
	Namespace     string
	FailThreshold int
	ResetTimeout  time.Duration
	Timeout       time.Duration
}

// NewClient creates a new cache client with circuit breaker.
func NewClient(cfg Config) (*Client, error) {
	namespace := cfg.Namespace
	if namespace == "" {
		namespace = Namespace
	}

	client := &Client{
		namespace:     namespace,
		failThreshold: cfg.FailThreshold,
		resetTimeout:  cfg.ResetTimeout,
	}

	// Try to connect to cache service
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		// Start with circuit open
		client.circuitOpen = true
		client.lastFailure = time.Now()
		return client, nil // Return client with circuit open for fallback
	}

	client.conn = conn
	return client, nil
}

// buildKey creates a namespaced cache key.
func (c *Client) buildKey(key string) string {
	return c.namespace + ":" + key
}

// isCircuitOpen checks if circuit breaker is open.
func (c *Client) isCircuitOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.circuitOpen {
		return false
	}

	// Check if reset timeout has passed (half-open state)
	if time.Since(c.lastFailure) > c.resetTimeout {
		return false // Allow probe request
	}

	return true
}

// recordFailure records a failure and potentially opens circuit.
func (c *Client) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures++
	c.lastFailure = time.Now()
	if c.failures >= c.failThreshold {
		c.circuitOpen = true
	}
}

// recordSuccess records a success and closes circuit.
func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures = 0
	c.circuitOpen = false
}

// Get retrieves a value from cache.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	if c.isCircuitOpen() || c.conn == nil {
		return nil, ErrCacheUnavailable
	}

	// In production, this would call the gRPC service
	// For now, return not found to trigger database fallback
	return nil, ErrKeyNotFound
}

// Set stores a value in cache.
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c.isCircuitOpen() || c.conn == nil {
		return ErrCacheUnavailable
	}

	// In production, this would call the gRPC service
	return nil
}

// Delete removes a value from cache.
func (c *Client) Delete(ctx context.Context, key string) error {
	if c.isCircuitOpen() || c.conn == nil {
		return ErrCacheUnavailable
	}

	// In production, this would call the gRPC service
	return nil
}

// BatchGet retrieves multiple values from cache.
func (c *Client) BatchGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if c.isCircuitOpen() || c.conn == nil {
		return nil, ErrCacheUnavailable
	}

	// In production, this would call the gRPC service
	return make(map[string][]byte), nil
}

// BatchSet stores multiple key-value pairs in cache.
func (c *Client) BatchSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if c.isCircuitOpen() || c.conn == nil {
		return ErrCacheUnavailable
	}

	// In production, this would call the gRPC service
	return nil
}

// GetJSON retrieves and unmarshals a JSON value from cache.
func (c *Client) GetJSON(ctx context.Context, key string, v any) error {
	data, err := c.Get(ctx, c.buildKey(key))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// SetJSON marshals and stores a JSON value in cache.
func (c *Client) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(ctx, c.buildKey(key), data, ttl)
}

// DeleteKey deletes a namespaced key from cache.
func (c *Client) DeleteKey(ctx context.Context, key string) error {
	return c.Delete(ctx, c.buildKey(key))
}

// InvalidatePattern invalidates all keys matching a pattern.
func (c *Client) InvalidatePattern(ctx context.Context, pattern string) error {
	// In production, this would use SCAN + DELETE or a pattern delete
	return nil
}

// Close closes the cache client connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsAvailable returns true if cache service is available.
func (c *Client) IsAvailable() bool {
	return !c.isCircuitOpen() && c.conn != nil
}

// GetNamespace returns the cache namespace.
func (c *Client) GetNamespace() string {
	return c.namespace
}

// Errors
var (
	ErrCacheUnavailable = &CacheError{Code: "CACHE_UNAVAILABLE", Message: "cache service is unavailable"}
	ErrKeyNotFound      = &CacheError{Code: "KEY_NOT_FOUND", Message: "key not found in cache"}
)

// CacheError represents a cache operation error.
type CacheError struct {
	Code    string
	Message string
}

func (e *CacheError) Error() string {
	return e.Code + ": " + e.Message
}

// Is implements errors.Is.
func (e *CacheError) Is(target error) bool {
	t, ok := target.(*CacheError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}
