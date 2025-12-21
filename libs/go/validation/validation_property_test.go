package validation

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func defaultTestParameters() *gopter.TestParameters {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	return params
}

// **Feature: resilience-lib-extraction, Property 6: Validation Correctness**
// **Validates: Requirements 4.1, 4.2**
func TestProperty_ValidationCorrectness(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("positive_validator_rejects_non_positive", prop.ForAll(
		func(value int) bool {
			validator := Positive[int]()
			err := validator("field", value)
			if value <= 0 {
				return err != nil
			}
			return err == nil
		},
		gen.IntRange(-100, 100),
	))

	props.Property("non_negative_validator_rejects_negative", prop.ForAll(
		func(value int) bool {
			validator := NonNegative[int]()
			err := validator("field", value)
			if value < 0 {
				return err != nil
			}
			return err == nil
		},
		gen.IntRange(-100, 100),
	))

	props.Property("non_empty_validator_rejects_empty_string", prop.ForAll(
		func(value string) bool {
			validator := NonEmpty()
			err := validator("field", value)
			if value == "" {
				return err != nil
			}
			return err == nil
		},
		gen.AnyString(),
	))

	props.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 7: Positive Validator Correctness**
// **Validates: Requirements 4.4**
func TestProperty_PositiveValidatorCorrectness(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("positive_int_validator", prop.ForAll(
		func(value int) bool {
			validator := Positive[int]()
			err := validator("field", value)
			return (value > 0 && err == nil) || (value <= 0 && err != nil)
		},
		gen.Int(),
	))

	props.Property("positive_int64_validator", prop.ForAll(
		func(value int64) bool {
			validator := Positive[int64]()
			err := validator("field", value)
			return (value > 0 && err == nil) || (value <= 0 && err != nil)
		},
		gen.Int64(),
	))

	props.Property("positive_float64_validator", prop.ForAll(
		func(value float64) bool {
			validator := Positive[float64]()
			err := validator("field", value)
			return (value > 0 && err == nil) || (value <= 0 && err != nil)
		},
		gen.Float64Range(-1000, 1000),
	))

	props.Property("positive_duration_validator", prop.ForAll(
		func(ms int64) bool {
			d := time.Duration(ms) * time.Millisecond
			validator := PositiveDuration()
			err := validator("field", d)
			return (d > 0 && err == nil) || (d <= 0 && err != nil)
		},
		gen.Int64Range(-1000, 1000),
	))

	props.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 8: InRange Validator Correctness**
// **Validates: Requirements 4.4**
func TestProperty_InRangeValidatorCorrectness(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("in_range_int_validator", prop.ForAll(
		func(value, min, max int) bool {
			if min > max {
				min, max = max, min
			}
			validator := InRange[int](min, max)
			err := validator("field", value)
			inRange := value >= min && value <= max
			return (inRange && err == nil) || (!inRange && err != nil)
		},
		gen.Int(),
		gen.IntRange(-100, 0),
		gen.IntRange(0, 100),
	))

	props.Property("in_range_float64_validator", prop.ForAll(
		func(value, min, max float64) bool {
			if min > max {
				min, max = max, min
			}
			validator := InRange[float64](min, max)
			err := validator("field", value)
			inRange := value >= min && value <= max
			return (inRange && err == nil) || (!inRange && err != nil)
		},
		gen.Float64Range(-100, 100),
		gen.Float64Range(-50, 0),
		gen.Float64Range(0, 50),
	))

	props.Property("duration_in_range_validator", prop.ForAll(
		func(valueMs, minMs, maxMs int64) bool {
			if minMs > maxMs {
				minMs, maxMs = maxMs, minMs
			}
			value := time.Duration(valueMs) * time.Millisecond
			min := time.Duration(minMs) * time.Millisecond
			max := time.Duration(maxMs) * time.Millisecond
			
			validator := DurationInRange(min, max)
			err := validator("field", value)
			inRange := value >= min && value <= max
			return (inRange && err == nil) || (!inRange && err != nil)
		},
		gen.Int64Range(-100, 100),
		gen.Int64Range(-50, 0),
		gen.Int64Range(0, 50),
	))

	props.TestingRun(t)
}
