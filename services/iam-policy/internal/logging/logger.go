// Package logging provides logging integration for IAM Policy Service.
package logging

import (
	"context"

	"github.com/auth-platform/iam-policy-service/internal/config"
	"github.com/authcorp/libs/go/src/logging"
	"github.com/authcorp/libs/go/src/observability"
)

// Logger wraps the logging client with IAM Policy Service specific functionality.
type Logger struct {
	client *logging.Client
}

// NewLogger creates a new logger from configuration.
func NewLogger(cfg config.LoggingConfig) (*Logger, error) {
	level := parseLevel(cfg.MinLevel)

	clientCfg := logging.ClientConfig{
		Address:       cfg.Address,
		ServiceName:   cfg.ServiceName,
		MinLevel:      level,
		LocalFallback: cfg.LocalFallback,
		BufferSize:    cfg.BufferSize,
		FlushInterval: cfg.FlushInterval,
	}

	client, err := logging.NewClient(clientCfg)
	if err != nil {
		// Fallback to local-only logging
		client = logging.LocalOnly(cfg.ServiceName)
	}

	return &Logger{client: client}, nil
}

// NewLocalLogger creates a local-only logger for testing.
func NewLocalLogger(serviceName string) *Logger {
	return &Logger{client: logging.LocalOnly(serviceName)}
}

// Debug logs at debug level.
func (l *Logger) Debug(ctx context.Context, msg string, fields ...logging.Field) {
	l.client.Debug(ctx, msg, fields...)
}

// Info logs at info level.
func (l *Logger) Info(ctx context.Context, msg string, fields ...logging.Field) {
	l.client.Info(ctx, msg, fields...)
}

// Warn logs at warn level.
func (l *Logger) Warn(ctx context.Context, msg string, fields ...logging.Field) {
	l.client.Warn(ctx, msg, fields...)
}

// Error logs at error level.
func (l *Logger) Error(ctx context.Context, msg string, fields ...logging.Field) {
	l.client.Error(ctx, msg, fields...)
}

// With returns a new logger with additional fields.
func (l *Logger) With(fields ...logging.Field) *Logger {
	return &Logger{client: l.client.With(fields...)}
}

// FromContext creates a logger enriched with context values.
func (l *Logger) FromContext(ctx context.Context) *Logger {
	return &Logger{client: l.client.FromContext(ctx)}
}

// WithCorrelationID adds correlation ID to the logger.
func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	return l.With(logging.String("correlation_id", correlationID))
}

// WithTraceContext adds trace context to the logger.
func (l *Logger) WithTraceContext(ctx context.Context) *Logger {
	traceID, spanID := observability.TraceContextFromContext(ctx)
	fields := make([]logging.Field, 0, 2)
	if traceID != "" {
		fields = append(fields, logging.String("trace_id", traceID))
	}
	if spanID != "" {
		fields = append(fields, logging.String("span_id", spanID))
	}
	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}

// Flush flushes buffered logs.
func (l *Logger) Flush() error {
	return l.client.Flush()
}

// Close closes the logger.
func (l *Logger) Close() error {
	return l.client.Close()
}

// Client returns the underlying logging client.
func (l *Logger) Client() *logging.Client {
	return l.client
}

func parseLevel(level string) logging.Level {
	switch level {
	case "debug":
		return logging.LevelDebug
	case "info":
		return logging.LevelInfo
	case "warn":
		return logging.LevelWarn
	case "error":
		return logging.LevelError
	default:
		return logging.LevelInfo
	}
}

// Field re-exports for convenience.
type Field = logging.Field

var (
	String   = logging.String
	Int      = logging.Int
	Int64    = logging.Int64
	Float64  = logging.Float64
	Bool     = logging.Bool
	Duration = logging.Duration
	Error    = logging.Error
	Any      = logging.Any
)
