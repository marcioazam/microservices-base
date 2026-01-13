// Package types provides property-based tests for Option type.
package types

import (
	"errors"
	"testing"

	"github.com/auth-platform/sdk-go/src/types"
	"pgregory.net/rapid"
)

// TestProperty_OptionSomeNoneExclusivity verifies that an Option is either Some or None, never both.
func TestProperty_OptionSomeNoneExclusivity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		opt := types.Some(value)

		// Some is exclusive
		if opt.IsSome() && opt.IsNone() {
			t.Fatal("Option cannot be both Some and None")
		}
		if !opt.IsSome() && !opt.IsNone() {
			t.Fatal("Option must be either Some or None")
		}
	})
}

// TestProperty_OptionNoneExclusivity verifies None behavior.
func TestProperty_OptionNoneExclusivity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		opt := types.None[int]()

		if opt.IsSome() && opt.IsNone() {
			t.Fatal("Option cannot be both Some and None")
		}
		if !opt.IsSome() && !opt.IsNone() {
			t.Fatal("Option must be either Some or None")
		}
		if !opt.IsNone() {
			t.Fatal("None() should create a None Option")
		}
	})
}

// TestProperty_OptionUnwrapOrDefault verifies UnwrapOr returns default for None.
func TestProperty_OptionUnwrapOrDefault(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		defaultVal := rapid.Int().Draw(t, "default")
		opt := types.None[int]()

		result := opt.UnwrapOr(defaultVal)
		if result != defaultVal {
			t.Fatalf("UnwrapOr on None should return default, got %d, want %d", result, defaultVal)
		}
	})
}

// TestProperty_OptionUnwrapOrValue verifies UnwrapOr returns value for Some.
func TestProperty_OptionUnwrapOrValue(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		defaultVal := rapid.Int().Draw(t, "default")
		opt := types.Some(value)

		result := opt.UnwrapOr(defaultVal)
		if result != value {
			t.Fatalf("UnwrapOr on Some should return value, got %d, want %d", result, value)
		}
	})
}

// TestProperty_MapOptionIdentity verifies MapOption with identity preserves value.
func TestProperty_MapOptionIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		opt := types.Some(value)

		mapped := types.MapOption(opt, func(v int) int { return v })
		if !mapped.IsSome() {
			t.Fatal("MapOption with identity should preserve Some")
		}
		if mapped.Unwrap() != value {
			t.Fatalf("MapOption with identity should preserve value, got %d, want %d", mapped.Unwrap(), value)
		}
	})
}

// TestProperty_MapOptionNone verifies MapOption on None returns None.
func TestProperty_MapOptionNone(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		opt := types.None[int]()

		mapped := types.MapOption(opt, func(v int) int { return v * 2 })
		if !mapped.IsNone() {
			t.Fatal("MapOption on None should return None")
		}
	})
}

// TestProperty_FlatMapOptionChaining verifies FlatMapOption chains correctly.
func TestProperty_FlatMapOptionChaining(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(1, 1000).Draw(t, "value")
		opt := types.Some(value)

		// Chain that doubles the value
		result := types.FlatMapOption(opt, func(v int) types.Option[int] {
			return types.Some(v * 2)
		})

		if !result.IsSome() {
			t.Fatal("FlatMapOption should preserve Some when function returns Some")
		}
		if result.Unwrap() != value*2 {
			t.Fatalf("FlatMapOption should apply function, got %d, want %d", result.Unwrap(), value*2)
		}
	})
}

// TestProperty_FlatMapOptionNoneShortCircuit verifies FlatMapOption short-circuits on None.
func TestProperty_FlatMapOptionNoneShortCircuit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		opt := types.None[int]()
		called := false

		result := types.FlatMapOption(opt, func(v int) types.Option[int] {
			called = true
			return types.Some(v * 2)
		})

		if called {
			t.Fatal("FlatMapOption should not call function on None")
		}
		if !result.IsNone() {
			t.Fatal("FlatMapOption on None should return None")
		}
	})
}

// TestProperty_FilterPreservesMatching verifies Filter keeps matching values.
func TestProperty_FilterPreservesMatching(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(1, 1000).Draw(t, "value")
		opt := types.Some(value)

		// Filter that always matches
		result := types.Filter(opt, func(v int) bool { return true })
		if !result.IsSome() {
			t.Fatal("Filter with true predicate should preserve Some")
		}
		if result.Unwrap() != value {
			t.Fatalf("Filter should preserve value, got %d, want %d", result.Unwrap(), value)
		}
	})
}

// TestProperty_FilterRemovesNonMatching verifies Filter removes non-matching values.
func TestProperty_FilterRemovesNonMatching(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		opt := types.Some(value)

		// Filter that never matches
		result := types.Filter(opt, func(v int) bool { return false })
		if !result.IsNone() {
			t.Fatal("Filter with false predicate should return None")
		}
	})
}

// TestProperty_OkOrConvertsToResult verifies OkOr converts Option to Result correctly.
func TestProperty_OkOrConvertsToResult(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		errMsg := rapid.String().Draw(t, "errMsg")
		testErr := errors.New(errMsg)

		// Some converts to Ok
		someOpt := types.Some(value)
		okResult := types.OkOr(someOpt, testErr)
		if !okResult.IsOk() {
			t.Fatal("OkOr on Some should return Ok Result")
		}
		if okResult.Unwrap() != value {
			t.Fatalf("OkOr should preserve value, got %d, want %d", okResult.Unwrap(), value)
		}

		// None converts to Err
		noneOpt := types.None[int]()
		errResult := types.OkOr(noneOpt, testErr)
		if !errResult.IsErr() {
			t.Fatal("OkOr on None should return Err Result")
		}
		if errResult.Error() != testErr {
			t.Fatal("OkOr should use provided error")
		}
	})
}

// TestProperty_ToOptionConvertsFromResult verifies ToOption converts Result to Option.
func TestProperty_ToOptionConvertsFromResult(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")

		// Ok converts to Some
		okResult := types.Ok(value)
		someOpt := types.ToOption(okResult)
		if !someOpt.IsSome() {
			t.Fatal("ToOption on Ok should return Some")
		}
		if someOpt.Unwrap() != value {
			t.Fatalf("ToOption should preserve value, got %d, want %d", someOpt.Unwrap(), value)
		}

		// Err converts to None
		errResult := types.Err[int](errors.New("test"))
		noneOpt := types.ToOption(errResult)
		if !noneOpt.IsNone() {
			t.Fatal("ToOption on Err should return None")
		}
	})
}

// TestProperty_OptionValueMethod verifies Value() returns correct tuple.
func TestProperty_OptionValueMethod(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")

		// Some returns (value, true)
		someOpt := types.Some(value)
		v, ok := someOpt.Value()
		if !ok {
			t.Fatal("Value() on Some should return true")
		}
		if v != value {
			t.Fatalf("Value() should return correct value, got %d, want %d", v, value)
		}

		// None returns (zero, false)
		noneOpt := types.None[int]()
		_, ok = noneOpt.Value()
		if ok {
			t.Fatal("Value() on None should return false")
		}
	})
}
