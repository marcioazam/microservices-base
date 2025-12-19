package async

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFuture(t *testing.T) {
	t.Run("Go starts async computation", func(t *testing.T) {
		f := Go(func() (int, error) {
			return 42, nil
		})
		v, err := f.Wait()
		if err != nil || v != 42 {
			t.Errorf("expected 42, got %d, err: %v", v, err)
		}
	})

	t.Run("Go handles errors", func(t *testing.T) {
		f := Go(func() (int, error) {
			return 0, errors.New("test error")
		})
		_, err := f.Wait()
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("WaitContext respects cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		f := Go(func() (int, error) {
			time.Sleep(time.Second)
			return 42, nil
		})
		cancel()
		_, err := f.WaitContext(ctx)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("Result returns Result type", func(t *testing.T) {
		f := Go(func() (int, error) {
			return 42, nil
		})
		r := f.Result()
		if !r.IsOk() || r.Unwrap() != 42 {
			t.Error("expected Ok(42)")
		}
	})
}

func TestParallel(t *testing.T) {
	t.Run("Parallel runs all functions", func(t *testing.T) {
		results, errs := Parallel(
			func() (int, error) { return 1, nil },
			func() (int, error) { return 2, nil },
			func() (int, error) { return 3, nil },
		)
		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}
		for _, err := range errs {
			if err != nil {
				t.Error("unexpected error")
			}
		}
	})

	t.Run("Parallel preserves order", func(t *testing.T) {
		results, _ := Parallel(
			func() (int, error) { time.Sleep(10 * time.Millisecond); return 1, nil },
			func() (int, error) { return 2, nil },
			func() (int, error) { time.Sleep(5 * time.Millisecond); return 3, nil },
		)
		if results[0] != 1 || results[1] != 2 || results[2] != 3 {
			t.Error("order not preserved")
		}
	})
}

func TestRace(t *testing.T) {
	t.Run("Race returns first result", func(t *testing.T) {
		v, err, idx := Race(
			func() (int, error) { time.Sleep(100 * time.Millisecond); return 1, nil },
			func() (int, error) { return 2, nil },
			func() (int, error) { time.Sleep(50 * time.Millisecond); return 3, nil },
		)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if v != 2 || idx != 1 {
			t.Errorf("expected 2 at index 1, got %d at %d", v, idx)
		}
	})
}

func TestWithTimeout(t *testing.T) {
	t.Run("WithTimeout returns result before timeout", func(t *testing.T) {
		v, err := WithTimeout(time.Second, func() (int, error) {
			return 42, nil
		})
		if err != nil || v != 42 {
			t.Errorf("expected 42, got %d, err: %v", v, err)
		}
	})

	t.Run("WithTimeout returns error on timeout", func(t *testing.T) {
		_, err := WithTimeout(10*time.Millisecond, func() (int, error) {
			time.Sleep(100 * time.Millisecond)
			return 42, nil
		})
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}
	})
}

func TestCollect(t *testing.T) {
	t.Run("Collect returns successful results", func(t *testing.T) {
		results := Collect(
			func() (int, error) { return 1, nil },
			func() (int, error) { return 0, errors.New("fail") },
			func() (int, error) { return 3, nil },
		)
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
	})
}

func TestFanOut(t *testing.T) {
	t.Run("FanOut processes all items", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		results, errs := FanOut(items, 2, func(x int) (int, error) {
			return x * 2, nil
		})
		if len(results) != 5 {
			t.Errorf("expected 5 results, got %d", len(results))
		}
		for i, r := range results {
			if r != items[i]*2 {
				t.Errorf("expected %d, got %d", items[i]*2, r)
			}
		}
		for _, err := range errs {
			if err != nil {
				t.Error("unexpected error")
			}
		}
	})

	t.Run("FanOut preserves order", func(t *testing.T) {
		items := []int{1, 2, 3}
		results, _ := FanOut(items, 3, func(x int) (int, error) {
			time.Sleep(time.Duration(4-x) * 10 * time.Millisecond)
			return x, nil
		})
		if results[0] != 1 || results[1] != 2 || results[2] != 3 {
			t.Error("order not preserved")
		}
	})
}

func TestAllAny(t *testing.T) {
	t.Run("All returns true when all succeed", func(t *testing.T) {
		f1 := Go(func() (int, error) { return 1, nil })
		f2 := Go(func() (int, error) { return 2, nil })
		if !All(f1, f2) {
			t.Error("expected true")
		}
	})

	t.Run("All returns false when any fails", func(t *testing.T) {
		f1 := Go(func() (int, error) { return 1, nil })
		f2 := Go(func() (int, error) { return 0, errors.New("fail") })
		if All(f1, f2) {
			t.Error("expected false")
		}
	})

	t.Run("Any returns true when any succeeds", func(t *testing.T) {
		f1 := Go(func() (int, error) { return 0, errors.New("fail") })
		f2 := Go(func() (int, error) { return 2, nil })
		if !Any(f1, f2) {
			t.Error("expected true")
		}
	})

	t.Run("Any returns false when all fail", func(t *testing.T) {
		f1 := Go(func() (int, error) { return 0, errors.New("fail1") })
		f2 := Go(func() (int, error) { return 0, errors.New("fail2") })
		if Any(f1, f2) {
			t.Error("expected false")
		}
	})
}
