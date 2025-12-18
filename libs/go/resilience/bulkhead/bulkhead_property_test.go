package bulkhead

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 7: Bulkhead Concurrency Limit**
// **Validates: Requirements 1.4**
func TestBulkheadConcurrencyLimit(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("at most MaxConcurrent operations active at any time", prop.ForAll(
		func(maxConcurrent int, numWorkers int) bool {
			if maxConcurrent < 1 {
				maxConcurrent = 1
			}
			if maxConcurrent > 20 {
				maxConcurrent = 20
			}
			if numWorkers < 1 {
				numWorkers = 1
			}
			if numWorkers > 50 {
				numWorkers = 50
			}

			b := New(Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      100,
				QueueTimeout:  time.Second,
			})

			var maxObserved int64
			var wg sync.WaitGroup
			ctx := context.Background()

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := b.Acquire(ctx); err != nil {
						return
					}
					defer b.Release()

					// Record current active count
					current := atomic.LoadInt64(&b.activeCount)
					for {
						old := atomic.LoadInt64(&maxObserved)
						if current <= old || atomic.CompareAndSwapInt64(&maxObserved, old, current) {
							break
						}
					}

					// Simulate work
					time.Sleep(time.Microsecond * 10)
				}()
			}

			wg.Wait()

			// Max observed should never exceed maxConcurrent
			return atomic.LoadInt64(&maxObserved) <= int64(maxConcurrent)
		},
		gen.IntRange(1, 20),
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}

func TestBulkheadAcquireRelease(t *testing.T) {
	b := New(Config{
		Name:          "test",
		MaxConcurrent: 2,
		MaxQueue:      0,
		QueueTimeout:  time.Millisecond * 100,
	})

	ctx := context.Background()

	// Acquire first permit
	if err := b.Acquire(ctx); err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}

	// Acquire second permit
	if err := b.Acquire(ctx); err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}

	// Third acquire should fail (no queue)
	if err := b.Acquire(ctx); err == nil {
		t.Error("third acquire should fail")
	}

	// Release one permit
	b.Release()

	// Now acquire should succeed
	if err := b.Acquire(ctx); err != nil {
		t.Fatalf("acquire after release failed: %v", err)
	}
}

func TestBulkheadMetrics(t *testing.T) {
	b := New(Config{
		Name:          "test",
		MaxConcurrent: 2,
		MaxQueue:      0,
	})

	ctx := context.Background()

	// Initial metrics
	metrics := b.GetMetrics()
	if metrics.ActiveCount != 0 {
		t.Errorf("expected 0 active, got %d", metrics.ActiveCount)
	}

	// Acquire and check
	b.Acquire(ctx)
	metrics = b.GetMetrics()
	if metrics.ActiveCount != 1 {
		t.Errorf("expected 1 active, got %d", metrics.ActiveCount)
	}

	// Release and check
	b.Release()
	metrics = b.GetMetrics()
	if metrics.ActiveCount != 0 {
		t.Errorf("expected 0 active after release, got %d", metrics.ActiveCount)
	}
}

func TestBulkheadManager(t *testing.T) {
	cfg := resilience.BulkheadConfig{
		MaxConcurrent: 5,
		MaxQueue:      10,
		QueueTimeout:  time.Second,
	}

	m := NewManager(cfg, nil)

	// Get bulkhead for partition
	b1 := m.GetBulkhead("partition-1")
	if b1 == nil {
		t.Fatal("expected non-nil bulkhead")
	}

	// Get same partition again
	b2 := m.GetBulkhead("partition-1")
	if b1 != b2 {
		t.Error("expected same bulkhead instance")
	}

	// Get different partition
	b3 := m.GetBulkhead("partition-2")
	if b1 == b3 {
		t.Error("expected different bulkhead instance")
	}

	// Check all metrics
	allMetrics := m.GetAllMetrics()
	if len(allMetrics) != 2 {
		t.Errorf("expected 2 partitions, got %d", len(allMetrics))
	}
}
