package logging

// Level represents log severity.
type Level int

const (
	// LevelDebug is for debug messages.
	LevelDebug Level = iota
	// LevelInfo is for informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
)

// String returns the string representation of Level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string into a Level.
func ParseLevel(s string) Level {
	switch s {
	case "DEBUG", "debug":
		return LevelDebug
	case "INFO", "info":
		return LevelInfo
	case "WARN", "warn", "WARNING", "warning":
		return LevelWarn
	case "ERROR", "error":
		return LevelError
	default:
		return LevelInfo
	}
}
