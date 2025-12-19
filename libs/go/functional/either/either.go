// Package either provides backward compatibility aliases.
// Deprecated: Use github.com/authcorp/libs/go/src/functional instead.
package either

import "github.com/authcorp/libs/go/src/functional"

// Either is an alias for functional.Either.
// Deprecated: Use functional.Either instead.
type Either[L, R any] = functional.Either[L, R]

// Left creates a Left value.
// Deprecated: Use functional.Left instead.
func Left[L, R any](value L) Either[L, R] {
	return functional.Left[L, R](value)
}

// Right creates a Right value.
// Deprecated: Use functional.Right instead.
func Right[L, R any](value R) Either[L, R] {
	return functional.Right[L](value)
}
