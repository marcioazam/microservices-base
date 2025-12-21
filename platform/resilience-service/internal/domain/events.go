// Package domain provides service-specific types for the resilience service.
package domain

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// EventType represents the type of resilience event.
type EventType string

const (
	EventCircuitOpen     EventType = "circuit_open"
	EventCircuitClosed   EventType = "circuit_closed"
	EventCircuitHalfOpen EventType = "circuit_half_open"
	EventRateLimited     EventType = "rate_limited"
	EventTimeout         EventType = "timeout"
	EventBulkheadFull    EventType = "bulkhead_full"
	EventRetryAttempt    EventType = "retry_attempt"
	EventRetryExhausted  EventType = "retry_exhausted"
)

// ResilienceEvent represents a service-specific event for local observability.
type ResilienceEvent struct {
	ID            string         `json:"id"`
	Type          EventType      `json:"type"`
	ServiceName   string         `json:"service_name"`
	Timestamp     time.Time      `json:"timestamp"`
	CorrelationID string         `json:"correlation_id"`
	TraceID       string         `json:"trace_id,omitempty"`
	SpanID        string         `json:"span_id,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
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

// GenerateEventID generates a UUID v7 compliant event ID.
func GenerateEventID() string {
	var uuid [16]byte
	now := time.Now().UnixMilli()
	binary.BigEndian.PutUint64(uuid[0:8], uint64(now)<<16)
	rand.Read(uuid[6:])
	uuid[6] = (uuid[6] & 0x0f) | 0x70
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// NowUTC returns current time in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// NewResilienceEvent creates a new resilience event with auto-generated ID and timestamp.
func NewResilienceEvent(eventType EventType, serviceName string) *ResilienceEvent {
	return &ResilienceEvent{
		ID:          GenerateEventID(),
		Type:        eventType,
		ServiceName: serviceName,
		Timestamp:   NowUTC(),
		Metadata:    make(map[string]any),
	}
}
