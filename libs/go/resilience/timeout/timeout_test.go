package timeout

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTimeoutManagerBasicOperations(t *testing.T) {
	t.Run("executes operation within timeout", func(t *testing.T) {
		tm := New[int](time.Second)

		result, err := tm.Execute(context.Background(), "test", func(ctx context.Context) (int, error) {
			return 42, nil
		})

		if err != nil || result != 42 {
			t.Errorf("expected 42, got %d, err: %v", result, err)
		}
	})

	t.Run("returns timeout error when exceeded", func(t *testing.T) {
		tm := New[int](50 * time.Millisecond)

		_, err := tm.Execute(context.Background(), "test", func(ctx context.Context) (int, error) {
			time.Sleep(200 * time.Millisecond)
			return 42, nil
		})

		if !errors.Is(err, ErrTimeout) {
			t.Errorf("expected ErrTimeout, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		tm := New[int](time.Hour)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := tm.Execute(ctx, "test", func(ctx context.Context) (int, error) {
			time.Sleep(time.Second)
			return 42, nil
		})

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestTimeoutManagerPerOperation(t *testing.T) {
	t.Run("uses per-operation timeout", func(t *testing.T) {
		tm := New[int](time.Hour)
		tm.SetOperationTimeout("fast", 50*time.Millisecond)

		_, err := tm.Execute(context.Background(), "fast", func(ctx context.Context) (int, error) {
			time.Sleep(200 * time.Millisecond)
			return 42, nil
		})

		if !errors.Is(err, ErrTimeout) {
			t.Errorf("expected ErrTimeout, got %v", err)
		}
	})

	t.Run("falls back to default for unknown operation", func(t *testing.T) {
		tm := New[int](50 * time.Millisecond)
		tm.SetOperationTimeout("fast", time.Hour)

		_, err := tm.Execute(context.Background(), "unknown", func(ctx context.Context) (int, error) {
			time.Sleep(200 * time.Millisecond)
			return 42, nil
		})

		if !errors.Is(err, ErrTimeout) {
			t.Errorf("expected ErrTimeout, got %v", err)
		}
	})

	t.Run("GetOperationTimeout returns correct value", func(t *testing.T) {
		tm := New[int](time.Second)
		tm.SetOperationTimeout("custom", 5*time.Second)

		if tm.GetOperationTimeout("custom") != 5*time.Second {
			t.Error("expected custom timeout")
		}
		if tm.GetOperationTimeout("unknown") != time.Second {
			t.Error("expected default timeout")
		}
	})
}

func TestTimeoutManagerConfiguration(t *testing.T) {
	t.Run("SetDefaultTimeout updates default", func(t *testing.T) {
		tm := New[int](time.Second)
		tm.SetDefaultTimeout(2 * time.Second)

		if tm.GetOperationTimeout("any") != 2*time.Second {
			t.Error("expected updated default timeout")
		}
	})

	t.Run("respects max timeout", func(t *testing.T) {
		tm := New[int](time.Second)
		tm.SetMaxTimeout(100 * time.Millisecond)
		tm.SetOperationTimeout("test", time.Hour)

		if tm.GetOperationTimeout("test") != 100*time.Millisecond {
			t.Error("expected timeout to be capped at max")
		}
	})

	t.Run("NewWithConfig creates manager with config", func(t *testing.T) {
		config := Config{
			Default: time.Second,
			Max:     time.Minute,
			PerOperation: map[string]time.Duration{
				"fast": 100 * time.Millisecond,
			},
		}
		tm := NewWithConfig[int](config)

		if tm.GetOperationTimeout("fast") != 100*time.Millisecond {
			t.Error("expected per-operation timeout from config")
		}
	})
}

func TestWithTimeout(t *testing.T) {
	t.Run("executes within timeout", func(t *testing.T) {
		result, err := WithTimeout(context.Background(), time.Second, func(ctx context.Context) (int, error) {
			return 42, nil
		})

		if err != nil || result != 42 {
			t.Errorf("expected 42, got %d, err: %v", result, err)
		}
	})

	t.Run("returns timeout error", func(t *testing.T) {
		_, err := WithTimeout(context.Background(), 50*time.Millisecond, func(ctx context.Context) (int, error) {
			time.Sleep(200 * time.Millisecond)
			return 42, nil
		})

		if !errors.Is(err, ErrTimeout) {
			t.Errorf("expected ErrTimeout, got %v", err)
		}
	})
}

func TestDo(t *testing.T) {
	t.Run("executes void operation", func(t *testing.T) {
		executed := false
		err := Do(context.Background(), time.Second, func(ctx context.Context) error {
			executed = true
			return nil
		})

		if err != nil || !executed {
			t.Errorf("expected execution, err: %v", err)
		}
	})

	t.Run("returns timeout error", func(t *testing.T) {
		err := Do(context.Background(), 50*time.Millisecond, func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})

		if !errors.Is(err, ErrTimeout) {
			t.Errorf("expected ErrTimeout, got %v", err)
		}
	})
}

func TestTimeoutPropagatesContext(t *testing.T) {
	tm := New[int](time.Second)

	result, err := tm.Execute(context.Background(), "test", func(ctx context.Context) (int, error) {
		// Check that context has deadline
		if _, ok := ctx.Deadline(); !ok {
			return 0, errors.New("context should have deadline")
		}
		return 42, nil
	})

	if err != nil || result != 42 {
		t.Errorf("expected 42, got %d, err: %v", result, err)
	}
}
