// Package timeout provides generic timeout management.
package timeout

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrTimeout = errors.New("operation timed out")
)

// Config holds timeout configuration.
type Config struct {
	Default      time.Duration
	Max          time.Duration
	PerOperation map[string]time.Duration
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Default:      30 * time.Second,
		Max:          5 * time.Minute,
		PerOperation: make(map[string]time.Duration),
	}
}

// TimeoutManager manages timeouts for operations.
type TimeoutManager[T any] struct {
	mu           sync.RWMutex
	defaultTimeout time.Duration
	maxTimeout     time.Duration
	perOperation   map[string]time.Duration
}

// New creates a new timeout manager.
func New[T any](defaultTimeout time.Duration) *TimeoutManager[T] {
	return &TimeoutManager[T]{
		defaultTimeout: defaultTimeout,
		maxTimeout:     5 * time.Minute,
		perOperation:   make(map[string]time.Duration),
	}
}

// NewWithConfig creates a new timeout manager with configuration.
func NewWithConfig[T any](config Config) *TimeoutManager[T] {
	perOp := make(map[string]time.Duration)
	for k, v := range config.PerOperation {
		perOp[k] = v
	}
	return &TimeoutManager[T]{
		defaultTimeout: config.Default,
		maxTimeout:     config.Max,
		perOperation:   perOp,
	}
}

// Execute runs the operation with timeout.
func (tm *TimeoutManager[T]) Execute(ctx context.Context, op string, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	timeout := tm.getTimeout(op)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel for result
	type result struct {
		value T
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		value, err := fn(ctx)
		resultCh <- result{value, err}
	}()

	select {
	case r := <-resultCh:
		return r.value, r.err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return zero, ErrTimeout
		}
		return zero, ctx.Err()
	}
}

// SetOperationTimeout sets a timeout for a specific operation.
func (tm *TimeoutManager[T]) SetOperationTimeout(op string, timeout time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if timeout > tm.maxTimeout {
		timeout = tm.maxTimeout
	}
	tm.perOperation[op] = timeout
}

// GetOperationTimeout returns the timeout for a specific operation.
func (tm *TimeoutManager[T]) GetOperationTimeout(op string) time.Duration {
	return tm.getTimeout(op)
}

// SetDefaultTimeout sets the default timeout.
func (tm *TimeoutManager[T]) SetDefaultTimeout(timeout time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if timeout > tm.maxTimeout {
		timeout = tm.maxTimeout
	}
	tm.defaultTimeout = timeout
}

// SetMaxTimeout sets the maximum allowed timeout.
func (tm *TimeoutManager[T]) SetMaxTimeout(timeout time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.maxTimeout = timeout
}

func (tm *TimeoutManager[T]) getTimeout(op string) time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if timeout, ok := tm.perOperation[op]; ok {
		return timeout
	}
	return tm.defaultTimeout
}

// WithTimeout is a simple helper to execute a function with timeout.
func WithTimeout[T any](ctx context.Context, timeout time.Duration, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		value T
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		value, err := fn(ctx)
		resultCh <- result{value, err}
	}()

	select {
	case r := <-resultCh:
		return r.value, r.err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return zero, ErrTimeout
		}
		return zero, ctx.Err()
	}
}

// Do is a simple helper for operations that don't return a value.
func Do(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) error) error {
	_, err := WithTimeout(ctx, timeout, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}
