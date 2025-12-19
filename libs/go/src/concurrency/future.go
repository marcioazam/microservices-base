package concurrency

import (
	"context"
	"sync"

	"github.com/authcorp/libs/go/src/functional"
)

// Future represents an asynchronous computation.
type Future[T any] struct {
	result functional.Result[T]
	done   chan struct{}
	once   sync.Once
}

// NewFuture creates a future from an async operation.
func NewFuture[T any](fn func() (T, error)) *Future[T] {
	f := &Future[T]{
		done: make(chan struct{}),
	}
	go func() {
		value, err := fn()
		if err != nil {
			f.result = functional.Err[T](err)
		} else {
			f.result = functional.Ok(value)
		}
		close(f.done)
	}()
	return f
}

// NewFutureWithContext creates a future with context support.
func NewFutureWithContext[T any](ctx context.Context, fn func(context.Context) (T, error)) *Future[T] {
	f := &Future[T]{
		done: make(chan struct{}),
	}
	go func() {
		value, err := fn(ctx)
		if err != nil {
			f.result = functional.Err[T](err)
		} else {
			f.result = functional.Ok(value)
		}
		close(f.done)
	}()
	return f
}

// Wait blocks until the future completes.
func (f *Future[T]) Wait() functional.Result[T] {
	<-f.done
	return f.result
}

// WaitContext blocks until future completes or context cancelled.
func (f *Future[T]) WaitContext(ctx context.Context) functional.Result[T] {
	select {
	case <-f.done:
		return f.result
	case <-ctx.Done():
		return functional.Err[T](ctx.Err())
	}
}

// IsDone returns true if future has completed.
func (f *Future[T]) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// Result returns the result if done, None otherwise.
func (f *Future[T]) Result() functional.Option[functional.Result[T]] {
	if f.IsDone() {
		return functional.Some(f.result)
	}
	return functional.None[functional.Result[T]]()
}

// Map transforms the future value.
func Map[T, U any](f *Future[T], fn func(T) U) *Future[U] {
	return NewFuture(func() (U, error) {
		result := f.Wait()
		if result.IsErr() {
			var zero U
			return zero, result.UnwrapErr()
		}
		return fn(result.Unwrap()), nil
	})
}

// FlatMap chains futures.
func FlatMap[T, U any](f *Future[T], fn func(T) *Future[U]) *Future[U] {
	return NewFuture(func() (U, error) {
		result := f.Wait()
		if result.IsErr() {
			var zero U
			return zero, result.UnwrapErr()
		}
		return fn(result.Unwrap()).Wait().Unwrap(), nil
	})
}

// All waits for all futures to complete.
func All[T any](futures ...*Future[T]) []functional.Result[T] {
	results := make([]functional.Result[T], len(futures))
	var wg sync.WaitGroup
	wg.Add(len(futures))
	for i, f := range futures {
		go func(idx int, fut *Future[T]) {
			defer wg.Done()
			results[idx] = fut.Wait()
		}(i, f)
	}
	wg.Wait()
	return results
}

// Race returns the first future to complete.
func Race[T any](futures ...*Future[T]) functional.Result[T] {
	result := make(chan functional.Result[T], 1)
	for _, f := range futures {
		go func(fut *Future[T]) {
			select {
			case result <- fut.Wait():
			default:
			}
		}(f)
	}
	return <-result
}

// Resolve creates a completed future with a value.
func Resolve[T any](value T) *Future[T] {
	f := &Future[T]{
		done:   make(chan struct{}),
		result: functional.Ok(value),
	}
	close(f.done)
	return f
}

// Reject creates a completed future with an error.
func Reject[T any](err error) *Future[T] {
	f := &Future[T]{
		done:   make(chan struct{}),
		result: functional.Err[T](err),
	}
	close(f.done)
	return f
}
