// Package functional provides unified functional programming types for Go.
// It consolidates Option, Result, Either, and related types with a common
// Functor interface for consistent mapping operations.
package functional

// Functor represents types that can be mapped over.
// All functional types (Option, Result, Either) implement this interface.
type Functor[A any] interface {
	// Map applies a function to the wrapped value if present.
	Map(fn func(A) A) Functor[A]
}

// Mappable is a type constraint for types that support Map operations.
type Mappable[T any] interface {
	Option[T] | Result[T] | Either[error, T]
}

// MapFunc applies a transformation function to any Functor.
func MapFunc[A, B any, F Functor[A]](f F, fn func(A) B) any {
	switch v := any(f).(type) {
	case Option[A]:
		return MapOption(v, fn)
	case Result[A]:
		return MapResult(v, fn)
	default:
		return nil
	}
}

// Type conversion functions

// OptionToResult converts Option[T] to Result[T] with provided error for None.
func OptionToResult[T any](opt Option[T], err error) Result[T] {
	if opt.IsSome() {
		return Ok(opt.Unwrap())
	}
	return Err[T](err)
}

// ResultToOption converts Result[T] to Option[T], discarding error.
func ResultToOption[T any](res Result[T]) Option[T] {
	if res.IsOk() {
		return Some(res.Unwrap())
	}
	return None[T]()
}

// IdentityFunc is an identity function for functor law testing.
func IdentityFunc[T any](v T) T {
	return v
}

// ComposeFunc composes two functions for functor law testing.
func ComposeFunc[A, B, C any](f func(A) B, g func(B) C) func(A) C {
	return func(a A) C {
		return g(f(a))
	}
}
