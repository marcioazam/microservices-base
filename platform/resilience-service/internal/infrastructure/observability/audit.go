// Package observability provides OpenTelemetry-based observability implementations.
package observability

import (
	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// ValidateAuditEvent validates that an audit event has required fields.
func ValidateAuditEvent(event domain.AuditEvent) []string {
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

// HasRequiredAuditFields checks if an audit event has all required fields.
func HasRequiredAuditFields(event domain.AuditEvent) bool {
	return event.ID != "" &&
		event.Type != "" &&
		!event.Timestamp.IsZero() &&
		event.CorrelationID != ""
}
