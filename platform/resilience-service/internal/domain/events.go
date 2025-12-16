package domain

import "time"

// ResilienceEventType represents the type of resilience event.
type ResilienceEventType string

const (
	EventCircuitStateChange ResilienceEventType = "circuit_state_change"
	EventRetryAttempt       ResilienceEventType = "retry_attempt"
	EventTimeout            ResilienceEventType = "timeout"
	EventRateLimitHit       ResilienceEventType = "rate_limit_hit"
	EventBulkheadRejection  ResilienceEventType = "bulkhead_rejection"
	EventHealthChange       ResilienceEventType = "health_change"
)

// ResilienceEvent represents an event for observability.
type ResilienceEvent struct {
	ID            string              `json:"id"`
	Type          ResilienceEventType `json:"type"`
	ServiceName   string              `json:"service_name"`
	Timestamp     time.Time           `json:"timestamp"`
	CorrelationID string              `json:"correlation_id"`
	TraceID       string              `json:"trace_id,omitempty"`
	SpanID        string              `json:"span_id,omitempty"`
	Metadata      map[string]any      `json:"metadata,omitempty"`
}

// AuditEvent represents an audit event for security logging.
type AuditEvent struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Timestamp     time.Time      `json:"timestamp"`
	CorrelationID string         `json:"correlation_id"`
	SpiffeID      string         `json:"spiffe_id,omitempty"`
	Action        string         `json:"action"`
	Resource      string         `json:"resource"`
	Outcome       string         `json:"outcome"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// EventEmitter emits resilience events.
type EventEmitter interface {
	// Emit emits a resilience event.
	Emit(event ResilienceEvent)

	// EmitAudit emits an audit event.
	EmitAudit(event AuditEvent)
}
