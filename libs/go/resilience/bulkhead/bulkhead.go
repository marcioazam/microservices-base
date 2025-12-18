// Package bulkhead implements the bulkhead isolation pattern.
package bulkhead

import (
	"context"
	"sync/atomic"
	"time"

	liberror "github.com/auth-platform/libs/go/error"
	"github.com/auth-platform/libs/go/resilience"
)

// Bulkhead implements semaphore-based concurrency limiting.
type Bulkhead struct {
	name          string
	maxConcurrent int
	maxQueue      int
	queueTimeout  time.Duration

	semaphore     chan struct{}
	queue         chan struct{}
	activeCount   int64
	queuedCount   int64
	rejectedCount int64

	eventEmitter  resilience.EventEmitter
	correlationFn func() string
}

// Config holds bulkhead creation options.
type Config struct {
	Name          string
	MaxConcurrent int
	MaxQueue      int
	QueueTimeout  time.Duration
	EventEmitter  resilience.EventEmitter
	CorrelationFn func() string
}

// New creates a new bulkhead.
func New(cfg Config) *Bulkhead {
	return &Bulkhead{
		name:          cfg.Name,
		maxConcurrent: cfg.MaxConcurrent,
		maxQueue:      cfg.MaxQueue,
		queueTimeout:  cfg.QueueTimeout,
		semaphore:     make(chan struct{}, cfg.MaxConcurrent),
		queue:         make(chan struct{}, cfg.MaxQueue),
		eventEmitter:  cfg.EventEmitter,
		correlationFn: resilience.EnsureCorrelationFunc(cfg.CorrelationFn),
	}
}

// Acquire attempts to acquire a permit.
func (b *Bulkhead) Acquire(ctx context.Context) error {
	// Try to acquire immediately
	select {
	case b.semaphore <- struct{}{}:
		atomic.AddInt64(&b.activeCount, 1)
		return nil
	default:
	}

	// Try to enter queue
	select {
	case b.queue <- struct{}{}:
		atomic.AddInt64(&b.queuedCount, 1)
	default:
		// Queue is full
		atomic.AddInt64(&b.rejectedCount, 1)
		b.emitRejectionEvent()
		return liberror.NewBulkheadFullError(b.name)
	}

	// Wait in queue for semaphore
	defer func() {
		<-b.queue
		atomic.AddInt64(&b.queuedCount, -1)
	}()

	// Create timeout context if configured
	waitCtx := ctx
	var cancel context.CancelFunc
	if b.queueTimeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, b.queueTimeout)
		defer cancel()
	}

	select {
	case b.semaphore <- struct{}{}:
		atomic.AddInt64(&b.activeCount, 1)
		return nil
	case <-waitCtx.Done():
		atomic.AddInt64(&b.rejectedCount, 1)
		b.emitRejectionEvent()
		return liberror.NewBulkheadFullError(b.name)
	}
}

// Release returns a permit.
func (b *Bulkhead) Release() {
	select {
	case <-b.semaphore:
		atomic.AddInt64(&b.activeCount, -1)
	default:
		// Semaphore was empty, shouldn't happen in normal use
	}
}

// GetMetrics returns current utilization.
func (b *Bulkhead) GetMetrics() resilience.BulkheadMetrics {
	return resilience.BulkheadMetrics{
		ActiveCount:   int(atomic.LoadInt64(&b.activeCount)),
		QueuedCount:   int(atomic.LoadInt64(&b.queuedCount)),
		RejectedCount: atomic.LoadInt64(&b.rejectedCount),
	}
}

// emitRejectionEvent emits a bulkhead rejection event.
func (b *Bulkhead) emitRejectionEvent() {
	metrics := b.GetMetrics()
	event := resilience.NewEvent(resilience.EventBulkheadRejection, b.name).
		WithCorrelationID(b.correlationFn()).
		WithMetadata("partition", b.name).
		WithMetadata("active_count", metrics.ActiveCount).
		WithMetadata("queued_count", metrics.QueuedCount).
		WithMetadata("max_concurrent", b.maxConcurrent).
		WithMetadata("max_queue", b.maxQueue)

	resilience.EmitEvent(b.eventEmitter, *event)
}
