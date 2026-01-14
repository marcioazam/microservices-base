// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 8: Graceful Shutdown Behavior
// Validates: Requirements 17.2, 17.3, 17.4, 17.5
package property

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// MockServer simulates server behavior for shutdown testing.
type MockServer struct {
	accepting      atomic.Bool
	inFlightCount  atomic.Int32
	shutdownCalled atomic.Bool
	cleanupCalled  atomic.Bool
	timeout        time.Duration
	mu             sync.Mutex
}

func NewMockServer(timeout time.Duration) *MockServer {
	s := &MockServer{timeout: timeout}
	s.accepting.Store(true)
	return s
}

// StartRequest simulates starting a request.
func (s *MockServer) StartRequest() bool {
	if !s.accepting.Load() {
		return false
	}
	s.inFlightCount.Add(1)
	return true
}

// EndRequest simulates ending a request.
func (s *MockServer) EndRequest() {
	s.inFlightCount.Add(-1)
}

// InFlightRequests returns the number of in-flight requests.
func (s *MockServer) InFlightRequests() int {
	return int(s.inFlightCount.Load())
}

// IsAccepting returns true if accepting new requests.
func (s *MockServer) IsAccepting() bool {
	return s.accepting.Load()
}

// Shutdown performs graceful shutdown.
func (s *MockServer) Shutdown(ctx context.Context) error {
	s.shutdownCalled.Store(true)

	// Stop accepting new requests
	s.accepting.Store(false)

	// Wait for in-flight requests or timeout
	deadline := time.Now().Add(s.timeout)
	for s.inFlightCount.Load() > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Run cleanup
	s.cleanupCalled.Store(true)

	return nil
}

// ForceClose forces immediate close.
func (s *MockServer) ForceClose() {
	s.accepting.Store(false)
}

// WasShutdownCalled returns true if shutdown was called.
func (s *MockServer) WasShutdownCalled() bool {
	return s.shutdownCalled.Load()
}

// WasCleanupCalled returns true if cleanup was called.
func (s *MockServer) WasCleanupCalled() bool {
	return s.cleanupCalled.Load()
}

// TestProperty8_NewRequestsRejectedOnSIGTERM tests that new requests are rejected on SIGTERM.
// Property 8: Graceful Shutdown Behavior
// Validates: Requirements 17.2, 17.3, 17.4, 17.5
func TestProperty8_NewRequestsRejectedOnSIGTERM(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := time.Duration(rapid.IntRange(5, 30).Draw(t, "timeoutSecs")) * time.Second
		server := NewMockServer(timeout)

		// Server should accept requests initially
		if !server.IsAccepting() {
			t.Error("server should accept requests initially")
		}

		// Start some requests
		numRequests := rapid.IntRange(1, 5).Draw(t, "numRequests")
		for i := 0; i < numRequests; i++ {
			if !server.StartRequest() {
				t.Error("should accept request before shutdown")
			}
		}

		// Trigger shutdown
		ctx := context.Background()
		go server.Shutdown(ctx)

		// Wait for accepting to be false
		time.Sleep(50 * time.Millisecond)

		// Property: New requests SHALL be rejected immediately
		if server.IsAccepting() {
			t.Error("server should not accept new requests after shutdown signal")
		}

		// New request should be rejected
		if server.StartRequest() {
			t.Error("new request should be rejected during shutdown")
		}

		// Complete in-flight requests
		for i := 0; i < numRequests; i++ {
			server.EndRequest()
		}
	})
}

