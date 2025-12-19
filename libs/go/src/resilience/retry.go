package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// Retry executes operation with retry logic.
func Retry(ctx context.Context, config RetryConfig, op func(context.Context) error) error {
	if err := config.Validate(); err != nil {
		return err
	}

	var lastErr error
	var errors []error
	startTime := time.Now()

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		lastErr = op(ctx)
		if lastErr == nil {
			return nil
		}

		errors = append(errors, lastErr)

		if !config.RetryIf(lastErr) {
			return lastErr
		}

		if attempt < config.MaxAttempts-1 {
			delay := calculateDelay(config, attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return NewRetryExhaustedError(
		"",
		"",
		config.MaxAttempts,
		time.Since(startTime),
		errors,
	)
}

// RetryWithResult executes operation returning Result[T].
func RetryWithResult[T any](ctx context.Context, config RetryConfig, op func(context.Context) (T, error)) functional.Result[T] {
	if err := config.Validate(); err != nil {
		return functional.Err[T](err)
	}

	var lastErr error
	var errors []error
	startTime := time.Now()

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return functional.Err[T](ctx.Err())
		}

		value, err := op(ctx)
		if err == nil {
			return functional.Ok(value)
		}

		lastErr = err
		errors = append(errors, lastErr)

		if !config.RetryIf(lastErr) {
			return functional.Err[T](lastErr)
		}

		if attempt < config.MaxAttempts-1 {
			delay := calculateDelay(config, attempt)
			select {
			case <-ctx.Done():
				return functional.Err[T](ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	return functional.Err[T](NewRetryExhaustedError(
		"",
		"",
		config.MaxAttempts,
		time.Since(startTime),
		errors,
	))
}

func calculateDelay(config RetryConfig, attempt int) time.Duration {
	base := float64(config.InitialInterval) * math.Pow(config.Multiplier, float64(attempt))
	delay := time.Duration(base)

	if delay > config.MaxInterval {
		delay = config.MaxInterval
	}

	return applyJitter(delay, config.JitterStrategy)
}

func applyJitter(delay time.Duration, strategy JitterStrategy) time.Duration {
	switch strategy {
	case NoJitter:
		return delay
	case FullJitter:
		return time.Duration(rand.Int63n(int64(delay)))
	case EqualJitter:
		half := delay / 2
		return half + time.Duration(rand.Int63n(int64(half)))
	case DecorrelatedJitter:
		return time.Duration(float64(delay) * (1 + rand.Float64()))
	default:
		return delay
	}
}

// RetryableFunc wraps a function for retry.
type RetryableFunc[T any] func(context.Context) (T, error)

// WithRetry creates a retryable version of a function.
func WithRetry[T any](config RetryConfig, fn RetryableFunc[T]) RetryableFunc[T] {
	return func(ctx context.Context) (T, error) {
		result := RetryWithResult(ctx, config, fn)
		if result.IsOk() {
			return result.Unwrap(), nil
		}
		var zero T
		return zero, result.UnwrapErr()
	}
}
