package workerpool_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/workerpool"
	"pgregory.net/rapid"
)

// TestPoolProcessesAllJobs verifies all submitted jobs are processed.
func TestPoolProcessesAllJobs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		workers := rapid.IntRange(1, 4).Draw(t, "workers")
		jobCount := rapid.IntRange(1, 20).Draw(t, "jobCount")

		var processed int64
		handler := func(ctx context.Context, data int) (int, error) {
			atomic.AddInt64(&processed, 1)
			return data * 2, nil
		}

		pool := workerpool.NewPool(workers, handler)
		pool.Start()

		for i := 0; i < jobCount; i++ {
			pool.Submit(workerpool.Job[int]{
				ID:   rapid.StringMatching(`job-[0-9]{3}`).Draw(t, "jobID"),
				Data: i,
			})
		}

		// Collect results
		collected := 0
		timeout := time.After(5 * time.Second)
	loop:
		for collected < jobCount {
			select {
			case <-pool.Results():
				collected++
			case <-timeout:
				break loop
			}
		}

		pool.Shutdown(time.Second)

		if atomic.LoadInt64(&processed) != int64(jobCount) {
			t.Errorf("expected %d processed, got %d", jobCount, processed)
		}
	})
}

// TestPoolStatsAccuracy verifies pool statistics are accurate.
func TestPoolStatsAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		workers := rapid.IntRange(1, 3).Draw(t, "workers")
		jobCount := rapid.IntRange(5, 15).Draw(t, "jobCount")

		handler := func(ctx context.Context, data int) (int, error) {
			return data, nil
		}

		pool := workerpool.NewPool(workers, handler)
		pool.Start()

		for i := 0; i < jobCount; i++ {
			pool.Submit(workerpool.Job[int]{ID: "job", Data: i})
		}

		// Wait for completion
		collected := 0
		timeout := time.After(5 * time.Second)
	loop:
		for collected < jobCount {
			select {
			case <-pool.Results():
				collected++
			case <-timeout:
				break loop
			}
		}

		pool.Shutdown(time.Second)

		stats := pool.Stats()
		if stats.Workers != workers {
			t.Errorf("expected %d workers, got %d", workers, stats.Workers)
		}
		if stats.Completed != int64(jobCount) {
			t.Errorf("expected %d completed, got %d", jobCount, stats.Completed)
		}
	})
}

// TestPoolHandlerResultCorrectness verifies handler results are correct.
func TestPoolHandlerResultCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.IntRange(1, 1000).Draw(t, "input")
		multiplier := rapid.IntRange(1, 10).Draw(t, "multiplier")

		handler := func(ctx context.Context, data int) (int, error) {
			return data * multiplier, nil
		}

		pool := workerpool.NewPool(1, handler)
		pool.Start()

		pool.Submit(workerpool.Job[int]{ID: "test", Data: input})

		select {
		case result := <-pool.Results():
			expected := input * multiplier
			if result.Data != expected {
				t.Errorf("expected %d, got %d", expected, result.Data)
			}
			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for result")
		}

		pool.Shutdown(time.Second)
	})
}

// TestPoolGracefulShutdown verifies graceful shutdown behavior.
func TestPoolGracefulShutdown(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		workers := rapid.IntRange(1, 3).Draw(t, "workers")

		handler := func(ctx context.Context, data int) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return data, nil
		}

		pool := workerpool.NewPool(workers, handler)
		pool.Start()

		// Submit a few jobs
		for i := 0; i < 3; i++ {
			pool.Submit(workerpool.Job[int]{ID: "job", Data: i})
		}

		// Shutdown should not panic
		pool.Shutdown(time.Second)
	})
}

// TestPoolPanicRecovery verifies panics in handlers are recovered.
func TestPoolPanicRecovery(t *testing.T) {
	handler := func(ctx context.Context, data int) (int, error) {
		if data == 42 {
			panic("test panic")
		}
		return data, nil
	}

	pool := workerpool.NewPool(1, handler)
	pool.Start()

	pool.Submit(workerpool.Job[int]{ID: "panic-job", Data: 42})

	select {
	case result := <-pool.Results():
		if result.Error == nil {
			t.Error("expected error from panic")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}

	pool.Shutdown(time.Second)

	stats := pool.Stats()
	if stats.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", stats.Failed)
	}
}

// TestPoolJobIDPreservation verifies job IDs are preserved in results.
func TestPoolJobIDPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		jobID := rapid.StringMatching(`[a-z]{5}-[0-9]{4}`).Draw(t, "jobID")

		handler := func(ctx context.Context, data int) (int, error) {
			return data, nil
		}

		pool := workerpool.NewPool(1, handler)
		pool.Start()

		pool.Submit(workerpool.Job[int]{ID: jobID, Data: 1})

		select {
		case result := <-pool.Results():
			if result.JobID != jobID {
				t.Errorf("expected job ID %s, got %s", jobID, result.JobID)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for result")
		}

		pool.Shutdown(time.Second)
	})
}
