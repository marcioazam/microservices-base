package resilience

import (
	"context"
	"sync"
)

// Bulkhead implements bulkhead isolation pattern.
type Bulkhead struct {
	config        BulkheadConfig
	semaphore     chan struct{}
	queue         chan struct{}
	mu            sync.RWMutex
	currentLoad   int
	correlationID string
	service       string
}

// NewBulkhead creates a new bulkhead.
func NewBulkhead(config BulkheadConfig) (*Bulkhead, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Bulkhead{
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrent),
		queue:     make(chan struct{}, config.QueueSize),
	}, nil
}

// Execute runs operation with bulkhead isolation.
func (b *Bulkhead) Execute(ctx context.Context, op func(context.Context) error) error {
	if err := b.acquire(ctx); err != nil {
		return err
	}
	defer b.release()

	return op(ctx)
}

func (b *Bulkhead) acquire(ctx context.Context) error {
	// Try immediate acquisition
	select {
	case b.semaphore <- struct{}{}:
		b.incrementLoad()
		return nil
	default:
	}

	// Try queue
	select {
	case b.queue <- struct{}{}:
	default:
		return b.bulkheadFullError()
	}
	defer func() { <-b.queue }()

	// Wait for semaphore with timeout
	waitCtx := ctx
	if b.config.MaxWait > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, b.config.MaxWait)
		defer cancel()
	}

	select {
	case b.semaphore <- struct{}{}:
		b.incrementLoad()
		return nil
	case <-waitCtx.Done():
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return b.bulkheadFullError()
	}
}

func (b *Bulkhead) release() {
	<-b.semaphore
	b.decrementLoad()
}

func (b *Bulkhead) incrementLoad() {
	b.mu.Lock()
	b.currentLoad++
	b.mu.Unlock()
}

func (b *Bulkhead) decrementLoad() {
	b.mu.Lock()
	b.currentLoad--
	b.mu.Unlock()
}

func (b *Bulkhead) bulkheadFullError() *BulkheadFullError {
	b.mu.RLock()
	load := b.currentLoad
	b.mu.RUnlock()

	return NewBulkheadFullError(
		b.service,
		b.correlationID,
		b.config.MaxConcurrent,
		b.config.QueueSize,
		load,
	)
}

// CurrentLoad returns current concurrent operations.
func (b *Bulkhead) CurrentLoad() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentLoad
}

// QueueLength returns current queue length.
func (b *Bulkhead) QueueLength() int {
	return len(b.queue)
}

// SetService sets service name for errors.
func (b *Bulkhead) SetService(service string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.service = service
}

// SetCorrelationID sets correlation ID for errors.
func (b *Bulkhead) SetCorrelationID(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.correlationID = id
}
