// Package cache provides the core cache service interfaces and implementations.
package cache

import (
	"context"
	"time"
)

// Service defines the main cache operations interface.
type Service interface {
	// Get retrieves a value from cache by namespace and key.
	Get(ctx context.Context, namespace, key string) (*Entry, error)

	// Set stores a value in cache with optional TTL and encryption.
	Set(ctx context.Context, namespace, key string, value []byte, ttl time.Duration, opts ...SetOption) error

	// Delete removes a value from cache.
	Delete(ctx context.Context, namespace, key string) (bool, error)

	// BatchGet retrieves multiple values from cache.
	// Returns found values, missing keys, and any error.
	BatchGet(ctx context.Context, namespace string, keys []string) (map[string][]byte, []string, error)

	// BatchSet stores multiple key-value pairs in cache.
	// Returns the count of successfully stored entries.
	BatchSet(ctx context.Context, namespace string, entries map[string][]byte, ttl time.Duration) (int, error)

	// Health returns the current health status of the cache service.
	Health(ctx context.Context) (*HealthStatus, error)
}

// RedisClient abstracts Redis operations for testability.
type RedisClient interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Del deletes one or more keys.
	Del(ctx context.Context, keys ...string) (int64, error)

	// MGet retrieves multiple values.
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// MSet stores multiple key-value pairs.
	MSet(ctx context.Context, pairs ...interface{}) error

	// Ping checks Redis connectivity.
	Ping(ctx context.Context) error

	// Close closes the Redis connection.
	Close() error
}

// CircuitBreaker wraps operations with failure protection.
type CircuitBreaker interface {
	// Execute runs the operation with circuit breaker protection.
	Execute(ctx context.Context, operation func() error) error

	// State returns the current circuit state (0=closed, 1=open, 2=half-open).
	State() int

	// Reset resets the circuit breaker to closed state.
	Reset()
}

// MessageBroker handles async cache invalidation events.
type MessageBroker interface {
	// Subscribe registers a handler for invalidation events.
	Subscribe(ctx context.Context, topic string, handler InvalidationHandler) error

	// Publish sends an invalidation event.
	Publish(ctx context.Context, topic string, event InvalidationEvent) error

	// Close closes the broker connection.
	Close() error
}

// InvalidationHandler processes cache invalidation events.
type InvalidationHandler func(event InvalidationEvent) error

// Encryptor handles value encryption/decryption.
type Encryptor interface {
	// Encrypt encrypts plaintext data.
	Encrypt(plaintext []byte) ([]byte, error)

	// Decrypt decrypts ciphertext data.
	Decrypt(ciphertext []byte) ([]byte, error)
}

// LocalCache provides in-memory caching with TTL and eviction.
type LocalCache interface {
	// Get retrieves a value from local cache.
	Get(key string) ([]byte, bool)

	// Set stores a value in local cache with TTL.
	Set(key string, value []byte, ttl time.Duration)

	// Delete removes a value from local cache.
	Delete(key string) bool

	// Clear removes all entries from local cache.
	Clear()

	// Size returns the number of entries in local cache.
	Size() int
}
