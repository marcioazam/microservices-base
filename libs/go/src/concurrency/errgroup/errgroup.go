// Package errgroup provides a generic error group with result collection.
package errgroup

import (
	"context"
	"sync"
)

// Group collects results from goroutines, stopping on first error.
type Group[T any] struct {
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	results []T
	err     error
	limit   chan struct{}
}

// New creates a new Group.
func New[T any]() *Group[T] {
	return &Group[T]{
		results: make([]T, 0),
	}
}

// WithContext creates a new Group with context.
func WithContext[T any](ctx context.Context) (*Group[T], context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group[T]{
		cancel:  cancel,
		results: make([]T, 0),
	}, ctx
}

// SetLimit sets the maximum number of concurrent goroutines.
func (g *Group[T]) SetLimit(n int) {
	if n <= 0 {
		g.limit = nil
		return
	}
	g.limit = make(chan struct{}, n)
}

// Go starts a goroutine that may return an error.
func (g *Group[T]) Go(fn func() (T, error)) {
	if g.limit != nil {
		g.limit <- struct{}{}
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if g.limit != nil {
			defer func() { <-g.limit }()
		}

		result, err := fn()
		g.mu.Lock()
		defer g.mu.Unlock()

		if err != nil {
			if g.err == nil {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			}
			return
		}

		if g.err == nil {
			g.results = append(g.results, result)
		}
	}()
}

// Wait waits for all goroutines and returns results and first error.
func (g *Group[T]) Wait() ([]T, error) {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.results, g.err
}

// TryGo starts a goroutine if limit allows, returns false if at limit.
func (g *Group[T]) TryGo(fn func() (T, error)) bool {
	if g.limit != nil {
		select {
		case g.limit <- struct{}{}:
		default:
			return false
		}
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if g.limit != nil {
			defer func() { <-g.limit }()
		}

		result, err := fn()
		g.mu.Lock()
		defer g.mu.Unlock()

		if err != nil {
			if g.err == nil {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			}
			return
		}

		if g.err == nil {
			g.results = append(g.results, result)
		}
	}()

	return true
}

// CollectGroup collects all results even on error.
type CollectGroup[T any] struct {
	wg      sync.WaitGroup
	mu      sync.Mutex
	results []T
	errors  []error
	limit   chan struct{}
}

// NewCollect creates a new CollectGroup.
func NewCollect[T any]() *CollectGroup[T] {
	return &CollectGroup[T]{
		results: make([]T, 0),
		errors:  make([]error, 0),
	}
}

// SetLimit sets the maximum number of concurrent goroutines.
func (g *CollectGroup[T]) SetLimit(n int) {
	if n <= 0 {
		g.limit = nil
		return
	}
	g.limit = make(chan struct{}, n)
}

// Go starts a goroutine.
func (g *CollectGroup[T]) Go(fn func() (T, error)) {
	if g.limit != nil {
		g.limit <- struct{}{}
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if g.limit != nil {
			defer func() { <-g.limit }()
		}

		result, err := fn()
		g.mu.Lock()
		defer g.mu.Unlock()

		if err != nil {
			g.errors = append(g.errors, err)
		} else {
			g.results = append(g.results, result)
		}
	}()
}

// Wait waits for all goroutines and returns all results and errors.
func (g *CollectGroup[T]) Wait() ([]T, []error) {
	g.wg.Wait()
	return g.results, g.errors
}

// HasErrors returns true if any goroutine returned an error.
func (g *CollectGroup[T]) HasErrors() bool {
	g.wg.Wait()
	return len(g.errors) > 0
}
