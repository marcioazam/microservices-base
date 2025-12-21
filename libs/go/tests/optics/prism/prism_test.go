package prism

import (
	"testing"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/authcorp/libs/go/src/optics"
)

func TestPrismBasicOperations(t *testing.T) {
	t.Run("GetOption returns Some for matching", func(t *testing.T) {
		p := optics.StringToInt()
		result := p.GetOption("123")
		if result.IsNone() || result.Unwrap() != 123 {
			t.Error("expected 123")
		}
	})

	t.Run("GetOption returns None for non-matching", func(t *testing.T) {
		p := optics.StringToInt()
		result := p.GetOption("abc")
		if result.IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("ReverseGet constructs source", func(t *testing.T) {
		p := optics.StringToInt()
		result := p.ReverseGet(42)
		if result != "42" {
			t.Errorf("expected 42, got %s", result)
		}
	})

	t.Run("Modify applies function when matching", func(t *testing.T) {
		p := optics.StringToInt()
		result := p.Modify("10", func(n int) int { return n * 2 })
		if result != "20" {
			t.Errorf("expected 20, got %s", result)
		}
	})

	t.Run("Modify returns source when not matching", func(t *testing.T) {
		p := optics.StringToInt()
		result := p.Modify("abc", func(n int) int { return n * 2 })
		if result != "abc" {
			t.Errorf("expected abc, got %s", result)
		}
	})

	t.Run("Set sets value when matching", func(t *testing.T) {
		p := optics.StringToInt()
		result := p.Set("10", 99)
		if result != "99" {
			t.Errorf("expected 99, got %s", result)
		}
	})

	t.Run("Set returns source when not matching", func(t *testing.T) {
		p := optics.StringToInt()
		result := p.Set("abc", 99)
		if result != "abc" {
			t.Errorf("expected abc, got %s", result)
		}
	})
}

func TestSomePrism(t *testing.T) {
	p := optics.SomePrism[int]()

	t.Run("GetOption on Some returns value", func(t *testing.T) {
		opt := functional.Some(42)
		result := p.GetOption(opt)
		if result.IsNone() || result.Unwrap() != 42 {
			t.Error("expected 42")
		}
	})

	t.Run("GetOption on None returns None", func(t *testing.T) {
		opt := functional.None[int]()
		result := p.GetOption(opt)
		if result.IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("ReverseGet creates Some", func(t *testing.T) {
		result := p.ReverseGet(42)
		if result.IsNone() || result.Unwrap() != 42 {
			t.Error("expected Some(42)")
		}
	})
}

func TestPrismComposition(t *testing.T) {
	// Create a prism that focuses on Option[string] -> Option[int]
	optString := optics.SomePrism[string]()
	strToInt := optics.StringToInt()

	// Compose: Option[string] -> string -> int
	composed := optics.ComposePrism(optString, strToInt)

	t.Run("Composed prism works on matching", func(t *testing.T) {
		opt := functional.Some("123")
		result := composed.GetOption(opt)
		if result.IsNone() || result.Unwrap() != 123 {
			t.Error("expected 123")
		}
	})

	t.Run("Composed prism returns None on outer mismatch", func(t *testing.T) {
		opt := functional.None[string]()
		result := composed.GetOption(opt)
		if result.IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("Composed prism returns None on inner mismatch", func(t *testing.T) {
		opt := functional.Some("abc")
		result := composed.GetOption(opt)
		if result.IsSome() {
			t.Error("expected None")
		}
	})
}

func TestStringToIntEdgeCases(t *testing.T) {
	p := optics.StringToInt()

	t.Run("Handles zero", func(t *testing.T) {
		result := p.GetOption("0")
		if result.IsNone() || result.Unwrap() != 0 {
			t.Error("expected 0")
		}
	})

	t.Run("ReverseGet handles zero", func(t *testing.T) {
		result := p.ReverseGet(0)
		if result != "0" {
			t.Errorf("expected 0, got %s", result)
		}
	})

	t.Run("Handles negative", func(t *testing.T) {
		result := p.ReverseGet(-42)
		if result != "-42" {
			t.Errorf("expected -42, got %s", result)
		}
	})

	t.Run("Empty string returns None", func(t *testing.T) {
		result := p.GetOption("")
		if result.IsSome() {
			t.Error("expected None")
		}
	})
}
