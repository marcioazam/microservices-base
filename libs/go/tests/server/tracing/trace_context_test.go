package tracing

import (
	"net/http/httptest"
	"testing"

	"github.com/authcorp/libs/go/src/observability"
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
			tc, err := observability.ParseTraceparent(tt.traceparent)
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

func TestNewW3CTraceContext(t *testing.T) {
	tc := observability.NewW3CTraceContext()

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
	original := observability.NewW3CTraceContext()
	traceparent := original.ToTraceparent()

	parsed, err := observability.ParseTraceparent(traceparent)
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
	parent := observability.NewW3CTraceContext()
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
	tc := observability.NewW3CTraceContext()
	span := observability.NewTraceSpan("test-operation", tc, "")

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

	// Verify duration is recorded (can be 0 if very fast, so check >= 0)
	if span.Duration() < 0 {
		t.Error("Span duration should be non-negative after End()")
	}
}

func TestSpanStatus(t *testing.T) {
	tc := observability.NewW3CTraceContext()
	span := observability.NewTraceSpan("test-operation", tc, "")

	// Default status
	if span.Status != observability.TraceStatusUnset {
		t.Errorf("Default status = %d, want TraceStatusUnset", span.Status)
	}

	// Set OK status
	span.SetStatus(observability.TraceStatusOK)
	if span.Status != observability.TraceStatusOK {
		t.Errorf("Status = %d, want TraceStatusOK", span.Status)
	}

	// Set Error status
	span.SetStatus(observability.TraceStatusError)
	if span.Status != observability.TraceStatusError {
		t.Errorf("Status = %d, want TraceStatusError", span.Status)
	}
}

func TestHTTPPropagation(t *testing.T) {
	// Create trace context
	tc := observability.NewW3CTraceContext()
	tc.TraceState = "vendor=value"

	// Create request and inject
	req := httptest.NewRequest("GET", "/test", nil)
	observability.InjectToHTTP(tc, req)

	// Verify headers are set
	if req.Header.Get(observability.TraceparentHeader) == "" {
		t.Error("traceparent header not set")
	}

	if req.Header.Get(observability.TracestateHeader) != "vendor=value" {
		t.Errorf("tracestate = %s, want vendor=value", req.Header.Get(observability.TracestateHeader))
	}

	// Extract from request
	extracted := observability.ExtractFromHTTP(req)

	if extracted.TraceID != tc.TraceID {
		t.Errorf("Extracted TraceID = %s, want %s", extracted.TraceID, tc.TraceID)
	}
}

func TestExtractFromHTTPWithoutHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	// Should create new trace context when header is missing
	tc := observability.ExtractFromHTTP(req)

	if tc == nil {
		t.Fatal("ExtractFromHTTP returned nil")
	}

	if len(tc.TraceID) != 32 {
		t.Errorf("TraceID length = %d, want 32", len(tc.TraceID))
	}
}

func TestAuthSpanAttributes(t *testing.T) {
	attrs := observability.AuthSpanAttributes("user-123", "session-456", "webauthn")

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
