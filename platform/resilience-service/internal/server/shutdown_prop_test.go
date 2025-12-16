package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 25: Graceful Shutdown Request Draining**
// **Validates: Requirements 10.4**
func TestProperty_GracefulShutdownRequestDraining(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("all_requests_complete_before_shutdown", prop.ForAll(
		func(requestCount int) bool {
			gs := NewGracefulShutdown(5 * time.Second)

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

			return err == nil && completed == requestCount
		},
		gen.IntRange(1, 20),
	))

	props.Property("new_requests_rejected_after_shutdown", prop.ForAll(
		func(requestCount int) bool {
			gs := NewGracefulShutdown(time.Second)

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
			return rejectedCount > 0
		},
		gen.IntRange(5, 20),
	))

	props.Property("in_flight_count_accurate", prop.ForAll(
		func(startCount int, finishCount int) bool {
			gs := NewGracefulShutdown(time.Second)

			// Start some requests
			actualStarted := 0
			for i := 0; i < startCount; i++ {
				if gs.RequestStarted() {
					actualStarted++
				}
			}

			// Verify count
			if gs.InFlightCount() != int64(actualStarted) {
				return false
			}

			// Finish some requests
			toFinish := min(finishCount, actualStarted)
			for i := 0; i < toFinish; i++ {
				gs.RequestFinished()
			}

			// Verify count again
			expected := int64(actualStarted - toFinish)
			return gs.InFlightCount() == expected
		},
		gen.IntRange(1, 20),
		gen.IntRange(0, 15),
	))

	props.Property("shutdown_waits_for_in_flight", prop.ForAll(
		func(requestCount int) bool {
			gs := NewGracefulShutdown(5 * time.Second)

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
				return false // Shutdown shouldn't complete yet
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
				return true
			case <-time.After(time.Second):
				return false
			}
		},
		gen.IntRange(1, 10),
	))

	props.TestingRun(t)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
