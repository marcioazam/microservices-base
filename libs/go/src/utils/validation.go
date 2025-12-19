package utils

import (
	"fmt"
	"strings"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validation represents validation result with error accumulation.
type Validation[T any] struct {
	value  T
	errors []ValidationError
	valid  bool
}

// Valid creates a valid Validation.
func Valid[T any](value T) Validation[T] {
	return Validation[T]{value: value, valid: true}
}

// Invalid creates an invalid Validation with errors.
func Invalid[T any](errors ...ValidationError) Validation[T] {
	return Validation[T]{errors: errors, valid: false}
}

// IsValid returns true if validation passed.
func (v Validation[T]) IsValid() bool {
	return v.valid
}

// Value returns the validated value.
func (v Validation[T]) Value() T {
	return v.value
}

// Errors returns all validation errors.
func (v Validation[T]) Errors() []ValidationError {
	return v.errors
}

// ErrorMessages returns error messages as strings.
func (v Validation[T]) ErrorMessages() []string {
	msgs := make([]string, len(v.errors))
	for i, e := range v.errors {
		msgs[i] = e.Error()
	}
	return msgs
}

// Combine merges two validations, accumulating errors.
func (v Validation[T]) Combine(other Validation[T]) Validation[T] {
	if v.valid && other.valid {
		return other
	}
	errors := append(v.errors, other.errors...)
	return Invalid[T](errors...)
}

// Rule represents a validation rule.
type Rule[T any] func(T) *ValidationError

// Validator validates values of type T.
type Validator[T any] struct {
	rules []Rule[T]
	field string
}

// NewValidator creates a new validator.
func NewValidator[T any](field string) *Validator[T] {
	return &Validator[T]{field: field}
}

// AddRule adds a validation rule.
func (v *Validator[T]) AddRule(rule Rule[T]) *Validator[T] {
	v.rules = append(v.rules, rule)
	return v
}

// Validate runs all rules and accumulates errors.
func (v *Validator[T]) Validate(value T) Validation[T] {
	var errors []ValidationError
	for _, rule := range v.rules {
		if err := rule(value); err != nil {
			err.Field = v.field
			errors = append(errors, *err)
		}
	}
	if len(errors) > 0 {
		return Invalid[T](errors...)
	}
	return Valid(value)
}

// And combines two validators.
func (v *Validator[T]) And(other *Validator[T]) *Validator[T] {
	combined := NewValidator[T](v.field)
	combined.rules = append(combined.rules, v.rules...)
	combined.rules = append(combined.rules, other.rules...)
	return combined
}

// Common validation rules

// Required checks that string is not empty.
func Required() Rule[string] {
	return func(s string) *ValidationError {
		if strings.TrimSpace(s) == "" {
			return &ValidationError{Message: "is required", Code: "required"}
		}
		return nil
	}
}

// MinLength checks minimum string length.
func MinLength(min int) Rule[string] {
	return func(s string) *ValidationError {
		if len(s) < min {
			return &ValidationError{
				Message: fmt.Sprintf("must be at least %d characters", min),
				Code:    "min_length",
			}
		}
		return nil
	}
}

// MaxLength checks maximum string length.
func MaxLength(max int) Rule[string] {
	return func(s string) *ValidationError {
		if len(s) > max {
			return &ValidationError{
				Message: fmt.Sprintf("must be at most %d characters", max),
				Code:    "max_length",
			}
		}
		return nil
	}
}

// Min checks minimum numeric value.
func Min[T ~int | ~int64 | ~float64](min T) Rule[T] {
	return func(v T) *ValidationError {
		if v < min {
			return &ValidationError{
				Message: fmt.Sprintf("must be at least %v", min),
				Code:    "min",
			}
		}
		return nil
	}
}

// Max checks maximum numeric value.
func Max[T ~int | ~int64 | ~float64](max T) Rule[T] {
	return func(v T) *ValidationError {
		if v > max {
			return &ValidationError{
				Message: fmt.Sprintf("must be at most %v", max),
				Code:    "max",
			}
		}
		return nil
	}
}

// Range checks value is within range.
func Range[T ~int | ~int64 | ~float64](min, max T) Rule[T] {
	return func(v T) *ValidationError {
		if v < min || v > max {
			return &ValidationError{
				Message: fmt.Sprintf("must be between %v and %v", min, max),
				Code:    "range",
			}
		}
		return nil
	}
}

// OneOf checks value is one of allowed values.
func OneOf[T comparable](allowed ...T) Rule[T] {
	return func(v T) *ValidationError {
		for _, a := range allowed {
			if v == a {
				return nil
			}
		}
		return &ValidationError{
			Message: "must be one of allowed values",
			Code:    "one_of",
		}
	}
}

// Custom creates a custom validation rule.
func Custom[T any](check func(T) bool, message, code string) Rule[T] {
	return func(v T) *ValidationError {
		if !check(v) {
			return &ValidationError{Message: message, Code: code}
		}
		return nil
	}
}
