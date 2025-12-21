// Feature: go-libs-state-of-art-2025, Property 5: Functor Laws
// Validates: Requirements 8.3, 8.4
package functional_test

import (
	"testing"

	"github.com/authcorp/libs/go/src/functional"
	"pgregory.net/rapid"
)

// TestOptionFunctorIdentity verifies Option.Map(id) == Option for identity function.
func TestOptionFunctorIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasSome := rapid.Bool().Draw(t, "hasSome")
		value := rapid.Int().Draw(t, "value")

		var opt functional.Option[int]
		if hasSome {
			opt = functional.Some(value)
		} else {
			opt = functional.None[int]()
		}

		// Map with identity
		mapped := opt.Map(functional.IdentityFunc[int]).(functional.Option[int])

		// Should be equal
		if opt.IsSome() != mapped.IsSome() {
			t.Fatal("identity law violated: presence changed")
		}
		if opt.IsSome() && opt.Unwrap() != mapped.Unwrap() {
			t.Fatalf("identity law violated: %d != %d", opt.Unwrap(), mapped.Unwrap())
		}
	})
}

// TestOptionFunctorComposition verifies Option.Map(f).Map(g) == Option.Map(f∘g).
func TestOptionFunctorComposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		addend := rapid.IntRange(1, 100).Draw(t, "addend")
		multiplier := rapid.IntRange(1, 10).Draw(t, "multiplier")

		opt := functional.Some(value)

		f := func(x int) int { return x + addend }
		g := func(x int) int { return x * multiplier }

		// Map(f).Map(g)
		result1 := opt.Map(f).(functional.Option[int]).Map(g).(functional.Option[int])

		// Map(f∘g)
		composed := functional.ComposeFunc(f, g)
		result2 := opt.Map(composed).(functional.Option[int])

		if result1.Unwrap() != result2.Unwrap() {
			t.Fatalf("composition law violated: %d != %d", result1.Unwrap(), result2.Unwrap())
		}
	})
}

// TestResultFunctorIdentity verifies Result.Map(id) == Result for identity function.
func TestResultFunctorIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		isOk := rapid.Bool().Draw(t, "isOk")
		value := rapid.Int().Draw(t, "value")

		var res functional.Result[int]
		if isOk {
			res = functional.Ok(value)
		} else {
			res = functional.Err[int](functional.NewError("test error"))
		}

		// Map with identity
		mapped := res.Map(functional.IdentityFunc[int]).(functional.Result[int])

		// Should be equal
		if res.IsOk() != mapped.IsOk() {
			t.Fatal("identity law violated: ok status changed")
		}
		if res.IsOk() && res.Unwrap() != mapped.Unwrap() {
			t.Fatalf("identity law violated: %d != %d", res.Unwrap(), mapped.Unwrap())
		}
	})
}

// TestResultFunctorComposition verifies Result.Map(f).Map(g) == Result.Map(f∘g).
func TestResultFunctorComposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		addend := rapid.IntRange(1, 100).Draw(t, "addend")
		multiplier := rapid.IntRange(1, 10).Draw(t, "multiplier")

		res := functional.Ok(value)

		f := func(x int) int { return x + addend }
		g := func(x int) int { return x * multiplier }

		// Map(f).Map(g)
		result1 := res.Map(f).(functional.Result[int]).Map(g).(functional.Result[int])

		// Map(f∘g)
		composed := functional.ComposeFunc(f, g)
		result2 := res.Map(composed).(functional.Result[int])

		if result1.Unwrap() != result2.Unwrap() {
			t.Fatalf("composition law violated: %d != %d", result1.Unwrap(), result2.Unwrap())
		}
	})
}

// TestOptionToResultConversion verifies OptionToResult preserves semantics.
func TestOptionToResultConversion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasSome := rapid.Bool().Draw(t, "hasSome")
		value := rapid.Int().Draw(t, "value")

		var opt functional.Option[int]
		if hasSome {
			opt = functional.Some(value)
		} else {
			opt = functional.None[int]()
		}

		err := functional.NewError("none error")
		result := functional.OptionToResult(opt, err)

		if hasSome {
			if !result.IsOk() {
				t.Fatal("Some should convert to Ok")
			}
			if result.Unwrap() != value {
				t.Fatalf("value mismatch: %d != %d", result.Unwrap(), value)
			}
		} else {
			if result.IsOk() {
				t.Fatal("None should convert to Err")
			}
		}
	})
}

// TestResultToOptionConversion verifies ResultToOption preserves semantics.
func TestResultToOptionConversion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		isOk := rapid.Bool().Draw(t, "isOk")
		value := rapid.Int().Draw(t, "value")

		var res functional.Result[int]
		if isOk {
			res = functional.Ok(value)
		} else {
			res = functional.Err[int](functional.NewError("test error"))
		}

		opt := functional.ResultToOption(res)

		if isOk {
			if opt.IsNone() {
				t.Fatal("Ok should convert to Some")
			}
			if opt.Unwrap() != value {
				t.Fatalf("value mismatch: %d != %d", opt.Unwrap(), value)
			}
		} else {
			if opt.IsSome() {
				t.Fatal("Err should convert to None")
			}
		}
	})
}

