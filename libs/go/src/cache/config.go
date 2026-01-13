// Package cache provides a distributed cache client for cache-service.
package cache

import (
	"time"

	"github.com/authcorp/libs/go/src/fault"
)

// ClientConfig configures the cache client.
type ClientConfig struct {
	// Address is the cache-service gRPC address.
	Address string

	// Namespace isolates cache keys for multi-tenant scenarios.
	Namespace string

	// Timeout for cache operations.
	Timeout time.Duration

	// MaxRetries for failed operations.
	MaxRetries int

	// CircuitBreaker configuration for resilience.
	CircuitBreaker fault.CircuitBreakerConfig

	// LocalFallback enables local cache when remote is unavailable.
	LocalFallback bool

	// LocalCacheSize is the max entries in local fallback cache.
	LocalCacheSize int

	// ConnectionPool configuration.
	ConnectionPool PoolConfig
}

// PoolConfig configures connection pooling.
type PoolConfig struct {
	// MaxConns is the maximum number of connections.
	MaxConns int

	// IdleTimeout is how long idle connections are kept.
	IdleTimeout time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Address:        "localhost:50051",
		Namespace:      "default",
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		LocalFallback:  true,
		LocalCacheSize: 10000,
		CircuitBreaker: fault.CircuitBreakerConfig{
			Name:             "cache-client",
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
		ConnectionPool: PoolConfig{
			MaxConns:    10,
			IdleTimeout: 5 * time.Minute,
		},
	}
}

// Validate validates the configuration.
func (c *ClientConfig) Validate() error {
	if c.Address == "" {
		return ErrInvalidConfig
	}
	if c.Namespace == "" {
		c.Namespace = "default"
	}
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second
	}
	if c.LocalCacheSize <= 0 {
		c.LocalCacheSize = 10000
	}
	return nil
}
