package result

import (
	"errors"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 8: Result Map Preserves Structure**
// **Validates: Requirements 11.6**
func TestResultMapPreservesStructure(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Test that Map on Ok returns Ok(fn(value))
	properties.Property("Map on Ok returns Ok(fn(value))", prop.ForAll(
		func(n int) bool {
			r := Ok(n)
			fn := func(x int) int { return x * 2 }
			mapped := Map(r, fn)
			return mapped.IsOk() && mapped.Unwrap() == fn(n)
		},
		gen.Int(),
	))

	// Test that Map on Err returns Err
	properties.Property("Map on Err returns Err", prop.ForAll(
		func(msg string) bool {
			err := errors.New(msg)
			r := Err[int](err)
			fn := func(x int) int { return x * 2 }
			mapped := Map(r, fn)
			return mapped.IsErr() && mapped.UnwrapErr() == err
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 9: Result FlatMap Monad Law**
// **Validates: Requirements 11.7**
func TestResultFlatMapMonadLaw(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Left identity: FlatMap(Ok(a), f) == f(a)
	properties.Property("left identity law", prop.ForAll(
		func(n int) bool {
			f := func(x int) Result[int] { return Ok(x * 2) }
			left := FlatMap(Ok(n), f)
			right := f(n)
			return left.IsOk() == right.IsOk() &&
				(!left.IsOk() || left.Unwrap() == right.Unwrap())
		},
		gen.Int(),
	))

	// Right identity: FlatMap(m, Ok) == m
	properties.Property("right identity law", prop.ForAll(
		func(n int) bool {
			m := Ok(n)
			result := FlatMap(m, func(x int) Result[int] { return Ok(x) })
			return result.IsOk() && result.Unwrap() == n
		},
		gen.Int(),
	))

	// Associativity: FlatMap(FlatMap(m, f), g) == FlatMap(m, x => FlatMap(f(x), g))
	properties.Property("associativity law", prop.ForAll(
		func(n int) bool {
			m := Ok(n)
			f := func(x int) Result[int] { return Ok(x + 1) }
			g := func(x int) Result[int] { return Ok(x * 2) }

			left := FlatMap(FlatMap(m, f), g)
			right := FlatMap(m, func(x int) Result[int] { return FlatMap(f(x), g) })

			return left.IsOk() == right.IsOk() &&
				(!left.IsOk() || left.Unwrap() == right.Unwrap())
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

func TestResultBasicOperations(t *testing.T) {
	t.Run("Ok creates successful result", func(t *testing.T) {
		r := Ok(42)
		if !r.IsOk() {
			t.Error("expected IsOk to be true")
		}
		if r.IsErr() {
			t.Error("expected IsErr to be false")
		}
		if r.Unwrap() != 42 {
			t.Errorf("expected 42, got %d", r.Unwrap())
		}
	})

	t.Run("Err creates failed result", func(t *testing.T) {
		err := errors.New("test error")
		r := Err[int](err)
		if r.IsOk() {
			t.Error("expected IsOk to be false")
		}
		if !r.IsErr() {
			t.Error("expected IsErr to be true")
		}
		if r.UnwrapErr() != err {
			t.Errorf("expected %v, got %v", err, r.UnwrapErr())
		}
	})

	t.Run("UnwrapOr returns default on error", func(t *testing.T) {
		r := Err[int](errors.New("error"))
		if r.UnwrapOr(100) != 100 {
			t.Error("expected default value")
		}
	})

	t.Run("UnwrapOr returns value on success", func(t *testing.T) {
		r := Ok(42)
		if r.UnwrapOr(100) != 42 {
			t.Error("expected actual value")
		}
	})
}

func TestTry(t *testing.T) {
	t.Run("Try wraps successful function", func(t *testing.T) {
		r := Try(func() (int, error) { return 42, nil })
		if !r.IsOk() || r.Unwrap() != 42 {
			t.Error("expected Ok(42)")
		}
	})

	t.Run("Try wraps failed function", func(t *testing.T) {
		err := errors.New("failed")
		r := Try(func() (int, error) { return 0, err })
		if !r.IsErr() || r.UnwrapErr() != err {
			t.Error("expected Err")
		}
	})
}

func TestCollect(t *testing.T) {
	t.Run("Collect all Ok returns Ok slice", func(t *testing.T) {
		results := []Result[int]{Ok(1), Ok(2), Ok(3)}
		collected := Collect(results)
		if !collected.IsOk() {
			t.Error("expected Ok")
		}
		vals := collected.Unwrap()
		if len(vals) != 3 || vals[0] != 1 || vals[1] != 2 || vals[2] != 3 {
			t.Errorf("unexpected values: %v", vals)
		}
	})

	t.Run("Collect with Err returns first Err", func(t *testing.T) {
		err := errors.New("error")
		results := []Result[int]{Ok(1), Err[int](err), Ok(3)}
		collected := Collect(results)
		if !collected.IsErr() {
			t.Error("expected Err")
		}
	})
}
