package redis

import (
	"context"
	"time"

	"github.com/auth-platform/cache-service/internal/cache"
)

// Operations provides high-level cache operations on top of Redis client.
type Operations struct {
	client     *Client
	defaultTTL time.Duration
}

// NewOperations creates a new Operations instance.
func NewOperations(client *Client, defaultTTL time.Duration) *Operations {
	return &Operations{
		client:     client,
		defaultTTL: defaultTTL,
	}
}

// Get retrieves a value from cache.
func (o *Operations) Get(ctx context.Context, key string) (*cache.Entry, error) {
	if key == "" {
		return nil, cache.ErrInvalidKeyError
	}

	value, err := o.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return &cache.Entry{
		Value:  value,
		Source: cache.SourceRedis,
	}, nil
}

// Set stores a value in cache with TTL.
func (o *Operations) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return cache.ErrInvalidKeyError
	}
	if len(value) == 0 {
		return cache.ErrInvalidValueError
	}

	if ttl <= 0 {
		ttl = o.defaultTTL
	}

	return o.client.Set(ctx, key, value, ttl)
}

// Delete removes a value from cache.
func (o *Operations) Delete(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, cache.ErrInvalidKeyError
	}

	count, err := o.client.Del(ctx, key)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// BatchGet retrieves multiple values from cache.
func (o *Operations) BatchGet(ctx context.Context, keys []string) (map[string][]byte, []string, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	for _, key := range keys {
		if key == "" {
			return nil, nil, cache.ErrInvalidKeyError
		}
	}

	results, err := o.client.MGet(ctx, keys...)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[string][]byte)
	var missing []string

	for i, result := range results {
		if result == nil {
			missing = append(missing, keys[i])
			continue
		}

		switch v := result.(type) {
		case string:
			found[keys[i]] = []byte(v)
		case []byte:
			found[keys[i]] = v
		default:
			missing = append(missing, keys[i])
		}
	}

	return found, missing, nil
}

// BatchSet stores multiple key-value pairs in cache.
func (o *Operations) BatchSet(ctx context.Context, entries map[string][]byte, ttl time.Duration) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	for key, value := range entries {
		if key == "" {
			return 0, cache.ErrInvalidKeyError
		}
		if len(value) == 0 {
			return 0, cache.ErrInvalidValueError
		}
	}

	if ttl <= 0 {
		ttl = o.defaultTTL
	}

	err := o.client.SetWithExpire(ctx, entries, ttl)
	if err != nil {
		return 0, err
	}

	return len(entries), nil
}

// Ping checks Redis connectivity.
func (o *Operations) Ping(ctx context.Context) error {
	return o.client.Ping(ctx)
}

// Close closes the Redis connection.
func (o *Operations) Close() error {
	return o.client.Close()
}
