// Package audit provides audit event logging.
package audit

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// Logger handles audit event logging.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new audit logger.
func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

// Log logs an audit event.
func (l *Logger) Log(event domain.AuditEvent) {
	// Ensure required fields
	if event.ID == "" {
		event.ID = resilience.GenerateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
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

// LogJSON logs an audit event as JSON.
func (l *Logger) LogJSON(event domain.AuditEvent) error {
	// Ensure required fields
	if event.ID == "" {
		event.ID = resilience.GenerateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	l.logger.Info(string(data))
	return nil
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
