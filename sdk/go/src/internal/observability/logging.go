package observability

import (
	"context"
	"log/slog"
	"os"
)

// Logger provides structured logging for the SDK.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new logger with default settings.
func NewLogger() *Logger {
	return &Logger{
		logger: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// NewLoggerWithHandler creates a logger with a custom handler.
func NewLoggerWithHandler(handler slog.Handler) *Logger {
	return &Logger{logger: slog.New(handler)}
}

// NewLoggerWithLevel creates a logger with a specific level.
func NewLoggerWithLevel(level slog.Level) *Logger {
	return &Logger{
		logger: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})),
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, l.filterArgs(args)...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, l.filterArgs(args)...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, l.filterArgs(args)...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, l.filterArgs(args)...)
}

// DebugContext logs a debug message with context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, l.filterArgs(args)...)
}

// InfoContext logs an info message with context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, l.filterArgs(args)...)
}

// WarnContext logs a warning message with context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, l.filterArgs(args)...)
}

// ErrorContext logs an error message with context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, l.filterArgs(args)...)
}

// With returns a logger with additional attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{logger: l.logger.With(l.filterArgs(args)...)}
}

// filterArgs filters sensitive data from log arguments.
func (l *Logger) filterArgs(args []any) []any {
	filtered := make([]any, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			filtered[i] = FilterSensitiveData(v)
		default:
			filtered[i] = arg
		}
	}
	return filtered
}

// LogTokenValidation logs a token validation event.
func (l *Logger) LogTokenValidation(ctx context.Context, success bool, err error) {
	if success {
		l.InfoContext(ctx, "token validation successful")
	} else {
		l.WarnContext(ctx, "token validation failed", "error", err)
	}
}

// LogJWKSRefresh logs a JWKS refresh event.
func (l *Logger) LogJWKSRefresh(ctx context.Context, uri string, success bool) {
	if success {
		l.DebugContext(ctx, "JWKS refreshed", "uri", uri)
	} else {
		l.WarnContext(ctx, "JWKS refresh failed", "uri", uri)
	}
}

// LogRetry logs a retry event.
func (l *Logger) LogRetry(ctx context.Context, attempt int, delay string) {
	l.DebugContext(ctx, "retrying operation", "attempt", attempt, "delay", delay)
}
