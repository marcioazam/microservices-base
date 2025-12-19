// Package bulkhead provides generic bulkhead (concurrency limiter) implementation.
package bulkhead

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrBulkheadFull = errors.New("bulkhead is full")
)

// Config holds bulkhead configuration.
type Config struct {
	MaxConcurrent int
	MaxQueue      int
	QueueTimeout  time.Duration
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		MaxConcurrent: 10,
		MaxQueue:      100,
		QueueTimeout:  5 * time.Second,
	}
}

// Option configures a bulkhead.
type Option func(*Config)

// WithMaxConcurrent sets the maximum concurrent executions.
func WithMaxConcurrent(n int) Option {
	return func(c *Config) { c.MaxConcurrent = n }
}

// WithMaxQueue sets the maximum queue size.
func WithMaxQueue(n int) Option {
	return func(c *Config) { c.MaxQueue = n }
}

// WithQueueTimeout sets the queue timeout.
func WithQueueTimeout(d time.Duration) Option {
	return func(c *Config) { c.QueueTimeout = d }
}

// Metrics holds bulkhead metrics.
type Metrics struct {
	ActiveCount   int64
	QueuedCount   int64
	CompletedCount int64
	RejectedCount int64
}

// Bulkhead is a generic bulkhead implementation.
type Bulkhead[T any] struct {
	name   string
	config Config

	semaphore chan struct{}
	queue     chan struct{}

	activeCount    int64
	queuedCount    int64
	completedCount int64
	rejectedCount  int64

	mu sync.RWMutex
}

// New creates a new bulkhead.
func New[T any](name string, maxConcurrent int, opts ...Option) *Bulkhead[T] {
	config := DefaultConfig()
	config.MaxConcurrent = maxConcurrent
	for _, opt := range opts {
		opt(&config)
	}

	return &Bulkhead[T]{
		name:      name,
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrent),
		queue:     make(chan struct{}, config.MaxQueue),
	}
}

// Execute runs the operation with bulkhead protection.
func (b *Bulkhead[T]) Execute(ctx context.Context, op func() (T, error)) (T, error) {
	var zero T

	// Try to acquire semaphore immediately
	select {
	case b.semaphore <- struct{}{}:
		atomic.AddInt64(&b.activeCount, 1)
		defer func() {
			<-b.semaphore
			atomic.AddInt64(&b.activeCount, -1)
			atomic.AddInt64(&b.completedCount, 1)
		}()
		return op()
	default:
	}

	// Try to queue
	select {
	case b.queue <- struct{}{}:
		atomic.AddInt64(&b.queuedCount, 1)
		defer func() {
			<-b.queue
			atomic.AddInt64(&b.queuedCount, -1)
		}()
	default:
		atomic.AddInt64(&b.rejectedCount, 1)
		return zero, ErrBulkheadFull
	}

	// Wait for semaphore with timeout
	timeout := b.config.QueueTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case b.semaphore <- struct{}{}:
		atomic.AddInt64(&b.activeCount, 1)
		defer func() {
			<-b.semaphore
			atomic.AddInt64(&b.activeCount, -1)
			atomic.AddInt64(&b.completedCount, 1)
		}()
		return op()
	case <-time.After(timeout):
		atomic.AddInt64(&b.rejectedCount, 1)
		return zero, ErrBulkheadFull
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// Metrics returns current bulkhead metrics.
func (b *Bulkhead[T]) Metrics() Metrics {
	return Metrics{
		ActiveCount:    atomic.LoadInt64(&b.activeCount),
		QueuedCount:    atomic.LoadInt64(&b.queuedCount),
		CompletedCount: atomic.LoadInt64(&b.completedCount),
		RejectedCount:  atomic.LoadInt64(&b.rejectedCount),
	}
}

// Name returns the bulkhead name.
func (b *Bulkhead[T]) Name() string {
	return b.name
}

// AvailablePermits returns the number of available permits.
func (b *Bulkhead[T]) AvailablePermits() int {
	return b.config.MaxConcurrent - int(atomic.LoadInt64(&b.activeCount))
}

// QueueSpace returns the available queue space.
func (b *Bulkhead[T]) QueueSpace() int {
	return b.config.MaxQueue - int(atomic.LoadInt64(&b.queuedCount))
}
