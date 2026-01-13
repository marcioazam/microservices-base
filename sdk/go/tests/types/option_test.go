package types_test

import (
	"errors"
	"testing"

	"github.com/auth-platform/sdk-go/src/types"
)

func TestSome(t *testing.T) {
	o := types.Some(42)
	if !o.IsSome() {
		t.Error("Some should be some")
	}
	if o.IsNone() {
		t.Error("Some should not be none")
	}
	if o.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", o.Unwrap())
	}
}

func TestNone(t *testing.T) {
	o := types.None[int]()
	if o.IsSome() {
		t.Error("None should not be some")
	}
	if !o.IsNone() {
		t.Error("None should be none")
	}
}

func TestOptionUnwrapOr(t *testing.T) {
	some := types.Some(42)
	if some.UnwrapOr(0) != 42 {
		t.Error("UnwrapOr on Some should return value")
	}

	none := types.None[int]()
	if none.UnwrapOr(99) != 99 {
		t.Error("UnwrapOr on None should return default")
	}
}

func TestOptionUnwrapPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Unwrap on None should panic")
		}
	}()
	o := types.None[int]()
	o.Unwrap()
}

func TestMapOption(t *testing.T) {
	some := types.Some(21)
	doubled := types.MapOption(some, func(x int) int { return x * 2 })
	if doubled.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", doubled.Unwrap())
	}

	none := types.None[int]()
	mapped := types.MapOption(none, func(x int) int { return x * 2 })
	if mapped.IsSome() {
		t.Error("MapOption on None should return None")
	}
}

func TestFlatMapOption(t *testing.T) {
	some := types.Some(21)
	result := types.FlatMapOption(some, func(x int) types.Option[int] {
		return types.Some(x * 2)
	})
	if result.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", result.Unwrap())
	}

	none := types.None[int]()
	result = types.FlatMapOption(none, func(x int) types.Option[int] {
		return types.Some(x * 2)
	})
	if result.IsSome() {
		t.Error("FlatMapOption on None should return None")
	}
}

func TestFilter(t *testing.T) {
	some := types.Some(42)
	filtered := types.Filter(some, func(x int) bool { return x > 40 })
	if !filtered.IsSome() {
		t.Error("Filter with true predicate should keep value")
	}

	filtered = types.Filter(some, func(x int) bool { return x > 50 })
	if filtered.IsSome() {
		t.Error("Filter with false predicate should return None")
	}

	none := types.None[int]()
	filtered = types.Filter(none, func(x int) bool { return true })
	if filtered.IsSome() {
		t.Error("Filter on None should return None")
	}
}

func TestOptionMatch(t *testing.T) {
	var someCalled, noneCalled bool

	some := types.Some(42)
	some.Match(func(v int) { someCalled = true }, func() { noneCalled = true })
	if !someCalled || noneCalled {
		t.Error("Match on Some should call onSome")
	}

	someCalled, noneCalled = false, false
	none := types.None[int]()
	none.Match(func(v int) { someCalled = true }, func() { noneCalled = true })
	if someCalled || !noneCalled {
		t.Error("Match on None should call onNone")
	}
}

func TestOkOr(t *testing.T) {
	some := types.Some(42)
	result := types.OkOr(some, errors.New("error"))
	if !result.IsOk() {
		t.Error("OkOr on Some should return Ok")
	}
	if result.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", result.Unwrap())
	}

	none := types.None[int]()
	result = types.OkOr(none, errors.New("error"))
	if result.IsOk() {
		t.Error("OkOr on None should return Err")
	}
}

func TestToOption(t *testing.T) {
	ok := types.Ok(42)
	opt := types.ToOption(ok)
	if !opt.IsSome() {
		t.Error("ToOption on Ok should return Some")
	}
	if opt.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", opt.Unwrap())
	}

	err := types.Err[int](errors.New("error"))
	opt = types.ToOption(err)
	if opt.IsSome() {
		t.Error("ToOption on Err should return None")
	}
}

func TestOptionValue(t *testing.T) {
	some := types.Some(42)
	v, ok := some.Value()
	if !ok || v != 42 {
		t.Error("Value on Some should return value and true")
	}

	none := types.None[int]()
	_, ok = none.Value()
	if ok {
		t.Error("Value on None should return false")
	}
}
