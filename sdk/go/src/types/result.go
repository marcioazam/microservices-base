// Package types provides generic functional types for the SDK.
package types

// Result represents the outcome of an operation that may fail.
type Result[T any] struct {
	value T
	err   error
	ok    bool
}

// Ok creates a successful Result with the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value, ok: true}
}

// Err creates a failed Result with the given error.
func Err[T any](err error) Result[T] {
	return Result[T]{err: err, ok: false}
}

// IsOk returns true if the Result is successful.
func (r Result[T]) IsOk() bool {
	return r.ok
}

// IsErr returns true if the Result is a failure.
func (r Result[T]) IsErr() bool {
	return !r.ok
}

// Unwrap returns the value if successful, panics otherwise.
func (r Result[T]) Unwrap() T {
	if !r.ok {
		panic("called Unwrap on an Err Result")
	}
	return r.value
}

// UnwrapOr returns the value if successful, or the default value otherwise.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if !r.ok {
		return defaultValue
	}
	return r.value
}

// UnwrapErr returns the error if failed, panics otherwise.
func (r Result[T]) UnwrapErr() error {
	if r.ok {
		panic("called UnwrapErr on an Ok Result")
	}
	return r.err
}

// Error returns the error if present, nil otherwise.
func (r Result[T]) Error() error {
	return r.err
}

// Value returns the value and a boolean indicating success.
func (r Result[T]) Value() (T, bool) {
	return r.value, r.ok
}

// Match applies the appropriate function based on success/failure.
func (r Result[T]) Match(onOk func(T), onErr func(error)) {
	if r.ok {
		onOk(r.value)
	} else {
		onErr(r.err)
	}
}

// MatchReturn applies the appropriate function and returns the result.
func MatchReturn[T, U any](r Result[T], onOk func(T) U, onErr func(error) U) U {
	if r.ok {
		return onOk(r.value)
	}
	return onErr(r.err)
}

// Map transforms the value if successful, preserving errors.
func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if !r.ok {
		return Err[U](r.err)
	}
	return Ok(fn(r.value))
}

// FlatMap chains operations that may fail.
func FlatMap[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if !r.ok {
		return Err[U](r.err)
	}
	return fn(r.value)
}

// MapErr transforms the error if failed, preserving success.
func MapErr[T any](r Result[T], fn func(error) error) Result[T] {
	if r.ok {
		return r
	}
	return Err[T](fn(r.err))
}

// And returns the other Result if this one is successful.
func And[T, U any](r Result[T], other Result[U]) Result[U] {
	if !r.ok {
		return Err[U](r.err)
	}
	return other
}

// Or returns this Result if successful, otherwise returns the other.
func Or[T any](r Result[T], other Result[T]) Result[T] {
	if r.ok {
		return r
	}
	return other
}
