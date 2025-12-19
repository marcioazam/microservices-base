// Package validator provides a generic composable validator.
package validator

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Rule    string
	Message string
}

// ValidationResult contains validation results.
type ValidationResult struct {
	Errors []ValidationError
}

// IsValid returns true if validation passed.
func (r ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// Error returns the first error message or empty string.
func (r ValidationResult) Error() string {
	if len(r.Errors) == 0 {
		return ""
	}
	return r.Errors[0].Message
}

// Messages returns all error messages.
func (r ValidationResult) Messages() []string {
	msgs := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		msgs[i] = e.Message
	}
	return msgs
}

// rule represents a validation rule.
type rule[T any] struct {
	name    string
	check   func(T) bool
	message string
}

// Validator validates values of type T.
type Validator[T any] struct {
	rules []rule[T]
}

// New creates a new Validator.
func New[T any]() *Validator[T] {
	return &Validator[T]{rules: make([]rule[T], 0)}
}

// Rule adds a validation rule.
func (v *Validator[T]) Rule(name string, check func(T) bool, message string) *Validator[T] {
	v.rules = append(v.rules, rule[T]{name: name, check: check, message: message})
	return v
}

// Validate validates a value.
func (v *Validator[T]) Validate(value T) ValidationResult {
	result := ValidationResult{Errors: make([]ValidationError, 0)}
	for _, r := range v.rules {
		if !r.check(value) {
			result.Errors = append(result.Errors, ValidationError{
				Rule:    r.name,
				Message: r.message,
			})
		}
	}
	return result
}

// And combines two validators.
func (v *Validator[T]) And(other *Validator[T]) *Validator[T] {
	v.rules = append(v.rules, other.rules...)
	return v
}

// Field validates a field of the value.
func Field[T, F any](v *Validator[T], name string, getter func(T) F, fieldValidator *Validator[F]) *Validator[T] {
	for _, r := range fieldValidator.rules {
		ruleCopy := r
		v.rules = append(v.rules, rule[T]{
			name: name + "." + ruleCopy.name,
			check: func(t T) bool {
				return ruleCopy.check(getter(t))
			},
			message: name + ": " + ruleCopy.message,
		})
	}
	return v
}

// ForEach creates a validator for slices.
func ForEach[T any](itemValidator *Validator[T]) *Validator[[]T] {
	return New[[]T]().Rule("items", func(items []T) bool {
		for _, item := range items {
			if !itemValidator.Validate(item).IsValid() {
				return false
			}
		}
		return true
	}, "one or more items failed validation")
}

// Required creates a rule that checks for non-zero value.
func Required[T comparable]() *Validator[T] {
	var zero T
	return New[T]().Rule("required", func(v T) bool {
		return v != zero
	}, "value is required")
}

// MinLength creates a rule for minimum string length.
func MinLength(min int) *Validator[string] {
	return New[string]().Rule("minLength", func(s string) bool {
		return len(s) >= min
	}, "value is too short")
}

// MaxLength creates a rule for maximum string length.
func MaxLength(max int) *Validator[string] {
	return New[string]().Rule("maxLength", func(s string) bool {
		return len(s) <= max
	}, "value is too long")
}

// Min creates a rule for minimum numeric value.
func Min[T interface{ ~int | ~int64 | ~float64 }](min T) *Validator[T] {
	return New[T]().Rule("min", func(v T) bool {
		return v >= min
	}, "value is too small")
}

// Max creates a rule for maximum numeric value.
func Max[T interface{ ~int | ~int64 | ~float64 }](max T) *Validator[T] {
	return New[T]().Rule("max", func(v T) bool {
		return v <= max
	}, "value is too large")
}

// Range creates a rule for value in range.
func Range[T interface{ ~int | ~int64 | ~float64 }](min, max T) *Validator[T] {
	return New[T]().Rule("range", func(v T) bool {
		return v >= min && v <= max
	}, "value is out of range")
}

// OneOf creates a rule for value in set.
func OneOf[T comparable](values ...T) *Validator[T] {
	set := make(map[T]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return New[T]().Rule("oneOf", func(v T) bool {
		_, ok := set[v]
		return ok
	}, "value is not in allowed set")
}

// Custom creates a custom validation rule.
func Custom[T any](name string, check func(T) bool, message string) *Validator[T] {
	return New[T]().Rule(name, check, message)
}
