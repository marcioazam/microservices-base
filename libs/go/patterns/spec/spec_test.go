package spec

import "testing"

func TestSpec(t *testing.T) {
	t.Run("IsSatisfiedBy", func(t *testing.T) {
		s := New(func(x int) bool { return x > 0 })
		if !s.IsSatisfiedBy(5) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(-5) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("And", func(t *testing.T) {
		positive := New(func(x int) bool { return x > 0 })
		even := New(func(x int) bool { return x%2 == 0 })
		both := positive.And(even)

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
		positive := New(func(x int) bool { return x > 0 })
		even := New(func(x int) bool { return x%2 == 0 })
		either := positive.Or(even)

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
		positive := New(func(x int) bool { return x > 0 })
		notPositive := positive.Not()

		if notPositive.IsSatisfiedBy(5) {
			t.Error("expected not satisfied")
		}
		if !notPositive.IsSatisfiedBy(-5) {
			t.Error("expected satisfied")
		}
	})

	t.Run("Filter", func(t *testing.T) {
		positive := New(func(x int) bool { return x > 0 })
		items := []int{-2, -1, 0, 1, 2}
		result := positive.Filter(items)
		if len(result) != 2 || result[0] != 1 || result[1] != 2 {
			t.Error("unexpected result")
		}
	})

	t.Run("FindFirst", func(t *testing.T) {
		even := New(func(x int) bool { return x%2 == 0 })
		items := []int{1, 3, 4, 5, 6}
		result := even.FindFirst(items)
		if result.IsNone() || result.Unwrap() != 4 {
			t.Error("expected 4")
		}
	})

	t.Run("Count", func(t *testing.T) {
		positive := New(func(x int) bool { return x > 0 })
		items := []int{-2, -1, 0, 1, 2}
		if positive.Count(items) != 2 {
			t.Error("expected 2")
		}
	})

	t.Run("Any", func(t *testing.T) {
		positive := New(func(x int) bool { return x > 0 })
		if !positive.Any([]int{-1, 0, 1}) {
			t.Error("expected true")
		}
		if positive.Any([]int{-1, 0, -2}) {
			t.Error("expected false")
		}
	})

	t.Run("All", func(t *testing.T) {
		positive := New(func(x int) bool { return x > 0 })
		if !positive.All([]int{1, 2, 3}) {
			t.Error("expected true")
		}
		if positive.All([]int{1, 0, 3}) {
			t.Error("expected false")
		}
	})

	t.Run("None", func(t *testing.T) {
		positive := New(func(x int) bool { return x > 0 })
		if !positive.None([]int{-1, -2, -3}) {
			t.Error("expected true")
		}
		if positive.None([]int{-1, 0, 1}) {
			t.Error("expected false")
		}
	})
}

func TestBuiltInSpecs(t *testing.T) {
	t.Run("True and False", func(t *testing.T) {
		if !True[int]().IsSatisfiedBy(42) {
			t.Error("True should always be satisfied")
		}
		if False[int]().IsSatisfiedBy(42) {
			t.Error("False should never be satisfied")
		}
	})

	t.Run("Equals", func(t *testing.T) {
		s := Equals(42)
		if !s.IsSatisfiedBy(42) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(43) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("NotEquals", func(t *testing.T) {
		s := NotEquals(42)
		if s.IsSatisfiedBy(42) {
			t.Error("expected not satisfied")
		}
		if !s.IsSatisfiedBy(43) {
			t.Error("expected satisfied")
		}
	})

	t.Run("GreaterThan", func(t *testing.T) {
		s := GreaterThan(10)
		if !s.IsSatisfiedBy(15) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(5) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("LessThan", func(t *testing.T) {
		s := LessThan(10)
		if !s.IsSatisfiedBy(5) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(15) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("Between", func(t *testing.T) {
		s := Between(1, 10)
		if !s.IsSatisfiedBy(5) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(15) {
			t.Error("expected not satisfied")
		}
	})

	t.Run("In", func(t *testing.T) {
		s := In(1, 2, 3)
		if !s.IsSatisfiedBy(2) {
			t.Error("expected satisfied")
		}
		if s.IsSatisfiedBy(4) {
			t.Error("expected not satisfied")
		}
	})
}
