// Package timeout implements timeout management with context cancellation.
// Re-exports from libs/go/resilience/timeout for backward compatibility.
package timeout

import (
	libto "github.com/auth-platform/libs/go/resilience/timeout"
)

// Manager implements the TimeoutManager interface.
// Re-exported from libs/go/resilience/timeout for backward compatibility.
type Manager = libto.Manager

// Config holds timeout manager creation options.
// Re-exported from libs/go/resilience/timeout for backward compatibility.
type Config = libto.Config

// New creates a new timeout manager.
// Re-exported from libs/go/resilience/timeout for backward compatibility.
var New = libto.New