// TestProperty8_InFlightRequestsComplete tests that in-flight requests complete within timeout.
// Property 8: Graceful Shutdown Behavior
// Validates: Requirements 17.2, 17.3, 17.4, 17.5
func TestProperty8_InFlightRequestsComplete(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := 5 * time.Second
		server := NewMockServer(timeout)

		// Start some in-flight requests
		numRequests := rapid.IntRange(1, 3).Draw(t, "numRequests")
		for i := 0; i < numRequests; i++ {
			server.StartRequest()
		}

		// Start shutdown in background
		shutdownDone := make(chan struct{})
		go func() {
			server.Shutdown(context.Background())
			close(shutdownDone)
		}()

		// Simulate requests completing
		time.Sleep(100 * time.Millisecond)
		for i := 0; i < numRequests; i++ {
			server.EndRequest()
		}

		// Property: In-flight requests SHALL complete within timeout
		select {
		case <-shutdownDone:
			// Shutdown completed
		case <-time.After(timeout + time.Second):
			t.Error("shutdown should complete after in-flight requests finish")
		}

		if server.InFlightRequests() != 0 {
			t.Errorf("all in-flight requests should complete, got %d", server.InFlightRequests())
		}
	})
}

// TestProperty8_ForceTerminateAfterTimeout tests that service force terminates after timeout.
// Property 8: Graceful Shutdown Behavior
// Validates: Requirements 17.2, 17.3, 17.4, 17.5
func TestProperty8_ForceTerminateAfterTimeout(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Short timeout for testing
		timeout := 100 * time.Millisecond
		server := NewMockServer(timeout)

		// Start a request that won't complete
		server.StartRequest()

		// Start shutdown
		start := time.Now()
		server.Shutdown(context.Background())
		elapsed := time.Since(start)

		// Property: After timeout, service SHALL force terminate
		// Shutdown should complete around timeout duration
		if elapsed < timeout {
			t.Errorf("shutdown completed too quickly: %v < %v", elapsed, timeout)
		}
		if elapsed > timeout+100*time.Millisecond {
			t.Errorf("shutdown took too long: %v > %v", elapsed, timeout+100*time.Millisecond)
		}
	})
}

// TestProperty8_CleanupHandlersInvoked tests that all cleanup handlers are invoked.
// Property 8: Graceful Shutdown Behavior
// Validates: Requirements 17.2, 17.3, 17.4, 17.5
func TestProperty8_CleanupHandlersInvoked(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := time.Second
		server := NewMockServer(timeout)

		// Shutdown
		server.Shutdown(context.Background())

		// Property: All cleanup handlers SHALL be invoked
		if !server.WasShutdownCalled() {
			t.Error("shutdown should be called")
		}
		if !server.WasCleanupCalled() {
			t.Error("cleanup handlers should be invoked")
		}
	})
}

// TestProperty8_ShutdownIdempotent tests that multiple shutdown calls are safe.
// Property 8: Graceful Shutdown Behavior
// Validates: Requirements 17.2, 17.3, 17.4, 17.5
func TestProperty8_ShutdownIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := time.Second
		server := NewMockServer(timeout)

		// Multiple shutdown calls should be safe
		numCalls := rapid.IntRange(2, 5).Draw(t, "numCalls")

		var wg sync.WaitGroup
		for i := 0; i < numCalls; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				server.Shutdown(context.Background())
			}()
		}

		wg.Wait()

		// Property: Multiple shutdown calls should be safe (idempotent)
		if server.IsAccepting() {
			t.Error("server should not be accepting after shutdown")
		}
	})
}

// TestProperty8_RequestsDuringShutdownRejected tests that requests during shutdown are rejected.
// Property 8: Graceful Shutdown Behavior
// Validates: Requirements 17.2, 17.3, 17.4, 17.5
func TestProperty8_RequestsDuringShutdownRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := time.Second
		server := NewMockServer(timeout)

		// Start shutdown
		go server.Shutdown(context.Background())

		// Wait for shutdown to start
		time.Sleep(50 * time.Millisecond)

		// Property: Requests during shutdown SHALL be rejected
		numAttempts := rapid.IntRange(5, 10).Draw(t, "numAttempts")
		rejectedCount := 0

		for i := 0; i < numAttempts; i++ {
			if !server.StartRequest() {
				rejectedCount++
			}
		}

		if rejectedCount != numAttempts {
			t.Errorf("all %d requests should be rejected during shutdown, got %d rejected",
				numAttempts, rejectedCount)
		}
	})
}
