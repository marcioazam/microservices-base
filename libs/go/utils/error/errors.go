package error

import "errors"

// Re-export standard library error functions for convenience.
var (
	// New returns an error that formats as the given text.
	New = errors.New

	// Is reports whether any error in err's tree matches target.
	Is = errors.Is

	// As finds the first error in err's tree that matches target.
	As = errors.As

	// Unwrap returns the result of calling the Unwrap method on err.
	Unwrap = errors.Unwrap

	// Join returns an error that wraps the given errors.
	Join = errors.Join
)
