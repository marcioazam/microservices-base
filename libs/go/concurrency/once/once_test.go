package once

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestOnceBasicOperations(t *testing.T) {
	t.Run("Get initializes value once", func(t *testing.T) {
		callCount := 0
		o := New(func() (int, error) {
			callCount++
			return 42, nil
		})

		v1, err := o.Get()
		if err != nil || v1 != 42 {
			t.Errorf("expected 42, got %d, err: %v", v1, err)
		}

		v2, err := o.Get()
		if err != nil || v2 != 42 {
			t.Errorf("expected 42, got %d, err: %v", v2, err)
		}

		if callCount != 1 {
			t.Errorf("expected 1 call, got %d", callCount)
		}
	})

	t.Run("Get retries on error", func(t *testing.T) {
		callCount := 0
		o := New(func() (int, error) {
			callCount++
			if callCount < 3 {
				return 0, errors.New("not ready")
			}
			return 42, nil
		})

		_, err := o.Get()
		if err == nil {
			t.Error("expected error on first call")
		}

		_, err = o.Get()
		if err == nil {
			t.Error("expected error on second call")
		}

		v, err := o.Get()
		if err != nil || v != 42 {
			t.Errorf("expected 42, got %d, err: %v", v, err)
		}

		if callCount != 3 {
			t.Errorf("expected 3 calls, got %d", callCount)
		}
	})

	t.Run("IsDone returns correct state", func(t *testing.T) {
		o := New(func() (int, error) {
			return 42, nil
		})

		if o.IsDone() {
			t.Error("expected not done before Get")
		}

		o.Get()

		if !o.IsDone() {
			t.Error("expected done after Get")
		}
	})

	t.Run("Reset allows re-initialization", func(t *testing.T) {
		callCount := 0
		o := New(func() (int, error) {
			callCount++
			return callCount * 10, nil
		})

		v1, _ := o.Get()
		if v1 != 10 {
			t.Errorf("expected 10, got %d", v1)
		}

		o.Reset()

		v2, _ := o.Get()
		if v2 != 20 {
			t.Errorf("expected 20, got %d", v2)
		}
	})
}

func TestNewSimple(t *testing.T) {
	callCount := 0
	o := NewSimple(func() int {
		callCount++
		return 42
	})

	v, err := o.Get()
	if err != nil || v != 42 {
		t.Errorf("expected 42, got %d, err: %v", v, err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestMustGet(t *testing.T) {
	t.Run("MustGet returns value on success", func(t *testing.T) {
		o := NewSimple(func() int { return 42 })
		v := o.MustGet()
		if v != 42 {
			t.Errorf("expected 42, got %d", v)
		}
	})

	t.Run("MustGet panics on error", func(t *testing.T) {
		o := New(func() (int, error) {
			return 0, errors.New("failed")
		})

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()

		o.MustGet()
	})
}

func TestResetWithFn(t *testing.T) {
	o := NewSimple(func() int { return 10 })

	v1, _ := o.Get()
	if v1 != 10 {
		t.Errorf("expected 10, got %d", v1)
	}

	o.ResetWithFn(func() (int, error) { return 20, nil })

	v2, _ := o.Get()
	if v2 != 20 {
		t.Errorf("expected 20, got %d", v2)
	}
}

func TestLazy(t *testing.T) {
	t.Run("Get initializes value once", func(t *testing.T) {
		callCount := 0
		l := NewLazy(func() int {
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
}

func TestConcurrentAccess(t *testing.T) {
	var callCount int32
	o := New(func() (int, error) {
		atomic.AddInt32(&callCount, 1)
		return 42, nil
	})

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := o.Get()
			if err != nil || v != 42 {
				t.Errorf("unexpected result: %d, %v", v, err)
			}
		}()
	}

	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}
