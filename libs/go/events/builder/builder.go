// Package events provides a generic event builder library.
package events

import (
	"context"
	"time"

	"github.com/auth-platform/libs/go/utils/uuid"
)

// Event represents a generic event.
type Event struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Source        string            `json:"source"`
	Timestamp     time.Time         `json:"timestamp"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
	SpanID        string            `json:"span_id,omitempty"`
	Data          interface{}       `json:"data,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Emitter is an interface for emitting events.
type Emitter interface {
	Emit(event Event) error
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
func (b *Builder) Build() Event {
	return Event{
		ID:            uuid.GenerateEventID(),
		Type:          b.eventType,
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
func (b *Builder) BuildWithContext(ctx context.Context) Event {
	event := b.Build()
	
	// Extract trace ID from context if available
	if traceID := extractTraceID(ctx); traceID != "" {
		event.TraceID = traceID
	}
	
	// Extract span ID from context if available
	if spanID := extractSpanID(ctx); spanID != "" {
		event.SpanID = spanID
	}
	
	return event
}

// Emit builds and emits the event.
func (b *Builder) Emit(emitter Emitter) error {
	if emitter == nil {
		return nil // Handle gracefully without panic
	}
	return emitter.Emit(b.Build())
}

// EmitWithContext builds and emits the event with context.
func (b *Builder) EmitWithContext(ctx context.Context, emitter Emitter) error {
	if emitter == nil {
		return nil // Handle gracefully without panic
	}
	return emitter.Emit(b.BuildWithContext(ctx))
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

// Context keys for trace extraction
type contextKey string

const (
	traceIDKey contextKey = "trace_id"
	spanIDKey  contextKey = "span_id"
)

// extractTraceID extracts trace ID from context.
func extractTraceID(ctx context.Context) string {
	if v := ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractSpanID extracts span ID from context.
func extractSpanID(ctx context.Context) string {
	if v := ctx.Value(spanIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithTraceContext adds trace info to context.
func WithTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, traceIDKey, traceID)
	ctx = context.WithValue(ctx, spanIDKey, spanID)
	return ctx
}
