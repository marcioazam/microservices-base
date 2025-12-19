package bulkhead

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkheadBasicOperations(t *testing.T) {
	t.Run("allows execution within limit", func(t *testing.T) {
		b := New[int]("test", 5)

		result, err := b.Execute(context.Background(), func() (int, error) {
			return 42, nil
		})

		if err != nil || result != 42 {
			t.Errorf("expected 42, got %d, err: %v", result, err)
		}
	})

	t.Run("tracks metrics", func(t *testing.T) {
		b := New[int]("test", 5)

		b.Execute(context.Background(), func() (int, error) {
			return 42, nil
		})

		metrics := b.Metrics()
		if metrics.CompletedCount != 1 {
			t.Errorf("expected 1 completed, got %d", metrics.CompletedCount)
		}
	})
}

func TestBulkheadConcurrencyLimit(t *testing.T) {
	t.Run("limits concurrent executions", func(t *testing.T) {
		b := New[int]("test", 2, WithMaxQueue(0))

		var activeCount int64
		var maxActive int64
		var wg sync.WaitGroup

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				b.Execute(context.Background(), func() (int, error) {
					current := atomic.AddInt64(&activeCount, 1)
					for {
						old := atomic.LoadInt64(&maxActive)
						if current <= old || atomic.CompareAndSwapInt64(&maxActive, old, current) {
							break
						}
					}
					time.Sleep(50 * time.Millisecond)
					atomic.AddInt64(&activeCount, -1)
					return 0, nil
				})
			}()
		}

		wg.Wait()

		if maxActive > 2 {
			t.Errorf("max concurrent should be 2, got %d", maxActive)
		}
	})
}

func TestBulkheadQueue(t *testing.T) {
	t.Run("queues requests when at capacity", func(t *testing.T) {
		b := New[int]("test", 1, WithMaxQueue(5), WithQueueTimeout(time.Second))

		var wg sync.WaitGroup
		results := make(chan int, 3)

		// Start first request (takes the semaphore)
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Execute(context.Background(), func() (int, error) {
				time.Sleep(100 * time.Millisecond)
				results <- 1
				return 1, nil
			})
		}()

		time.Sleep(10 * time.Millisecond) // Let first request start

		// Start second request (should queue)
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Execute(context.Background(), func() (int, error) {
				results <- 2
				return 2, nil
			})
		}()

		wg.Wait()
		close(results)

		count := 0
		for range results {
			count++
		}
		if count != 2 {
			t.Errorf("expected 2 results, got %d", count)
		}
	})

	t.Run("rejects when queue is full", func(t *testing.T) {
		b := New[int]("test", 1, WithMaxQueue(0))

		// Block the semaphore
		started := make(chan struct{})
		done := make(chan struct{})
		go func() {
			b.Execute(context.Background(), func() (int, error) {
				close(started)
				<-done
				return 0, nil
			})
		}()

		<-started

		// Try another request
		_, err := b.Execute(context.Background(), func() (int, error) {
			return 42, nil
		})

		close(done)

		if !errors.Is(err, ErrBulkheadFull) {
			t.Errorf("expected ErrBulkheadFull, got %v", err)
		}
	})
}

func TestBulkheadTimeout(t *testing.T) {
	t.Run("times out waiting in queue", func(t *testing.T) {
		b := New[int]("test", 1, WithMaxQueue(5), WithQueueTimeout(50*time.Millisecond))

		// Block the semaphore
		started := make(chan struct{})
		done := make(chan struct{})
		go func() {
			b.Execute(context.Background(), func() (int, error) {
				close(started)
				<-done
				return 0, nil
			})
		}()

		<-started

		// Try another request that will timeout
		_, err := b.Execute(context.Background(), func() (int, error) {
			return 42, nil
		})

		close(done)

		if !errors.Is(err, ErrBulkheadFull) {
			t.Errorf("expected ErrBulkheadFull, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		b := New[int]("test", 1, WithMaxQueue(5), WithQueueTimeout(time.Hour))

		// Block the semaphore
		started := make(chan struct{})
		done := make(chan struct{})
		go func() {
			b.Execute(context.Background(), func() (int, error) {
				close(started)
				<-done
				return 0, nil
			})
		}()

		<-started

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := b.Execute(ctx, func() (int, error) {
			return 42, nil
		})

		close(done)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}

func TestBulkheadMetrics(t *testing.T) {
	b := New[int]("test", 2, WithMaxQueue(0))

	// Successful executions
	for i := 0; i < 3; i++ {
		b.Execute(context.Background(), func() (int, error) {
			return 0, nil
		})
	}

	// Block and reject
	started := make(chan struct{})
	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Execute(context.Background(), func() (int, error) {
				started <- struct{}{}
				<-done
				return 0, nil
			})
		}()
	}

	<-started
	<-started

	// This should be rejected
	b.Execute(context.Background(), func() (int, error) {
		return 0, nil
	})

	close(done)
	wg.Wait()

	metrics := b.Metrics()
	if metrics.CompletedCount != 5 {
		t.Errorf("expected 5 completed, got %d", metrics.CompletedCount)
	}
	if metrics.RejectedCount != 1 {
		t.Errorf("expected 1 rejected, got %d", metrics.RejectedCount)
	}
}

func TestBulkheadHelpers(t *testing.T) {
	b := New[int]("test-bulkhead", 5, WithMaxQueue(10))

	if b.Name() != "test-bulkhead" {
		t.Errorf("expected test-bulkhead, got %s", b.Name())
	}

	if b.AvailablePermits() != 5 {
		t.Errorf("expected 5 available permits, got %d", b.AvailablePermits())
	}

	if b.QueueSpace() != 10 {
		t.Errorf("expected 10 queue space, got %d", b.QueueSpace())
	}
}
