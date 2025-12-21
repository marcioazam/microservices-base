package property

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/bulkhead"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 15: Bulkhead Concurrent Request Enforcement**
// **Validates: Requirements 5.1, 5.2**
func TestProperty_BulkheadConcurrentRequestEnforcement(t *testing.T) {
	t.Run("active_plus_queued_never_exceeds_limits", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(2, 10).Draw(t, "maxConcurrent")
			maxQueue := rapid.IntRange(2, 10).Draw(t, "maxQueue")
			requests := rapid.IntRange(5, 30).Draw(t, "requests")

			b := bulkhead.New(bulkhead.Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      maxQueue,
				QueueTimeout:  time.Second,
			})

			ctx := context.Background()
			var wg sync.WaitGroup
			var maxActive int64
			var maxQueued int64

			done := make(chan struct{})
			go func() {
				ticker := time.NewTicker(time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						metrics := b.GetMetrics()
						for {
							current := atomic.LoadInt64(&maxActive)
							if int64(metrics.ActiveCount) <= current {
								break
							}
							if atomic.CompareAndSwapInt64(&maxActive, current, int64(metrics.ActiveCount)) {
								break
							}
						}
						for {
							current := atomic.LoadInt64(&maxQueued)
							if int64(metrics.QueuedCount) <= current {
								break
							}
							if atomic.CompareAndSwapInt64(&maxQueued, current, int64(metrics.QueuedCount)) {
								break
							}
						}
					}
				}
			}()

			for i := 0; i < requests; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := b.Acquire(ctx); err == nil {
						time.Sleep(10 * time.Millisecond)
						b.Release()
					}
				}()
			}

			wg.Wait()
			close(done)

			if atomic.LoadInt64(&maxActive) > int64(maxConcurrent) {
				t.Fatalf("active count %d exceeded max concurrent %d", maxActive, maxConcurrent)
			}
			if atomic.LoadInt64(&maxQueued) > int64(maxQueue) {
				t.Fatalf("queued count %d exceeded max queue %d", maxQueued, maxQueue)
			}
		})
	})

	t.Run("requests_beyond_capacity_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(2, 10).Draw(t, "maxConcurrent")
			maxQueue := rapid.IntRange(2, 10).Draw(t, "maxQueue")

			b := bulkhead.New(bulkhead.Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      maxQueue,
				QueueTimeout:  10 * time.Millisecond,
			})

			ctx := context.Background()
			totalCapacity := maxConcurrent + maxQueue

			acquired := 0
			for i := 0; i < totalCapacity+5; i++ {
				if err := b.Acquire(ctx); err == nil {
					acquired++
				}
			}

			if acquired > totalCapacity {
				t.Fatalf("acquired %d permits but capacity is %d", acquired, totalCapacity)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 16: Bulkhead Partition Isolation**
// **Validates: Requirements 5.3**
func TestProperty_BulkheadPartitionIsolation(t *testing.T) {
	t.Run("partitions_independent", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(2, 10).Draw(t, "maxConcurrent")

			cfg := resilience.BulkheadConfig{
				MaxConcurrent: maxConcurrent,
				MaxQueue:      5,
				QueueTimeout:  time.Second,
			}

			manager := bulkhead.NewManager(cfg, nil)

			ctx := context.Background()

			partitionA := manager.GetBulkhead("partition-a")
			for i := 0; i < maxConcurrent; i++ {
				if err := partitionA.Acquire(ctx); err != nil {
					t.Fatalf("failed to acquire partition A at %d: %v", i, err)
				}
			}

			partitionB := manager.GetBulkhead("partition-b")
			for i := 0; i < maxConcurrent; i++ {
				if err := partitionB.Acquire(ctx); err != nil {
					t.Fatalf("failed to acquire partition B at %d: %v", i, err)
				}
			}

			metricsA := partitionA.GetMetrics()
			metricsB := partitionB.GetMetrics()

			if metricsA.ActiveCount != maxConcurrent {
				t.Fatalf("partition A active count %d != %d", metricsA.ActiveCount, maxConcurrent)
			}
			if metricsB.ActiveCount != maxConcurrent {
				t.Fatalf("partition B active count %d != %d", metricsB.ActiveCount, maxConcurrent)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 17: Bulkhead Metrics Accuracy**
// **Validates: Requirements 5.4**
func TestProperty_BulkheadMetricsAccuracy(t *testing.T) {
	t.Run("metrics_reflect_actual_state", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(5, 20).Draw(t, "maxConcurrent")
			acquireCount := rapid.IntRange(1, 15).Draw(t, "acquireCount")

			b := bulkhead.New(bulkhead.Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      10,
				QueueTimeout:  time.Second,
			})

			ctx := context.Background()

			actualAcquired := 0
			for i := 0; i < acquireCount && i < maxConcurrent; i++ {
				if err := b.Acquire(ctx); err == nil {
					actualAcquired++
				}
			}

			metrics := b.GetMetrics()
			if metrics.ActiveCount != actualAcquired {
				t.Fatalf("active count %d != acquired %d", metrics.ActiveCount, actualAcquired)
			}

			for i := 0; i < actualAcquired; i++ {
				b.Release()
			}

			metrics = b.GetMetrics()
			if metrics.ActiveCount != 0 {
				t.Fatalf("active count %d != 0 after release", metrics.ActiveCount)
			}
		})
	})

	t.Run("rejected_count_accurate", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(2, 10).Draw(t, "maxConcurrent")

			b := bulkhead.New(bulkhead.Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      0,
				QueueTimeout:  time.Millisecond,
			})

			ctx := context.Background()

			for i := 0; i < maxConcurrent; i++ {
				b.Acquire(ctx)
			}

			rejections := 0
			for i := 0; i < 5; i++ {
				if err := b.Acquire(ctx); err != nil {
					rejections++
				}
			}

			metrics := b.GetMetrics()
			if metrics.RejectedCount != int64(rejections) {
				t.Fatalf("rejected count %d != %d", metrics.RejectedCount, rejections)
			}
		})
	})
}
