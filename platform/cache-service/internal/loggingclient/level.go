package loggingclient

// Level represents log severity levels.
type Level int32

const (
	// LevelUnspecified is the default unspecified level.
	LevelUnspecified Level = 0
	// LevelDebug is for debug messages.
	LevelDebug Level = 1
	// LevelInfo is for informational messages.
	LevelInfo Level = 2
	// LevelWarn is for warning messages.
	LevelWarn Level = 3
	// LevelError is for error messages.
	LevelError Level = 4
	// LevelFatal is for fatal messages.
	LevelFatal Level = 5
)

// String returns the string representation of the level.
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
	case LevelFatal:
		return "FATAL"
	default:
		return "UNSPECIFIED"
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
	case "FATAL", "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}
