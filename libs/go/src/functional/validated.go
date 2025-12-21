// Package functional provides validated applicative functor for error accumulation.
package functional

// Validated represents a validation result that accumulates errors.
type Validated[E, A any] struct {
	value  A
	errors []E
	valid  bool
}

// Valid creates a valid result.
func Valid[E, A any](value A) Validated[E, A] {
	return Validated[E, A]{value: value, valid: true}
}

// Invalid creates an invalid result with errors.
func Invalid[E, A any](errors ...E) Validated[E, A] {
	return Validated[E, A]{errors: errors, valid: false}
}

// IsValid returns true if the validation passed.
func (v Validated[E, A]) IsValid() bool {
	return v.valid
}

// IsInvalid returns true if the validation failed.
func (v Validated[E, A]) IsInvalid() bool {
	return !v.valid
}

// GetValue returns the value (panics if invalid).
func (v Validated[E, A]) GetValue() A {
	if !v.valid {
		panic("cannot get value from invalid Validated")
	}
	return v.value
}

// GetErrors returns the errors (empty if valid).
func (v Validated[E, A]) GetErrors() []E {
	return v.errors
}

// GetOrElse returns the value or a default if invalid.
func (v Validated[E, A]) GetOrElse(defaultVal A) A {
	if v.valid {
		return v.value
	}
	return defaultVal
}

// MapValidated applies a function to the value if valid.
func MapValidated[E, A, B any](v Validated[E, A], fn func(A) B) Validated[E, B] {
	if !v.valid {
		return Validated[E, B]{errors: v.errors, valid: false}
	}
	return Validated[E, B]{value: fn(v.value), valid: true}
}

// MapValidatedErrors applies a function to the errors if invalid.
func MapValidatedErrors[E, F, A any](v Validated[E, A], fn func(E) F) Validated[F, A] {
	if v.valid {
		return Validated[F, A]{value: v.value, valid: true}
	}
	newErrors := make([]F, len(v.errors))
	for i, e := range v.errors {
		newErrors[i] = fn(e)
	}
	return Validated[F, A]{errors: newErrors, valid: false}
}

// CombineValidated combines two validated values, accumulating errors.
func CombineValidated[E, A, B, C any](va Validated[E, A], vb Validated[E, B], fn func(A, B) C) Validated[E, C] {
	if va.valid && vb.valid {
		return Valid[E, C](fn(va.value, vb.value))
	}
	errors := make([]E, 0, len(va.errors)+len(vb.errors))
	errors = append(errors, va.errors...)
	errors = append(errors, vb.errors...)
	return Invalid[E, C](errors...)
}

// CombineValidated3 combines three validated values.
func CombineValidated3[E, A, B, C, D any](va Validated[E, A], vb Validated[E, B], vc Validated[E, C], fn func(A, B, C) D) Validated[E, D] {
	if va.valid && vb.valid && vc.valid {
		return Valid[E, D](fn(va.value, vb.value, vc.value))
	}
	errors := make([]E, 0, len(va.errors)+len(vb.errors)+len(vc.errors))
	errors = append(errors, va.errors...)
	errors = append(errors, vb.errors...)
	errors = append(errors, vc.errors...)
	return Invalid[E, D](errors...)
}

// ValidatedToResult converts to a Result, using the first error if invalid.
func ValidatedToResult[A any](v Validated[error, A]) Result[A] {
	if v.valid {
		return Ok(v.value)
	}
	if len(v.errors) > 0 {
		return Err[A](v.errors[0])
	}
	return Err[A](NewError("validation failed"))
}

// ResultToValidated converts a Result to a Validated.
func ResultToValidated[A any](r Result[A]) Validated[error, A] {
	if r.IsOk() {
		return Valid[error, A](r.Unwrap())
	}
	return Invalid[error, A](r.UnwrapErr())
}

// SequenceValidated converts a slice of Validated to a Validated of slice.
func SequenceValidated[E, A any](vs []Validated[E, A]) Validated[E, []A] {
	values := make([]A, 0, len(vs))
	errors := make([]E, 0)
	allValid := true

	for _, v := range vs {
		if v.valid {
			values = append(values, v.value)
		} else {
			allValid = false
			errors = append(errors, v.errors...)
		}
	}

	if allValid {
		return Valid[E, []A](values)
	}
	return Invalid[E, []A](errors...)
}

// TraverseValidated applies a function to each element and sequences the results.
func TraverseValidated[E, A, B any](items []A, fn func(A) Validated[E, B]) Validated[E, []B] {
	results := make([]Validated[E, B], len(items))
	for i, item := range items {
		results[i] = fn(item)
	}
	return SequenceValidated(results)
}

// FoldValidated applies one of two functions based on validity.
func FoldValidated[E, A, B any](v Validated[E, A], onInvalid func([]E) B, onValid func(A) B) B {
	if v.valid {
		return onValid(v.value)
	}
	return onInvalid(v.errors)
}
