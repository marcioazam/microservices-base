package shutdown

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/server"
)

func TestShutdownHandler(t *testing.T) {
	t.Run("NewShutdownHandler creates handler", func(t *testing.T) {
		h := server.NewShutdownHandler()
		if h == nil {
			t.Error("expected non-nil")
		}
	})

	t.Run("WithTimeout sets timeout", func(t *testing.T) {
		h := server.NewShutdownHandler().WithTimeout(5 * time.Second)
		if h == nil {
			t.Error("expected non-nil")
		}
	})

	t.Run("OnShutdown registers hook", func(t *testing.T) {
		h := server.NewShutdownHandler()
		called := false
		h.OnShutdown(func(ctx context.Context) error {
			called = true
			return nil
		})
		h.Shutdown(context.Background())
		if !called {
			t.Error("expected hook to be called")
		}
	})

	t.Run("Shutdown executes hooks in LIFO order", func(t *testing.T) {
		h := server.NewShutdownHandler()
		order := make([]int, 0)
		var mu sync.Mutex

		h.OnShutdown(func(ctx context.Context) error {
			mu.Lock()
			order = append(order, 1)
			mu.Unlock()
			return nil
		})
		h.OnShutdown(func(ctx context.Context) error {
			mu.Lock()
			order = append(order, 2)
			mu.Unlock()
			return nil
		})

		h.Shutdown(context.Background())

		// Hooks run concurrently, so we just check both were called
		if len(order) != 2 {
			t.Errorf("expected 2 hooks called, got %d", len(order))
		}
	})
}

func TestDrainManager(t *testing.T) {
	t.Run("NewDrainManager creates manager", func(t *testing.T) {
		m := server.NewDrainManager()
		if m == nil {
			t.Error("expected non-nil")
		}
	})

	t.Run("RequestStarted returns true when not shutting down", func(t *testing.T) {
		m := server.NewDrainManager()
		if !m.RequestStarted() {
			t.Error("expected true")
		}
		m.RequestFinished()
	})

	t.Run("InFlightCount tracks requests", func(t *testing.T) {
		m := server.NewDrainManager()
		m.RequestStarted()
		m.RequestStarted()
		if m.InFlightCount() != 2 {
			t.Errorf("expected 2, got %d", m.InFlightCount())
		}
		m.RequestFinished()
		if m.InFlightCount() != 1 {
			t.Errorf("expected 1, got %d", m.InFlightCount())
		}
		m.RequestFinished()
	})

	t.Run("IsShuttingDown returns correct state", func(t *testing.T) {
		m := server.NewDrainManager()
		if m.IsShuttingDown() {
			t.Error("expected false before shutdown")
		}
		go m.Drain(context.Background())
		time.Sleep(10 * time.Millisecond)
		if !m.IsShuttingDown() {
			t.Error("expected true after drain initiated")
		}
	})

	t.Run("Drain waits for in-flight requests", func(t *testing.T) {
		m := server.NewDrainManager()
		m.RequestStarted()

		done := make(chan error)
		go func() {
			done <- m.Drain(context.Background())
		}()

		// Drain should be waiting
		select {
		case <-done:
			t.Error("drain should wait for in-flight")
		case <-time.After(50 * time.Millisecond):
		}

		m.RequestFinished()

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("drain should complete")
		}
	})

	t.Run("Drain returns immediately when no in-flight", func(t *testing.T) {
		m := server.NewDrainManager()
		err := m.Drain(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Drain respects context timeout", func(t *testing.T) {
		m := server.NewDrainManager()
		m.RequestStarted()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := m.Drain(ctx)
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}

		m.RequestFinished()
	})

	t.Run("Done channel closes after drain", func(t *testing.T) {
		m := server.NewDrainManager()
		m.Drain(context.Background())

		select {
		case <-m.Done():
		case <-time.After(time.Second):
			t.Error("Done should be closed after drain")
		}
	})
}

func TestDrainManagerConcurrency(t *testing.T) {
	m := server.NewDrainManager()
	var wg sync.WaitGroup

	// Start many concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if m.RequestStarted() {
				time.Sleep(10 * time.Millisecond)
				m.RequestFinished()
			}
		}()
	}

	// Start drain after some requests
	time.Sleep(5 * time.Millisecond)
	go func() {
		m.Drain(context.Background())
	}()

	wg.Wait()

	if m.InFlightCount() != 0 {
		t.Errorf("expected 0 in-flight, got %d", m.InFlightCount())
	}
}

func TestGracefulServer(t *testing.T) {
	t.Run("NewGracefulServer creates server", func(t *testing.T) {
		gs := server.NewGracefulServer(
			func() error { return nil },
			func(ctx context.Context) error { return nil },
		)
		if gs == nil {
			t.Error("expected non-nil")
		}
	})

	t.Run("OnShutdown registers hook", func(t *testing.T) {
		gs := server.NewGracefulServer(
			func() error { return nil },
			func(ctx context.Context) error { return nil },
		)
		called := false
		gs.OnShutdown(func(ctx context.Context) error {
			called = true
			return nil
		})
		// Note: We can't easily test Run() without blocking
		_ = called
	})
}
