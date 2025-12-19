package syncmap

import (
	"sync"
	"testing"
)

func TestConcurrentMapBasicOperations(t *testing.T) {
	t.Run("New creates empty map", func(t *testing.T) {
		m := New[string, int]()
		if m.Len() != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		m := New[string, int]()
		m.Set("key", 42)
		if v, ok := m.Get("key"); !ok || v != 42 {
			t.Error("expected value to be stored")
		}
	})

	t.Run("Get returns false for missing key", func(t *testing.T) {
		m := New[string, int]()
		if _, ok := m.Get("missing"); ok {
			t.Error("expected false for missing key")
		}
	})

	t.Run("Delete removes value", func(t *testing.T) {
		m := New[string, int]()
		m.Set("key", 42)
		v, ok := m.Delete("key")
		if !ok || v != 42 || m.Has("key") {
			t.Error("expected value to be deleted")
		}
	})

	t.Run("Clear removes all entries", func(t *testing.T) {
		m := New[string, int]()
		m.Set("a", 1)
		m.Set("b", 2)
		m.Clear()
		if m.Len() != 0 {
			t.Error("expected empty map after clear")
		}
	})
}

func TestConcurrentMapGetOrSet(t *testing.T) {
	m := New[string, int]()

	// First call should set
	v1, existed := m.GetOrSet("key", 42)
	if existed || v1 != 42 {
		t.Error("expected new value to be set")
	}

	// Second call should get existing
	v2, existed := m.GetOrSet("key", 100)
	if !existed || v2 != 42 {
		t.Error("expected existing value")
	}
}

func TestConcurrentMapComputeIfAbsent(t *testing.T) {
	m := New[string, int]()
	computed := 0

	// First call should compute
	v1 := m.ComputeIfAbsent("key", func() int {
		computed++
		return 42
	})
	if v1 != 42 || computed != 1 {
		t.Error("expected computation to run once")
	}

	// Second call should not compute
	v2 := m.ComputeIfAbsent("key", func() int {
		computed++
		return 100
	})
	if v2 != 42 || computed != 1 {
		t.Error("expected no additional computation")
	}
}

func TestConcurrentMapUpdate(t *testing.T) {
	m := New[string, int]()

	// Update non-existent key
	v1 := m.Update("key", func(old int, exists bool) int {
		if exists {
			return old + 1
		}
		return 1
	})
	if v1 != 1 {
		t.Errorf("expected 1, got %d", v1)
	}

	// Update existing key
	v2 := m.Update("key", func(old int, exists bool) int {
		return old + 1
	})
	if v2 != 2 {
		t.Errorf("expected 2, got %d", v2)
	}
}

func TestConcurrentMapCompute(t *testing.T) {
	m := New[string, int]()
	m.Set("key", 42)

	// Compute and keep
	v1, kept := m.Compute("key", func(k string, v int, exists bool) (int, bool) {
		return v * 2, true
	})
	if !kept || v1 != 84 {
		t.Error("expected computed value to be kept")
	}

	// Compute and remove
	_, kept = m.Compute("key", func(k string, v int, exists bool) (int, bool) {
		return 0, false
	})
	if kept || m.Has("key") {
		t.Error("expected key to be removed")
	}
}

func TestConcurrentMapThreadSafety(t *testing.T) {
	m := New[int, int]()
	var wg sync.WaitGroup
	numGoroutines := 100
	numOps := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				m.Set(j, idx)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				m.Get(j)
			}
		}()
	}

	wg.Wait()
	// If we get here without panic/race, test passes
}

func TestConcurrentMapClone(t *testing.T) {
	m := New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)

	clone := m.Clone()
	m.Set("c", 3)

	if clone.Has("c") {
		t.Error("clone should be independent")
	}
}
