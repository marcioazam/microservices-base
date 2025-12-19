package functional_test

import (
	"testing"

	"github.com/authcorp/libs/go/src/functional"
	"pgregory.net/rapid"
)

// Property 1: Either-Result Round Trip
// For any Either[error, T], converting to Result[T] and back preserves the value.
func TestProperty_EitherResultRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random int value
		value := rapid.Int().Draw(t, "value")
		
		// Test Right case: Either -> Result -> Either
		rightEither := functional.Right[error](value)
		result := functional.EitherToResult(rightEither)
		backToEither := functional.ResultToEither(result)
		
		// Verify round trip preserves value
		if !backToEither.IsRight() {
			t.Fatalf("Expected Right after round trip, got Left")
		}
		if backToEither.RightValue() != value {
			t.Fatalf("Value changed: expected %d, got %d", value, backToEither.RightValue())
		}
	})
}

// Property 1b: Either-Result Round Trip for Left/Error case
func TestProperty_EitherResultRoundTrip_Error(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random error message
		errMsg := rapid.String().Draw(t, "errMsg")
		err := functional.NewError(errMsg)
		
		// Test Left case: Either -> Result -> Either
		leftEither := functional.Left[error, int](err)
		result := functional.EitherToResult(leftEither)
		backToEither := functional.ResultToEither(result)
		
		// Verify round trip preserves error
		if !backToEither.IsLeft() {
			t.Fatalf("Expected Left after round trip, got Right")
		}
		if backToEither.LeftValue().Error() != errMsg {
			t.Fatalf("Error changed: expected %s, got %s", errMsg, backToEither.LeftValue().Error())
		}
	})
}

// Property: Option Some/None round trip through Match
func TestProperty_OptionMatchExhaustive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		useSome := rapid.Bool().Draw(t, "useSome")
		
		var opt functional.Option[int]
		if useSome {
			opt = functional.Some(value)
		} else {
			opt = functional.None[int]()
		}
		
		// Match must handle exactly one case
		matchedSome := false
		matchedNone := false
		
		opt.Match(
			func(v int) {
				matchedSome = true
				if v != value {
					t.Fatalf("Some value mismatch: expected %d, got %d", value, v)
				}
			},
			func() {
				matchedNone = true
			},
		)
		
		// Exactly one branch must execute
		if matchedSome == matchedNone {
			t.Fatalf("Match must execute exactly one branch")
		}
		if useSome && !matchedSome {
			t.Fatalf("Some option should match Some branch")
		}
		if !useSome && !matchedNone {
			t.Fatalf("None option should match None branch")
		}
	})
}

// Property: Result Ok/Err round trip through Match
func TestProperty_ResultMatchExhaustive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		useOk := rapid.Bool().Draw(t, "useOk")
		errMsg := rapid.String().Draw(t, "errMsg")
		
		var res functional.Result[int]
		if useOk {
			res = functional.Ok(value)
		} else {
			res = functional.Err[int](functional.NewError(errMsg))
		}
		
		matchedOk := false
		matchedErr := false
		
		res.Match(
			func(v int) {
				matchedOk = true
				if v != value {
					t.Fatalf("Ok value mismatch: expected %d, got %d", value, v)
				}
			},
			func(e error) {
				matchedErr = true
				if e.Error() != errMsg {
					t.Fatalf("Err message mismatch: expected %s, got %s", errMsg, e.Error())
				}
			},
		)
		
		if matchedOk == matchedErr {
			t.Fatalf("Match must execute exactly one branch")
		}
	})
}

// Property: Map preserves structure (Functor law)
func TestProperty_MapPreservesStructure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		
		// Option: Map over Some preserves Some
		opt := functional.Some(value)
		mapped := functional.MapOption(opt, func(x int) int { return x * 2 })
		if !mapped.IsSome() {
			t.Fatalf("Map over Some should preserve Some")
		}
		if mapped.Unwrap() != value*2 {
			t.Fatalf("Map should apply function: expected %d, got %d", value*2, mapped.Unwrap())
		}
		
		// Option: Map over None preserves None
		none := functional.None[int]()
		mappedNone := functional.MapOption(none, func(x int) int { return x * 2 })
		if mappedNone.IsSome() {
			t.Fatalf("Map over None should preserve None")
		}
	})
}

// Property: FlatMap associativity (Monad law)
func TestProperty_FlatMapAssociativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(0, 100).Draw(t, "value")
		
		f := func(x int) functional.Option[int] {
			return functional.Some(x + 1)
		}
		g := func(x int) functional.Option[int] {
			return functional.Some(x * 2)
		}
		
		opt := functional.Some(value)
		
		// (m >>= f) >>= g
		left := functional.FlatMapOption(functional.FlatMapOption(opt, f), g)
		
		// m >>= (\x -> f x >>= g)
		right := functional.FlatMapOption(opt, func(x int) functional.Option[int] {
			return functional.FlatMapOption(f(x), g)
		})
		
		if left.IsSome() != right.IsSome() {
			t.Fatalf("FlatMap associativity violated: structure differs")
		}
		if left.IsSome() && left.Unwrap() != right.Unwrap() {
			t.Fatalf("FlatMap associativity violated: %d != %d", left.Unwrap(), right.Unwrap())
		}
	})
}

// Property: Iterator Map-Collect identity
func TestProperty_IteratorMapIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slice := rapid.SliceOf(rapid.Int()).Draw(t, "slice")
		
		iter := functional.FromSlice(slice)
		mapped := functional.Map(iter, func(x int) int { return x })
		collected := functional.Collect(mapped)
		
		if len(collected) != len(slice) {
			t.Fatalf("Length mismatch: expected %d, got %d", len(slice), len(collected))
		}
		for i, v := range collected {
			if v != slice[i] {
				t.Fatalf("Value mismatch at %d: expected %d, got %d", i, slice[i], v)
			}
		}
	})
}

// Property: Stream memoization consistency
func TestProperty_StreamMemoization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slice := rapid.SliceOfN(rapid.Int(), 1, 10).Draw(t, "slice")
		
		stream := functional.StreamFromSlice(slice)
		
		// Access tail multiple times
		tail1 := stream.Tail()
		tail2 := stream.Tail()
		
		// Should be same reference (memoized)
		if tail1 != tail2 {
			t.Fatalf("Stream tail should be memoized")
		}
	})
}
