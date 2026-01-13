package types_test

import (
	"errors"
	"testing"

	"github.com/auth-platform/sdk-go/src/types"
)

func TestOk(t *testing.T) {
	r := types.Ok(42)
	if !r.IsOk() {
		t.Error("Ok result should be ok")
	}
	if r.IsErr() {
		t.Error("Ok result should not be err")
	}
	if r.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", r.Unwrap())
	}
}

func TestErr(t *testing.T) {
	err := errors.New("test error")
	r := types.Err[int](err)
	if r.IsOk() {
		t.Error("Err result should not be ok")
	}
	if !r.IsErr() {
		t.Error("Err result should be err")
	}
	if r.Error() != err {
		t.Error("Error() should return the error")
	}
}

func TestUnwrapOr(t *testing.T) {
	ok := types.Ok(42)
	if ok.UnwrapOr(0) != 42 {
		t.Error("UnwrapOr on Ok should return value")
	}

	err := types.Err[int](errors.New("error"))
	if err.UnwrapOr(99) != 99 {
		t.Error("UnwrapOr on Err should return default")
	}
}

func TestUnwrapPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Unwrap on Err should panic")
		}
	}()
	r := types.Err[int](errors.New("error"))
	r.Unwrap()
}

func TestUnwrapErrPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("UnwrapErr on Ok should panic")
		}
	}()
	r := types.Ok(42)
	r.UnwrapErr()
}

func TestMap(t *testing.T) {
	ok := types.Ok(21)
	doubled := types.Map(ok, func(x int) int { return x * 2 })
	if doubled.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", doubled.Unwrap())
	}

	err := types.Err[int](errors.New("error"))
	mapped := types.Map(err, func(x int) int { return x * 2 })
	if mapped.IsOk() {
		t.Error("Map on Err should preserve error")
	}
}

func TestFlatMap(t *testing.T) {
	ok := types.Ok(21)
	result := types.FlatMap(ok, func(x int) types.Result[int] {
		return types.Ok(x * 2)
	})
	if result.Unwrap() != 42 {
		t.Errorf("expected 42, got %d", result.Unwrap())
	}

	err := types.Err[int](errors.New("error"))
	result = types.FlatMap(err, func(x int) types.Result[int] {
		return types.Ok(x * 2)
	})
	if result.IsOk() {
		t.Error("FlatMap on Err should preserve error")
	}
}

func TestMapErr(t *testing.T) {
	err := types.Err[int](errors.New("original"))
	mapped := types.MapErr(err, func(e error) error {
		return errors.New("wrapped: " + e.Error())
	})
	if mapped.Error().Error() != "wrapped: original" {
		t.Errorf("expected wrapped error, got %s", mapped.Error())
	}

	ok := types.Ok(42)
	mapped = types.MapErr(ok, func(e error) error {
		return errors.New("should not be called")
	})
	if !mapped.IsOk() {
		t.Error("MapErr on Ok should preserve success")
	}
}

func TestMatch(t *testing.T) {
	var okCalled, errCalled bool

	ok := types.Ok(42)
	ok.Match(func(v int) { okCalled = true }, func(e error) { errCalled = true })
	if !okCalled || errCalled {
		t.Error("Match on Ok should call onOk")
	}

	okCalled, errCalled = false, false
	err := types.Err[int](errors.New("error"))
	err.Match(func(v int) { okCalled = true }, func(e error) { errCalled = true })
	if okCalled || !errCalled {
		t.Error("Match on Err should call onErr")
	}
}

func TestMatchReturn(t *testing.T) {
	ok := types.Ok(42)
	result := types.MatchReturn(ok,
		func(v int) string { return "ok" },
		func(e error) string { return "err" },
	)
	if result != "ok" {
		t.Errorf("expected 'ok', got '%s'", result)
	}

	err := types.Err[int](errors.New("error"))
	result = types.MatchReturn(err,
		func(v int) string { return "ok" },
		func(e error) string { return "err" },
	)
	if result != "err" {
		t.Errorf("expected 'err', got '%s'", result)
	}
}

func TestAnd(t *testing.T) {
	ok1 := types.Ok(1)
	ok2 := types.Ok("two")
	result := types.And(ok1, ok2)
	if result.Unwrap() != "two" {
		t.Error("And with two Ok should return second")
	}

	err := types.Err[int](errors.New("error"))
	result = types.And(err, ok2)
	if result.IsOk() {
		t.Error("And with first Err should return Err")
	}
}

func TestOr(t *testing.T) {
	ok := types.Ok(1)
	other := types.Ok(2)
	result := types.Or(ok, other)
	if result.Unwrap() != 1 {
		t.Error("Or with first Ok should return first")
	}

	err := types.Err[int](errors.New("error"))
	result = types.Or(err, other)
	if result.Unwrap() != 2 {
		t.Error("Or with first Err should return second")
	}
}
