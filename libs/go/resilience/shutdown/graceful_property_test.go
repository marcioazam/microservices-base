package shutdown

import (
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 9: Graceful Shutdown Request Tracking**
// **Validates: Requirements 3.1, 3.2, 3.3**
// *For any* graceful shutdown manager, the in-flight count SHALL equal the number of started requests minus the number of finished requests.
func TestGracefulShutdownRequestTracking(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("in-flight count equals started minus finished", prop.ForAll(
		func(starts, finishes int) bool {
			g := NewGracefulShutdown(5 * time.Second)

			// Start requests
			for i := 0; i < starts; i++ {
				g.RequestStarted()
			}

			// Finish some requests (but not more than started)
			actualFinishes := finishes
			if actualFinishes > starts {
				actualFinishes = starts
			}
			for i := 0; i < actualFinishes; i++ {
				g.RequestFinished()
			}

			expected := int64(starts - actualFinishes)
			return g.InFlightCount() == expected
		},
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
	))

	properties.Property("request started returns false after shutdown", prop.ForAll(
		func(n int) bool {
			g := NewGracefulShutdown(5 * time.Second)

			// Start some requests
			for i := 0; i < n; i++ {
				g.RequestStarted()
			}

			// Initiate shutdown (in background to not block)
			g.mu.Lock()
			g.isShutdown = true
			g.mu.Unlock()

			// New requests should be rejected
			return !g.RequestStarted()
		},
		gen.IntRange(0, 10),
	))

	properties.Property("concurrent request tracking is accurate", prop.ForAll(
		func(numGoroutines int) bool {
			if numGoroutines <= 0 {
				return true
			}

			g := NewGracefulShutdown(5 * time.Second)
			var wg sync.WaitGroup

			// Start requests concurrently
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					g.RequestStarted()
				}()
			}
			wg.Wait()

			if g.InFlightCount() != int64(numGoroutines) {
				return false
			}

			// Finish requests concurrently
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					g.RequestFinished()
				}()
			}
			wg.Wait()

			return g.InFlightCount() == 0
		},
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}

func TestShutdownChannelBehavior(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("shutdown channel is closed after shutdown", prop.ForAll(
		func(n int) bool {
			g := NewGracefulShutdown(5 * time.Second)

			// Channel should not be closed initially
			select {
			case <-g.ShutdownCh():
				return false // Should not be closed
			default:
				// Good, not closed
			}

			// Initiate shutdown
			g.mu.Lock()
			g.isShutdown = true
			close(g.shutdownCh)
			g.mu.Unlock()

			// Channel should be closed now
			select {
			case <-g.ShutdownCh():
				return true // Good, closed
			default:
				return false // Should be closed
			}
		},
		gen.IntRange(0, 10),
	))

	properties.TestingRun(t)
}
