package stream

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 18: Stream Collect Materializes All**
// **Validates: Requirements 62.7**
func TestStreamCollectMaterializesAll(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Collect returns all elements in order", prop.ForAll(
		func(items []int) bool {
			s := FromSlice(items)
			collected := s.Collect()

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

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 19: Stream GroupBy Partitions Correctly**
// **Validates: Requirements 62.15**
func TestStreamGroupByPartitionsCorrectly(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("GroupBy partitions elements by key", prop.ForAll(
		func(items []int) bool {
			s := FromSlice(items)
			groups := GroupBy(s, func(x int) int { return x % 3 })

			// Check all elements are in groups
			totalCount := 0
			for _, group := range groups {
				totalCount += len(group)
			}
			if totalCount != len(items) {
				return false
			}

			// Check elements are in correct groups
			for key, group := range groups {
				for _, item := range group {
					if item%3 != key {
						return false
					}
				}
			}
			return true
		},
		gen.SliceOf(gen.Int()),
	))

	properties.TestingRun(t)
}

func TestStreamBasicOperations(t *testing.T) {
	t.Run("Of creates stream from values", func(t *testing.T) {
		s := Of(1, 2, 3)
		if s.Count() != 3 {
			t.Errorf("expected 3, got %d", s.Count())
		}
	})

	t.Run("FromSlice creates stream from slice", func(t *testing.T) {
		s := FromSlice([]int{1, 2, 3})
		if s.Count() != 3 {
			t.Errorf("expected 3, got %d", s.Count())
		}
	})

	t.Run("Empty creates empty stream", func(t *testing.T) {
		s := Empty[int]()
		if s.Count() != 0 {
			t.Errorf("expected 0, got %d", s.Count())
		}
	})

	t.Run("Generate creates stream from generator", func(t *testing.T) {
		s := Generate(5, func(i int) int { return i * 2 })
		collected := s.Collect()
		if len(collected) != 5 {
			t.Errorf("expected 5, got %d", len(collected))
		}
		if collected[2] != 4 {
			t.Errorf("expected 4, got %d", collected[2])
		}
	})
}

func TestStreamTransformations(t *testing.T) {
	t.Run("Map transforms elements", func(t *testing.T) {
		s := Of(1, 2, 3).Map(func(x int) int { return x * 2 })
		collected := s.Collect()
		if collected[0] != 2 || collected[1] != 4 || collected[2] != 6 {
			t.Error("unexpected values")
		}
	})

	t.Run("MapTo transforms to different type", func(t *testing.T) {
		s := Of(1, 2, 3)
		mapped := MapTo(s, func(x int) string { return string(rune('a' + x - 1)) })
		collected := mapped.Collect()
		if collected[0] != "a" || collected[1] != "b" || collected[2] != "c" {
			t.Error("unexpected values")
		}
	})

	t.Run("Filter keeps matching elements", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5).Filter(func(x int) bool { return x%2 == 0 })
		collected := s.Collect()
		if len(collected) != 2 || collected[0] != 2 || collected[1] != 4 {
			t.Error("unexpected values")
		}
	})

	t.Run("FlatMap transforms and flattens", func(t *testing.T) {
		s := Of(1, 2, 3)
		flattened := FlatMap(s, func(x int) Stream[int] {
			return Of(x, x*10)
		})
		collected := flattened.Collect()
		if len(collected) != 6 {
			t.Errorf("expected 6, got %d", len(collected))
		}
	})
}

func TestStreamTerminalOperations(t *testing.T) {
	t.Run("Reduce folds elements", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5)
		sum := s.Reduce(0, func(a, b int) int { return a + b })
		if sum != 15 {
			t.Errorf("expected 15, got %d", sum)
		}
	})

	t.Run("FindFirst returns first element", func(t *testing.T) {
		s := Of(1, 2, 3)
		first := s.FindFirst()
		if first.IsNone() || first.Unwrap() != 1 {
			t.Error("expected 1")
		}
	})

	t.Run("FindFirst returns None for empty", func(t *testing.T) {
		s := Empty[int]()
		first := s.FindFirst()
		if first.IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("Find returns matching element", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5)
		found := s.Find(func(x int) bool { return x > 3 })
		if found.IsNone() || found.Unwrap() != 4 {
			t.Error("expected 4")
		}
	})

	t.Run("AnyMatch returns true if any match", func(t *testing.T) {
		s := Of(1, 2, 3)
		if !s.AnyMatch(func(x int) bool { return x == 2 }) {
			t.Error("expected true")
		}
	})

	t.Run("AllMatch returns true if all match", func(t *testing.T) {
		s := Of(2, 4, 6)
		if !s.AllMatch(func(x int) bool { return x%2 == 0 }) {
			t.Error("expected true")
		}
	})

	t.Run("NoneMatch returns true if none match", func(t *testing.T) {
		s := Of(1, 3, 5)
		if !s.NoneMatch(func(x int) bool { return x%2 == 0 }) {
			t.Error("expected true")
		}
	})
}

func TestStreamAdvancedOperations(t *testing.T) {
	t.Run("Sorted sorts elements", func(t *testing.T) {
		s := Of(3, 1, 4, 1, 5).Sorted(func(a, b int) bool { return a < b })
		collected := s.Collect()
		if collected[0] != 1 || collected[4] != 5 {
			t.Error("unexpected order")
		}
	})

	t.Run("Distinct removes duplicates", func(t *testing.T) {
		s := Of(1, 2, 2, 3, 3, 3)
		distinct := Distinct(s)
		if distinct.Count() != 3 {
			t.Errorf("expected 3, got %d", distinct.Count())
		}
	})

	t.Run("Limit takes first n elements", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5).Limit(3)
		if s.Count() != 3 {
			t.Errorf("expected 3, got %d", s.Count())
		}
	})

	t.Run("Skip skips first n elements", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5).Skip(2)
		collected := s.Collect()
		if len(collected) != 3 || collected[0] != 3 {
			t.Error("unexpected values")
		}
	})

	t.Run("Partition splits elements", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5)
		even, odd := s.Partition(func(x int) bool { return x%2 == 0 })
		if len(even) != 2 || len(odd) != 3 {
			t.Error("unexpected partition")
		}
	})

	t.Run("Reverse reverses elements", func(t *testing.T) {
		s := Of(1, 2, 3).Reverse()
		collected := s.Collect()
		if collected[0] != 3 || collected[2] != 1 {
			t.Error("unexpected order")
		}
	})

	t.Run("Concat concatenates streams", func(t *testing.T) {
		s1 := Of(1, 2)
		s2 := Of(3, 4)
		concatenated := Concat(s1, s2)
		if concatenated.Count() != 4 {
			t.Errorf("expected 4, got %d", concatenated.Count())
		}
	})
}

func TestStreamForEach(t *testing.T) {
	sum := 0
	Of(1, 2, 3).ForEach(func(x int) {
		sum += x
	})
	if sum != 6 {
		t.Errorf("expected 6, got %d", sum)
	}
}

func TestStreamPeek(t *testing.T) {
	sum := 0
	s := Of(1, 2, 3).Peek(func(x int) {
		sum += x
	})
	collected := s.Collect()
	if sum != 6 {
		t.Errorf("expected 6, got %d", sum)
	}
	if len(collected) != 3 {
		t.Error("peek should not modify stream")
	}
}
