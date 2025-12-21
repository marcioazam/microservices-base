package resilience

import "time"

// EventType represents the type of resilience event.
type EventType string

const (
	EventCircuitStateChange EventType = "circuit_state_change"
	EventRetryAttempt       EventType = "retry_attempt"
	EventTimeout            EventType = "timeout"
	EventRateLimitHit       EventType = "rate_limit_hit"
	EventBulkheadRejection  EventType = "bulkhead_rejection"
	EventHealthChange       EventType = "health_change"
	EventPolicyUpdated      EventType = "policy_updated"
	EventShutdownInitiated  EventType = "shutdown_initiated"
)

// Event represents a resilience event for observability.
type Event struct {
	ID            string         `json:"id"`
	Type          EventType      `json:"type"`
	ServiceName   string         `json:"service_name"`
	Timestamp     time.Time      `json:"timestamp"`
	CorrelationID string         `json:"correlation_id"`
	TraceID       string         `json:"trace_id,omitempty"`
	SpanID        string         `json:"span_id,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// NewEvent creates a new resilience event with auto-generated ID and timestamp.
func NewEvent(eventType EventType, serviceName string) *Event {
	return &Event{
		ID:          GenerateEventID(),
		Type:        eventType,
		ServiceName: serviceName,
		Timestamp:   NowUTC(),
		Metadata:    make(map[string]any),
	}
}

// WithCorrelationID sets the correlation ID.
func (e *Event) WithCorrelationID(id string) *Event {
	e.CorrelationID = id
	return e
}

// WithTraceContext sets trace and span IDs.
func (e *Event) WithTraceContext(traceID, spanID string) *Event {
	e.TraceID = traceID
	e.SpanID = spanID
	return e
}

// WithMetadata adds metadata to the event.
func (e *Event) WithMetadata(key string, value any) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
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

// NewAuditEvent creates a new audit event with auto-generated ID and timestamp.
func NewAuditEvent(action, resource, outcome string) *AuditEvent {
	return &AuditEvent{
		ID:        GenerateEventID(),
		Type:      "audit",
		Timestamp: NowUTC(),
		Action:    action,
		Resource:  resource,
		Outcome:   outcome,
		Metadata:  make(map[string]any),
	}
}

// EventEmitter emits resilience events.
type EventEmitter interface {
	// Emit emits a resilience event.
	Emit(event Event)

	// EmitAudit emits an audit event.
	EmitAudit(event AuditEvent)
}

// EmitEvent safely emits a resilience event, handling nil emitter.
func EmitEvent(emitter EventEmitter, event Event) {
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

// NoOpEmitter is an EventEmitter that does nothing.
type NoOpEmitter struct{}

func (NoOpEmitter) Emit(event Event)           {}
func (NoOpEmitter) EmitAudit(event AuditEvent) {}

// ChannelEmitter is an EventEmitter that sends events to channels.
type ChannelEmitter struct {
	Events      chan Event
	AuditEvents chan AuditEvent
}

// NewChannelEmitter creates a new channel-based emitter.
func NewChannelEmitter(bufferSize int) *ChannelEmitter {
	return &ChannelEmitter{
		Events:      make(chan Event, bufferSize),
		AuditEvents: make(chan AuditEvent, bufferSize),
	}
}

func (e *ChannelEmitter) Emit(event Event) {
	select {
	case e.Events <- event:
	default:
		// Channel full, drop event
	}
}

func (e *ChannelEmitter) EmitAudit(event AuditEvent) {
	select {
	case e.AuditEvents <- event:
	default:
		// Channel full, drop event
	}
}

// Close closes the emitter channels.
func (e *ChannelEmitter) Close() {
	close(e.Events)
	close(e.AuditEvents)
}
