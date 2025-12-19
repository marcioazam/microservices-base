package iterator

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestIteratorBasicOperations(t *testing.T) {
	t.Run("FromSlice creates iterator", func(t *testing.T) {
		it := FromSlice([]int{1, 2, 3})
		if !it.HasNext() {
			t.Error("expected HasNext to be true")
		}
	})

	t.Run("Of creates iterator from values", func(t *testing.T) {
		it := Of(1, 2, 3)
		if it.Remaining() != 3 {
			t.Errorf("expected 3, got %d", it.Remaining())
		}
	})

	t.Run("Next returns elements in order", func(t *testing.T) {
		it := Of(1, 2, 3)
		if it.Next().Unwrap() != 1 {
			t.Error("expected 1")
		}
		if it.Next().Unwrap() != 2 {
			t.Error("expected 2")
		}
		if it.Next().Unwrap() != 3 {
			t.Error("expected 3")
		}
		if it.HasNext() {
			t.Error("expected no more elements")
		}
	})

	t.Run("Next returns None when exhausted", func(t *testing.T) {
		it := Of[int]()
		if it.Next().IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("Peek returns without advancing", func(t *testing.T) {
		it := Of(1, 2, 3)
		if it.Peek().Unwrap() != 1 {
			t.Error("expected 1")
		}
		if it.Peek().Unwrap() != 1 {
			t.Error("expected 1 again")
		}
	})

	t.Run("Reset resets iterator", func(t *testing.T) {
		it := Of(1, 2, 3)
		it.Next()
		it.Next()
		it.Reset()
		if it.Next().Unwrap() != 1 {
			t.Error("expected 1 after reset")
		}
	})
}

func TestIteratorTransformations(t *testing.T) {
	t.Run("Map transforms elements", func(t *testing.T) {
		it := Of(1, 2, 3)
		mapped := Map(it, func(x int) int { return x * 2 })
		collected := mapped.Collect()
		if collected[0] != 2 || collected[1] != 4 || collected[2] != 6 {
			t.Error("unexpected values")
		}
	})

	t.Run("Filter keeps matching elements", func(t *testing.T) {
		it := Of(1, 2, 3, 4, 5)
		filtered := Filter(it, func(x int) bool { return x%2 == 0 })
		collected := filtered.Collect()
		if len(collected) != 2 || collected[0] != 2 || collected[1] != 4 {
			t.Error("unexpected values")
		}
	})

	t.Run("Take returns first n elements", func(t *testing.T) {
		it := Of(1, 2, 3, 4, 5)
		taken := Take(it, 3)
		if taken.Remaining() != 3 {
			t.Errorf("expected 3, got %d", taken.Remaining())
		}
	})

	t.Run("Skip skips first n elements", func(t *testing.T) {
		it := Of(1, 2, 3, 4, 5)
		skipped := Skip(it, 2)
		collected := skipped.Collect()
		if len(collected) != 3 || collected[0] != 3 {
			t.Error("unexpected values")
		}
	})
}

func TestIteratorTerminalOperations(t *testing.T) {
	t.Run("Collect returns all elements", func(t *testing.T) {
		it := Of(1, 2, 3)
		collected := it.Collect()
		if len(collected) != 3 {
			t.Errorf("expected 3, got %d", len(collected))
		}
	})

	t.Run("Find returns matching element", func(t *testing.T) {
		it := Of(1, 2, 3, 4, 5)
		found := it.Find(func(x int) bool { return x > 3 })
		if found.IsNone() || found.Unwrap() != 4 {
			t.Error("expected 4")
		}
	})

	t.Run("Any returns true if any match", func(t *testing.T) {
		it := Of(1, 2, 3)
		if !it.Any(func(x int) bool { return x == 2 }) {
			t.Error("expected true")
		}
	})

	t.Run("All returns true if all match", func(t *testing.T) {
		it := Of(2, 4, 6)
		if !it.All(func(x int) bool { return x%2 == 0 }) {
			t.Error("expected true")
		}
	})

	t.Run("Count returns element count", func(t *testing.T) {
		it := Of(1, 2, 3, 4, 5)
		if it.Count() != 5 {
			t.Error("expected 5")
		}
	})

	t.Run("Reduce folds elements", func(t *testing.T) {
		it := Of(1, 2, 3, 4, 5)
		sum := Reduce(it, 0, func(a, b int) int { return a + b })
		if sum != 15 {
			t.Errorf("expected 15, got %d", sum)
		}
	})
}

func TestIteratorCombinators(t *testing.T) {
	t.Run("Chain concatenates iterators", func(t *testing.T) {
		it1 := Of(1, 2)
		it2 := Of(3, 4)
		chained := Chain(it1, it2)
		if chained.Remaining() != 4 {
			t.Errorf("expected 4, got %d", chained.Remaining())
		}
	})

	t.Run("ForEach applies function", func(t *testing.T) {
		sum := 0
		it := Of(1, 2, 3)
		it.ForEach(func(x int) { sum += x })
		if sum != 6 {
			t.Errorf("expected 6, got %d", sum)
		}
	})
}

func TestIteratorPropertyBased(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Collect returns all elements", prop.ForAll(
		func(items []int) bool {
			it := FromSlice(items)
			collected := it.Collect()
			if len(collected) != len(items) {
				return false
			}
			for i, item := range items {
				if collected[i] != item {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Int()),
	))

	properties.Property("Take returns at most n elements", prop.ForAll(
		func(items []int, n int) bool {
			if n < 0 {
				return true
			}
			it := FromSlice(items)
			taken := Take(it, n)
			expected := n
			if len(items) < n {
				expected = len(items)
			}
			return taken.Remaining() == expected
		},
		gen.SliceOf(gen.Int()),
		gen.IntRange(0, 20),
	))

	properties.TestingRun(t)
}
