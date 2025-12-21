package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
)

// ObjectPool is a generic object pool.
type ObjectPool[T any] struct {
	factory  func() T
	reset    func(T)
	items    chan T
	capacity int
	hits     int64
	misses   int64
	mu       sync.Mutex
}

// PoolStats contains pool statistics.
type PoolStats struct {
	Hits     int64
	Misses   int64
	Size     int
	Capacity int
}

// NewObjectPool creates a new Pool with a factory and reset function.
func NewObjectPool[T any](factory func() T, reset func(T)) *ObjectPool[T] {
	return &ObjectPool[T]{
		factory:  factory,
		reset:    reset,
		items:    make(chan T, 100),
		capacity: 100,
	}
}

// WithCapacity sets the pool capacity.
func (p *ObjectPool[T]) WithCapacity(n int) *ObjectPool[T] {
	p.mu.Lock()
	defer p.mu.Unlock()

	newItems := make(chan T, n)
	close(p.items)
	for item := range p.items {
		select {
		case newItems <- item:
		default:
		}
	}

	p.items = newItems
	p.capacity = n
	return p
}

// Acquire gets an object from the pool or creates a new one.
func (p *ObjectPool[T]) Acquire() T {
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
func (p *ObjectPool[T]) AcquireContext(ctx context.Context) (T, error) {
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
func (p *ObjectPool[T]) Release(item T) {
	if p.reset != nil {
		p.reset(item)
	}
	select {
	case p.items <- item:
	default:
	}
}

// Stats returns pool statistics.
func (p *ObjectPool[T]) Stats() PoolStats {
	return PoolStats{
		Hits:     atomic.LoadInt64(&p.hits),
		Misses:   atomic.LoadInt64(&p.misses),
		Size:     len(p.items),
		Capacity: p.capacity,
	}
}

// Drain removes all items from the pool.
func (p *ObjectPool[T]) Drain() {
	for {
		select {
		case <-p.items:
		default:
			return
		}
	}
}

// Size returns the current number of items in the pool.
func (p *ObjectPool[T]) Size() int {
	return len(p.items)
}

// Use acquires an object, calls fn, and releases it.
func (p *ObjectPool[T]) Use(fn func(T)) {
	item := p.Acquire()
	defer p.Release(item)
	fn(item)
}

// UsePoolWithResult acquires an object, calls fn, releases it, and returns the result.
func UsePoolWithResult[T, R any](p *ObjectPool[T], fn func(T) R) R {
	item := p.Acquire()
	defer p.Release(item)
	return fn(item)
}

// UsePoolWithError acquires an object, calls fn, releases it, and returns the result and error.
func UsePoolWithError[T, R any](p *ObjectPool[T], fn func(T) (R, error)) (R, error) {
	item := p.Acquire()
	defer p.Release(item)
	return fn(item)
}
