package events

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// DomainEvent represents a generic domain event with metadata.
type DomainEvent struct {
	ID            string            `json:"id"`
	EventType     string            `json:"type"`
	Source        string            `json:"source"`
	Timestamp     time.Time         `json:"timestamp"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
	SpanID        string            `json:"span_id,omitempty"`
	Data          interface{}       `json:"data,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Type implements Event interface.
func (e DomainEvent) Type() string {
	return e.EventType
}

// Emitter is an interface for emitting events.
type Emitter interface {
	Emit(event DomainEvent) error
}

// Builder builds events with automatic field population.
type Builder struct {
	serviceName   string
	correlationID string
	traceID       string
	spanID        string
	eventType     string
	data          interface{}
	metadata      map[string]string
}

// NewBuilder creates a new event builder.
func NewBuilder(serviceName string) *Builder {
	return &Builder{
		serviceName: serviceName,
		metadata:    make(map[string]string),
	}
}

// WithCorrelationID sets the correlation ID.
func (b *Builder) WithCorrelationID(id string) *Builder {
	b.correlationID = id
	return b
}

// WithType sets the event type.
func (b *Builder) WithType(eventType string) *Builder {
	b.eventType = eventType
	return b
}

// WithData sets the event data.
func (b *Builder) WithData(data interface{}) *Builder {
	b.data = data
	return b
}

// WithMetadata adds metadata.
func (b *Builder) WithMetadata(key, value string) *Builder {
	b.metadata[key] = value
	return b
}

// WithTraceID sets the trace ID.
func (b *Builder) WithTraceID(traceID string) *Builder {
	b.traceID = traceID
	return b
}

// WithSpanID sets the span ID.
func (b *Builder) WithSpanID(spanID string) *Builder {
	b.spanID = spanID
	return b
}

// Build creates the event with auto-generated ID and timestamp.
func (b *Builder) Build() DomainEvent {
	return DomainEvent{
		ID:            generateEventID(),
		EventType:     b.eventType,
		Source:        b.serviceName,
		Timestamp:     time.Now().UTC(),
		CorrelationID: b.correlationID,
		TraceID:       b.traceID,
		SpanID:        b.spanID,
		Data:          b.data,
		Metadata:      b.metadata,
	}
}

// BuildWithContext creates the event extracting trace info from context.
func (b *Builder) BuildWithContext(ctx context.Context) DomainEvent {
	event := b.Build()
	if traceID := extractTraceIDFromCtx(ctx); traceID != "" {
		event.TraceID = traceID
	}
	if spanID := extractSpanIDFromCtx(ctx); spanID != "" {
		event.SpanID = spanID
	}
	return event
}

// Emit builds and emits the event.
func (b *Builder) Emit(emitter Emitter) error {
	if emitter == nil {
		return nil
	}
	return emitter.Emit(b.Build())
}

// Reset resets the builder for reuse.
func (b *Builder) Reset() *Builder {
	b.correlationID = ""
	b.traceID = ""
	b.spanID = ""
	b.eventType = ""
	b.data = nil
	b.metadata = make(map[string]string)
	return b
}

// Clone creates a copy of the builder.
func (b *Builder) Clone() *Builder {
	metadata := make(map[string]string, len(b.metadata))
	for k, v := range b.metadata {
		metadata[k] = v
	}
	return &Builder{
		serviceName:   b.serviceName,
		correlationID: b.correlationID,
		traceID:       b.traceID,
		spanID:        b.spanID,
		eventType:     b.eventType,
		data:          b.data,
		metadata:      metadata,
	}
}

type eventContextKey string

const (
	eventTraceIDKey eventContextKey = "event_trace_id"
	eventSpanIDKey  eventContextKey = "event_span_id"
)

func extractTraceIDFromCtx(ctx context.Context) string {
	if v := ctx.Value(eventTraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractSpanIDFromCtx(ctx context.Context) string {
	if v := ctx.Value(eventSpanIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithEventTraceContext adds trace info to context for events.
func WithEventTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, eventTraceIDKey, traceID)
	ctx = context.WithValue(ctx, eventSpanIDKey, spanID)
	return ctx
}

func generateEventID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
