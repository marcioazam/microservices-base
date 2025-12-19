package errgroup

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	t.Run("Go collects results", func(t *testing.T) {
		g := New[int]()
		g.Go(func() (int, error) { return 1, nil })
		g.Go(func() (int, error) { return 2, nil })
		g.Go(func() (int, error) { return 3, nil })
		results, err := g.Wait()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}
	})

	t.Run("Go stops on first error", func(t *testing.T) {
		g := New[int]()
		g.Go(func() (int, error) { return 1, nil })
		g.Go(func() (int, error) { return 0, errors.New("fail") })
		g.Go(func() (int, error) { time.Sleep(10 * time.Millisecond); return 3, nil })
		_, err := g.Wait()
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("WithContext cancels on error", func(t *testing.T) {
		g, ctx := WithContext[int](context.Background())
		g.Go(func() (int, error) { return 0, errors.New("fail") })
		g.Go(func() (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		})
		_, err := g.Wait()
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("SetLimit limits concurrency", func(t *testing.T) {
		g := New[int]()
		g.SetLimit(2)

		var concurrent int32
		var maxConcurrent int32

		for i := 0; i < 10; i++ {
			g.Go(func() (int, error) {
				c := atomic.AddInt32(&concurrent, 1)
				for {
					old := atomic.LoadInt32(&maxConcurrent)
					if c <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, c) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&concurrent, -1)
				return 1, nil
			})
		}

		g.Wait()
		if maxConcurrent > 2 {
			t.Errorf("expected max 2 concurrent, got %d", maxConcurrent)
		}
	})
}

func TestCollectGroup(t *testing.T) {
	t.Run("Go collects all results and errors", func(t *testing.T) {
		g := NewCollect[int]()
		g.Go(func() (int, error) { return 1, nil })
		g.Go(func() (int, error) { return 0, errors.New("fail") })
		g.Go(func() (int, error) { return 3, nil })
		results, errs := g.Wait()
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
	})

	t.Run("HasErrors returns true when errors exist", func(t *testing.T) {
		g := NewCollect[int]()
		g.Go(func() (int, error) { return 0, errors.New("fail") })
		g.Wait()
		if !g.HasErrors() {
			t.Error("expected HasErrors to be true")
		}
	})

	t.Run("SetLimit limits concurrency", func(t *testing.T) {
		g := NewCollect[int]()
		g.SetLimit(2)

		var concurrent int32
		var maxConcurrent int32

		for i := 0; i < 10; i++ {
			g.Go(func() (int, error) {
				c := atomic.AddInt32(&concurrent, 1)
				for {
					old := atomic.LoadInt32(&maxConcurrent)
					if c <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, c) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&concurrent, -1)
				return 1, nil
			})
		}

		g.Wait()
		if maxConcurrent > 2 {
			t.Errorf("expected max 2 concurrent, got %d", maxConcurrent)
		}
	})
}

func TestTryGo(t *testing.T) {
	g := New[int]()
	g.SetLimit(1)

	// First should succeed
	if !g.TryGo(func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 1, nil
	}) {
		t.Error("first TryGo should succeed")
	}

	// Give time for goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Second should fail (at limit)
	if g.TryGo(func() (int, error) { return 2, nil }) {
		t.Error("second TryGo should fail when at limit")
	}

	g.Wait()
}
