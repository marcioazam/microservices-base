// Package redis provides Redis client for distributed state.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

// Client wraps Redis operations for resilience state.
type Client struct {
	rdb    *redis.Client
	prefix string
}

// Config holds Redis client configuration.
type Config struct {
	URL      string
	Password string
	DB       int
	Prefix   string
}

// NewClient creates a new Redis client.
func NewClient(cfg Config) (*Client, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	opt.DB = cfg.DB

	rdb := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "resilience:"
	}

	return &Client{
		rdb:    rdb,
		prefix: prefix,
	}, nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// SaveCircuitState saves circuit breaker state.
func (c *Client) SaveCircuitState(ctx context.Context, state domain.CircuitBreakerState) error {
	key := c.prefix + "circuit:" + state.ServiceName

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal circuit state: %w", err)
	}

	return c.rdb.Set(ctx, key, data, 0).Err()
}

// LoadCircuitState loads circuit breaker state.
func (c *Client) LoadCircuitState(ctx context.Context, serviceName string) (*domain.CircuitBreakerState, error) {
	key := c.prefix + "circuit:" + serviceName

	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get circuit state: %w", err)
	}

	var state domain.CircuitBreakerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal circuit state: %w", err)
	}

	return &state, nil
}

// IncrementRateLimit increments rate limit counter.
func (c *Client) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	fullKey := c.prefix + "ratelimit:" + key

	pipe := c.rdb.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("increment rate limit: %w", err)
	}

	return incr.Val(), nil
}

// GetRateLimitCount gets current rate limit count.
func (c *Client) GetRateLimitCount(ctx context.Context, key string) (int64, error) {
	fullKey := c.prefix + "ratelimit:" + key

	count, err := c.rdb.Get(ctx, fullKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get rate limit count: %w", err)
	}

	return count, nil
}

// GetRateLimitTTL gets remaining TTL for rate limit key.
func (c *Client) GetRateLimitTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.prefix + "ratelimit:" + key

	ttl, err := c.rdb.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("get rate limit ttl: %w", err)
	}

	return ttl, nil
}

// HealthCheck checks Redis connectivity.
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}
