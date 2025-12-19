// Package pool provides a generic object pool.
package pool

import (
	"context"
	"sync"
	"sync/atomic"
)

// Pool is a generic object pool.
type Pool[T any] struct {
	factory  func() T
	reset    func(T)
	items    chan T
	capacity int
	hits     int64
	misses   int64
	mu       sync.Mutex
}

// Stats contains pool statistics.
type Stats struct {
	Hits     int64
	Misses   int64
	Size     int
	Capacity int
}

// New creates a new Pool with a factory and reset function.
func New[T any](factory func() T, reset func(T)) *Pool[T] {
	return &Pool[T]{
		factory:  factory,
		reset:    reset,
		items:    make(chan T, 100),
		capacity: 100,
	}
}

// WithCapacity sets the pool capacity.
func (p *Pool[T]) WithCapacity(n int) *Pool[T] {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create new channel with new capacity
	newItems := make(chan T, n)

	// Transfer existing items
	close(p.items)
	for item := range p.items {
		select {
		case newItems <- item:
		default:
			// Discard if new capacity is smaller
		}
	}

	p.items = newItems
	p.capacity = n
	return p
}

// Acquire gets an object from the pool or creates a new one.
func (p *Pool[T]) Acquire() T {
	select {
	case item := <-p.items:
		atomic.AddInt64(&p.hits, 1)
		return item
	default:
		atomic.AddInt64(&p.misses, 1)
		return p.factory()
	}
}

// AcquireContext gets an object with context support.
func (p *Pool[T]) AcquireContext(ctx context.Context) (T, error) {
	select {
	case item := <-p.items:
		atomic.AddInt64(&p.hits, 1)
		return item, nil
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	default:
		atomic.AddInt64(&p.misses, 1)
		return p.factory(), nil
	}
}

// Release returns an object to the pool.
func (p *Pool[T]) Release(item T) {
	if p.reset != nil {
		p.reset(item)
	}
	select {
	case p.items <- item:
	default:
		// Pool is full, discard
	}
}

// Stats returns pool statistics.
func (p *Pool[T]) Stats() Stats {
	return Stats{
		Hits:     atomic.LoadInt64(&p.hits),
		Misses:   atomic.LoadInt64(&p.misses),
		Size:     len(p.items),
		Capacity: p.capacity,
	}
}

// Drain removes all items from the pool.
func (p *Pool[T]) Drain() {
	for {
		select {
		case <-p.items:
		default:
			return
		}
	}
}

// Size returns the current number of items in the pool.
func (p *Pool[T]) Size() int {
	return len(p.items)
}

// Use acquires an object, calls fn, and releases it.
func (p *Pool[T]) Use(fn func(T)) {
	item := p.Acquire()
	defer p.Release(item)
	fn(item)
}

// UseWithResult acquires an object, calls fn, releases it, and returns the result.
func UseWithResult[T, R any](p *Pool[T], fn func(T) R) R {
	item := p.Acquire()
	defer p.Release(item)
	return fn(item)
}

// UseWithError acquires an object, calls fn, releases it, and returns the result and error.
func UseWithError[T, R any](p *Pool[T], fn func(T) (R, error)) (R, error) {
	item := p.Acquire()
	defer p.Release(item)
	return fn(item)
}
