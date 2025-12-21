package resilience

import (
	"context"
	"testing"
	"time"
)

func TestGenerateEventID(t *testing.T) {
	id := GenerateEventID()
	if len(id) != 23 {
		t.Errorf("expected ID length 23, got %d", len(id))
	}
	if id[14] != '-' {
		t.Errorf("expected dash at position 14, got %c", id[14])
	}
}

func TestGenerateEventIDUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateEventID()
		if seen[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestGenerateEventIDWithPrefix(t *testing.T) {
	id := GenerateEventIDWithPrefix("cb")
	if len(id) < 26 {
		t.Errorf("expected ID length >= 26, got %d", len(id))
	}
	if id[:3] != "cb-" {
		t.Errorf("expected prefix 'cb-', got %s", id[:3])
	}
}

func TestEnsureCorrelationFunc(t *testing.T) {
	// Test with nil
	fn := EnsureCorrelationFunc(nil)
	if fn() != "" {
		t.Error("expected empty string from default correlation func")
	}

	// Test with custom func
	custom := func() string { return "test-id" }
	fn = EnsureCorrelationFunc(custom)
	if fn() != "test-id" {
		t.Error("expected 'test-id' from custom correlation func")
	}
}

func TestCorrelationIDContext(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if id := CorrelationIDFromContext(ctx); id != "" {
		t.Errorf("expected empty string, got %s", id)
	}

	// Test with correlation ID
	ctx = ContextWithCorrelationID(ctx, "test-correlation-id")
	if id := CorrelationIDFromContext(ctx); id != "test-correlation-id" {
		t.Errorf("expected 'test-correlation-id', got %s", id)
	}
}

func TestMarshalUnmarshalTime(t *testing.T) {
	original := time.Now().UTC().Truncate(time.Nanosecond)

	marshaled := MarshalTime(original)
	unmarshaled, err := UnmarshalTime(marshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal time: %v", err)
	}

	if !original.Equal(unmarshaled) {
		t.Errorf("time mismatch: original=%v, unmarshaled=%v", original, unmarshaled)
	}
}

func TestMarshalUnmarshalTimePtr(t *testing.T) {
	// Test nil
	if s := MarshalTimePtr(nil); s != "" {
		t.Errorf("expected empty string for nil, got %s", s)
	}

	ptr, err := UnmarshalTimePtr("")
	if err != nil {
		t.Fatalf("failed to unmarshal empty string: %v", err)
	}
	if ptr != nil {
		t.Error("expected nil for empty string")
	}

	// Test non-nil
	now := time.Now().UTC().Truncate(time.Nanosecond)
	s := MarshalTimePtr(&now)
	ptr, err = UnmarshalTimePtr(s)
	if err != nil {
		t.Fatalf("failed to unmarshal time: %v", err)
	}
	if !now.Equal(*ptr) {
		t.Errorf("time mismatch: original=%v, unmarshaled=%v", now, *ptr)
	}
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventCircuitStateChange, "test-service")

	if event.ID == "" {
		t.Error("expected non-empty ID")
	}
	if event.Type != EventCircuitStateChange {
		t.Errorf("expected type %s, got %s", EventCircuitStateChange, event.Type)
	}
	if event.ServiceName != "test-service" {
		t.Errorf("expected service 'test-service', got %s", event.ServiceName)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestEventWithMethods(t *testing.T) {
	event := NewEvent(EventRetryAttempt, "test-service").
		WithCorrelationID("corr-123").
		WithTraceContext("trace-456", "span-789").
		WithMetadata("attempt", 3)

	if event.CorrelationID != "corr-123" {
		t.Errorf("expected correlation ID 'corr-123', got %s", event.CorrelationID)
	}
	if event.TraceID != "trace-456" {
		t.Errorf("expected trace ID 'trace-456', got %s", event.TraceID)
	}
	if event.SpanID != "span-789" {
		t.Errorf("expected span ID 'span-789', got %s", event.SpanID)
	}
	if event.Metadata["attempt"] != 3 {
		t.Errorf("expected metadata attempt=3, got %v", event.Metadata["attempt"])
	}
}

func TestEmitEventNilSafety(t *testing.T) {
	// Should not panic
	EmitEvent(nil, Event{})
	EmitAuditEvent(nil, AuditEvent{})
}

func TestChannelEmitter(t *testing.T) {
	emitter := NewChannelEmitter(10)
	defer emitter.Close()

	event := *NewEvent(EventCircuitStateChange, "test")
	emitter.Emit(event)

	select {
	case received := <-emitter.Events:
		if received.Type != event.Type {
			t.Errorf("expected type %s, got %s", event.Type, received.Type)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}
