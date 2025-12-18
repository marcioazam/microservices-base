// Package shutdown provides server lifecycle management.
package shutdown

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// GracefulShutdown manages graceful shutdown with request draining.
type GracefulShutdown struct {
	inFlight     int64
	shutdownCh   chan struct{}
	drainTimeout time.Duration
	mu           sync.Mutex
	isShutdown   bool
}

// NewGracefulShutdown creates a new graceful shutdown manager.
func NewGracefulShutdown(drainTimeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		shutdownCh:   make(chan struct{}),
		drainTimeout: drainTimeout,
	}
}

// RequestStarted marks a request as started.
// Returns false if shutdown has been initiated.
func (g *GracefulShutdown) RequestStarted() bool {
	g.mu.Lock()
	if g.isShutdown {
		g.mu.Unlock()
		return false
	}
	g.mu.Unlock()

	atomic.AddInt64(&g.inFlight, 1)
	return true
}

// RequestFinished marks a request as finished.
func (g *GracefulShutdown) RequestFinished() {
	atomic.AddInt64(&g.inFlight, -1)
}

// InFlightCount returns the number of in-flight requests.
func (g *GracefulShutdown) InFlightCount() int64 {
	return atomic.LoadInt64(&g.inFlight)
}

// Shutdown initiates graceful shutdown and waits for requests to drain.
func (g *GracefulShutdown) Shutdown(ctx context.Context) error {
	g.mu.Lock()
	g.isShutdown = true
	g.mu.Unlock()

	close(g.shutdownCh)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, g.drainTimeout)
	defer cancel()

	// Wait for in-flight requests to complete
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if atomic.LoadInt64(&g.inFlight) == 0 {
				return nil
			}
		}
	}
}

// IsShutdown returns whether shutdown has been initiated.
func (g *GracefulShutdown) IsShutdown() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isShutdown
}

// ShutdownCh returns a channel that's closed when shutdown is initiated.
func (g *GracefulShutdown) ShutdownCh() <-chan struct{} {
	return g.shutdownCh
}
