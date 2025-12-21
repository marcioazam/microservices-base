package functional

import (
	"errors"
	"iter"
)

// NewError creates a new error with the given message.
func NewError(msg string) error {
	return errors.New(msg)
}

// Result represents the outcome of an operation that may fail.
// It contains either a success value or an error.
type Result[T any] struct {
	value T
	err   error
	ok    bool
}

// Ok creates a successful Result.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value, ok: true}
}

// Err creates a failed Result.
func Err[T any](err error) Result[T] {
	return Result[T]{err: err, ok: false}
}

// IsOk returns true if the Result is successful.
func (r Result[T]) IsOk() bool {
	return r.ok
}

// IsErr returns true if the Result is an error.
func (r Result[T]) IsErr() bool {
	return !r.ok
}

// Unwrap returns the success value or panics on error.
func (r Result[T]) Unwrap() T {
	if !r.ok {
		panic("called Unwrap on Err: " + r.err.Error())
	}
	return r.value
}

// UnwrapErr returns the error or panics on success.
func (r Result[T]) UnwrapErr() error {
	if r.ok {
		panic("called UnwrapErr on Ok")
	}
	return r.err
}

// UnwrapOr returns the success value or a default.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.ok {
		return r.value
	}
	return defaultValue
}

// UnwrapOrElse returns the success value or computes a default from error.
func (r Result[T]) UnwrapOrElse(fn func(error) T) T {
	if r.ok {
		return r.value
	}
	return fn(r.err)
}

// Map applies a function to the success value.
func (r Result[T]) Map(fn func(T) T) Functor[T] {
	if r.ok {
		return Ok(fn(r.value))
	}
	return Err[T](r.err)
}

// MapResult applies a transformation function to Result.
func MapResult[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.ok {
		return Ok(fn(r.value))
	}
	return Err[U](r.err)
}

// MapErr applies a function to the error.
func MapResultErr[T any](r Result[T], fn func(error) error) Result[T] {
	if r.ok {
		return r
	}
	return Err[T](fn(r.err))
}

// FlatMap applies a function that returns a Result.
func FlatMapResult[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.ok {
		return fn(r.value)
	}
	return Err[U](r.err)
}

// Match executes one of two functions based on Result state.
func (r Result[T]) Match(onOk func(T), onErr func(error)) {
	if r.ok {
		onOk(r.value)
	} else {
		onErr(r.err)
	}
}

// MatchReturn executes one of two functions and returns the result.
func MatchResult[T, U any](r Result[T], onOk func(T) U, onErr func(error) U) U {
	if r.ok {
		return onOk(r.value)
	}
	return onErr(r.err)
}

// ToOption converts Result to Option, discarding error.
func (r Result[T]) ToOption() Option[T] {
	if r.ok {
		return Some(r.value)
	}
	return None[T]()
}

// All returns a Go 1.23+ iterator over the Result (0 or 1 element).
func (r Result[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if r.ok {
			yield(r.value)
		}
	}
}

// Collect returns the Result value as a slice (empty if error).
func (r Result[T]) Collect() []T {
	if r.ok {
		return []T{r.value}
	}
	return []T{}
}

// FromOption creates a Result from an Option.
func FromOption[T any](o Option[T], err error) Result[T] {
	if o.IsSome() {
		return Ok(o.Unwrap())
	}
	return Err[T](err)
}

// Try wraps a function that may return an error.
func Try[T any](fn func() (T, error)) Result[T] {
	value, err := fn()
	if err != nil {
		return Err[T](err)
	}
	return Ok(value)
}

// TryFunc wraps a function call with error handling.
func TryFunc[T any](value T, err error) Result[T] {
	if err != nil {
		return Err[T](err)
	}
	return Ok(value)
}
