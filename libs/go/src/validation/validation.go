// Package validation provides composable validation with error accumulation.
// This is the single authoritative validation implementation (consolidated from utils/).
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Value   any    `json:"value,omitempty"`
}

func (e ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Result represents validation result with error accumulation.
type Result struct {
	errors []ValidationError
}

// NewResult creates an empty validation result.
func NewResult() *Result {
	return &Result{}
}

// AddError adds a validation error.
func (r *Result) AddError(err ValidationError) *Result {
	r.errors = append(r.errors, err)
	return r
}

// AddFieldError adds a validation error for a specific field.
func (r *Result) AddFieldError(field, message, code string) *Result {
	return r.AddError(ValidationError{Field: field, Message: message, Code: code})
}

// Merge combines another result into this one.
func (r *Result) Merge(other *Result) *Result {
	if other != nil {
		r.errors = append(r.errors, other.errors...)
	}
	return r
}

// IsValid returns true if no errors.
func (r *Result) IsValid() bool {
	return len(r.errors) == 0
}

// Errors returns all validation errors.
func (r *Result) Errors() []ValidationError {
	return r.errors
}

// ErrorMap returns errors grouped by field.
func (r *Result) ErrorMap() map[string][]string {
	m := make(map[string][]string)
	for _, e := range r.errors {
		key := e.Field
		if e.Path != "" {
			key = e.Path
		}
		m[key] = append(m[key], e.Message)
	}
	return m
}

// ErrorMessages returns all error messages as strings.
func (r *Result) ErrorMessages() []string {
	msgs := make([]string, len(r.errors))
	for i, e := range r.errors {
		msgs[i] = e.Error()
	}
	return msgs
}

// Validator is a function that validates a value.
type Validator[T any] func(T) *ValidationError

// And combines validators with AND logic (all must pass).
func And[T any](validators ...Validator[T]) Validator[T] {
	return func(v T) *ValidationError {
		for _, validator := range validators {
			if err := validator(v); err != nil {
				return err
			}
		}
		return nil
	}
}

// Or combines validators with OR logic (at least one must pass).
func Or[T any](validators ...Validator[T]) Validator[T] {
	return func(v T) *ValidationError {
		var lastErr *ValidationError
		for _, validator := range validators {
			if err := validator(v); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		return lastErr
	}
}

// Not negates a validator.
func Not[T any](validator Validator[T], message, code string) Validator[T] {
	return func(v T) *ValidationError {
		if err := validator(v); err == nil {
			return &ValidationError{Message: message, Code: code}
		}
		return nil
	}
}

// Field validates a field value and tracks the field path.
func Field[T any](field string, value T, validators ...Validator[T]) *Result {
	result := NewResult()
	for _, validator := range validators {
		if err := validator(value); err != nil {
			err.Field = field
			if err.Path == "" {
				err.Path = field
			}
			result.AddError(*err)
		}
	}
	return result
}

// NestedField validates a nested field with path tracking.
func NestedField[T any](parentPath, field string, value T, validators ...Validator[T]) *Result {
	result := NewResult()
	path := field
	if parentPath != "" {
		path = parentPath + "." + field
	}
	for _, validator := range validators {
		if err := validator(value); err != nil {
			err.Field = field
			err.Path = path
			result.AddError(*err)
		}
	}
	return result
}

// ValidateAll runs all validators and accumulates errors.
func ValidateAll[T any](value T, validators ...Validator[T]) *Result {
	result := NewResult()
	for _, validator := range validators {
		if err := validator(value); err != nil {
			result.AddError(*err)
		}
	}
	return result
}

// String validators

// Required checks that string is not empty.
func Required() Validator[string] {
	return func(s string) *ValidationError {
		if strings.TrimSpace(s) == "" {
			return &ValidationError{Message: "is required", Code: "required"}
		}
		return nil
	}
}

// MinLength checks minimum string length.
func MinLength(min int) Validator[string] {
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
func MaxLength(max int) Validator[string] {
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

// MatchesRegex checks string matches pattern.
func MatchesRegex(pattern *regexp.Regexp, message string) Validator[string] {
	return func(s string) *ValidationError {
		if !pattern.MatchString(s) {
			return &ValidationError{Message: message, Code: "pattern"}
		}
		return nil
	}
}

// OneOf checks value is one of allowed values.
func OneOf[T comparable](allowed ...T) Validator[T] {
	return func(v T) *ValidationError {
		for _, a := range allowed {
			if v == a {
				return nil
			}
		}
		return &ValidationError{Message: "must be one of allowed values", Code: "one_of"}
	}
}

// Numeric validators

// Positive checks value is positive.
func Positive[T ~int | ~int64 | ~float64]() Validator[T] {
	return func(v T) *ValidationError {
		if v <= 0 {
			return &ValidationError{Message: "must be positive", Code: "positive"}
		}
		return nil
	}
}

// NonZero checks value is not zero.
func NonZero[T ~int | ~int64 | ~float64]() Validator[T] {
	return func(v T) *ValidationError {
		if v == 0 {
			return &ValidationError{Message: "must not be zero", Code: "non_zero"}
		}
		return nil
	}
}

// InRange checks value is within range.
func InRange[T ~int | ~int64 | ~float64](min, max T) Validator[T] {
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

// Min checks minimum numeric value.
func Min[T ~int | ~int64 | ~float64](min T) Validator[T] {
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
func Max[T ~int | ~int64 | ~float64](max T) Validator[T] {
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

// Collection validators

// MinSize checks minimum collection size.
func MinSize[T any](min int) Validator[[]T] {
	return func(s []T) *ValidationError {
		if len(s) < min {
			return &ValidationError{
				Message: fmt.Sprintf("must have at least %d items", min),
				Code:    "min_size",
			}
		}
		return nil
	}
}

// MaxSize checks maximum collection size.
func MaxSize[T any](max int) Validator[[]T] {
	return func(s []T) *ValidationError {
		if len(s) > max {
			return &ValidationError{
				Message: fmt.Sprintf("must have at most %d items", max),
				Code:    "max_size",
			}
		}
		return nil
	}
}

// UniqueElements checks all elements are unique.
func UniqueElements[T comparable]() Validator[[]T] {
	return func(s []T) *ValidationError {
		seen := make(map[T]bool)
		for _, v := range s {
			if seen[v] {
				return &ValidationError{Message: "must have unique elements", Code: "unique"}
			}
			seen[v] = true
		}
		return nil
	}
}

// Custom creates a custom validator.
func Custom[T any](check func(T) bool, message, code string) Validator[T] {
	return func(v T) *ValidationError {
		if !check(v) {
			return &ValidationError{Message: message, Code: code}
		}
		return nil
	}
}
