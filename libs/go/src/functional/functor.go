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
