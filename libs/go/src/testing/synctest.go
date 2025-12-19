package testing

import (
	"context"
	"sync"
	"time"
)

// SyncTestHelper provides deterministic concurrency testing utilities.
// Uses Go 1.25 testing/synctest patterns.
type SyncTestHelper struct {
	mu       sync.Mutex
	events   []string
	barriers map[string]chan struct{}
}

// NewSyncTestHelper creates a new sync test helper.
func NewSyncTestHelper() *SyncTestHelper {
	return &SyncTestHelper{
		barriers: make(map[string]chan struct{}),
	}
}

// RecordEvent records an event for verification.
func (h *SyncTestHelper) RecordEvent(event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
}

// Events returns recorded events.
func (h *SyncTestHelper) Events() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, len(h.events))
	copy(result, h.events)
	return result
}

// ClearEvents clears recorded events.
func (h *SyncTestHelper) ClearEvents() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = nil
}

// CreateBarrier creates a named barrier.
func (h *SyncTestHelper) CreateBarrier(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.barriers[name] = make(chan struct{})
}

// WaitBarrier waits for a barrier to be released.
func (h *SyncTestHelper) WaitBarrier(ctx context.Context, name string) error {
	h.mu.Lock()
	barrier, ok := h.barriers[name]
	h.mu.Unlock()

	if !ok {
		return nil
	}

	select {
	case <-barrier:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReleaseBarrier releases a barrier.
func (h *SyncTestHelper) ReleaseBarrier(name string) {
	h.mu.Lock()
	barrier, ok := h.barriers[name]
	h.mu.Unlock()

	if ok {
		close(barrier)
	}
}

// WaitGroup wraps sync.WaitGroup with timeout support.
type WaitGroup struct {
	wg sync.WaitGroup
}

// NewWaitGroup creates a new wait group.
func NewWaitGroup() *WaitGroup {
	return &WaitGroup{}
}

// Add adds to the wait group counter.
func (w *WaitGroup) Add(delta int) {
	w.wg.Add(delta)
}

// Done decrements the wait group counter.
func (w *WaitGroup) Done() {
	w.wg.Done()
}

// Wait waits for all goroutines to complete.
func (w *WaitGroup) Wait() {
	w.wg.Wait()
}

// WaitTimeout waits with timeout.
func (w *WaitGroup) WaitTimeout(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// WaitContext waits with context.
func (w *WaitGroup) WaitContext(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// OrderedExecution ensures operations execute in order.
type OrderedExecution struct {
	current int
	mu      sync.Mutex
	cond    *sync.Cond
}

// NewOrderedExecution creates a new ordered execution helper.
func NewOrderedExecution() *OrderedExecution {
	o := &OrderedExecution{}
	o.cond = sync.NewCond(&o.mu)
	return o
}

// WaitForTurn waits until it's this operation's turn.
func (o *OrderedExecution) WaitForTurn(order int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for o.current != order {
		o.cond.Wait()
	}
}

// Done signals this operation is complete.
func (o *OrderedExecution) Done() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.current++
	o.cond.Broadcast()
}
