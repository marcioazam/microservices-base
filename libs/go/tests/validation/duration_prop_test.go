package validation_test

import (
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/validation"
	"pgregory.net/rapid"
)

// Property: DurationRange accepts values within range
func TestDurationRangeAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		minMs := rapid.Int64Range(1, 1000).Draw(t, "min_ms")
		maxMs := rapid.Int64Range(minMs+1, minMs+10000).Draw(t, "max_ms")
		valueMs := rapid.Int64Range(minMs, maxMs).Draw(t, "value_ms")

		min := time.Duration(minMs) * time.Millisecond
		max := time.Duration(maxMs) * time.Millisecond
		value := time.Duration(valueMs) * time.Millisecond

		validator := validation.DurationRange(min, max)
		err := validator(value)

		if err != nil {
			t.Errorf("DurationRange(%v, %v)(%v) should pass, got error: %v", min, max, value, err)
		}
	})
}

// Property: DurationRange rejects values outside range
func TestDurationRangeRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		minMs := rapid.Int64Range(100, 1000).Draw(t, "min_ms")
		maxMs := rapid.Int64Range(minMs+100, minMs+10000).Draw(t, "max_ms")
		belowMin := rapid.Bool().Draw(t, "below_min")

		min := time.Duration(minMs) * time.Millisecond
		max := time.Duration(maxMs) * time.Millisecond

		var value time.Duration
		if belowMin {
			value = time.Duration(rapid.Int64Range(1, minMs-1).Draw(t, "value_ms")) * time.Millisecond
		} else {
			value = time.Duration(rapid.Int64Range(maxMs+1, maxMs+10000).Draw(t, "value_ms")) * time.Millisecond
		}

		validator := validation.DurationRange(min, max)
		err := validator(value)

		if err == nil {
			t.Errorf("DurationRange(%v, %v)(%v) should fail", min, max, value)
		}
	})
}

// Property: DurationMin accepts values >= min
func TestDurationMinAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		minMs := rapid.Int64Range(1, 1000).Draw(t, "min_ms")
		valueMs := rapid.Int64Range(minMs, minMs+10000).Draw(t, "value_ms")

		min := time.Duration(minMs) * time.Millisecond
		value := time.Duration(valueMs) * time.Millisecond

		validator := validation.DurationMin(min)
		err := validator(value)

		if err != nil {
			t.Errorf("DurationMin(%v)(%v) should pass, got error: %v", min, value, err)
		}
	})
}

// Property: DurationMax accepts values <= max
func TestDurationMaxAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxMs := rapid.Int64Range(100, 10000).Draw(t, "max_ms")
		valueMs := rapid.Int64Range(1, maxMs).Draw(t, "value_ms")

		max := time.Duration(maxMs) * time.Millisecond
		value := time.Duration(valueMs) * time.Millisecond

		validator := validation.DurationMax(max)
		err := validator(value)

		if err != nil {
			t.Errorf("DurationMax(%v)(%v) should pass, got error: %v", max, value, err)
		}
	})
}

// Property: DurationPositive accepts positive durations
func TestDurationPositiveAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		valueMs := rapid.Int64Range(1, 100000).Draw(t, "value_ms")
		value := time.Duration(valueMs) * time.Millisecond

		validator := validation.DurationPositive()
		err := validator(value)

		if err != nil {
			t.Errorf("DurationPositive()(%v) should pass, got error: %v", value, err)
		}
	})
}

// Property: DurationPositive rejects zero and negative
func TestDurationPositiveRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		valueMs := rapid.Int64Range(-10000, 0).Draw(t, "value_ms")
		value := time.Duration(valueMs) * time.Millisecond

		validator := validation.DurationPositive()
		err := validator(value)

		if err == nil {
			t.Errorf("DurationPositive()(%v) should fail", value)
		}
	})
}

// Property: FloatRange accepts values within range
func TestFloatRangeAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		min := rapid.Float64Range(0.0, 100.0).Draw(t, "min")
		max := rapid.Float64Range(min+0.1, min+100.0).Draw(t, "max")
		value := rapid.Float64Range(min, max).Draw(t, "value")

		validator := validation.FloatRange(min, max)
		err := validator(value)

		if err != nil {
			t.Errorf("FloatRange(%v, %v)(%v) should pass, got error: %v", min, max, value, err)
		}
	})
}

// Property: FloatRange rejects values outside range
func TestFloatRangeRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		min := rapid.Float64Range(10.0, 100.0).Draw(t, "min")
		max := rapid.Float64Range(min+10.0, min+100.0).Draw(t, "max")
		belowMin := rapid.Bool().Draw(t, "below_min")

		var value float64
		if belowMin {
			value = rapid.Float64Range(0.0, min-0.1).Draw(t, "value")
		} else {
			value = rapid.Float64Range(max+0.1, max+100.0).Draw(t, "value")
		}

		validator := validation.FloatRange(min, max)
		err := validator(value)

		if err == nil {
			t.Errorf("FloatRange(%v, %v)(%v) should fail", min, max, value)
		}
	})
}

// Property: FloatPositive accepts positive floats
func TestFloatPositiveAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Float64Range(0.001, 1000000.0).Draw(t, "value")

		validator := validation.FloatPositive()
		err := validator(value)

		if err != nil {
			t.Errorf("FloatPositive()(%v) should pass, got error: %v", value, err)
		}
	})
}

// Property: FloatPositive rejects zero and negative
func TestFloatPositiveRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Float64Range(-1000000.0, 0.0).Draw(t, "value")

		validator := validation.FloatPositive()
		err := validator(value)

		if err == nil {
			t.Errorf("FloatPositive()(%v) should fail", value)
		}
	})
}

// Property: FloatNonNegative accepts zero and positive
func TestFloatNonNegativeAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Float64Range(0.0, 1000000.0).Draw(t, "value")

		validator := validation.FloatNonNegative()
		err := validator(value)

		if err != nil {
			t.Errorf("FloatNonNegative()(%v) should pass, got error: %v", value, err)
		}
	})
}

// Property: FloatNonNegative rejects negative
func TestFloatNonNegativeRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Float64Range(-1000000.0, -0.001).Draw(t, "value")

		validator := validation.FloatNonNegative()
		err := validator(value)

		if err == nil {
			t.Errorf("FloatNonNegative()(%v) should fail", value)
		}
	})
}

// Property: IntRange is consistent with InRange
func TestIntRangeConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		min := rapid.IntRange(0, 100).Draw(t, "min")
		max := rapid.IntRange(min+1, min+100).Draw(t, "max")
		value := rapid.IntRange(min-50, max+50).Draw(t, "value")

		intRangeValidator := validation.IntRange(min, max)
		inRangeValidator := validation.InRange(min, max)

		intRangeErr := intRangeValidator(value)
		inRangeErr := inRangeValidator(value)

		intRangeValid := intRangeErr == nil
		inRangeValid := inRangeErr == nil

		if intRangeValid != inRangeValid {
			t.Errorf("IntRange and InRange should be consistent: IntRange=%v, InRange=%v for value=%d",
				intRangeValid, inRangeValid, value)
		}
	})
}
