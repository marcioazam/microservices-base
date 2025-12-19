package cache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	t.Run("Set and Get", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		c.Set("a", 1)
		v, ok := c.Get("a")
		if !ok || v != 1 {
			t.Error("expected 1")
		}
	})

	t.Run("Get returns false for missing key", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		_, ok := c.Get("missing")
		if ok {
			t.Error("expected false")
		}
	})

	t.Run("Get returns false for expired key", func(t *testing.T) {
		c := New[string, int](10 * time.Millisecond)
		defer c.Close()

		c.Set("a", 1)
		time.Sleep(20 * time.Millisecond)

		_, ok := c.Get("a")
		if ok {
			t.Error("expected false for expired key")
		}
	})

	t.Run("SetWithTTL uses custom TTL", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		c.SetWithTTL("a", 1, 10*time.Millisecond)
		time.Sleep(20 * time.Millisecond)

		_, ok := c.Get("a")
		if ok {
			t.Error("expected false for expired key")
		}
	})

	t.Run("GetOrCompute computes on miss", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		computed := false
		v := c.GetOrCompute("a", func() int {
			computed = true
			return 42
		})

		if !computed || v != 42 {
			t.Error("expected computed value")
		}

		// Second call should not compute
		computed = false
		v = c.GetOrCompute("a", func() int {
			computed = true
			return 99
		})

		if computed || v != 42 {
			t.Error("expected cached value")
		}
	})

	t.Run("Delete removes key", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		c.Set("a", 1)
		c.Delete("a")

		_, ok := c.Get("a")
		if ok {
			t.Error("expected key to be deleted")
		}
	})

	t.Run("Clear removes all keys", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		c.Set("a", 1)
		c.Set("b", 2)
		c.Clear()

		if c.Len() != 0 {
			t.Error("expected empty cache")
		}
	})

	t.Run("Contains", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		c.Set("a", 1)
		if !c.Contains("a") {
			t.Error("expected a to exist")
		}
		if c.Contains("b") {
			t.Error("expected b to not exist")
		}
	})

	t.Run("Keys returns non-expired keys", func(t *testing.T) {
		c := New[string, int](time.Minute)
		defer c.Close()

		c.Set("a", 1)
		c.Set("b", 2)
		c.SetWithTTL("c", 3, time.Nanosecond)
		time.Sleep(time.Millisecond)

		keys := c.Keys()
		if len(keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(keys))
		}
	})

	t.Run("Refresh updates expiration", func(t *testing.T) {
		c := New[string, int](50 * time.Millisecond)
		defer c.Close()

		c.Set("a", 1)
		time.Sleep(30 * time.Millisecond)
		c.Refresh("a")
		time.Sleep(30 * time.Millisecond)

		_, ok := c.Get("a")
		if !ok {
			t.Error("expected key to still exist after refresh")
		}
	})
}