// TestEitherToResultConversion verifies EitherToResult preserves semantics.
func TestEitherToResultConversion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		isRight := rapid.Bool().Draw(t, "isRight")
		value := rapid.Int().Draw(t, "value")

		var either functional.Either[error, int]
		if isRight {
			either = functional.Right[error, int](value)
		} else {
			either = functional.Left[error, int](functional.NewError("left error"))
		}

		result := functional.EitherToResult(either)

		if isRight {
			if !result.IsOk() {
				t.Fatal("Right should convert to Ok")
			}
			if result.Unwrap() != value {
				t.Fatalf("value mismatch: %d != %d", result.Unwrap(), value)
			}
		} else {
			if result.IsOk() {
				t.Fatal("Left should convert to Err")
			}
		}
	})
}

// TestResultToEitherConversion verifies ResultToEither preserves semantics.
func TestResultToEitherConversion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		isOk := rapid.Bool().Draw(t, "isOk")
		value := rapid.Int().Draw(t, "value")

		var res functional.Result[int]
		if isOk {
			res = functional.Ok(value)
		} else {
			res = functional.Err[int](functional.NewError("test error"))
		}

		either := functional.ResultToEither(res)

		if isOk {
			if !either.IsRight() {
				t.Fatal("Ok should convert to Right")
			}
			if either.RightValue() != value {
				t.Fatalf("value mismatch: %d != %d", either.RightValue(), value)
			}
		} else {
			if either.IsRight() {
				t.Fatal("Err should convert to Left")
			}
		}
	})
}

// TestMapOptionTransformation verifies MapOption transforms values correctly.
func TestMapOptionTransformation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		multiplier := rapid.IntRange(1, 10).Draw(t, "multiplier")

		opt := functional.Some(value)
		mapped := functional.MapOption(opt, func(x int) int { return x * multiplier })

		expected := value * multiplier
		if mapped.Unwrap() != expected {
			t.Fatalf("MapOption: got %d, expected %d", mapped.Unwrap(), expected)
		}
	})
}

// TestMapResultTransformation verifies MapResult transforms values correctly.
func TestMapResultTransformation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		addend := rapid.IntRange(1, 100).Draw(t, "addend")

		res := functional.Ok(value)
		mapped := functional.MapResult(res, func(x int) int { return x + addend })

		expected := value + addend
		if mapped.Unwrap() != expected {
			t.Fatalf("MapResult: got %d, expected %d", mapped.Unwrap(), expected)
		}
	})
}

// TestFlatMapOptionChaining verifies FlatMapOption chains correctly.
func TestFlatMapOptionChaining(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(1, 100).Draw(t, "value")
		divisor := rapid.IntRange(1, 10).Draw(t, "divisor")

		opt := functional.Some(value)

		// FlatMap that may return None
		result := functional.FlatMapOption(opt, func(x int) functional.Option[int] {
			if x%divisor == 0 {
				return functional.Some(x / divisor)
			}
			return functional.None[int]()
		})

		if value%divisor == 0 {
			if result.IsNone() {
				t.Fatal("should be Some when divisible")
			}
			if result.Unwrap() != value/divisor {
				t.Fatalf("wrong value: %d != %d", result.Unwrap(), value/divisor)
			}
		} else {
			if result.IsSome() {
				t.Fatal("should be None when not divisible")
			}
		}
	})
}

// TestFlatMapResultChaining verifies FlatMapResult chains correctly.
func TestFlatMapResultChaining(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(0, 100).Draw(t, "value")

		res := functional.Ok(value)

		// FlatMap that may return Err
		result := functional.FlatMapResult(res, func(x int) functional.Result[int] {
			if x > 50 {
				return functional.Err[int](functional.NewError("too large"))
			}
			return functional.Ok(x * 2)
		})

		if value > 50 {
			if result.IsOk() {
				t.Fatal("should be Err when value > 50")
			}
		} else {
			if result.IsErr() {
				t.Fatal("should be Ok when value <= 50")
			}
			if result.Unwrap() != value*2 {
				t.Fatalf("wrong value: %d != %d", result.Unwrap(), value*2)
			}
		}
	})
}

// TestNoneFunctorIdentity verifies None.Map(id) == None.
func TestNoneFunctorIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		opt := functional.None[int]()
		mapped := opt.Map(functional.IdentityFunc[int]).(functional.Option[int])

		if mapped.IsSome() {
			t.Fatal("None.Map(id) should be None")
		}
	})
}

// TestErrFunctorIdentity verifies Err.Map(id) == Err.
func TestErrFunctorIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		errMsg := rapid.String().Draw(t, "errMsg")
		res := functional.Err[int](functional.NewError(errMsg))
		mapped := res.Map(functional.IdentityFunc[int]).(functional.Result[int])

		if mapped.IsOk() {
			t.Fatal("Err.Map(id) should be Err")
		}
	})
}
