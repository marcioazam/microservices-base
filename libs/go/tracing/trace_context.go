package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// TraceContext represents W3C Trace Context
type TraceContext struct {
	Version    string
	TraceID    string // 128-bit (32 hex chars)
	SpanID     string // 64-bit (16 hex chars)
	TraceFlags byte
	TraceState string
}

// Span represents a single operation in a trace
type Span struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	StartTime    time.Time
	EndTime      time.Time
	Status       SpanStatus
	Attributes   map[string]string
	Events       []SpanEvent
}

// SpanStatus represents the status of a span
type SpanStatus int

const (
	StatusUnset SpanStatus = iota
	StatusOK
	StatusError
)

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]string
}

// NewTraceContext creates a new trace context with random IDs
func NewTraceContext() *TraceContext {
	return &TraceContext{
		Version:    "00",
		TraceID:    generateTraceID(),
		SpanID:     generateSpanID(),
		TraceFlags: 0x01, // Sampled
		TraceState: "",
	}
}

// ParseTraceparent parses a W3C traceparent header
func ParseTraceparent(header string) (*TraceContext, error) {
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

	flags, err := hex.DecodeString(parts[3])
	if err != nil || len(flags) != 1 {
		return nil, fmt.Errorf("invalid trace-flags")
	}

	return &TraceContext{
		Version:    version,
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags[0],
	}, nil
}

// ToTraceparent formats the context as a W3C traceparent header
func (tc *TraceContext) ToTraceparent() string {
	return fmt.Sprintf("%s-%s-%s-%02x", tc.Version, tc.TraceID, tc.SpanID, tc.TraceFlags)
}

// IsSampled returns true if the trace is sampled
func (tc *TraceContext) IsSampled() bool {
	return tc.TraceFlags&0x01 != 0
}

// CreateChildSpan creates a new span as a child of this context
func (tc *TraceContext) CreateChildSpan() *TraceContext {
	return &TraceContext{
		Version:    tc.Version,
		TraceID:    tc.TraceID,
		SpanID:     generateSpanID(),
		TraceFlags: tc.TraceFlags,
		TraceState: tc.TraceState,
	}
}

// NewSpan creates a new span
func NewSpan(name string, traceCtx *TraceContext, parentSpanID string) *Span {
	return &Span{
		TraceID:      traceCtx.TraceID,
		SpanID:       traceCtx.SpanID,
		ParentSpanID: parentSpanID,
		Name:         name,
		StartTime:    time.Now(),
		Status:       StatusUnset,
		Attributes:   make(map[string]string),
		Events:       []SpanEvent{},
	}
}

// End marks the span as complete
func (s *Span) End() {
	s.EndTime = time.Now()
}

// SetStatus sets the span status
func (s *Span) SetStatus(status SpanStatus) {
	s.Status = status
}

// SetAttribute sets a span attribute
func (s *Span) SetAttribute(key, value string) {
	s.Attributes[key] = value
}

// AddEvent adds an event to the span
func (s *Span) AddEvent(name string, attrs map[string]string) {
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

// Duration returns the span duration
func (s *Span) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// Context key for trace context
type traceContextKey struct{}

// ContextWithTrace adds trace context to a context
func ContextWithTrace(ctx context.Context, tc *TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey{}, tc)
}

// TraceFromContext extracts trace context from a context
func TraceFromContext(ctx context.Context) *TraceContext {
	if tc, ok := ctx.Value(traceContextKey{}).(*TraceContext); ok {
		return tc
	}
	return nil
}

// ExtractFromHTTP extracts trace context from HTTP headers
func ExtractFromHTTP(r *http.Request) *TraceContext {
	traceparent := r.Header.Get(TraceparentHeader)
	if traceparent == "" {
		return NewTraceContext()
	}

	tc, err := ParseTraceparent(traceparent)
	if err != nil {
		return NewTraceContext()
	}

	tc.TraceState = r.Header.Get(TracestateHeader)
	return tc
}

// InjectToHTTP injects trace context into HTTP headers
func InjectToHTTP(tc *TraceContext, r *http.Request) {
	r.Header.Set(TraceparentHeader, tc.ToTraceparent())
	if tc.TraceState != "" {
		r.Header.Set(TracestateHeader, tc.TraceState)
	}
}

// Helper functions
func generateTraceID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateSpanID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// AuthSpanAttributes returns common attributes for auth spans
func AuthSpanAttributes(userID, sessionID, authMethod string) map[string]string {
	return map[string]string{
		"user.id":     userID,
		"session.id":  sessionID,
		"auth.method": authMethod,
	}
}
