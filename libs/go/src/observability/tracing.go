package observability

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// W3C Trace Context headers
const (
	TraceparentHeader = "traceparent"
	TracestateHeader  = "tracestate"
	BaggageHeader     = "baggage"
)

// W3CTraceContext represents W3C Trace Context
type W3CTraceContext struct {
	Version    string
	TraceID    string // 128-bit (32 hex chars)
	SpanID     string // 64-bit (16 hex chars)
	TraceFlags byte
	TraceState string
}

// TraceSpan represents a single operation in a trace
type TraceSpan struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	StartTime    time.Time
	EndTime      time.Time
	Status       TraceSpanStatus
	Attributes   map[string]string
	Events       []TraceSpanEvent
}

// TraceSpanStatus represents the status of a span
type TraceSpanStatus int

const (
	TraceStatusUnset TraceSpanStatus = iota
	TraceStatusOK
	TraceStatusError
)

// TraceSpanEvent represents an event within a span
type TraceSpanEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]string
}

// NewW3CTraceContext creates a new trace context with random IDs
func NewW3CTraceContext() *W3CTraceContext {
	return &W3CTraceContext{
		Version:    "00",
		TraceID:    GenerateTraceID(),
		SpanID:     GenerateSpanID(),
		TraceFlags: 0x01, // Sampled
		TraceState: "",
	}
}

// ParseTraceparent parses a W3C traceparent header
func ParseTraceparent(header string) (*W3CTraceContext, error) {
	parts := strings.Split(header, "-")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid traceparent format")
	}

	version := parts[0]
	if version != "00" {
		return nil, fmt.Errorf("unsupported traceparent version: %s", version)
	}

	traceID := parts[1]
	if len(traceID) != 32 {
		return nil, fmt.Errorf("invalid trace-id length")
	}

	spanID := parts[2]
	if len(spanID) != 16 {
		return nil, fmt.Errorf("invalid span-id length")
	}

	if len(parts[3]) != 2 {
		return nil, fmt.Errorf("invalid trace-flags")
	}

	var flags byte
	for _, c := range parts[3] {
		flags = flags << 4
		if c >= '0' && c <= '9' {
			flags |= byte(c - '0')
		} else if c >= 'a' && c <= 'f' {
			flags |= byte(c - 'a' + 10)
		} else {
			return nil, fmt.Errorf("invalid trace-flags")
		}
	}

	return &W3CTraceContext{
		Version:    version,
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
	}, nil
}

// ToTraceparent formats the context as a W3C traceparent header
func (tc *W3CTraceContext) ToTraceparent() string {
	return fmt.Sprintf("%s-%s-%s-%02x", tc.Version, tc.TraceID, tc.SpanID, tc.TraceFlags)
}

// IsSampled returns true if the trace is sampled
func (tc *W3CTraceContext) IsSampled() bool {
	return tc.TraceFlags&0x01 != 0
}

// CreateChildSpan creates a new span as a child of this context
func (tc *W3CTraceContext) CreateChildSpan() *W3CTraceContext {
	return &W3CTraceContext{
		Version:    tc.Version,
		TraceID:    tc.TraceID,
		SpanID:     GenerateSpanID(),
		TraceFlags: tc.TraceFlags,
		TraceState: tc.TraceState,
	}
}

// NewTraceSpan creates a new span
func NewTraceSpan(name string, traceCtx *W3CTraceContext, parentSpanID string) *TraceSpan {
	return &TraceSpan{
		TraceID:      traceCtx.TraceID,
		SpanID:       traceCtx.SpanID,
		ParentSpanID: parentSpanID,
		Name:         name,
		StartTime:    time.Now(),
		Status:       TraceStatusUnset,
		Attributes:   make(map[string]string),
		Events:       []TraceSpanEvent{},
	}
}

// End marks the span as complete
func (s *TraceSpan) End() {
	s.EndTime = time.Now()
}

// SetStatus sets the span status
func (s *TraceSpan) SetStatus(status TraceSpanStatus) {
	s.Status = status
}

// SetAttribute sets a span attribute
func (s *TraceSpan) SetAttribute(key, value string) {
	s.Attributes[key] = value
}

// AddEvent adds an event to the span
func (s *TraceSpan) AddEvent(name string, attrs map[string]string) {
	s.Events = append(s.Events, TraceSpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

// Duration returns the span duration
func (s *TraceSpan) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// ExtractFromHTTP extracts trace context from HTTP headers
func ExtractFromHTTP(r *http.Request) *W3CTraceContext {
	traceparent := r.Header.Get(TraceparentHeader)
	if traceparent == "" {
		return NewW3CTraceContext()
	}

	tc, err := ParseTraceparent(traceparent)
	if err != nil {
		return NewW3CTraceContext()
	}

	tc.TraceState = r.Header.Get(TracestateHeader)
	return tc
}

// InjectToHTTP injects trace context into HTTP headers
func InjectToHTTP(tc *W3CTraceContext, r *http.Request) {
	r.Header.Set(TraceparentHeader, tc.ToTraceparent())
	if tc.TraceState != "" {
		r.Header.Set(TracestateHeader, tc.TraceState)
	}
}

// AuthSpanAttributes returns common attributes for auth spans
func AuthSpanAttributes(userID, sessionID, authMethod string) map[string]string {
	return map[string]string{
		"user.id":     userID,
		"session.id":  sessionID,
		"auth.method": authMethod,
	}
}
