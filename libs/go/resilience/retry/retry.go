// Package retry provides generic retry functionality with configurable backoff.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration.
type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	Jitter      float64
	RetryIf     func(error) bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.1,
		RetryIf:     func(error) bool { return true },
	}
}

// Option configures retry behavior.
type Option func(*Config)

// WithMaxAttempts sets the maximum number of retry attempts.
func WithMaxAttempts(n int) Option {
	return func(c *Config) { c.MaxAttempts = n }
}

// WithBaseDelay sets the base delay between retries.
func WithBaseDelay(d time.Duration) Option {
	return func(c *Config) { c.BaseDelay = d }
}

// WithMaxDelay sets the maximum delay between retries.
func WithMaxDelay(d time.Duration) Option {
	return func(c *Config) { c.MaxDelay = d }
}

// WithMultiplier sets the exponential backoff multiplier.
func WithMultiplier(m float64) Option {
	return func(c *Config) { c.Multiplier = m }
}

// WithJitter sets the jitter percentage (0-1).
func WithJitter(percent float64) Option {
	return func(c *Config) { c.Jitter = percent }
}

// WithRetryIf sets the retry predicate.
func WithRetryIf(predicate func(error) bool) Option {
	return func(c *Config) { c.RetryIf = predicate }
}

// RetryError wraps the last error with attempt information.
type RetryError struct {
	Attempts int
	LastErr  error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("all %d retry attempts exhausted: %v", e.Attempts, e.LastErr)
}

func (e *RetryError) Unwrap() error {
	return e.LastErr
}

// Retry executes the operation with retry policy.
func Retry[T any](ctx context.Context, op func() (T, error), opts ...Option) (T, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}

	var zero T
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result, err := op()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if !config.RetryIf(err) {
			return zero, err
		}

		// Check if this was the last attempt
		if attempt >= config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(attempt, config)

		// Wait or check for context cancellation
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, &RetryError{
		Attempts: config.MaxAttempts,
		LastErr:  lastErr,
	}
}

// RetryWithContext executes the operation with retry policy and context-aware operation.
func RetryWithContext[T any](ctx context.Context, op func(context.Context) (T, error), opts ...Option) (T, error) {
	return Retry(ctx, func() (T, error) {
		return op(ctx)
	}, opts...)
}

func calculateDelay(attempt int, config Config) time.Duration {
	// Exponential backoff
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter
	if config.Jitter > 0 {
		jitter := delay * config.Jitter * (rand.Float64()*2 - 1)
		delay += jitter
	}

	return time.Duration(delay)
}

// IsRetryError checks if the error is a RetryError.
func IsRetryError(err error) bool {
	var retryErr *RetryError
	return errors.As(err, &retryErr)
}

// GetRetryAttempts returns the number of attempts from a RetryError.
func GetRetryAttempts(err error) int {
	var retryErr *RetryError
	if errors.As(err, &retryErr) {
		return retryErr.Attempts
	}
	return 0
}

// Do is a simpler version of Retry for operations that don't return a value.
func Do(ctx context.Context, op func() error, opts ...Option) error {
	_, err := Retry(ctx, func() (struct{}, error) {
		return struct{}{}, op()
	}, opts...)
	return err
}
