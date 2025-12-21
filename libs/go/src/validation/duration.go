// Package validation provides duration and float validators.
package validation

import (
	"fmt"
	"time"
)

// DurationRange checks duration is within range.
func DurationRange(min, max time.Duration) Validator[time.Duration] {
	return func(d time.Duration) *ValidationError {
		if d < min || d > max {
			return &ValidationError{
				Message: fmt.Sprintf("must be between %v and %v", min, max),
				Code:    "duration_range",
			}
		}
		return nil
	}
}

// DurationMin checks minimum duration.
func DurationMin(min time.Duration) Validator[time.Duration] {
	return func(d time.Duration) *ValidationError {
		if d < min {
			return &ValidationError{
				Message: fmt.Sprintf("must be at least %v", min),
				Code:    "duration_min",
			}
		}
		return nil
	}
}

// DurationMax checks maximum duration.
func DurationMax(max time.Duration) Validator[time.Duration] {
	return func(d time.Duration) *ValidationError {
		if d > max {
			return &ValidationError{
				Message: fmt.Sprintf("must be at most %v", max),
				Code:    "duration_max",
			}
		}
		return nil
	}
}

// DurationPositive checks duration is positive.
func DurationPositive() Validator[time.Duration] {
	return func(d time.Duration) *ValidationError {
		if d <= 0 {
			return &ValidationError{
				Message: "must be positive",
				Code:    "duration_positive",
			}
		}
		return nil
	}
}

// DurationNonZero checks duration is not zero.
func DurationNonZero() Validator[time.Duration] {
	return func(d time.Duration) *ValidationError {
		if d == 0 {
			return &ValidationError{
				Message: "must not be zero",
				Code:    "duration_non_zero",
			}
		}
		return nil
	}
}

// FloatRange checks float is within range.
func FloatRange(min, max float64) Validator[float64] {
	return func(f float64) *ValidationError {
		if f < min || f > max {
			return &ValidationError{
				Message: fmt.Sprintf("must be between %v and %v", min, max),
				Code:    "float_range",
			}
		}
		return nil
	}
}

// FloatMin checks minimum float value.
func FloatMin(min float64) Validator[float64] {
	return func(f float64) *ValidationError {
		if f < min {
			return &ValidationError{
				Message: fmt.Sprintf("must be at least %v", min),
				Code:    "float_min",
			}
		}
		return nil
	}
}

// FloatMax checks maximum float value.
func FloatMax(max float64) Validator[float64] {
	return func(f float64) *ValidationError {
		if f > max {
			return &ValidationError{
				Message: fmt.Sprintf("must be at most %v", max),
				Code:    "float_max",
			}
		}
		return nil
	}
}

// FloatPositive checks float is positive.
func FloatPositive() Validator[float64] {
	return func(f float64) *ValidationError {
		if f <= 0 {
			return &ValidationError{
				Message: "must be positive",
				Code:    "float_positive",
			}
		}
		return nil
	}
}

// FloatNonNegative checks float is non-negative.
func FloatNonNegative() Validator[float64] {
	return func(f float64) *ValidationError {
		if f < 0 {
			return &ValidationError{
				Message: "must be non-negative",
				Code:    "float_non_negative",
			}
		}
		return nil
	}
}

// IntRange checks int is within range (alias for InRange with int).
func IntRange(min, max int) Validator[int] {
	return InRange(min, max)
}

// IntMin checks minimum int value (alias for Min with int).
func IntMin(min int) Validator[int] {
	return Min(min)
}

// IntMax checks maximum int value (alias for Max with int).
func IntMax(max int) Validator[int] {
	return Max(max)
}

// IntPositive checks int is positive (alias for Positive with int).
func IntPositive() Validator[int] {
	return Positive[int]()
}
