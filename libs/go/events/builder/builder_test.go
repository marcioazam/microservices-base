// Package events provides tests for the event builder library.
package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/utils/uuid"
)

// mockEmitter is a test emitter that records emitted events.
type mockEmitter struct {
	events []Event
	err    error
}

func (m *mockEmitter) Emit(event Event) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, event)
	return nil
}

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("test-service")
	if builder == nil {
		t.Fatal("NewBuilder returned nil")
	}
	if builder.serviceName != "test-service" {
		t.Errorf("expected serviceName 'test-service', got '%s'", builder.serviceName)
	}
}

func TestBuilder_Build_AutoGeneratesID(t *testing.T) {
	builder := NewBuilder("test-service")
	event := builder.Build()

	// Verify ID is a valid UUID v7
	if !uuid.IsValidUUIDv7(event.ID) {
		t.Errorf("expected valid UUID v7, got '%s'", event.ID)
	}
}

func TestBuilder_Build_AutoPopulatesTimestamp(t *testing.T) {
	before := time.Now().UTC()
	builder := NewBuilder("test-service")
	event := builder.Build()
	after := time.Now().UTC()

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Errorf("timestamp %v not between %v and %v", event.Timestamp, before, after)
	}
}

func TestBuilder_Build_IncludesServiceName(t *testing.T) {
	builder := NewBuilder("my-service")
	event := builder.Build()

	if event.Source != "my-service" {
		t.Errorf("expected Source 'my-service', got '%s'", event.Source)
	}
}

func TestBuilder_WithCorrelationID(t *testing.T) {
	builder := NewBuilder("test-service")
	event := builder.WithCorrelationID("corr-123").Build()

	if event.CorrelationID != "corr-123" {
		t.Errorf("expected CorrelationID 'corr-123', got '%s'", event.CorrelationID)
	}
}

func TestBuilder_WithType(t *testing.T) {
	builder := NewBuilder("test-service")
	event := builder.WithType("user.created").Build()

	if event.Type != "user.created" {
		t.Errorf("expected Type 'user.created', got '%s'", event.Type)
	}
}

func TestBuilder_WithData(t *testing.T) {
	builder := NewBuilder("test-service")
	data := map[string]string{"key": "value"}
	event := builder.WithData(data).Build()

	if event.Data == nil {
		t.Fatal("expected Data to be set")
	}
	dataMap, ok := event.Data.(map[string]string)
	if !ok {
		t.Fatal("expected Data to be map[string]string")
	}
	if dataMap["key"] != "value" {
		t.Errorf("expected Data['key'] = 'value', got '%s'", dataMap["key"])
	}
}

func TestBuilder_WithMetadata(t *testing.T) {
	builder := NewBuilder("test-service")
	event := builder.
		WithMetadata("env", "prod").
		WithMetadata("version", "1.0").
		Build()

	if event.Metadata["env"] != "prod" {
		t.Errorf("expected Metadata['env'] = 'prod', got '%s'", event.Metadata["env"])
	}
	if event.Metadata["version"] != "1.0" {
		t.Errorf("expected Metadata['version'] = '1.0', got '%s'", event.Metadata["version"])
	}
}

func TestBuilder_WithTraceID(t *testing.T) {
	builder := NewBuilder("test-service")
	event := builder.WithTraceID("trace-abc").Build()

	if event.TraceID != "trace-abc" {
		t.Errorf("expected TraceID 'trace-abc', got '%s'", event.TraceID)
	}
}

func TestBuilder_WithSpanID(t *testing.T) {
	builder := NewBuilder("test-service")
	event := builder.WithSpanID("span-xyz").Build()

	if event.SpanID != "span-xyz" {
		t.Errorf("expected SpanID 'span-xyz', got '%s'", event.SpanID)
	}
}

func TestBuilder_BuildWithContext_ExtractsTraceInfo(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceContext(ctx, "trace-from-ctx", "span-from-ctx")

	builder := NewBuilder("test-service")
	event := builder.BuildWithContext(ctx)

	if event.TraceID != "trace-from-ctx" {
		t.Errorf("expected TraceID 'trace-from-ctx', got '%s'", event.TraceID)
	}
	if event.SpanID != "span-from-ctx" {
		t.Errorf("expected SpanID 'span-from-ctx', got '%s'", event.SpanID)
	}
}

func TestBuilder_BuildWithContext_EmptyContext(t *testing.T) {
	ctx := context.Background()

	builder := NewBuilder("test-service")
	event := builder.BuildWithContext(ctx)

	// Should still generate valid event
	if !uuid.IsValidUUIDv7(event.ID) {
		t.Errorf("expected valid UUID v7, got '%s'", event.ID)
	}
	if event.TraceID != "" {
		t.Errorf("expected empty TraceID, got '%s'", event.TraceID)
	}
}

func TestBuilder_Emit_NilEmitter(t *testing.T) {
	builder := NewBuilder("test-service")
	err := builder.Emit(nil)

	// Should handle gracefully without panic
	if err != nil {
		t.Errorf("expected nil error for nil emitter, got %v", err)
	}
}

