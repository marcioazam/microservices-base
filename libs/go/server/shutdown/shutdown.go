// Package shutdown provides graceful shutdown utilities.
package shutdown

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Manager manages graceful shutdown with request draining.
type Manager struct {
	mu          sync.RWMutex
	inFlight    int64
	shuttingDown int32
	done        chan struct{}
}

// New creates a new shutdown manager.
func New() *Manager {
	return &Manager{
		done: make(chan struct{}),
	}
}

// RequestStarted should be called when a request starts.
// Returns true if the request is allowed, false if shutting down.
func (m *Manager) RequestStarted() bool {
	if atomic.LoadInt32(&m.shuttingDown) == 1 {
		return false
	}
	atomic.AddInt64(&m.inFlight, 1)
	return true
}

// RequestFinished should be called when a request finishes.
func (m *Manager) RequestFinished() {
	count := atomic.AddInt64(&m.inFlight, -1)
	if count == 0 && atomic.LoadInt32(&m.shuttingDown) == 1 {
		m.mu.Lock()
		select {
		case <-m.done:
		default:
			close(m.done)
		}
		m.mu.Unlock()
	}
}

// Shutdown initiates graceful shutdown and waits for in-flight requests to drain.
// Returns nil if all requests drain before timeout, or context.DeadlineExceeded.
func (m *Manager) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&m.shuttingDown, 1)

	// Check if already drained
	if atomic.LoadInt64(&m.inFlight) == 0 {
		m.mu.Lock()
		select {
		case <-m.done:
		default:
			close(m.done)
		}
		m.mu.Unlock()
		return nil
	}

	select {
	case <-m.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ShutdownWithTimeout initiates graceful shutdown with a timeout.
func (m *Manager) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.Shutdown(ctx)
}

// IsShuttingDown returns true if shutdown has been initiated.
func (m *Manager) IsShuttingDown() bool {
	return atomic.LoadInt32(&m.shuttingDown) == 1
}

// InFlightCount returns the number of in-flight requests.
func (m *Manager) InFlightCount() int64 {
	return atomic.LoadInt64(&m.inFlight)
}

// Done returns a channel that is closed when shutdown is complete.
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// Reset resets the manager for reuse (mainly for testing).
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.StoreInt32(&m.shuttingDown, 0)
	atomic.StoreInt64(&m.inFlight, 0)
	m.done = make(chan struct{})
}

// Middleware returns a middleware function that tracks requests.
type Middleware func(next func()) func()

// NewMiddleware creates a middleware that tracks requests.
func (m *Manager) NewMiddleware() Middleware {
	return func(next func()) func() {
		return func() {
			if !m.RequestStarted() {
				return // Reject request during shutdown
			}
			defer m.RequestFinished()
			next()
		}
	}
}

// WaitForDrain waits for all in-flight requests to complete.
func (m *Manager) WaitForDrain(ctx context.Context) error {
	for {
		if atomic.LoadInt64(&m.inFlight) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
