package lazy

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestLazyBasicOperations(t *testing.T) {
	t.Run("Get initializes value once", func(t *testing.T) {
		callCount := 0
		l := New(func() int {
			callCount++
			return 42
		})

		v1 := l.Get()
		v2 := l.Get()

		if v1 != 42 || v2 != 42 {
			t.Errorf("expected 42, got %d and %d", v1, v2)
		}
		if callCount != 1 {
			t.Errorf("expected 1 call, got %d", callCount)
		}
	})

	t.Run("IsInitialized returns correct state", func(t *testing.T) {
		l := New(func() int { return 42 })

		if l.IsInitialized() {
			t.Error("expected not initialized before Get")
		}

		l.Get()

		if !l.IsInitialized() {
			t.Error("expected initialized after Get")
		}
	})
}

func TestLazyWithError(t *testing.T) {
	t.Run("Get initializes value once on success", func(t *testing.T) {
		callCount := 0
		l := NewWithError(func() (int, error) {
			callCount++
			return 42, nil
		})

		v1, err1 := l.Get()
		v2, err2 := l.Get()

		if err1 != nil || err2 != nil {
			t.Error("expected no errors")
		}
		if v1 != 42 || v2 != 42 {
			t.Errorf("expected 42, got %d and %d", v1, v2)
		}
		if callCount != 1 {
			t.Errorf("expected 1 call, got %d", callCount)
		}
	})

	t.Run("Get retries on error", func(t *testing.T) {
		callCount := 0
		l := NewWithError(func() (int, error) {
			callCount++
			if callCount < 3 {
				return 0, errors.New("not ready")
			}
			return 42, nil
		})

		_, err := l.Get()
		if err == nil {
			t.Error("expected error on first call")
		}

		_, err = l.Get()
		if err == nil {
			t.Error("expected error on second call")
		}

		v, err := l.Get()
		if err != nil || v != 42 {
			t.Errorf("expected 42, got %d, err: %v", v, err)
		}

		if callCount != 3 {
			t.Errorf("expected 3 calls, got %d", callCount)
		}
	})

	t.Run("MustGet panics on error", func(t *testing.T) {
		l := NewWithError(func() (int, error) {
			return 0, errors.New("failed")
		})

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()

		l.MustGet()
	})

	t.Run("Reset allows re-initialization", func(t *testing.T) {
		callCount := 0
		l := NewWithError(func() (int, error) {
			callCount++
			return callCount * 10, nil
		})

		v1, _ := l.Get()
		if v1 != 10 {
			t.Errorf("expected 10, got %d", v1)
		}

		l.Reset()

		v2, _ := l.Get()
		if v2 != 20 {
			t.Errorf("expected 20, got %d", v2)
		}
	})

	t.Run("ResetWithFn changes function", func(t *testing.T) {
		l := NewWithError(func() (int, error) {
			return 10, nil
		})

		v1, _ := l.Get()
		if v1 != 10 {
			t.Errorf("expected 10, got %d", v1)
		}

		l.ResetWithFn(func() (int, error) {
			return 20, nil
		})

		v2, _ := l.Get()
		if v2 != 20 {
			t.Errorf("expected 20, got %d", v2)
		}
	})
}

func TestLazyConcurrency(t *testing.T) {
	var callCount int32
	l := New(func() int {
		atomic.AddInt32(&callCount, 1)
		return 42
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := l.Get()
			if v != 42 {
				t.Errorf("expected 42, got %d", v)
			}
		}()
	}

	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestMemoize(t *testing.T) {
	t.Run("Memoize caches result", func(t *testing.T) {
		callCount := 0
		fn := Memoize(func() int {
			callCount++
			return 42
		})

		v1 := fn()
		v2 := fn()

		if v1 != 42 || v2 != 42 {
			t.Errorf("expected 42, got %d and %d", v1, v2)
		}
		if callCount != 1 {
			t.Errorf("expected 1 call, got %d", callCount)
		}
	})

	t.Run("MemoizeWithError caches result", func(t *testing.T) {
		callCount := 0
		fn := MemoizeWithError(func() (int, error) {
			callCount++
			return 42, nil
		})

		v1, _ := fn()
		v2, _ := fn()

		if v1 != 42 || v2 != 42 {
			t.Errorf("expected 42, got %d and %d", v1, v2)
		}
		if callCount != 1 {
			t.Errorf("expected 1 call, got %d", callCount)
		}
	})
}

func TestValue(t *testing.T) {
	l := Value(42)

	if !l.IsInitialized() {
		t.Error("expected initialized")
	}

	if l.Get() != 42 {
		t.Errorf("expected 42, got %d", l.Get())
	}
}
