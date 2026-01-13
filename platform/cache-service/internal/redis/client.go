// Package redis provides Redis client implementation.
package redis

import (
	"context"
	"time"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/auth-platform/cache-service/internal/config"
	"github.com/auth-platform/cache-service/internal/loggingclient"
	"github.com/redis/go-redis/v9"
)

// Client implements the cache.RedisClient interface.
type Client struct {
	client      redis.UniversalClient
	clusterMode bool
	logger      *loggingclient.Client
}

// NewClient creates a new Redis client based on configuration.
func NewClient(cfg config.RedisConfig, logger *loggingclient.Client) (*Client, error) {
	if logger == nil {
		logger = loggingclient.NewNoop()
	}

	var client redis.UniversalClient

	if cfg.ClusterMode {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Addresses,
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Addresses[0],
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error(ctx, "redis connection failed", loggingclient.Error(err))
		return nil, cache.WrapError(cache.ErrRedisDown, "failed to connect to redis", err)
	}

	logger.Info(ctx, "redis client connected",
		loggingclient.Bool("cluster_mode", cfg.ClusterMode),
		loggingclient.Int("pool_size", cfg.PoolSize),
	)

	return &Client{
		client:      client,
		clusterMode: cfg.ClusterMode,
		logger:      logger,
	}, nil
}

// Get retrieves a value by key.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, cache.ErrNotFound
		}
		return nil, cache.WrapError(cache.ErrRedisDown, "redis get failed", err)
	}
	return result, nil
}

// Set stores a value with TTL.
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := c.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return cache.WrapError(cache.ErrRedisDown, "redis set failed", err)
	}
	return nil
}

// Del deletes one or more keys.
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	result, err := c.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, cache.WrapError(cache.ErrRedisDown, "redis del failed", err)
	}
	return result, nil
}

// MGet retrieves multiple values.
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	result, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, cache.WrapError(cache.ErrRedisDown, "redis mget failed", err)
	}
	return result, nil
}

// MSet stores multiple key-value pairs.
func (c *Client) MSet(ctx context.Context, pairs ...interface{}) error {
	err := c.client.MSet(ctx, pairs...).Err()
	if err != nil {
		return cache.WrapError(cache.ErrRedisDown, "redis mset failed", err)
	}
	return nil
}

// SetWithExpire stores multiple key-value pairs with individual TTL using pipeline.
func (c *Client) SetWithExpire(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	pipe := c.client.Pipeline()
	for key, value := range entries {
		pipe.Set(ctx, key, value, ttl)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return cache.WrapError(cache.ErrRedisDown, "redis pipeline set failed", err)
	}
	return nil
}

// Ping checks Redis connectivity.
func (c *Client) Ping(ctx context.Context) error {
	err := c.client.Ping(ctx).Err()
	if err != nil {
		return cache.WrapError(cache.ErrRedisDown, "redis ping failed", err)
	}
	return nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.client.Close()
}

// IsClusterMode returns whether the client is in cluster mode.
func (c *Client) IsClusterMode() bool {
	return c.clusterMode
}
