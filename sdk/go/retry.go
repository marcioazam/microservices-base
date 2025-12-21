package authplatform

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// RetryPolicy configures retry behavior for operations.
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts (not including initial).
	MaxRetries int
	// BaseDelay is the initial delay before the first retry.
	BaseDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// Jitter adds randomness to delays (0.0 to 1.0).
	Jitter float64
	// RetryableStatusCodes are HTTP status codes that trigger retry.
	RetryableStatusCodes []int
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:           3,
		BaseDelay:            time.Second,
		MaxDelay:             30 * time.Second,
		Jitter:               0.2,
		RetryableStatusCodes: []int{429, 502, 503, 504},
	}
}

// RetryOption configures a RetryPolicy.
type RetryOption func(*RetryPolicy)

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) RetryOption {
	return func(p *RetryPolicy) {
		p.MaxRetries = n
	}
}

// WithBaseDelay sets the base delay.
func WithBaseDelay(d time.Duration) RetryOption {
	return func(p *RetryPolicy) {
		p.BaseDelay = d
	}
}

// WithMaxDelay sets the maximum delay.
func WithMaxDelay(d time.Duration) RetryOption {
	return func(p *RetryPolicy) {
		p.MaxDelay = d
	}
}

// WithJitter sets the jitter factor.
func WithJitter(j float64) RetryOption {
	return func(p *RetryPolicy) {
		p.Jitter = j
	}
}

// NewRetryPolicy creates a new retry policy with options.
func NewRetryPolicy(opts ...RetryOption) *RetryPolicy {
	p := DefaultRetryPolicy()
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// CalculateDelay computes the delay for a given attempt using exponential backoff.
func (p *RetryPolicy) CalculateDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Exponential backoff: baseDelay * 2^attempt
	delay := float64(p.BaseDelay) * math.Pow(2, float64(attempt))

	// Apply jitter
	if p.Jitter > 0 {
		jitterRange := delay * p.Jitter
		delay = delay - jitterRange + (rand.Float64() * 2 * jitterRange)
	}

	// Clamp to bounds
	if delay < float64(p.BaseDelay) {
		delay = float64(p.BaseDelay)
	}
	if delay > float64(p.MaxDelay) {
		delay = float64(p.MaxDelay)
	}

	return time.Duration(delay)
}

// IsRetryable checks if an HTTP status code should trigger a retry.
func (p *RetryPolicy) IsRetryable(statusCode int) bool {
	for _, code := range p.RetryableStatusCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

// IsRetryableError checks if an error should trigger a retry.
func (p *RetryPolicy) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Network errors are retryable
	if IsNetwork(err) {
		return true
	}
	// Rate limit errors are retryable
	if IsRateLimited(err) {
		return true
	}
	return false
}

// ParseRetryAfter parses the Retry-After header value.
func ParseRetryAfter(header string) (time.Duration, bool) {
	if header == "" {
		return 0, false
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second, true
	}

	// Try parsing as HTTP-date
	if t, err := http.ParseTime(header); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay, true
		}
	}

	return 0, false
}

// RetryResult represents the outcome of a retry operation.
type RetryResult[T any] struct {
	Value    T
	Err      error
	Attempts int
}

// Retry executes an operation with retry logic.
func Retry[T any](ctx context.Context, p *RetryPolicy, op func(ctx context.Context) (T, error)) RetryResult[T] {
	var result RetryResult[T]
	var lastErr error

	maxAttempts := p.MaxRetries + 1 // initial + retries

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result.Attempts = attempt + 1

		// Check context before attempting
		if err := ctx.Err(); err != nil {
			result.Err = WrapError(ErrCodeNetwork, "context cancelled", err)
			return result
		}

		value, err := op(ctx)
		if err == nil {
			result.Value = value
			return result
		}

		lastErr = err

		// Check if error is retryable
		if !p.IsRetryableError(err) {
			result.Err = err
			return result
		}

		// Don't sleep after the last attempt
		if attempt < maxAttempts-1 {
			delay := p.CalculateDelay(attempt)

			select {
			case <-ctx.Done():
				result.Err = WrapError(ErrCodeNetwork, "context cancelled during retry", ctx.Err())
				return result
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	result.Err = lastErr
	return result
}

// RetryWithResponse executes an HTTP operation with retry logic.
func RetryWithResponse(ctx context.Context, p *RetryPolicy, op func(ctx context.Context) (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	maxAttempts := p.MaxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check context before attempting
		if err := ctx.Err(); err != nil {
			return nil, WrapError(ErrCodeNetwork, "context cancelled", err)
		}

		resp, err := op(ctx)
		if err != nil {
			lastErr = WrapError(ErrCodeNetwork, "request failed", err)

			if attempt < maxAttempts-1 {
				delay := p.CalculateDelay(attempt)
				select {
				case <-ctx.Done():
					return nil, WrapError(ErrCodeNetwork, "context cancelled during retry", ctx.Err())
				case <-time.After(delay):
				}
			}
			continue
		}

		// Check if status code is retryable
		if !p.IsRetryable(resp.StatusCode) {
			return resp, nil
		}

		// Close body for retryable responses
		lastResp = resp
		resp.Body.Close()

		// Determine delay
		delay := p.CalculateDelay(attempt)

		// Check Retry-After header
		if retryAfter, ok := ParseRetryAfter(resp.Header.Get("Retry-After")); ok {
			if retryAfter > delay {
				delay = retryAfter
			}
			// Clamp to max delay
			if delay > p.MaxDelay {
				delay = p.MaxDelay
			}
		}

		lastErr = NewError(ErrCodeRateLimited, "rate limited")

		if attempt < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, WrapError(ErrCodeNetwork, "context cancelled during retry", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return lastResp, nil
}

// RetryableFunc is a function that can be retried.
type RetryableFunc[T any] func(ctx context.Context) (T, error)

// WithRetry wraps a function with retry logic.
func WithRetry[T any](p *RetryPolicy, fn RetryableFunc[T]) RetryableFunc[T] {
	return func(ctx context.Context) (T, error) {
		result := Retry(ctx, p, fn)
		return result.Value, result.Err
	}
}
