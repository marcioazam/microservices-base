package property

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience/shutdown"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 25: Graceful Shutdown Request Draining**
// **Validates: Requirements 10.4**
func TestProperty_GracefulShutdownRequestDraining(t *testing.T) {
	t.Run("all_requests_complete_before_shutdown", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			requestCount := rapid.IntRange(1, 20).Draw(t, "requestCount")

			gs := shutdown.NewGracefulShutdown(5 * time.Second)

			var wg sync.WaitGroup
			completedCount := 0
			var mu sync.Mutex

			// Start requests
			for i := 0; i < requestCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					if !gs.RequestStarted() {
						return
					}
					defer gs.RequestFinished()

					// Simulate work
					time.Sleep(10 * time.Millisecond)

					mu.Lock()
					completedCount++
					mu.Unlock()
				}()
			}

			// Wait a bit for requests to start
			time.Sleep(5 * time.Millisecond)

			// Initiate shutdown
			ctx := context.Background()
			err := gs.Shutdown(ctx)

			// Wait for all goroutines
			wg.Wait()

			// All requests should have completed
			mu.Lock()
			completed := completedCount
			mu.Unlock()

			if err != nil {
				t.Fatalf("shutdown error: %v", err)
			}
			if completed != requestCount {
				t.Fatalf("completed %d != %d", completed, requestCount)
			}
		})
	})

	t.Run("new_requests_rejected_after_shutdown", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			requestCount := rapid.IntRange(5, 20).Draw(t, "requestCount")

			gs := shutdown.NewGracefulShutdown(time.Second)

			// Initiate shutdown immediately
			go gs.Shutdown(context.Background())

			// Wait for shutdown to be initiated
			time.Sleep(10 * time.Millisecond)

			// Try to start new requests
			rejectedCount := 0
			for i := 0; i < requestCount; i++ {
				if !gs.RequestStarted() {
					rejectedCount++
				} else {
					gs.RequestFinished()
				}
			}

			// All or most requests should be rejected
			if rejectedCount == 0 {
				t.Fatal("expected some requests to be rejected")
			}
		})
	})

	t.Run("in_flight_count_accurate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			startCount := rapid.IntRange(1, 20).Draw(t, "startCount")
			finishCount := rapid.IntRange(0, 15).Draw(t, "finishCount")

			gs := shutdown.NewGracefulShutdown(time.Second)

			// Start some requests
			actualStarted := 0
			for i := 0; i < startCount; i++ {
				if gs.RequestStarted() {
					actualStarted++
				}
			}

			// Verify count
			if gs.InFlightCount() != int64(actualStarted) {
				t.Fatalf("in-flight count %d != %d", gs.InFlightCount(), actualStarted)
			}

			// Finish some requests
			toFinish := minIntShutdown(finishCount, actualStarted)
			for i := 0; i < toFinish; i++ {
				gs.RequestFinished()
			}

			// Verify count again
			expected := int64(actualStarted - toFinish)
			if gs.InFlightCount() != expected {
				t.Fatalf("in-flight count %d != %d after finish", gs.InFlightCount(), expected)
			}
		})
	})

	t.Run("shutdown_waits_for_in_flight", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			requestCount := rapid.IntRange(1, 10).Draw(t, "requestCount")

			gs := shutdown.NewGracefulShutdown(5 * time.Second)

			// Start long-running requests
			for i := 0; i < requestCount; i++ {
				gs.RequestStarted()
			}

			// Start shutdown in background
			shutdownDone := make(chan struct{})
			go func() {
				gs.Shutdown(context.Background())
				close(shutdownDone)
			}()

			// Verify shutdown is waiting
			select {
			case <-shutdownDone:
				t.Fatal("shutdown shouldn't complete yet")
			case <-time.After(50 * time.Millisecond):
				// Good, shutdown is waiting
			}

			// Finish all requests
			for i := 0; i < requestCount; i++ {
				gs.RequestFinished()
			}

			// Now shutdown should complete
			select {
			case <-shutdownDone:
				// Good
			case <-time.After(time.Second):
				t.Fatal("shutdown should have completed")
			}
		})
	})
}

func minIntShutdown(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// **Feature: resilience-service-state-of-art-2025, Property 10: Shutdown Request Blocking**
// **Validates: Requirements 14.1**
func TestProperty_ShutdownRequestBlocking(t *testing.T) {
	t.Run("request_started_returns_false_after_shutdown", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			drainTimeoutMs := rapid.IntRange(100, 5000).Draw(t, "drainTimeoutMs")

			gs := shutdown.NewGracefulShutdown(time.Duration(drainTimeoutMs) * time.Millisecond)

			// Initiate shutdown
			go gs.Shutdown(context.Background())

			// Wait for shutdown to be initiated
			time.Sleep(10 * time.Millisecond)

			// RequestStarted should return false
			if gs.RequestStarted() {
				t.Fatal("RequestStarted should return false after shutdown")
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 11: Shutdown Drain with Timeout**
// **Validates: Requirements 14.2, 14.3**
func TestProperty_ShutdownDrainWithTimeout(t *testing.T) {
	t.Run("shutdown_returns_error_after_timeout_with_in_flight", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			requestCount := rapid.IntRange(1, 5).Draw(t, "requestCount")

			// Very short timeout
			gs := shutdown.NewGracefulShutdown(50 * time.Millisecond)

			// Start requests that won't finish
			for i := 0; i < requestCount; i++ {
				gs.RequestStarted()
			}

			// Shutdown should timeout
			ctx := context.Background()
			err := gs.Shutdown(ctx)

			// Should return context deadline exceeded
			if err == nil {
				t.Fatal("expected error from shutdown timeout")
			}
		})
	})
}
