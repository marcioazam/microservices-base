package set

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 15: Set Union Contains All**
// **Validates: Requirements 34.8**
func TestSetUnionContainsAll(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Union contains all elements from both sets", prop.ForAll(
		func(a, b []int) bool {
			setA := FromSlice(a)
			setB := FromSlice(b)
			union := setA.Union(setB)

			// Check all elements from A are in union
			for _, elem := range a {
				if !union.Contains(elem) {
					return false
				}
			}
			// Check all elements from B are in union
			for _, elem := range b {
				if !union.Contains(elem) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Int()),
		gen.SliceOf(gen.Int()),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 16: Set Intersection Contains Common**
// **Validates: Requirements 34.9**
func TestSetIntersectionContainsCommon(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Intersection contains only elements in both sets", prop.ForAll(
		func(a, b []int) bool {
			setA := FromSlice(a)
			setB := FromSlice(b)
			intersection := setA.Intersection(setB)

			// Check all elements in intersection are in both A and B
			for _, elem := range intersection.ToSlice() {
				if !setA.Contains(elem) || !setB.Contains(elem) {
					return false
				}
			}
			// Check all common elements are in intersection
			for _, elem := range a {
				if setB.Contains(elem) && !intersection.Contains(elem) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Int()),
		gen.SliceOf(gen.Int()),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 17: Set Difference Excludes Second**
// **Validates: Requirements 34.10**
func TestSetDifferenceExcludesSecond(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Difference contains elements in A but not in B", prop.ForAll(
		func(a, b []int) bool {
			setA := FromSlice(a)
			setB := FromSlice(b)
			diff := setA.Difference(setB)

			// Check all elements in diff are in A but not in B
			for _, elem := range diff.ToSlice() {
				if !setA.Contains(elem) || setB.Contains(elem) {
					return false
				}
			}
			// Check all elements in A that are not in B are in diff
			for _, elem := range a {
				if !setB.Contains(elem) && !diff.Contains(elem) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Int()),
		gen.SliceOf(gen.Int()),
	))

	properties.TestingRun(t)
}

func TestSetBasicOperations(t *testing.T) {
	t.Run("New creates empty set", func(t *testing.T) {
		s := New[int]()
		if !s.IsEmpty() {
			t.Error("expected empty set")
		}
	})

	t.Run("Of creates set with elements", func(t *testing.T) {
		s := Of(1, 2, 3)
		if s.Len() != 3 {
			t.Errorf("expected 3 elements, got %d", s.Len())
		}
	})

	t.Run("Add adds element", func(t *testing.T) {
		s := New[int]()
		added := s.Add(1)
		if !added || !s.Contains(1) {
			t.Error("expected element to be added")
		}
	})

	t.Run("Add returns false for duplicate", func(t *testing.T) {
		s := Of(1)
		added := s.Add(1)
		if added {
			t.Error("expected false for duplicate")
		}
	})

	t.Run("Remove removes element", func(t *testing.T) {
		s := Of(1, 2, 3)
		removed := s.Remove(2)
		if !removed || s.Contains(2) {
			t.Error("expected element to be removed")
		}
	})

	t.Run("Remove returns false for non-existent", func(t *testing.T) {
		s := Of(1)
		removed := s.Remove(2)
		if removed {
			t.Error("expected false for non-existent")
		}
	})

	t.Run("Clear removes all elements", func(t *testing.T) {
		s := Of(1, 2, 3)
		s.Clear()
		if !s.IsEmpty() {
			t.Error("expected empty set after clear")
		}
	})
}

func TestSetOperations(t *testing.T) {
	t.Run("Union combines sets", func(t *testing.T) {
		a := Of(1, 2, 3)
		b := Of(3, 4, 5)
		union := a.Union(b)
		if union.Len() != 5 {
			t.Errorf("expected 5 elements, got %d", union.Len())
		}
	})

	t.Run("Intersection finds common elements", func(t *testing.T) {
		a := Of(1, 2, 3)
		b := Of(2, 3, 4)
		intersection := a.Intersection(b)
		if intersection.Len() != 2 {
			t.Errorf("expected 2 elements, got %d", intersection.Len())
		}
	})

	t.Run("Difference finds elements in first but not second", func(t *testing.T) {
		a := Of(1, 2, 3)
		b := Of(2, 3, 4)
		diff := a.Difference(b)
		if diff.Len() != 1 || !diff.Contains(1) {
			t.Error("expected {1}")
		}
	})

	t.Run("SymmetricDifference finds elements in either but not both", func(t *testing.T) {
		a := Of(1, 2, 3)
		b := Of(2, 3, 4)
		symDiff := a.SymmetricDifference(b)
		if symDiff.Len() != 2 || !symDiff.Contains(1) || !symDiff.Contains(4) {
			t.Error("expected {1, 4}")
		}
	})
}

func TestSetRelations(t *testing.T) {
	t.Run("IsSubset returns true for subset", func(t *testing.T) {
		a := Of(1, 2)
		b := Of(1, 2, 3)
		if !a.IsSubset(b) {
			t.Error("expected a to be subset of b")
		}
	})

	t.Run("IsSuperset returns true for superset", func(t *testing.T) {
		a := Of(1, 2, 3)
		b := Of(1, 2)
		if !a.IsSuperset(b) {
			t.Error("expected a to be superset of b")
		}
	})

	t.Run("Equal returns true for equal sets", func(t *testing.T) {
		a := Of(1, 2, 3)
		b := Of(3, 2, 1)
		if !a.Equal(b) {
			t.Error("expected sets to be equal")
		}
	})
}

func TestSetCloneAndFilter(t *testing.T) {
	t.Run("Clone creates independent copy", func(t *testing.T) {
		a := Of(1, 2, 3)
		b := a.Clone()
		a.Add(4)
		if b.Contains(4) {
			t.Error("clone should be independent")
		}
	})

	t.Run("Filter returns matching elements", func(t *testing.T) {
		s := Of(1, 2, 3, 4, 5)
		filtered := s.Filter(func(x int) bool { return x > 2 })
		if filtered.Len() != 3 {
			t.Errorf("expected 3 elements, got %d", filtered.Len())
		}
	})
}

func TestSetMap(t *testing.T) {
	s := Of(1, 2, 3)
	mapped := Map(s, func(x int) string {
		return string(rune('a' + x - 1))
	})
	if mapped.Len() != 3 {
		t.Errorf("expected 3 elements, got %d", mapped.Len())
	}
	if !mapped.Contains("a") || !mapped.Contains("b") || !mapped.Contains("c") {
		t.Error("expected {a, b, c}")
	}
}
