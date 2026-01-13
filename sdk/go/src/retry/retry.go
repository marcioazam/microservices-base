package retry

import (
	"context"
	"net/http"
	"time"

	"github.com/auth-platform/sdk-go/src/errors"
)

// Result represents the outcome of a retry operation.
type Result[T any] struct {
	Value    T
	Err      error
	Attempts int
}

// IsRetryableError checks if an error should trigger a retry.
func (p *Policy) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.IsNetwork(err) {
		return true
	}
	if errors.IsRateLimited(err) {
		return true
	}
	return false
}

// Retry executes an operation with retry logic.
func Retry[T any](ctx context.Context, p *Policy, op func(ctx context.Context) (T, error)) Result[T] {
	var result Result[T]
	var lastErr error

	maxAttempts := p.MaxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result.Attempts = attempt + 1

		if err := ctx.Err(); err != nil {
			result.Err = errors.WrapError(errors.ErrCodeNetwork, "context cancelled", err)
			return result
		}

		value, err := op(ctx)
		if err == nil {
			result.Value = value
			return result
		}

		lastErr = err

		if !p.IsRetryableError(err) {
			result.Err = err
			return result
		}

		if attempt < maxAttempts-1 {
			delay := p.CalculateDelay(attempt)
			select {
			case <-ctx.Done():
				result.Err = errors.WrapError(errors.ErrCodeNetwork, "context cancelled during retry", ctx.Err())
				return result
			case <-time.After(delay):
			}
		}
	}

	result.Err = lastErr
	return result
}

// RetryWithResponse executes an HTTP operation with retry logic.
func RetryWithResponse(ctx context.Context, p *Policy, op func(ctx context.Context) (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	maxAttempts := p.MaxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, errors.WrapError(errors.ErrCodeNetwork, "context cancelled", err)
		}

		resp, err := op(ctx)
		if err != nil {
			lastErr = errors.WrapError(errors.ErrCodeNetwork, "request failed", err)
			if attempt < maxAttempts-1 {
				delay := p.CalculateDelay(attempt)
				select {
				case <-ctx.Done():
					return nil, errors.WrapError(errors.ErrCodeNetwork, "context cancelled during retry", ctx.Err())
				case <-time.After(delay):
				}
			}
			continue
		}

		if !p.IsRetryable(resp.StatusCode) {
			return resp, nil
		}

		lastResp = resp
		resp.Body.Close()

		delay := p.CalculateDelay(attempt)
		if retryAfter, ok := ParseRetryAfter(resp.Header.Get("Retry-After")); ok {
			if retryAfter > delay {
				delay = retryAfter
			}
			if delay > p.MaxDelay {
				delay = p.MaxDelay
			}
		}

		lastErr = errors.NewError(errors.ErrCodeRateLimited, "rate limited")

		if attempt < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, errors.WrapError(errors.ErrCodeNetwork, "context cancelled during retry", ctx.Err())
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
func WithRetry[T any](p *Policy, fn RetryableFunc[T]) RetryableFunc[T] {
	return func(ctx context.Context) (T, error) {
		result := Retry(ctx, p, fn)
		return result.Value, result.Err
	}
}
