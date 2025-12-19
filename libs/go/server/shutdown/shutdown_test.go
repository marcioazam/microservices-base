package shutdown

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestManagerBasicOperations(t *testing.T) {
	t.Run("RequestStarted returns true when not shutting down", func(t *testing.T) {
		m := New()
		if !m.RequestStarted() {
			t.Error("expected true")
		}
		m.RequestFinished()
	})

	t.Run("RequestStarted returns false when shutting down", func(t *testing.T) {
		m := New()
		go m.Shutdown(context.Background())
		time.Sleep(10 * time.Millisecond)
		if m.RequestStarted() {
			t.Error("expected false during shutdown")
		}
	})

	t.Run("InFlightCount tracks requests", func(t *testing.T) {
		m := New()
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
		m := New()
		if m.IsShuttingDown() {
			t.Error("expected false before shutdown")
		}
		go m.Shutdown(context.Background())
		time.Sleep(10 * time.Millisecond)
		if !m.IsShuttingDown() {
			t.Error("expected true after shutdown initiated")
		}
	})
}

func TestManagerShutdown(t *testing.T) {
	t.Run("Shutdown waits for in-flight requests", func(t *testing.T) {
		m := New()
		m.RequestStarted()

		done := make(chan error)
		go func() {
			done <- m.Shutdown(context.Background())
		}()

		// Shutdown should be waiting
		select {
		case <-done:
			t.Error("shutdown should wait for in-flight")
		case <-time.After(50 * time.Millisecond):
		}

		m.RequestFinished()

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("shutdown should complete")
		}
	})

	t.Run("Shutdown returns immediately when no in-flight", func(t *testing.T) {
		m := New()
		err := m.Shutdown(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Shutdown respects context timeout", func(t *testing.T) {
		m := New()
		m.RequestStarted()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := m.Shutdown(ctx)
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}

		m.RequestFinished()
	})

	t.Run("ShutdownWithTimeout works", func(t *testing.T) {
		m := New()
		m.RequestStarted()

		err := m.ShutdownWithTimeout(50 * time.Millisecond)
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}

		m.RequestFinished()
	})
}

func TestManagerConcurrency(t *testing.T) {
	m := New()
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

	// Start shutdown after some requests
	time.Sleep(5 * time.Millisecond)
	go func() {
		m.Shutdown(context.Background())
	}()

	wg.Wait()

	if m.InFlightCount() != 0 {
		t.Errorf("expected 0 in-flight, got %d", m.InFlightCount())
	}
}

func TestManagerDone(t *testing.T) {
	m := New()

	select {
	case <-m.Done():
		t.Error("Done should not be closed before shutdown")
	default:
	}

	m.Shutdown(context.Background())

	select {
	case <-m.Done():
	case <-time.After(time.Second):
		t.Error("Done should be closed after shutdown")
	}
}

func TestManagerReset(t *testing.T) {
	m := New()
	m.RequestStarted()
	m.RequestFinished()
	m.Shutdown(context.Background())

	m.Reset()

	if m.IsShuttingDown() {
		t.Error("expected not shutting down after reset")
	}
	if m.InFlightCount() != 0 {
		t.Error("expected 0 in-flight after reset")
	}
	if !m.RequestStarted() {
		t.Error("expected RequestStarted to work after reset")
	}
	m.RequestFinished()
}

func TestManagerMiddleware(t *testing.T) {
	m := New()
	middleware := m.NewMiddleware()

	executed := false
	wrapped := middleware(func() {
		executed = true
	})

	wrapped()

	if !executed {
		t.Error("expected handler to be executed")
	}
}

func TestWaitForDrain(t *testing.T) {
	t.Run("returns immediately when no in-flight", func(t *testing.T) {
		m := New()
		err := m.WaitForDrain(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("waits for in-flight to drain", func(t *testing.T) {
		m := New()
		m.RequestStarted()

		go func() {
			time.Sleep(50 * time.Millisecond)
			m.RequestFinished()
		}()

		err := m.WaitForDrain(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("respects context timeout", func(t *testing.T) {
		m := New()
		m.RequestStarted()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := m.WaitForDrain(ctx)
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}

		m.RequestFinished()
	})
}
