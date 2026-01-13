package redis

import (
	"context"
	"time"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/auth-platform/cache-service/internal/loggingclient"
	"github.com/authcorp/libs/go/src/fault"
)

// ProtectedClient wraps a Redis client with circuit breaker protection.
type ProtectedClient struct {
	client  *Client
	breaker *fault.CircuitBreaker
	local   cache.LocalCache
	logger  *loggingclient.Client
}

// NewProtectedClient creates a new protected Redis client.
func NewProtectedClient(client *Client, breaker *fault.CircuitBreaker, local cache.LocalCache, logger *loggingclient.Client) *ProtectedClient {
	if logger == nil {
		logger = loggingclient.NewNoop()
	}
	return &ProtectedClient{
		client:  client,
		breaker: breaker,
		local:   local,
		logger:  logger,
	}
}

// Get retrieves a value with circuit breaker protection.
func (p *ProtectedClient) Get(ctx context.Context, key string) ([]byte, cache.Source, error) {
	var result []byte
	var source cache.Source = cache.SourceRedis

	err := p.breaker.Execute(ctx, func(ctx context.Context) error {
		var err error
		result, err = p.client.Get(ctx, key)
		return err
	})

	if err != nil {
		if p.local != nil {
			if val, ok := p.local.Get(key); ok {
				p.logger.Debug(ctx, "redis fallback to local cache",
					loggingclient.String("key", key),
				)
				return val, cache.SourceLocal, nil
			}
		}

		if isCircuitOpenError(err) {
			p.logger.Warn(ctx, "circuit breaker open for redis get",
				loggingclient.String("key", key),
			)
			return nil, source, cache.ErrCircuitBreakerOpen
		}
		return nil, source, err
	}

	if p.local != nil {
		p.local.Set(key, result, time.Hour)
	}

	return result, source, nil
}

// Set stores a value with circuit breaker protection.
func (p *ProtectedClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := p.breaker.Execute(ctx, func(ctx context.Context) error {
		return p.client.Set(ctx, key, value, ttl)
	})

	if err != nil {
		if p.local != nil && isCircuitOpenError(err) {
			p.local.Set(key, value, ttl)
			p.logger.Warn(ctx, "circuit open, stored in local cache",
				loggingclient.String("key", key),
			)
			return nil
		}
		return err
	}

	if p.local != nil {
		p.local.Set(key, value, ttl)
	}

	return nil
}

// Del deletes keys with circuit breaker protection.
func (p *ProtectedClient) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64

	err := p.breaker.Execute(ctx, func(ctx context.Context) error {
		var err error
		count, err = p.client.Del(ctx, keys...)
		return err
	})

	if err != nil {
		// Delete from local cache
		if p.local != nil {
			for _, key := range keys {
				p.local.Delete(key)
			}
		}

		if isCircuitOpenError(err) {
			return int64(len(keys)), nil // Assume success in degraded mode
		}
		return 0, err
	}

	// Delete from local cache
	if p.local != nil {
		for _, key := range keys {
			p.local.Delete(key)
		}
	}

	return count, nil
}

// MGet retrieves multiple values with circuit breaker protection.
func (p *ProtectedClient) MGet(ctx context.Context, keys ...string) ([]interface{}, cache.Source, error) {
	var results []interface{}
	var source cache.Source = cache.SourceRedis

	err := p.breaker.Execute(ctx, func(ctx context.Context) error {
		var err error
		results, err = p.client.MGet(ctx, keys...)
		return err
	})

	if err != nil {
		// Try local cache fallback
		if p.local != nil {
			results = make([]interface{}, len(keys))
			allFound := true
			for i, key := range keys {
				if val, ok := p.local.Get(key); ok {
					results[i] = string(val)
				} else {
					results[i] = nil
					allFound = false
				}
			}
			if allFound || isCircuitOpenError(err) {
				return results, cache.SourceLocal, nil
			}
		}
		return nil, source, err
	}

	return results, source, nil
}

// SetWithExpire stores multiple values with circuit breaker protection.
func (p *ProtectedClient) SetWithExpire(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	err := p.breaker.Execute(ctx, func(ctx context.Context) error {
		return p.client.SetWithExpire(ctx, entries, ttl)
	})

	if err != nil {
		// Store in local cache as fallback
		if p.local != nil && isCircuitOpenError(err) {
			for key, value := range entries {
				p.local.Set(key, value, ttl)
			}
			return nil
		}
		return err
	}

	// Update local cache
	if p.local != nil {
		for key, value := range entries {
			p.local.Set(key, value, ttl)
		}
	}

	return nil
}

// Ping checks connectivity with circuit breaker protection.
func (p *ProtectedClient) Ping(ctx context.Context) error {
	return p.breaker.Execute(ctx, func(ctx context.Context) error {
		return p.client.Ping(ctx)
	})
}

// Close closes the Redis connection.
func (p *ProtectedClient) Close() error {
	return p.client.Close()
}

// CircuitState returns the current circuit breaker state.
func (p *ProtectedClient) CircuitState() fault.State {
	return p.breaker.State()
}

// ResetCircuit resets the circuit breaker.
func (p *ProtectedClient) ResetCircuit() {
	p.breaker.Reset()
}

// isCircuitOpenError checks if the error is a circuit open error.
func isCircuitOpenError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*fault.CircuitOpenError)
	return ok
}
