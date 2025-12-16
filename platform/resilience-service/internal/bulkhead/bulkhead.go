// Package bulkhead implements the bulkhead isolation pattern.
package bulkhead

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
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

	eventEmitter  domain.EventEmitter
	correlationFn func() string
}

// Config holds bulkhead creation options.
type Config struct {
	Name          string
	MaxConcurrent int
	MaxQueue      int
	QueueTimeout  time.Duration
	EventEmitter  domain.EventEmitter
	CorrelationFn func() string
}

// New creates a new bulkhead.
func New(cfg Config) *Bulkhead {
	correlationFn := cfg.CorrelationFn
	if correlationFn == nil {
		correlationFn = func() string { return "" }
	}

	return &Bulkhead{
		name:          cfg.Name,
		maxConcurrent: cfg.MaxConcurrent,
		maxQueue:      cfg.MaxQueue,
		queueTimeout:  cfg.QueueTimeout,
		semaphore:     make(chan struct{}, cfg.MaxConcurrent),
		queue:         make(chan struct{}, cfg.MaxQueue),
		eventEmitter:  cfg.EventEmitter,
		correlationFn: correlationFn,
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
		return domain.NewBulkheadFullError(b.name)
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
		return domain.NewBulkheadFullError(b.name)
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
func (b *Bulkhead) GetMetrics() domain.BulkheadMetrics {
	return domain.BulkheadMetrics{
		ActiveCount:   int(atomic.LoadInt64(&b.activeCount)),
		QueuedCount:   int(atomic.LoadInt64(&b.queuedCount)),
		RejectedCount: atomic.LoadInt64(&b.rejectedCount),
	}
}

// emitRejectionEvent emits a bulkhead rejection event.
func (b *Bulkhead) emitRejectionEvent() {
	if b.eventEmitter == nil {
		return
	}

	metrics := b.GetMetrics()
	event := domain.ResilienceEvent{
		ID:            generateEventID(),
		Type:          domain.EventBulkheadRejection,
		ServiceName:   b.name,
		Timestamp:     time.Now(),
		CorrelationID: b.correlationFn(),
		Metadata: map[string]any{
			"partition":      b.name,
			"active_count":   metrics.ActiveCount,
			"queued_count":   metrics.QueuedCount,
			"max_concurrent": b.maxConcurrent,
			"max_queue":      b.maxQueue,
		},
	}

	b.eventEmitter.Emit(event)
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	return time.Now().Format("20060102150405.000000000")
}

// Manager manages multiple bulkhead partitions.
type Manager struct {
	mu         sync.RWMutex
	partitions map[string]*Bulkhead
	config     domain.BulkheadConfig
	emitter    domain.EventEmitter
}

// NewManager creates a new bulkhead manager.
func NewManager(cfg domain.BulkheadConfig, emitter domain.EventEmitter) *Manager {
	return &Manager{
		partitions: make(map[string]*Bulkhead),
		config:     cfg,
		emitter:    emitter,
	}
}

// GetBulkhead returns the bulkhead for a partition.
func (m *Manager) GetBulkhead(partition string) domain.Bulkhead {
	m.mu.RLock()
	if b, ok := m.partitions[partition]; ok {
		m.mu.RUnlock()
		return b
	}
	m.mu.RUnlock()

	// Create new bulkhead
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if b, ok := m.partitions[partition]; ok {
		return b
	}

	b := New(Config{
		Name:          partition,
		MaxConcurrent: m.config.MaxConcurrent,
		MaxQueue:      m.config.MaxQueue,
		QueueTimeout:  m.config.QueueTimeout,
		EventEmitter:  m.emitter,
	})

	m.partitions[partition] = b
	return b
}

// GetAllMetrics returns metrics for all partitions.
func (m *Manager) GetAllMetrics() map[string]domain.BulkheadMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]domain.BulkheadMetrics, len(m.partitions))
	for name, b := range m.partitions {
		result[name] = b.GetMetrics()
	}
	return result
}
