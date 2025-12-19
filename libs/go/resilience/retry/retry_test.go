package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySuccess(t *testing.T) {
	t.Run("returns immediately on success", func(t *testing.T) {
		result, err := Retry(context.Background(), func() (int, error) {
			return 42, nil
		})

		if err != nil || result != 42 {
			t.Errorf("expected 42, got %d, err: %v", result, err)
		}
	})

	t.Run("succeeds after retries", func(t *testing.T) {
		attempts := 0
		result, err := Retry(context.Background(), func() (int, error) {
			attempts++
			if attempts < 3 {
				return 0, errors.New("not yet")
			}
			return 42, nil
		}, WithMaxAttempts(5), WithBaseDelay(time.Millisecond))

		if err != nil || result != 42 {
			t.Errorf("expected 42, got %d, err: %v", result, err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
}

func TestRetryFailure(t *testing.T) {
	t.Run("returns error after max attempts", func(t *testing.T) {
		attempts := 0
		_, err := Retry(context.Background(), func() (int, error) {
			attempts++
			return 0, errors.New("always fails")
		}, WithMaxAttempts(3), WithBaseDelay(time.Millisecond))

		if err == nil {
			t.Error("expected error")
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
		if !IsRetryError(err) {
			t.Error("expected RetryError")
		}
	})

	t.Run("stops on non-retryable error", func(t *testing.T) {
		nonRetryable := errors.New("non-retryable")
		attempts := 0

		_, err := Retry(context.Background(), func() (int, error) {
			attempts++
			return 0, nonRetryable
		}, WithMaxAttempts(5), WithRetryIf(func(err error) bool {
			return err != nonRetryable
		}))

		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
		if err != nonRetryable {
			t.Errorf("expected non-retryable error, got %v", err)
		}
	})
}

func TestRetryContext(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		attempts := 0
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := Retry(ctx, func() (int, error) {
			attempts++
			return 0, errors.New("fail")
		}, WithMaxAttempts(100), WithBaseDelay(20*time.Millisecond))

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("respects context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := Retry(ctx, func() (int, error) {
			return 0, errors.New("fail")
		}, WithMaxAttempts(100), WithBaseDelay(20*time.Millisecond))

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}

func TestRetryOptions(t *testing.T) {
	t.Run("WithMaxAttempts", func(t *testing.T) {
		attempts := 0
		Retry(context.Background(), func() (int, error) {
			attempts++
			return 0, errors.New("fail")
		}, WithMaxAttempts(5), WithBaseDelay(time.Millisecond))

		if attempts != 5 {
			t.Errorf("expected 5 attempts, got %d", attempts)
		}
	})

	t.Run("WithBaseDelay", func(t *testing.T) {
		start := time.Now()
		Retry(context.Background(), func() (int, error) {
			return 0, errors.New("fail")
		}, WithMaxAttempts(2), WithBaseDelay(50*time.Millisecond))

		elapsed := time.Since(start)
		if elapsed < 40*time.Millisecond {
			t.Errorf("expected at least 40ms delay, got %v", elapsed)
		}
	})

	t.Run("WithMaxDelay caps delay", func(t *testing.T) {
		start := time.Now()
		Retry(context.Background(), func() (int, error) {
			return 0, errors.New("fail")
		}, WithMaxAttempts(3), WithBaseDelay(100*time.Millisecond),
			WithMultiplier(10), WithMaxDelay(50*time.Millisecond))

		elapsed := time.Since(start)
		// With max delay of 50ms, total should be around 100ms (2 delays)
		if elapsed > 200*time.Millisecond {
			t.Errorf("delay should be capped, got %v", elapsed)
		}
	})
}

func TestRetryError(t *testing.T) {
	t.Run("wraps last error", func(t *testing.T) {
		lastErr := errors.New("last error")
		_, err := Retry(context.Background(), func() (int, error) {
			return 0, lastErr
		}, WithMaxAttempts(2), WithBaseDelay(time.Millisecond))

		var retryErr *RetryError
		if !errors.As(err, &retryErr) {
			t.Fatal("expected RetryError")
		}
		if retryErr.Attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", retryErr.Attempts)
		}
		if !errors.Is(err, lastErr) {
			t.Error("should unwrap to last error")
		}
	})

	t.Run("GetRetryAttempts returns count", func(t *testing.T) {
		_, err := Retry(context.Background(), func() (int, error) {
			return 0, errors.New("fail")
		}, WithMaxAttempts(3), WithBaseDelay(time.Millisecond))

		if GetRetryAttempts(err) != 3 {
			t.Errorf("expected 3 attempts, got %d", GetRetryAttempts(err))
		}
	})
}

func TestDo(t *testing.T) {
	t.Run("Do works for void operations", func(t *testing.T) {
		attempts := 0
		err := Do(context.Background(), func() error {
			attempts++
			if attempts < 2 {
				return errors.New("not yet")
			}
			return nil
		}, WithMaxAttempts(3), WithBaseDelay(time.Millisecond))

		if err != nil {
			t.Errorf("expected success, got %v", err)
		}
		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}

func TestRetryWithContext(t *testing.T) {
	t.Run("passes context to operation", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "key", "value")

		result, err := RetryWithContext(ctx, func(c context.Context) (string, error) {
			return c.Value("key").(string), nil
		})

		if err != nil || result != "value" {
			t.Errorf("expected value, got %s, err: %v", result, err)
		}
	})
}
