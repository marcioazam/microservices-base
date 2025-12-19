package atomic

import (
	"sync"
	"testing"
)

func TestValueBasicOperations(t *testing.T) {
	t.Run("New creates value with initial", func(t *testing.T) {
		v := New(42)
		if v.Load() != 42 {
			t.Errorf("expected 42, got %d", v.Load())
		}
	})

	t.Run("Store updates value", func(t *testing.T) {
		v := New(0)
		v.Store(42)
		if v.Load() != 42 {
			t.Errorf("expected 42, got %d", v.Load())
		}
	})

	t.Run("Swap returns old value", func(t *testing.T) {
		v := New(42)
		old := v.Swap(100)
		if old != 42 {
			t.Errorf("expected old value 42, got %d", old)
		}
		if v.Load() != 100 {
			t.Errorf("expected new value 100, got %d", v.Load())
		}
	})

	t.Run("Update applies function", func(t *testing.T) {
		v := New(10)
		result := v.Update(func(x int) int { return x * 2 })
		if result != 20 {
			t.Errorf("expected 20, got %d", result)
		}
	})

	t.Run("GetAndUpdate returns old value", func(t *testing.T) {
		v := New(10)
		old := v.GetAndUpdate(func(x int) int { return x * 2 })
		if old != 10 {
			t.Errorf("expected old value 10, got %d", old)
		}
		if v.Load() != 20 {
			t.Errorf("expected new value 20, got %d", v.Load())
		}
	})
}

func TestValueCompareAndSwap(t *testing.T) {
	t.Run("CAS succeeds when values match", func(t *testing.T) {
		v := New(42)
		swapped := v.CompareAndSwap(42, 100, func(a, b int) bool { return a == b })
		if !swapped {
			t.Error("expected swap to succeed")
		}
		if v.Load() != 100 {
			t.Errorf("expected 100, got %d", v.Load())
		}
	})

	t.Run("CAS fails when values don't match", func(t *testing.T) {
		v := New(42)
		swapped := v.CompareAndSwap(50, 100, func(a, b int) bool { return a == b })
		if swapped {
			t.Error("expected swap to fail")
		}
		if v.Load() != 42 {
			t.Errorf("expected 42, got %d", v.Load())
		}
	})
}

func TestInt64Operations(t *testing.T) {
	t.Run("Add adds delta", func(t *testing.T) {
		v := NewInt64(10)
		result := v.Add(5)
		if result != 15 {
			t.Errorf("expected 15, got %d", result)
		}
	})

	t.Run("Sub subtracts delta", func(t *testing.T) {
		v := NewInt64(10)
		result := v.Sub(3)
		if result != 7 {
			t.Errorf("expected 7, got %d", result)
		}
	})

	t.Run("Inc increments by 1", func(t *testing.T) {
		v := NewInt64(10)
		result := v.Inc()
		if result != 11 {
			t.Errorf("expected 11, got %d", result)
		}
	})

	t.Run("Dec decrements by 1", func(t *testing.T) {
		v := NewInt64(10)
		result := v.Dec()
		if result != 9 {
			t.Errorf("expected 9, got %d", result)
		}
	})
}

func TestBoolOperations(t *testing.T) {
	t.Run("Toggle flips value", func(t *testing.T) {
		v := NewBool(false)
		result := v.Toggle()
		if !result {
			t.Error("expected true after toggle")
		}
		result = v.Toggle()
		if result {
			t.Error("expected false after second toggle")
		}
	})
}

func TestStringOperations(t *testing.T) {
	t.Run("Store and Load string", func(t *testing.T) {
		v := NewString("hello")
		if v.Load() != "hello" {
			t.Errorf("expected hello, got %s", v.Load())
		}
		v.Store("world")
		if v.Load() != "world" {
			t.Errorf("expected world, got %s", v.Load())
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	v := NewInt64(0)
	var wg sync.WaitGroup
	numGoroutines := 100
	incrementsPerGoroutine := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				v.Inc()
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * incrementsPerGoroutine)
	if v.Load() != expected {
		t.Errorf("expected %d, got %d", expected, v.Load())
	}
}
