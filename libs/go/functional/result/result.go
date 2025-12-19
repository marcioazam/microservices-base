// Package result provides backward compatibility aliases.
// Deprecated: Use github.com/authcorp/libs/go/src/functional instead.
package result

import "github.com/authcorp/libs/go/src/functional"

// Result is an alias for functional.Result.
// Deprecated: Use functional.Result instead.
type Result[T any] = functional.Result[T]

// Ok creates an Ok value.
// Deprecated: Use functional.Ok instead.
func Ok[T any](value T) Result[T] {
	return functional.Ok(value)
}

// Err creates an Err value.
// Deprecated: Use functional.Err instead.
func Err[T any](err error) Result[T] {
	return functional.Err[T](err)
}
