package fault

import (
	"context"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// Timeout wraps an operation with timeout.
func Timeout(ctx context.Context, config TimeoutConfig, op func(context.Context) error) error {
	if err := config.Validate(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	done := make(chan error, 1)
	start := time.Now()

	go func() {
		done <- op(timeoutCtx)
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		if config.OnTimeout != nil {
			config.OnTimeout()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return NewTimeoutError("", "", config.Timeout, time.Since(start), timeoutCtx.Err())
	}
}

// TimeoutWithResult wraps operation returning Result[T].
func TimeoutWithResult[T any](ctx context.Context, config TimeoutConfig, op func(context.Context) (T, error)) functional.Result[T] {
	if err := config.Validate(); err != nil {
		return functional.Err[T](err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	type result struct {
		value T
		err   error
	}
	done := make(chan result, 1)
	start := time.Now()

	go func() {
		v, err := op(timeoutCtx)
		done <- result{v, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			return functional.Err[T](r.err)
		}
		return functional.Ok(r.value)
	case <-timeoutCtx.Done():
		if config.OnTimeout != nil {
			config.OnTimeout()
		}
		if ctx.Err() != nil {
			return functional.Err[T](ctx.Err())
		}
		return functional.Err[T](NewTimeoutError("", "", config.Timeout, time.Since(start), timeoutCtx.Err()))
	}
}

// WithTimeout creates a timeout-wrapped function.
func WithTimeout[T any](config TimeoutConfig, fn func(context.Context) (T, error)) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		result := TimeoutWithResult(ctx, config, fn)
		if result.IsOk() {
			return result.Unwrap(), nil
		}
		var zero T
		return zero, result.UnwrapErr()
	}
}
