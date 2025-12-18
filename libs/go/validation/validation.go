// Package validation provides generic validation utilities for configuration and input validation.
package validation

import (
	"fmt"
	"time"
)

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validator is a function that validates a value.
type Validator[T any] func(field string, value T) error

// Positive returns a validator that checks if a number is positive.
func Positive[T ~int | ~int64 | ~float64]() Validator[T] {
	return func(field string, value T) error {
		if value <= 0 {
			return &ValidationError{Field: field, Message: "must be positive"}
		}
		return nil
	}
}

// NonNegative returns a validator that checks if a number is non-negative.
func NonNegative[T ~int | ~int64 | ~float64]() Validator[T] {
	return func(field string, value T) error {
		if value < 0 {
			return &ValidationError{Field: field, Message: "must be >= 0"}
		}
		return nil
	}
}

// InRange returns a validator that checks if a value is in range [min, max].
func InRange[T ~int | ~int64 | ~float64](min, max T) Validator[T] {
	return func(field string, value T) error {
		if value < min || value > max {
			return &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("must be in range [%v, %v]", min, max),
			}
		}
		return nil
	}
}

// NonEmpty returns a validator that checks if a string is non-empty.
func NonEmpty() Validator[string] {
	return func(field string, value string) error {
		if value == "" {
			return &ValidationError{Field: field, Message: "must not be empty"}
		}
		return nil
	}
}

// MinLength returns a validator that checks if a string has minimum length.
func MinLength(min int) Validator[string] {
	return func(field string, value string) error {
		if len(value) < min {
			return &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("must have at least %d characters", min),
			}
		}
		return nil
	}
}

// MaxLength returns a validator that checks if a string has maximum length.
func MaxLength(max int) Validator[string] {
	return func(field string, value string) error {
		if len(value) > max {
			return &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("must have at most %d characters", max),
			}
		}
		return nil
	}
}

// PositiveDuration returns a validator that checks if a duration is positive.
func PositiveDuration() Validator[time.Duration] {
	return func(field string, value time.Duration) error {
		if value <= 0 {
			return &ValidationError{Field: field, Message: "must be positive"}
		}
		return nil
	}
}

// DurationInRange returns a validator that checks if a duration is in range.
func DurationInRange(min, max time.Duration) Validator[time.Duration] {
	return func(field string, value time.Duration) error {
		if value < min || value > max {
			return &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("must be in range [%v, %v]", min, max),
			}
		}
		return nil
	}
}

// OneOf returns a validator that checks if a value is one of the allowed values.
func OneOf[T comparable](allowed ...T) Validator[T] {
	return func(field string, value T) error {
		for _, a := range allowed {
			if value == a {
				return nil
			}
		}
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be one of %v", allowed),
		}
	}
}

// NotNil returns a validator that checks if a pointer is not nil.
func NotNil[T any]() Validator[*T] {
	return func(field string, value *T) error {
		if value == nil {
			return &ValidationError{Field: field, Message: "must not be nil"}
		}
		return nil
	}
}

// Compose combines multiple validators into one.
func Compose[T any](validators ...Validator[T]) Validator[T] {
	return func(field string, value T) error {
		for _, v := range validators {
			if err := v(field, value); err != nil {
				return err
			}
		}
		return nil
	}
}

// ValidateAll runs all validations and returns all errors.
type ValidationErrors []error

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d validation errors: %v", len(e), e[0])
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Builder provides a fluent API for building validations.
type Builder struct {
	errors ValidationErrors
}

// NewBuilder creates a new validation builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Validate adds a validation to the builder.
func (b *Builder) Validate(err error) *Builder {
	if err != nil {
		b.errors = append(b.errors, err)
	}
	return b
}

// Build returns the validation errors or nil if no errors.
func (b *Builder) Build() error {
	if len(b.errors) == 0 {
		return nil
	}
	if len(b.errors) == 1 {
		return b.errors[0]
	}
	return b.errors
}
