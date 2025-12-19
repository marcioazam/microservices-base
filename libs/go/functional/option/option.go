// Package option provides backward compatibility aliases.
// Deprecated: Use github.com/authcorp/libs/go/src/functional instead.
package option

import "github.com/authcorp/libs/go/src/functional"

// Option is an alias for functional.Option.
// Deprecated: Use functional.Option instead.
type Option[T any] = functional.Option[T]

// Some creates a Some value.
// Deprecated: Use functional.Some instead.
func Some[T any](value T) Option[T] {
	return functional.Some(value)
}

// None creates a None value.
// Deprecated: Use functional.None instead.
func None[T any]() Option[T] {
	return functional.None[T]()
}
