// Package audit provides audit event logging with structured JSON output.
package audit

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// Logger handles audit event logging with structured JSON output.
type Logger struct {
	logger *slog.Logger
}

// Config holds logger configuration.
type Config struct {
	Output io.Writer // Output destination (defaults to os.Stdout)
	Level  slog.Level
}

// NewLogger creates a new audit logger with structured JSON output.
func NewLogger(cfg Config) *Logger {
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: cfg.Level,
	})

	return &Logger{
		logger: slog.New(handler),
	}
}

// NewLoggerWithHandler creates a new audit logger with a custom handler.
func NewLoggerWithHandler(handler slog.Handler) *Logger {
	return &Logger{
		logger: slog.New(handler),
	}
}

// Log logs an audit event with structured JSON output.
func (l *Logger) Log(event domain.AuditEvent) {
	// Ensure required fields using centralized UUID v7
	if event.ID == "" {
		event.ID = domain.GenerateEventID()
	}

	l.logger.Info("audit_event",
		slog.String("event_id", event.ID),
		slog.String("type", event.Type),
		slog.Time("timestamp", event.Timestamp),
		slog.String("correlation_id", event.CorrelationID),
		slog.String("spiffe_id", event.SpiffeID),
		slog.String("action", event.Action),
		slog.String("resource", event.Resource),
		slog.String("outcome", event.Outcome),
		slog.Any("metadata", event.Metadata),
	)
}

// LogWithContext logs an audit event with context for trace propagation.
func (l *Logger) LogWithContext(ctx context.Context, event domain.AuditEvent) {
	// Ensure required fields using centralized UUID v7
	if event.ID == "" {
		event.ID = domain.GenerateEventID()
	}

	l.logger.InfoContext(ctx, "audit_event",
		slog.String("event_id", event.ID),
		slog.String("type", event.Type),
		slog.Time("timestamp", event.Timestamp),
		slog.String("correlation_id", event.CorrelationID),
		slog.String("spiffe_id", event.SpiffeID),
		slog.String("action", event.Action),
		slog.String("resource", event.Resource),
		slog.String("outcome", event.Outcome),
		slog.Any("metadata", event.Metadata),
	)
}

// LogJSON logs an audit event as raw JSON.
func (l *Logger) LogJSON(event domain.AuditEvent) error {
	// Ensure required fields using centralized UUID v7
	if event.ID == "" {
		event.ID = domain.GenerateEventID()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	l.logger.Info(string(data))
	return nil
}

// LogResilienceEvent logs a resilience event.
func (l *Logger) LogResilienceEvent(event domain.ResilienceEvent) {
	l.logger.Info("resilience_event",
		slog.String("event_id", event.ID),
		slog.String("type", string(event.Type)),
		slog.String("service_name", event.ServiceName),
		slog.Time("timestamp", event.Timestamp),
		slog.String("correlation_id", event.CorrelationID),
		slog.String("trace_id", event.TraceID),
		slog.String("span_id", event.SpanID),
		slog.Any("metadata", event.Metadata),
	)
}

// Error logs an error with structured output.
func (l *Logger) Error(msg string, err error, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs)+1)
	args = append(args, slog.String("error", err.Error()))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	l.logger.Error(msg, args...)
}

// Warn logs a warning with structured output.
func (l *Logger) Warn(msg string, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	l.logger.Warn(msg, args...)
}

// Info logs an info message with structured output.
func (l *Logger) Info(msg string, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	l.logger.Info(msg, args...)
}

// Debug logs a debug message with structured output.
func (l *Logger) Debug(msg string, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	l.logger.Debug(msg, args...)
}

// ValidateEvent validates that an audit event has required fields.
func ValidateEvent(event domain.AuditEvent) []string {
	var missing []string

	if event.ID == "" {
		missing = append(missing, "id")
	}
	if event.Type == "" {
		missing = append(missing, "type")
	}
	if event.Timestamp.IsZero() {
		missing = append(missing, "timestamp")
	}
	if event.CorrelationID == "" {
		missing = append(missing, "correlation_id")
	}

	return missing
}

// HasRequiredFields checks if an audit event has all required fields.
func HasRequiredFields(event domain.AuditEvent) bool {
	return event.ID != "" &&
		event.Type != "" &&
		!event.Timestamp.IsZero() &&
		event.CorrelationID != ""
}
