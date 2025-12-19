package lru

import "testing"

func TestLRUCache(t *testing.T) {
	t.Run("Set and Get", func(t *testing.T) {
		c := New[string, int](3)
		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3)

		v, ok := c.Get("a")
		if !ok || v != 1 {
			t.Error("expected 1")
		}
	})

	t.Run("Evicts oldest on capacity", func(t *testing.T) {
		c := New[string, int](2)
		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3) // Should evict "a"

		_, ok := c.Get("a")
		if ok {
			t.Error("expected a to be evicted")
		}

		v, ok := c.Get("b")
		if !ok || v != 2 {
			t.Error("expected b to exist")
		}
	})

	t.Run("Get updates recency", func(t *testing.T) {
		c := New[string, int](2)
		c.Set("a", 1)
		c.Set("b", 2)
		c.Get("a")    // Make "a" most recent
		c.Set("c", 3) // Should evict "b"

		_, ok := c.Get("b")
		if ok {
			t.Error("expected b to be evicted")
		}

		v, ok := c.Get("a")
		if !ok || v != 1 {
			t.Error("expected a to exist")
		}
	})

	t.Run("Peek does not update recency", func(t *testing.T) {
		c := New[string, int](2)
		c.Set("a", 1)
		c.Set("b", 2)
		c.Peek("a")   // Should not update recency
		c.Set("c", 3) // Should evict "a"

		_, ok := c.Peek("a")
		if ok {
			t.Error("expected a to be evicted")
		}
	})

	t.Run("Remove", func(t *testing.T) {
		c := New[string, int](3)
		c.Set("a", 1)
		c.Set("b", 2)

		if !c.Remove("a") {
			t.Error("expected remove to return true")
		}

		_, ok := c.Get("a")
		if ok {
			t.Error("expected a to be removed")
		}

		if c.Remove("nonexistent") {
			t.Error("expected remove to return false")
		}
	})

	t.Run("Contains", func(t *testing.T) {
		c := New[string, int](3)
		c.Set("a", 1)

		if !c.Contains("a") {
			t.Error("expected a to exist")
		}
		if c.Contains("b") {
			t.Error("expected b to not exist")
		}
	})

	t.Run("Keys returns in order", func(t *testing.T) {
		c := New[string, int](3)
		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3)
		c.Get("a") // Make "a" most recent

		keys := c.Keys()
		if keys[0] != "a" {
			t.Error("expected a to be first")
		}
	})

	t.Run("Len", func(t *testing.T) {
		c := New[string, int](3)
		if c.Len() != 0 {
			t.Error("expected 0")
		}
		c.Set("a", 1)
		c.Set("b", 2)
		if c.Len() != 2 {
			t.Errorf("expected 2, got %d", c.Len())
		}
	})

	t.Run("Clear", func(t *testing.T) {
		c := New[string, int](3)
		c.Set("a", 1)
		c.Set("b", 2)
		c.Clear()
		if c.Len() != 0 {
			t.Error("expected empty after clear")
		}
	})

	t.Run("Resize", func(t *testing.T) {
		c := New[string, int](3)
		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3)
		c.Resize(2) // Should evict oldest

		if c.Len() != 2 {
			t.Errorf("expected 2, got %d", c.Len())
		}
	})

	t.Run("GetOrSet", func(t *testing.T) {
		c := New[string, int](3)

		v, existed := c.GetOrSet("a", 1)
		if existed || v != 1 {
			t.Error("expected new value")
		}

		v, existed = c.GetOrSet("a", 2)
		if !existed || v != 1 {
			t.Error("expected existing value")
		}
	})

	t.Run("Evict callback", func(t *testing.T) {
		evicted := make(map[string]int)
		c := New[string, int](2).WithEvictCallback(func(k string, v int) {
			evicted[k] = v
		})

		c.Set("a", 1)
		c.Set("b", 2)
		c.Set("c", 3) // Should evict "a"

		if evicted["a"] != 1 {
			t.Error("expected a to be evicted with value 1")
		}
	})
}
