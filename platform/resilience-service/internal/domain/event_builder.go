package domain

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// EventBuilder constructs resilience events with automatic field population.
type EventBuilder struct {
	emitter       EventEmitter
	serviceName   string
	correlationFn func() string
}

// NewEventBuilder creates a new EventBuilder.
func NewEventBuilder(emitter EventEmitter, serviceName string, correlationFn func() string) *EventBuilder {
	if correlationFn == nil {
		correlationFn = DefaultCorrelationFn()
	}
	return &EventBuilder{
		emitter:       emitter,
		serviceName:   serviceName,
		correlationFn: correlationFn,
	}
}

// Build creates a ResilienceEvent with automatic ID, Timestamp, and Type.
func (b *EventBuilder) Build(eventType ResilienceEventType, metadata map[string]any) ResilienceEvent {
	return ResilienceEvent{
		ID:            GenerateEventID(),
		Type:          eventType,
		ServiceName:   b.serviceName,
		Timestamp:     time.Now(),
		CorrelationID: b.correlationFn(),
		Metadata:      metadata,
	}
}

// BuildWithContext creates a ResilienceEvent with trace context propagation.
func (b *EventBuilder) BuildWithContext(ctx context.Context, eventType ResilienceEventType, metadata map[string]any) ResilienceEvent {
	event := b.Build(eventType, metadata)

	// Extract trace context if available
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		event.TraceID = spanCtx.TraceID().String()
		event.SpanID = spanCtx.SpanID().String()
	}

	return event
}

// Emit builds and emits an event. Safe to call with nil emitter.
func (b *EventBuilder) Emit(eventType ResilienceEventType, metadata map[string]any) {
	if b == nil || b.emitter == nil {
		return
	}
	event := b.Build(eventType, metadata)
	b.emitter.Emit(event)
}

// EmitWithContext builds and emits an event with trace context. Safe to call with nil emitter.
func (b *EventBuilder) EmitWithContext(ctx context.Context, eventType ResilienceEventType, metadata map[string]any) {
	if b == nil || b.emitter == nil {
		return
	}
	event := b.BuildWithContext(ctx, eventType, metadata)
	b.emitter.Emit(event)
}

// GetServiceName returns the service name configured in the builder.
func (b *EventBuilder) GetServiceName() string {
	if b == nil {
		return ""
	}
	return b.serviceName
}

// GetCorrelationID returns the current correlation ID.
func (b *EventBuilder) GetCorrelationID() string {
	if b == nil || b.correlationFn == nil {
		return ""
	}
	return b.correlationFn()
}
