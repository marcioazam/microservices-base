package registry

import (
	"sync"
	"testing"

	"github.com/authcorp/libs/go/src/patterns"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 23: Registry Thread-Safety**
// **Validates: Requirements 13.11**
func TestRegistryThreadSafety(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("concurrent operations complete without data races", prop.ForAll(
		func(keys []string, values []int) bool {
			if len(keys) == 0 || len(values) == 0 {
				return true
			}

			r := patterns.NewRegistry[string, int]()
			var wg sync.WaitGroup
			numGoroutines := 10

			// Concurrent writes
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					for j, key := range keys {
						r.Register(key, values[j%len(values)]+idx)
					}
				}(i)
			}

			// Concurrent reads
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for _, key := range keys {
						r.Get(key)
						r.Has(key)
					}
				}()
			}

			// Concurrent iterations
			for i := 0; i < numGoroutines/2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					r.Keys()
					r.Values()
					r.Size()
				}()
			}

			wg.Wait()
			return true // If we get here without panic/race, test passes
		},
		gen.SliceOfN(10, gen.AnyString()),
		gen.SliceOfN(10, gen.Int()),
	))

	properties.TestingRun(t)
}

func TestRegistryBasicOperations(t *testing.T) {
	t.Run("NewRegistry creates empty registry", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		if r.Size() != 0 {
			t.Error("expected empty registry")
		}
	})

	t.Run("Register stores value", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		r.Register("key", 42)
		opt := r.Get("key")
		if opt.IsNone() || opt.Unwrap() != 42 {
			t.Error("expected value to be stored")
		}
	})

	t.Run("Get returns None for missing key", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		if r.Get("missing").IsSome() {
			t.Error("expected None for missing key")
		}
	})

	t.Run("GetOrDefault returns default for missing key", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		if r.GetOrDefault("missing", 100) != 100 {
			t.Error("expected default value")
		}
	})

	t.Run("Has returns true for existing key", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		r.Register("key", 42)
		if !r.Has("key") {
			t.Error("expected Has to return true")
		}
	})

	t.Run("Unregister removes value", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		r.Register("key", 42)
		removed := r.Unregister("key")
		if !removed || r.Has("key") {
			t.Error("expected value to be removed")
		}
	})

	t.Run("Unregister returns false for missing key", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		if r.Unregister("missing") {
			t.Error("expected false for missing key")
		}
	})

	t.Run("Clear removes all entries", func(t *testing.T) {
		r := patterns.NewRegistry[string, int]()
		r.Register("a", 1)
		r.Register("b", 2)
		r.Clear()
		if r.Size() != 0 {
			t.Error("expected empty registry after clear")
		}
	})
}

func TestRegistryKeysValues(t *testing.T) {
	r := patterns.NewRegistry[string, int]()
	r.Register("a", 1)
	r.Register("b", 2)
	r.Register("c", 3)

	keys := r.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	values := r.Values()
	if len(values) != 3 {
		t.Errorf("expected 3 values, got %d", len(values))
	}
}

func TestRegistryForEach(t *testing.T) {
	r := patterns.NewRegistry[string, int]()
	r.Register("a", 1)
	r.Register("b", 2)

	sum := 0
	r.ForEach(func(k string, v int) {
		sum += v
	})

	if sum != 3 {
		t.Errorf("expected sum 3, got %d", sum)
	}
}

func TestRegistryGetOrRegister(t *testing.T) {
	r := patterns.NewRegistry[string, int]()

	// First call should register
	v1 := r.GetOrRegister("key", func() int { return 42 })
	if v1 != 42 {
		t.Errorf("expected 42, got %d", v1)
	}

	// Second call should return existing
	v2 := r.GetOrRegister("key", func() int { return 100 })
	if v2 != 42 {
		t.Errorf("expected 42, got %d", v2)
	}
}

func TestRegistryUpdate(t *testing.T) {
	r := patterns.NewRegistry[string, int]()

	// Update non-existent key
	v1 := r.Update("key", func(old int, exists bool) int {
		if exists {
			return old + 1
		}
		return 1
	})
	if v1 != 1 {
		t.Errorf("expected 1, got %d", v1)
	}

	// Update existing key
	v2 := r.Update("key", func(old int, exists bool) int {
		if exists {
			return old + 1
		}
		return 1
	})
	if v2 != 2 {
		t.Errorf("expected 2, got %d", v2)
	}
}

func TestRegistryFilter(t *testing.T) {
	r := patterns.NewRegistry[string, int]()
	r.Register("a", 1)
	r.Register("b", 2)
	r.Register("c", 3)

	filtered := r.FilterRegistry(func(k string, v int) bool {
		return v > 1
	})

	if filtered.Size() != 2 {
		t.Errorf("expected 2 entries, got %d", filtered.Size())
	}
}

func TestRegistryClone(t *testing.T) {
	r := patterns.NewRegistry[string, int]()
	r.Register("a", 1)
	r.Register("b", 2)

	clone := r.Clone()
	r.Register("c", 3)

	if clone.Has("c") {
		t.Error("clone should be independent")
	}
}
