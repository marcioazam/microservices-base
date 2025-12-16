package bulkhead

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 15: Bulkhead Concurrent Request Enforcement**
// **Validates: Requirements 5.1, 5.2**
func TestProperty_BulkheadConcurrentRequestEnforcement(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("active_plus_queued_never_exceeds_limits", prop.ForAll(
		func(maxConcurrent int, maxQueue int, requests int) bool {
			b := New(Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      maxQueue,
				QueueTimeout:  time.Second,
			})

			ctx := context.Background()
			var wg sync.WaitGroup
			var maxActive int64
			var maxQueued int64

			// Track max values during execution
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

			// Launch concurrent requests
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

			// Verify limits were respected
			return atomic.LoadInt64(&maxActive) <= int64(maxConcurrent) &&
				atomic.LoadInt64(&maxQueued) <= int64(maxQueue)
		},
		gen.IntRange(2, 10),
		gen.IntRange(2, 10),
		gen.IntRange(5, 30),
	))

	props.Property("requests_beyond_capacity_rejected", prop.ForAll(
		func(maxConcurrent int, maxQueue int) bool {
			b := New(Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      maxQueue,
				QueueTimeout:  10 * time.Millisecond,
			})

			ctx := context.Background()
			totalCapacity := maxConcurrent + maxQueue

			// Fill up the bulkhead
			acquired := 0
			for i := 0; i < totalCapacity+5; i++ {
				if err := b.Acquire(ctx); err == nil {
					acquired++
				}
			}

			// Should have acquired at most totalCapacity
			return acquired <= totalCapacity
		},
		gen.IntRange(2, 10),
		gen.IntRange(2, 10),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 16: Bulkhead Partition Isolation**
// **Validates: Requirements 5.3**
func TestProperty_BulkheadPartitionIsolation(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("partitions_independent", prop.ForAll(
		func(maxConcurrent int) bool {
			cfg := domain.BulkheadConfig{
				MaxConcurrent: maxConcurrent,
				MaxQueue:      5,
				QueueTimeout:  time.Second,
			}

			manager := NewManager(cfg, nil)

			ctx := context.Background()

			// Fill partition A completely
			partitionA := manager.GetBulkhead("partition-a")
			for i := 0; i < maxConcurrent; i++ {
				if err := partitionA.Acquire(ctx); err != nil {
					return false
				}
			}

			// Partition B should still be available
			partitionB := manager.GetBulkhead("partition-b")
			for i := 0; i < maxConcurrent; i++ {
				if err := partitionB.Acquire(ctx); err != nil {
					return false
				}
			}

			// Verify metrics are independent
			metricsA := partitionA.GetMetrics()
			metricsB := partitionB.GetMetrics()

			return metricsA.ActiveCount == maxConcurrent &&
				metricsB.ActiveCount == maxConcurrent
		},
		gen.IntRange(2, 10),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 17: Bulkhead Metrics Accuracy**
// **Validates: Requirements 5.4**
func TestProperty_BulkheadMetricsAccuracy(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("metrics_reflect_actual_state", prop.ForAll(
		func(maxConcurrent int, acquireCount int) bool {
			b := New(Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      10,
				QueueTimeout:  time.Second,
			})

			ctx := context.Background()

			// Acquire some permits
			actualAcquired := 0
			for i := 0; i < acquireCount && i < maxConcurrent; i++ {
				if err := b.Acquire(ctx); err == nil {
					actualAcquired++
				}
			}

			// Metrics should match
			metrics := b.GetMetrics()
			if metrics.ActiveCount != actualAcquired {
				return false
			}

			// Release all
			for i := 0; i < actualAcquired; i++ {
				b.Release()
			}

			// Active should be 0
			metrics = b.GetMetrics()
			return metrics.ActiveCount == 0
		},
		gen.IntRange(5, 20),
		gen.IntRange(1, 15),
	))

	props.Property("rejected_count_accurate", prop.ForAll(
		func(maxConcurrent int) bool {
			b := New(Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      0, // No queue
				QueueTimeout:  time.Millisecond,
			})

			ctx := context.Background()

			// Fill up
			for i := 0; i < maxConcurrent; i++ {
				b.Acquire(ctx)
			}

			// Try to acquire more (should be rejected)
			rejections := 0
			for i := 0; i < 5; i++ {
				if err := b.Acquire(ctx); err != nil {
					rejections++
				}
			}

			metrics := b.GetMetrics()
			return metrics.RejectedCount == int64(rejections)
		},
		gen.IntRange(2, 10),
	))

	props.TestingRun(t)
}
