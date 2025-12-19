package either

import (
	"errors"
	"testing"
)

func TestEitherBasicOperations(t *testing.T) {
	t.Run("Left creates left value", func(t *testing.T) {
		e := Left[string, int]("error")
		if !e.IsLeft() || e.IsRight() {
			t.Error("expected Left")
		}
		if e.LeftValue() != "error" {
			t.Errorf("expected error, got %s", e.LeftValue())
		}
	})

	t.Run("Right creates right value", func(t *testing.T) {
		e := Right[string, int](42)
		if e.IsLeft() || !e.IsRight() {
			t.Error("expected Right")
		}
		if e.RightValue() != 42 {
			t.Errorf("expected 42, got %d", e.RightValue())
		}
	})

	t.Run("LeftValue panics on Right", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		e := Right[string, int](42)
		e.LeftValue()
	})

	t.Run("RightValue panics on Left", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		e := Left[string, int]("error")
		e.RightValue()
	})
}

func TestEitherDefaults(t *testing.T) {
	t.Run("LeftOr returns left value", func(t *testing.T) {
		e := Left[string, int]("error")
		if e.LeftOr("default") != "error" {
			t.Error("expected left value")
		}
	})

	t.Run("LeftOr returns default on Right", func(t *testing.T) {
		e := Right[string, int](42)
		if e.LeftOr("default") != "default" {
			t.Error("expected default")
		}
	})

	t.Run("RightOr returns right value", func(t *testing.T) {
		e := Right[string, int](42)
		if e.RightOr(0) != 42 {
			t.Error("expected right value")
		}
	})

	t.Run("RightOr returns default on Left", func(t *testing.T) {
		e := Left[string, int]("error")
		if e.RightOr(100) != 100 {
			t.Error("expected default")
		}
	})
}

func TestMapLeft(t *testing.T) {
	t.Run("maps left value", func(t *testing.T) {
		e := Left[int, string](10)
		mapped := MapLeft(e, func(x int) int { return x * 2 })
		if !mapped.IsLeft() || mapped.LeftValue() != 20 {
			t.Error("expected mapped left value")
		}
	})

	t.Run("preserves right value", func(t *testing.T) {
		e := Right[int, string]("hello")
		mapped := MapLeft(e, func(x int) int { return x * 2 })
		if !mapped.IsRight() || mapped.RightValue() != "hello" {
			t.Error("expected preserved right value")
		}
	})
}

func TestMapRight(t *testing.T) {
	t.Run("maps right value", func(t *testing.T) {
		e := Right[string, int](10)
		mapped := MapRight(e, func(x int) int { return x * 2 })
		if !mapped.IsRight() || mapped.RightValue() != 20 {
			t.Error("expected mapped right value")
		}
	})

	t.Run("preserves left value", func(t *testing.T) {
		e := Left[string, int]("error")
		mapped := MapRight(e, func(x int) int { return x * 2 })
		if !mapped.IsLeft() || mapped.LeftValue() != "error" {
			t.Error("expected preserved left value")
		}
	})
}

func TestFold(t *testing.T) {
	t.Run("folds left value", func(t *testing.T) {
		e := Left[int, string](10)
		result := Fold(e,
			func(l int) string { return "left" },
			func(r string) string { return "right" },
		)
		if result != "left" {
			t.Errorf("expected left, got %s", result)
		}
	})

	t.Run("folds right value", func(t *testing.T) {
		e := Right[int, string]("hello")
		result := Fold(e,
			func(l int) string { return "left" },
			func(r string) string { return "right" },
		)
		if result != "right" {
			t.Errorf("expected right, got %s", result)
		}
	})
}

func TestFlatMap(t *testing.T) {
	t.Run("FlatMapRight chains operations", func(t *testing.T) {
		e := Right[string, int](10)
		result := FlatMapRight(e, func(x int) Either[string, int] {
			if x > 5 {
				return Right[string, int](x * 2)
			}
			return Left[string, int]("too small")
		})
		if !result.IsRight() || result.RightValue() != 20 {
			t.Error("expected chained right value")
		}
	})

	t.Run("FlatMapRight preserves left", func(t *testing.T) {
		e := Left[string, int]("error")
		result := FlatMapRight(e, func(x int) Either[string, int] {
			return Right[string, int](x * 2)
		})
		if !result.IsLeft() || result.LeftValue() != "error" {
			t.Error("expected preserved left value")
		}
	})
}

func TestSwap(t *testing.T) {
	t.Run("swaps left to right", func(t *testing.T) {
		e := Left[int, string](42)
		swapped := e.Swap()
		if !swapped.IsRight() || swapped.RightValue() != 42 {
			t.Error("expected swapped value")
		}
	})

	t.Run("swaps right to left", func(t *testing.T) {
		e := Right[int, string]("hello")
		swapped := e.Swap()
		if !swapped.IsLeft() || swapped.LeftValue() != "hello" {
			t.Error("expected swapped value")
		}
	})
}

func TestToSlice(t *testing.T) {
	t.Run("returns slice with right value", func(t *testing.T) {
		e := Right[string, int](42)
		slice := e.ToSlice()
		if len(slice) != 1 || slice[0] != 42 {
			t.Error("expected slice with value")
		}
	})

	t.Run("returns empty slice for left", func(t *testing.T) {
		e := Left[string, int]("error")
		slice := e.ToSlice()
		if len(slice) != 0 {
			t.Error("expected empty slice")
		}
	})
}

func TestGetOrElse(t *testing.T) {
	t.Run("returns right value", func(t *testing.T) {
		e := Right[string, int](42)
		result := e.GetOrElse(func(s string) int { return 0 })
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("computes default from left", func(t *testing.T) {
		e := Left[string, int]("error")
		result := e.GetOrElse(func(s string) int { return len(s) })
		if result != 5 {
			t.Errorf("expected 5, got %d", result)
		}
	})
}

func TestFromError(t *testing.T) {
	t.Run("creates Right on nil error", func(t *testing.T) {
		e := FromError(42, nil)
		if !e.IsRight() || e.RightValue() != 42 {
			t.Error("expected Right with value")
		}
	})

	t.Run("creates Left on error", func(t *testing.T) {
		err := errors.New("failed")
		e := FromError(0, err)
		if !e.IsLeft() || e.LeftValue() != err {
			t.Error("expected Left with error")
		}
	})
}

func TestToError(t *testing.T) {
	t.Run("returns value and nil on Right", func(t *testing.T) {
		e := Right[error, int](42)
		val, err := ToError(e)
		if err != nil || val != 42 {
			t.Error("expected value and nil error")
		}
	})

	t.Run("returns zero and error on Left", func(t *testing.T) {
		origErr := errors.New("failed")
		e := Left[error, int](origErr)
		val, err := ToError(e)
		if err != origErr || val != 0 {
			t.Error("expected zero and error")
		}
	})
}
