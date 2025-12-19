// Package async provides generic async utilities for concurrent operations.
package async

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/libs/go/functional/result"
)

// Future represents an async computation.
type Future[T any] struct {
	done   chan struct{}
	value  T
	err    error
	once   sync.Once
}

// Go starts an async computation.
func Go[T any](fn func() (T, error)) *Future[T] {
	f := &Future[T]{done: make(chan struct{})}
	go func() {
		f.value, f.err = fn()
		close(f.done)
	}()
	return f
}

// GoContext starts an async computation with context.
func GoContext[T any](ctx context.Context, fn func(context.Context) (T, error)) *Future[T] {
	f := &Future[T]{done: make(chan struct{})}
	go func() {
		f.value, f.err = fn(ctx)
		close(f.done)
	}()
	return f
}

// Wait blocks until the future completes.
func (f *Future[T]) Wait() (T, error) {
	<-f.done
	return f.value, f.err
}

// WaitContext blocks until the future completes or context is cancelled.
func (f *Future[T]) WaitContext(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.value, f.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Done returns a channel that closes when the future completes.
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// Result returns the result as a Result type.
func (f *Future[T]) Result() result.Result[T] {
	<-f.done
	if f.err != nil {
		return result.Err[T](f.err)
	}
	return result.Ok(f.value)
}

// Parallel runs multiple functions concurrently and returns all results.
func Parallel[T any](fns ...func() (T, error)) ([]T, []error) {
	results := make([]T, len(fns))
	errors := make([]error, len(fns))
	var wg sync.WaitGroup
	wg.Add(len(fns))

	for i, fn := range fns {
		go func(idx int, f func() (T, error)) {
			defer wg.Done()
			results[idx], errors[idx] = f()
		}(i, fn)
	}

	wg.Wait()
	return results, errors
}

// ParallelContext runs multiple functions concurrently with context.
func ParallelContext[T any](ctx context.Context, fns ...func(context.Context) (T, error)) ([]T, []error) {
	results := make([]T, len(fns))
	errors := make([]error, len(fns))
	var wg sync.WaitGroup
	wg.Add(len(fns))

	for i, fn := range fns {
		go func(idx int, f func(context.Context) (T, error)) {
			defer wg.Done()
			results[idx], errors[idx] = f(ctx)
		}(i, fn)
	}

	wg.Wait()
	return results, errors
}

// Race runs multiple functions and returns the first result.
func Race[T any](fns ...func() (T, error)) (T, error, int) {
	type raceResult struct {
		value T
		err   error
		index int
	}

	ch := make(chan raceResult, len(fns))
	for i, fn := range fns {
		go func(idx int, f func() (T, error)) {
			v, e := f()
			ch <- raceResult{value: v, err: e, index: idx}
		}(i, fn)
	}

	result := <-ch
	return result.value, result.err, result.index
}

// RaceContext runs multiple functions and returns the first result with context.
func RaceContext[T any](ctx context.Context, fns ...func(context.Context) (T, error)) (T, error, int) {
	type raceResult struct {
		value T
		err   error
		index int
	}

	ch := make(chan raceResult, len(fns))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, fn := range fns {
		go func(idx int, f func(context.Context) (T, error)) {
			v, e := f(ctx)
			select {
			case ch <- raceResult{value: v, err: e, index: idx}:
			case <-ctx.Done():
			}
		}(i, fn)
	}

	select {
	case result := <-ch:
		return result.value, result.err, result.index
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err(), -1
	}
}

// WithTimeout runs a function with a timeout.
func WithTimeout[T any](timeout time.Duration, fn func() (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan struct{})
	var value T
	var err error

	go func() {
		value, err = fn()
		close(done)
	}()

	select {
	case <-done:
		return value, err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Collect runs functions and collects successful results.
func Collect[T any](fns ...func() (T, error)) []T {
	results, errors := Parallel(fns...)
	collected := make([]T, 0, len(results))
	for i, r := range results {
		if errors[i] == nil {
			collected = append(collected, r)
		}
	}
	return collected
}

// FanOut distributes items to workers and collects results.
func FanOut[T, R any](items []T, workers int, fn func(T) (R, error)) ([]R, []error) {
	if workers <= 0 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}

	type workItem struct {
		index int
		item  T
	}

	type workResult struct {
		index int
		value R
		err   error
	}

	jobs := make(chan workItem, len(items))
	results := make(chan workResult, len(items))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				v, e := fn(job.item)
				results <- workResult{index: job.index, value: v, err: e}
			}
		}()
	}

	// Send jobs
	for i, item := range items {
		jobs <- workItem{index: i, item: item}
	}
	close(jobs)

	// Wait and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	values := make([]R, len(items))
	errors := make([]error, len(items))
	for r := range results {
		values[r.index] = r.value
		errors[r.index] = r.err
	}

	return values, errors
}

// All returns true if all futures complete successfully.
func All[T any](futures ...*Future[T]) bool {
	for _, f := range futures {
		if _, err := f.Wait(); err != nil {
			return false
		}
	}
	return true
}

// Any returns true if any future completes successfully.
func Any[T any](futures ...*Future[T]) bool {
	for _, f := range futures {
		if _, err := f.Wait(); err == nil {
			return true
		}
	}
	return false
}
