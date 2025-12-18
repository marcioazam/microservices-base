// Package domain provides service-specific types for the resilience service.
package domain

import (
	"time"

	"github.com/auth-platform/libs/go/resilience"
)

// ResilienceEvent represents a service-specific event for local observability.
type ResilienceEvent struct {
	ID            string               `json:"id"`
	Type          resilience.EventType `json:"type"`
	ServiceName   string               `json:"service_name"`
	Timestamp     time.Time            `json:"timestamp"`
	CorrelationID string               `json:"correlation_id"`
	TraceID       string               `json:"trace_id,omitempty"`
	SpanID        string               `json:"span_id,omitempty"`
	Metadata      map[string]any       `json:"metadata,omitempty"`
}

// AuditEvent represents a service-specific audit event for security logging.
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

// EventEmitter emits service-specific resilience events.
type EventEmitter interface {
	// Emit emits a resilience event.
	Emit(event ResilienceEvent)

	// EmitAudit emits an audit event.
	EmitAudit(event AuditEvent)
}

// EmitEvent safely emits a resilience event, handling nil emitter.
func EmitEvent(emitter EventEmitter, event ResilienceEvent) {
	if emitter == nil {
		return
	}
	emitter.Emit(event)
}

// EmitAuditEvent safely emits an audit event, handling nil emitter.
func EmitAuditEvent(emitter EventEmitter, event AuditEvent) {
	if emitter == nil {
		return
	}
	emitter.EmitAudit(event)
}

// NewResilienceEvent creates a new resilience event with auto-generated ID and timestamp.
func NewResilienceEvent(eventType resilience.EventType, serviceName string) *ResilienceEvent {
	return &ResilienceEvent{
		ID:          resilience.GenerateEventID(),
		Type:        eventType,
		ServiceName: serviceName,
		Timestamp:   resilience.NowUTC(),
		Metadata:    make(map[string]any),
	}
}
