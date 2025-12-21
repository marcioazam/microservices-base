package observability

import (
	"context"
	"log/slog"
	"os"
)

// LogLevel represents logging severity levels.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Logger interface for SDK logging.
type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}

// DefaultLogger implements Logger using slog.
type DefaultLogger struct {
	logger *slog.Logger
	level  LogLevel
}

// NewDefaultLogger creates a new default logger.
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	opts := &slog.HandlerOptions{
		Level: mapToSlogLevel(level),
	}
	handler := slog.NewJSONHandler(os.Stderr, opts)
	return &DefaultLogger{
		logger: slog.New(handler),
		level:  level,
	}
}

// NewLoggerWithHandler creates a logger with a custom handler.
func NewLoggerWithHandler(handler slog.Handler) *DefaultLogger {
	return &DefaultLogger{
		logger: slog.New(handler),
		level:  LogLevelInfo,
	}
}

func mapToSlogLevel(level LogLevel) slog.Level {
	switch level {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug logs a debug message.
func (l *DefaultLogger) Debug(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, filterSensitiveArgs(args)...)
}

// Info logs an info message.
func (l *DefaultLogger) Info(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, filterSensitiveArgs(args)...)
}

// Warn logs a warning message.
func (l *DefaultLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, filterSensitiveArgs(args)...)
}

// Error logs an error message.
func (l *DefaultLogger) Error(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, filterSensitiveArgs(args)...)
}

// filterSensitiveArgs filters out sensitive data from log arguments.
func filterSensitiveArgs(args []any) []any {
	filtered := make([]any, 0, len(args))
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			break
		}
		key, ok := args[i].(string)
		if !ok {
			filtered = append(filtered, args[i], args[i+1])
			continue
		}
		if IsSensitiveKey(key) {
			filtered = append(filtered, key, "[REDACTED]")
		} else {
			filtered = append(filtered, key, args[i+1])
		}
	}
	return filtered
}

// NopLogger is a no-op logger for testing or disabling logs.
type NopLogger struct{}

// Debug does nothing.
func (NopLogger) Debug(_ context.Context, _ string, _ ...any) {}

// Info does nothing.
func (NopLogger) Info(_ context.Context, _ string, _ ...any) {}

// Warn does nothing.
func (NopLogger) Warn(_ context.Context, _ string, _ ...any) {}

// Error does nothing.
func (NopLogger) Error(_ context.Context, _ string, _ ...any) {}

// LogTokenValidation logs token validation events.
func LogTokenValidation(l Logger, ctx context.Context, success bool, err error) {
	if success {
		l.Debug(ctx, "token validation successful")
	} else {
		l.Warn(ctx, "token validation failed", "error", err.Error())
	}
}

// LogTokenRefresh logs token refresh events.
func LogTokenRefresh(l Logger, ctx context.Context, success bool, err error) {
	if success {
		l.Info(ctx, "token refreshed successfully")
	} else {
		l.Error(ctx, "token refresh failed", "error", err.Error())
	}
}

// LogJWKSFetch logs JWKS fetch events.
func LogJWKSFetch(l Logger, ctx context.Context, uri string, success bool, err error) {
	if success {
		l.Debug(ctx, "JWKS fetched successfully", "uri_host", SanitizeURI(uri))
	} else {
		l.Warn(ctx, "JWKS fetch failed", "uri_host", SanitizeURI(uri), "error", err.Error())
	}
}
