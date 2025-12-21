package rand

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 13: Deterministic Random Source Reproducibility**
// **Validates: Requirements 7.3**
func TestDeterministicRandSourceReproducibility(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("same seed produces identical sequences", prop.ForAll(
		func(seed int64, count int) bool {
			if count < 1 {
				count = 1
			}
			if count > 100 {
				count = 100
			}

			src1 := NewDeterministicRandSource(seed)
			src2 := NewDeterministicRandSource(seed)

			for i := 0; i < count; i++ {
				v1 := src1.Float64()
				v2 := src2.Float64()
				if v1 != v2 {
					return false
				}
			}
			return true
		},
		gen.Int64(),
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 14: Random Source Value Range**
// **Validates: Requirements 7.1, 7.2**
func TestRandomSourceValueRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("CryptoRandSource returns values in [0, 1)", prop.ForAll(
		func(_ int) bool {
			src := NewCryptoRandSource()
			for i := 0; i < 100; i++ {
				v := src.Float64()
				if v < 0.0 || v >= 1.0 {
					return false
				}
			}
			return true
		},
		gen.Int(),
	))

	properties.Property("DeterministicRandSource returns values in [0, 1)", prop.ForAll(
		func(seed int64) bool {
			src := NewDeterministicRandSource(seed)
			for i := 0; i < 100; i++ {
				v := src.Float64()
				if v < 0.0 || v >= 1.0 {
					return false
				}
			}
			return true
		},
		gen.Int64(),
	))

	properties.Property("FixedRandSource returns values in [0, 1)", prop.ForAll(
		func(value float64) bool {
			src := NewFixedRandSource(value)
			v := src.Float64()
			return v >= 0.0 && v < 1.0
		},
		gen.Float64Range(-10.0, 10.0),
	))

	properties.TestingRun(t)
}

func TestFixedRandSourceClamps(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{-1.0, 0.0},
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 0.9999999999},
		{2.0, 0.9999999999},
	}

	for _, tt := range tests {
		src := NewFixedRandSource(tt.input)
		got := src.Float64()
		if got != tt.expected {
			t.Errorf("NewFixedRandSource(%v).Float64() = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