func TestBuilder_Emit_Success(t *testing.T) {
	emitter := &mockEmitter{}
	builder := NewBuilder("test-service")
	
	err := builder.WithType("test.event").Emit(emitter)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitter.events))
	}
	if emitter.events[0].Type != "test.event" {
		t.Errorf("expected Type 'test.event', got '%s'", emitter.events[0].Type)
	}
}

func TestBuilder_Emit_Error(t *testing.T) {
	expectedErr := errors.New("emit failed")
	emitter := &mockEmitter{err: expectedErr}
	builder := NewBuilder("test-service")

	err := builder.Emit(emitter)

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestBuilder_EmitWithContext_NilEmitter(t *testing.T) {
	ctx := context.Background()
	builder := NewBuilder("test-service")
	
	err := builder.EmitWithContext(ctx, nil)

	if err != nil {
		t.Errorf("expected nil error for nil emitter, got %v", err)
	}
}

func TestBuilder_EmitWithContext_Success(t *testing.T) {
	ctx := WithTraceContext(context.Background(), "trace-123", "span-456")
	emitter := &mockEmitter{}
	builder := NewBuilder("test-service")

	err := builder.WithType("ctx.event").EmitWithContext(ctx, emitter)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitter.events))
	}
	if emitter.events[0].TraceID != "trace-123" {
		t.Errorf("expected TraceID 'trace-123', got '%s'", emitter.events[0].TraceID)
	}
}

func TestBuilder_Reset(t *testing.T) {
	builder := NewBuilder("test-service")
	builder.
		WithCorrelationID("corr-1").
		WithType("type-1").
		WithData("data").
		WithMetadata("key", "value").
		WithTraceID("trace-1").
		WithSpanID("span-1")

	builder.Reset()
	event := builder.Build()

	if event.CorrelationID != "" {
		t.Errorf("expected empty CorrelationID after reset, got '%s'", event.CorrelationID)
	}
	if event.Type != "" {
		t.Errorf("expected empty Type after reset, got '%s'", event.Type)
	}
	if event.Data != nil {
		t.Errorf("expected nil Data after reset, got %v", event.Data)
	}
	if len(event.Metadata) != 0 {
		t.Errorf("expected empty Metadata after reset, got %v", event.Metadata)
	}
	// Service name should be preserved
	if event.Source != "test-service" {
		t.Errorf("expected Source 'test-service' after reset, got '%s'", event.Source)
	}
}

func TestBuilder_Clone(t *testing.T) {
	original := NewBuilder("test-service")
	original.
		WithCorrelationID("corr-1").
		WithType("type-1").
		WithMetadata("key", "value")

	cloned := original.Clone()

	// Modify original
	original.WithCorrelationID("corr-2")
	original.WithMetadata("key", "modified")

	// Clone should be independent
	clonedEvent := cloned.Build()
	if clonedEvent.CorrelationID != "corr-1" {
		t.Errorf("expected cloned CorrelationID 'corr-1', got '%s'", clonedEvent.CorrelationID)
	}
	if clonedEvent.Metadata["key"] != "value" {
		t.Errorf("expected cloned Metadata['key'] = 'value', got '%s'", clonedEvent.Metadata["key"])
	}
}

func TestBuilder_Chaining(t *testing.T) {
	builder := NewBuilder("test-service")
	event := builder.
		WithType("user.created").
		WithCorrelationID("corr-123").
		WithData(map[string]string{"user_id": "u-1"}).
		WithMetadata("env", "test").
		WithTraceID("trace-abc").
		WithSpanID("span-xyz").
		Build()

	if event.Type != "user.created" {
		t.Errorf("expected Type 'user.created', got '%s'", event.Type)
	}
	if event.CorrelationID != "corr-123" {
		t.Errorf("expected CorrelationID 'corr-123', got '%s'", event.CorrelationID)
	}
	if event.TraceID != "trace-abc" {
		t.Errorf("expected TraceID 'trace-abc', got '%s'", event.TraceID)
	}
	if event.SpanID != "span-xyz" {
		t.Errorf("expected SpanID 'span-xyz', got '%s'", event.SpanID)
	}
}

func TestWithTraceContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceContext(ctx, "trace-id", "span-id")

	traceID := extractTraceID(ctx)
	spanID := extractSpanID(ctx)

	if traceID != "trace-id" {
		t.Errorf("expected traceID 'trace-id', got '%s'", traceID)
	}
	if spanID != "span-id" {
		t.Errorf("expected spanID 'span-id', got '%s'", spanID)
	}
}

func TestExtractTraceID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	traceID := extractTraceID(ctx)

	if traceID != "" {
		t.Errorf("expected empty traceID, got '%s'", traceID)
	}
}

func TestExtractSpanID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	spanID := extractSpanID(ctx)

	if spanID != "" {
		t.Errorf("expected empty spanID, got '%s'", spanID)
	}
}

func TestBuilder_MultipleBuilds_UniqueIDs(t *testing.T) {
	builder := NewBuilder("test-service")
	
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		event := builder.Build()
		if ids[event.ID] {
			t.Errorf("duplicate ID generated: %s", event.ID)
		}
		ids[event.ID] = true
	}
}
