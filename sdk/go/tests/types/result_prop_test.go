package types_test

import (
	"errors"
	"testing"

	"github.com/auth-platform/sdk-go/src/types"
	"pgregory.net/rapid"
)

// Feature: go-sdk-state-of-art-2025, Property 5: Result/Option Functor Laws
func TestProperty_ResultFunctorLaws(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Int().Draw(t, "value")
		f := func(x int) int { return x * 2 }
		g := func(x int) int { return x + 1 }

		r := types.Ok(v)

		// Law 1: Map(Ok(v), f) == Ok(f(v))
		mapped := types.Map(r, f)
		if mapped.Unwrap() != f(v) {
			t.Errorf("Map(Ok(v), f) should equal Ok(f(v)): got %d, expected %d", mapped.Unwrap(), f(v))
		}

		// Law 2: Map(Map(r, f), g) == Map(r, compose(g, f))
		composed := types.Map(types.Map(r, f), g)
		direct := types.Map(r, func(x int) int { return g(f(x)) })
		if composed.Unwrap() != direct.Unwrap() {
			t.Errorf("Functor composition law violated: %d != %d", composed.Unwrap(), direct.Unwrap())
		}

		// Law 3: Map(Err(e), f) == Err(e)
		err := types.Err[int](errors.New("test"))
		errMapped := types.Map(err, f)
		if errMapped.IsOk() {
			t.Error("Map on Err should preserve error")
		}
	})
}

// Feature: go-sdk-state-of-art-2025, Property 6: Result/Option Conversion Round-Trip
func TestProperty_ResultOptionConversion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Int().Draw(t, "value")

		// Successful Result -> Option should be Some
		okResult := types.Ok(v)
		opt := types.ToOption(okResult)
		if !opt.IsSome() {
			t.Error("ToOption on Ok should return Some")
		}
		if opt.Unwrap() != v {
			t.Errorf("ToOption should preserve value: got %d, expected %d", opt.Unwrap(), v)
		}

		// Failed Result -> Option should be None
		errResult := types.Err[int](errors.New("error"))
		opt = types.ToOption(errResult)
		if opt.IsSome() {
			t.Error("ToOption on Err should return None")
		}

		// Some -> Result should be Ok
		some := types.Some(v)
		result := types.OkOr(some, errors.New("error"))
		if !result.IsOk() {
			t.Error("OkOr on Some should return Ok")
		}
		if result.Unwrap() != v {
			t.Errorf("OkOr should preserve value: got %d, expected %d", result.Unwrap(), v)
		}

		// None -> Result should be Err
		none := types.None[int]()
		result = types.OkOr(none, errors.New("error"))
		if result.IsOk() {
			t.Error("OkOr on None should return Err")
		}
	})
}

func TestProperty_ResultMonadLaws(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Int().Draw(t, "value")
		f := func(x int) types.Result[int] { return types.Ok(x * 2) }
		g := func(x int) types.Result[int] { return types.Ok(x + 1) }

		// Left identity: FlatMap(Ok(v), f) == f(v)
		left := types.FlatMap(types.Ok(v), f)
		right := f(v)
		if left.Unwrap() != right.Unwrap() {
			t.Errorf("Left identity violated: %d != %d", left.Unwrap(), right.Unwrap())
		}

		// Right identity: FlatMap(r, Ok) == r
		r := types.Ok(v)
		flatMapped := types.FlatMap(r, func(x int) types.Result[int] { return types.Ok(x) })
		if flatMapped.Unwrap() != r.Unwrap() {
			t.Errorf("Right identity violated: %d != %d", flatMapped.Unwrap(), r.Unwrap())
		}

		// Associativity: FlatMap(FlatMap(r, f), g) == FlatMap(r, x => FlatMap(f(x), g))
		leftAssoc := types.FlatMap(types.FlatMap(r, f), g)
		rightAssoc := types.FlatMap(r, func(x int) types.Result[int] {
			return types.FlatMap(f(x), g)
		})
		if leftAssoc.Unwrap() != rightAssoc.Unwrap() {
			t.Errorf("Associativity violated: %d != %d", leftAssoc.Unwrap(), rightAssoc.Unwrap())
		}
	})
}
