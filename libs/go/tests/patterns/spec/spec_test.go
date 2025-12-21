package spec

import (
	"testing"

	"github.com/authcorp/libs/go/src/patterns"
)

func TestSpec(t *testing.T) {
	t.Run("IsSatisfiedBy", func(t *testing.T) {
		s := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		if !s.IsSatisfiedBy(5) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(-5) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("And", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		even := patterns.SpecFunc[int](func(x int) bool { return x%2 == 0 })
		both := patterns.And[int](positive, even)

		if !both.IsSatisfiedBy(4) {
			t.Error("expected satisfied")
		}
		if both.IsSatisfiedBy(3) {
			t.Error("expected not satisfied")
		}
		if both.IsSatisfiedBy(-4) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("Or", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		even := patterns.SpecFunc[int](func(x int) bool { return x%2 == 0 })
		either := patterns.Or[int](positive, even)

		if !either.IsSatisfiedBy(3) {
			t.Error("expected satisfied")
		}
		if !either.IsSatisfiedBy(-4) {
			t.Error("expected satisfied")
		}
		if either.IsSatisfiedBy(-3) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("Not", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		notPositive := patterns.Not[int](positive)

		if notPositive.IsSatisfiedBy(5) {
			t.Error("expected not satisfied")
		}
		if !notPositive.IsSatisfiedBy(-5) {
			t.Error("expected satisfied")
		}
	})

	t.Run("Filter", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		items := []int{-2, -1, 0, 1, 2}
		result := patterns.Filter(items, positive)
		if len(result) != 2 || result[0] != 1 || result[1] != 2 {
			t.Error("unexpected result")
		}
	})

	t.Run("FindFirst", func(t *testing.T) {
		even := patterns.SpecFunc[int](func(x int) bool { return x%2 == 0 })
		items := []int{1, 3, 4, 5, 6}
		result, found := patterns.FindFirst(items, even)
		if !found || result != 4 {
			t.Error("expected 4")
		}
	})

	t.Run("Count", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		items := []int{-2, -1, 0, 1, 2}
		if patterns.Count(items, positive) != 2 {
			t.Error("expected 2")
		}
	})

	t.Run("Any", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		anySpec := patterns.Any[int](positive)
		if !anySpec.IsSatisfiedBy(1) {
			t.Error("expected true")
		}
	})

	t.Run("All", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		allSpec := patterns.All[int](positive)
		if !allSpec.IsSatisfiedBy(1) {
			t.Error("expected true")
		}
	})

	t.Run("None", func(t *testing.T) {
		positive := patterns.SpecFunc[int](func(x int) bool { return x > 0 })
		noneSpec := patterns.None[int](positive)
		if !noneSpec.IsSatisfiedBy(-1) {
			t.Error("expected true")
		}
	})
}

func TestBuiltInSpecs(t *testing.T) {
	t.Run("True and False", func(t *testing.T) {
		if !patterns.True[int]().IsSatisfiedBy(42) {
			t.Error("True should always be satisfied")
		}
		if patterns.False[int]().IsSatisfiedBy(42) {
			t.Error("False should never be satisfied")
		}
	})

	t.Run("Equals", func(t *testing.T) {
		s := patterns.Equals(42)
		if !s.IsSatisfiedBy(42) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(43) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("NotEquals", func(t *testing.T) {
		s := patterns.NotEquals(42)
		if s.IsSatisfiedBy(42) {
			t.Error("expected not satisfied")
		}
		if !s.IsSatisfiedBy(43) {
			t.Error("expected satisfied")
		}
	})

	t.Run("GreaterThan", func(t *testing.T) {
		s := patterns.GreaterThan(10)
		if !s.IsSatisfiedBy(15) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(5) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("LessThan", func(t *testing.T) {
		s := patterns.LessThan(10)
		if !s.IsSatisfiedBy(5) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(15) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("Between", func(t *testing.T) {
		s := patterns.Between(1, 10)
		if !s.IsSatisfiedBy(5) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(15) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("In", func(t *testing.T) {
		s := patterns.In(1, 2, 3)
		if !s.IsSatisfiedBy(2) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(4) {
			t.Error("expected not satisfied")
		}
	})
}
