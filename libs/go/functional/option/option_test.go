package option

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 10: Option Map Preserves Structure**
// **Validates: Requirements 12.7**
func TestOptionMapPreservesStructure(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Test that Map on Some returns Some(fn(value))
	properties.Property("Map on Some returns Some(fn(value))", prop.ForAll(
		func(n int) bool {
			o := Some(n)
			fn := func(x int) int { return x * 2 }
			mapped := Map(o, fn)
			return mapped.IsSome() && mapped.Unwrap() == fn(n)
		},
		gen.Int(),
	))

	// Test that Map on None returns None
	properties.Property("Map on None returns None", prop.ForAll(
		func(n int) bool {
			o := None[int]()
			fn := func(x int) int { return x * 2 }
			mapped := Map(o, fn)
			return mapped.IsNone()
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 11: Option Pointer Round-Trip**
// **Validates: Requirements 12.9, 12.10**
func TestOptionPointerRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Test non-nil pointer round-trip
	properties.Property("FromPtr(ptr).ToPtr() returns equal value for non-nil", prop.ForAll(
		func(n int) bool {
			ptr := &n
			opt := FromPtr(ptr)
			result := opt.ToPtr()
			return result != nil && *result == n
		},
		gen.Int(),
	))

	// Test nil pointer round-trip
	properties.Property("FromPtr(nil).ToPtr() returns nil", prop.ForAll(
		func() bool {
			var ptr *int = nil
			opt := FromPtr(ptr)
			return opt.ToPtr() == nil
		},
	))

	properties.TestingRun(t)
}


func TestOptionBasicOperations(t *testing.T) {
	t.Run("Some creates present option", func(t *testing.T) {
		o := Some(42)
		if !o.IsSome() {
			t.Error("expected IsSome to be true")
		}
		if o.IsNone() {
			t.Error("expected IsNone to be false")
		}
		if o.Unwrap() != 42 {
			t.Errorf("expected 42, got %d", o.Unwrap())
		}
	})

	t.Run("None creates empty option", func(t *testing.T) {
		o := None[int]()
		if o.IsSome() {
			t.Error("expected IsSome to be false")
		}
		if !o.IsNone() {
			t.Error("expected IsNone to be true")
		}
	})

	t.Run("UnwrapOr returns default on None", func(t *testing.T) {
		o := None[int]()
		if o.UnwrapOr(100) != 100 {
			t.Error("expected default value")
		}
	})

	t.Run("UnwrapOr returns value on Some", func(t *testing.T) {
		o := Some(42)
		if o.UnwrapOr(100) != 42 {
			t.Error("expected actual value")
		}
	})

	t.Run("Filter keeps matching values", func(t *testing.T) {
		o := Some(42)
		filtered := o.Filter(func(x int) bool { return x > 0 })
		if !filtered.IsSome() || filtered.Unwrap() != 42 {
			t.Error("expected Some(42)")
		}
	})

	t.Run("Filter removes non-matching values", func(t *testing.T) {
		o := Some(42)
		filtered := o.Filter(func(x int) bool { return x < 0 })
		if !filtered.IsNone() {
			t.Error("expected None")
		}
	})

	t.Run("Filter on None returns None", func(t *testing.T) {
		o := None[int]()
		filtered := o.Filter(func(x int) bool { return true })
		if !filtered.IsNone() {
			t.Error("expected None")
		}
	})
}

func TestFlatMap(t *testing.T) {
	t.Run("FlatMap on Some applies function", func(t *testing.T) {
		o := Some(42)
		result := FlatMap(o, func(x int) Option[int] { return Some(x * 2) })
		if !result.IsSome() || result.Unwrap() != 84 {
			t.Error("expected Some(84)")
		}
	})

	t.Run("FlatMap on None returns None", func(t *testing.T) {
		o := None[int]()
		result := FlatMap(o, func(x int) Option[int] { return Some(x * 2) })
		if !result.IsNone() {
			t.Error("expected None")
		}
	})
}

func TestZip(t *testing.T) {
	t.Run("Zip two Some values", func(t *testing.T) {
		a := Some(1)
		b := Some("hello")
		result := Zip(a, b)
		if !result.IsSome() {
			t.Error("expected Some")
		}
		pair := result.Unwrap()
		if pair.First != 1 || pair.Second != "hello" {
			t.Error("unexpected pair values")
		}
	})

	t.Run("Zip with None returns None", func(t *testing.T) {
		a := Some(1)
		b := None[string]()
		result := Zip(a, b)
		if !result.IsNone() {
			t.Error("expected None")
		}
	})
}

func TestString(t *testing.T) {
	if Some(42).String() != "Some(42)" {
		t.Error("unexpected string for Some")
	}
	if None[int]().String() != "None" {
		t.Error("unexpected string for None")
	}
}
