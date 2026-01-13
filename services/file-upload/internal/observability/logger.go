package observability

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	tenantIDKey      contextKey = "tenant_id"
	userIDKey        contextKey = "user_id"
)

// Logger wraps zerolog for structured JSON logging
type Logger struct {
	zl zerolog.Logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Level         string    `json:"level"`
	Message       string    `json:"message"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	TenantID      string    `json:"tenant_id,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	Component     string    `json:"component,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// NewLogger creates a new structured logger
func NewLogger(level string) *Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano
	
	zl := zerolog.New(os.Stdout).
		Level(lvl).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{zl: zl}
}

// NewLoggerWithWriter creates a logger with custom writer (for testing)
func NewLoggerWithWriter(w io.Writer, level string) *Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano
	
	zl := zerolog.New(w).
		Level(lvl).
		With().
		Timestamp().
		Logger()

	return &Logger{zl: zl}
}

// WithContext returns a logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	zl := l.zl.With().Logger()

	if correlationID, ok := ctx.Value(correlationIDKey).(string); ok && correlationID != "" {
		zl = zl.With().Str("correlation_id", correlationID).Logger()
	}
	if tenantID, ok := ctx.Value(tenantIDKey).(string); ok && tenantID != "" {
		zl = zl.With().Str("tenant_id", tenantID).Logger()
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		zl = zl.With().Str("user_id", userID).Logger()
	}

	return &Logger{zl: zl}
}

// WithComponent returns a logger with component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		zl: l.zl.With().Str("component", component).Logger(),
	}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		zl: l.zl.With().Interface(key, value).Logger(),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.zl.Debug().Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.zl.Info().Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.zl.Warn().Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error) {
	l.zl.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error) {
	l.zl.Fatal().Err(err).Msg(msg)
}

// Context helpers

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// WithTenantID adds tenant ID to context
func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// GetCorrelationID retrieves correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// GetTenantID retrieves tenant ID from context
func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(tenantIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
