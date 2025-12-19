package tracing

import (
	"context"
	"net/http/httptest"
	"testing"
)

// **Feature: auth-platform-2025-enhancements, Property 33: Trace Context Propagation**
// **Validates: Requirements 17.1, 17.2**
func TestTraceContextPropagation(t *testing.T) {
	tests := []struct {
		name        string
		traceparent string
		wantErr     bool
	}{
		{
			name:        "valid traceparent",
			traceparent: "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			wantErr:     false,
		},
		{
			name:        "invalid version",
			traceparent: "ff-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			wantErr:     true,
		},
		{
			name:        "invalid trace-id length",
			traceparent: "00-0af7651916cd43dd-b7ad6b7169203331-01",
			wantErr:     true,
		},
		{
			name:        "invalid span-id length",
			traceparent: "00-0af7651916cd43dd8448eb211c80319c-b7ad6b71-01",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc, err := ParseTraceparent(tt.traceparent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTraceparent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tc == nil {
				t.Error("ParseTraceparent() returned nil without error")
			}
		})
	}
}

func TestNewTraceContext(t *testing.T) {
	tc := NewTraceContext()

	// Verify trace-id is 128 bits (32 hex chars)
	if len(tc.TraceID) != 32 {
		t.Errorf("TraceID length = %d, want 32", len(tc.TraceID))
	}

	// Verify span-id is 64 bits (16 hex chars)
	if len(tc.SpanID) != 16 {
		t.Errorf("SpanID length = %d, want 16", len(tc.SpanID))
	}

	// Verify version
	if tc.Version != "00" {
		t.Errorf("Version = %s, want 00", tc.Version)
	}

	// Verify sampled flag is set
	if !tc.IsSampled() {
		t.Error("New trace context should be sampled by default")
	}
}

func TestTraceContextRoundTrip(t *testing.T) {
	original := NewTraceContext()
	traceparent := original.ToTraceparent()

	parsed, err := ParseTraceparent(traceparent)
	if err != nil {
		t.Fatalf("ParseTraceparent() error = %v", err)
	}

	if parsed.TraceID != original.TraceID {
		t.Errorf("TraceID = %s, want %s", parsed.TraceID, original.TraceID)
	}

	if parsed.SpanID != original.SpanID {
		t.Errorf("SpanID = %s, want %s", parsed.SpanID, original.SpanID)
	}

	if parsed.TraceFlags != original.TraceFlags {
		t.Errorf("TraceFlags = %d, want %d", parsed.TraceFlags, original.TraceFlags)
	}
}

func TestCreateChildSpan(t *testing.T) {
	parent := NewTraceContext()
	child := parent.CreateChildSpan()

	// Child should have same trace-id
	if child.TraceID != parent.TraceID {
		t.Errorf("Child TraceID = %s, want %s", child.TraceID, parent.TraceID)
	}

	// Child should have different span-id
	if child.SpanID == parent.SpanID {
		t.Error("Child SpanID should be different from parent")
	}

	// Child should inherit trace flags
	if child.TraceFlags != parent.TraceFlags {
		t.Errorf("Child TraceFlags = %d, want %d", child.TraceFlags, parent.TraceFlags)
	}
}

// **Feature: auth-platform-2025-enhancements, Property 34: Span Attribute Recording**
// **Validates: Requirements 17.3, 17.4**
func TestSpanAttributeRecording(t *testing.T) {
	tc := NewTraceContext()
	span := NewSpan("test-operation", tc, "")

	// Set attributes
	span.SetAttribute("user.id", "user-123")
	span.SetAttribute("session.id", "session-456")
	span.SetAttribute("auth.method", "webauthn")

	// Verify attributes are recorded
	if span.Attributes["user.id"] != "user-123" {
		t.Errorf("user.id = %s, want user-123", span.Attributes["user.id"])
	}

	if span.Attributes["session.id"] != "session-456" {
		t.Errorf("session.id = %s, want session-456", span.Attributes["session.id"])
	}

	// Add event
	span.AddEvent("authentication_complete", map[string]string{
		"result": "success",
	})

	if len(span.Events) != 1 {
		t.Errorf("Events count = %d, want 1", len(span.Events))
	}

	// End span
	span.End()

	// Verify duration is recorded
	if span.Duration() <= 0 {
		t.Error("Span duration should be positive after End()")
	}
}

func TestSpanStatus(t *testing.T) {
	tc := NewTraceContext()
	span := NewSpan("test-operation", tc, "")

	// Default status
	if span.Status != StatusUnset {
		t.Errorf("Default status = %d, want StatusUnset", span.Status)
	}

	// Set OK status
	span.SetStatus(StatusOK)
	if span.Status != StatusOK {
		t.Errorf("Status = %d, want StatusOK", span.Status)
	}

	// Set Error status
	span.SetStatus(StatusError)
	if span.Status != StatusError {
		t.Errorf("Status = %d, want StatusError", span.Status)
	}
}

func TestHTTPPropagation(t *testing.T) {
	// Create trace context
	tc := NewTraceContext()
	tc.TraceState = "vendor=value"

	// Create request and inject
	req := httptest.NewRequest("GET", "/test", nil)
	InjectToHTTP(tc, req)

	// Verify headers are set
	if req.Header.Get(TraceparentHeader) == "" {
		t.Error("traceparent header not set")
	}

	if req.Header.Get(TracestateHeader) != "vendor=value" {
		t.Errorf("tracestate = %s, want vendor=value", req.Header.Get(TracestateHeader))
	}

	// Extract from request
	extracted := ExtractFromHTTP(req)

	if extracted.TraceID != tc.TraceID {
		t.Errorf("Extracted TraceID = %s, want %s", extracted.TraceID, tc.TraceID)
	}
}

func TestContextPropagation(t *testing.T) {
	tc := NewTraceContext()
	ctx := context.Background()

	// Add to context
	ctx = ContextWithTrace(ctx, tc)

	// Extract from context
	extracted := TraceFromContext(ctx)

	if extracted == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	if extracted.TraceID != tc.TraceID {
		t.Errorf("Extracted TraceID = %s, want %s", extracted.TraceID, tc.TraceID)
	}
}

func TestExtractFromHTTPWithoutHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	// Should create new trace context when header is missing
	tc := ExtractFromHTTP(req)

	if tc == nil {
		t.Fatal("ExtractFromHTTP returned nil")
	}

	if len(tc.TraceID) != 32 {
		t.Errorf("TraceID length = %d, want 32", len(tc.TraceID))
	}
}

func TestAuthSpanAttributes(t *testing.T) {
	attrs := AuthSpanAttributes("user-123", "session-456", "webauthn")

	if attrs["user.id"] != "user-123" {
		t.Errorf("user.id = %s, want user-123", attrs["user.id"])
	}

	if attrs["session.id"] != "session-456" {
		t.Errorf("session.id = %s, want session-456", attrs["session.id"])
	}

	if attrs["auth.method"] != "webauthn" {
		t.Errorf("auth.method = %s, want webauthn", attrs["auth.method"])
	}
}
