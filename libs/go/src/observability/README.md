# Observability

Structured logging, tracing, and metrics for Go applications.

## Features

- Structured JSON logging with levels
- W3C Trace Context propagation
- Correlation ID management
- PII redaction in logs
- OpenTelemetry integration

## Installation

```go
import "github.com/authcorp/libs/go/src/observability"
```

## W3C Trace Context

The package provides full W3C Trace Context support for distributed tracing:

### Creating Trace Context

```go
// Create new trace context with random IDs
tc := observability.NewW3CTraceContext()

// Parse from traceparent header
tc, err := observability.ParseTraceparent("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
if err != nil {
    // Handle invalid format
}

// Format as traceparent header
header := tc.ToTraceparent() // "00-{trace-id}-{span-id}-{flags}"
```

### HTTP Integration

```go
// Extract trace context from incoming request
tc := observability.ExtractFromHTTP(r)

// Inject trace context into outgoing request
observability.InjectToHTTP(tc, outgoingReq)
```

### Creating Child Spans

```go
// Create child span (same trace, new span ID)
childCtx := tc.CreateChildSpan()

// Check if trace is sampled
if tc.IsSampled() {
    // Record span data
}
```

### Span Management

```go
// Create a new span
span := observability.NewTraceSpan("operation-name", tc, parentSpanID)

// Set attributes
span.SetAttribute("user.id", userID)
span.SetAttribute("http.method", "POST")

// Add events
span.AddEvent("cache_miss", map[string]string{"key": cacheKey})

// Set status
span.SetStatus(observability.TraceStatusOK)

// End span (records duration)
span.End()

// Get duration
duration := span.Duration()
```

### Auth-Specific Attributes

```go
// Get standard auth span attributes
attrs := observability.AuthSpanAttributes(userID, sessionID, "oauth2")
// Returns: {"user.id": userID, "session.id": sessionID, "auth.method": "oauth2"}
```

## Types

### W3CTraceContext

```go
type W3CTraceContext struct {
    Version    string // "00" for current version
    TraceID    string // 128-bit (32 hex chars)
    SpanID     string // 64-bit (16 hex chars)
    TraceFlags byte   // Sampling flags
    TraceState string // Vendor-specific state
}
```

### TraceSpan

```go
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
```

### TraceSpanStatus

```go
const (
    TraceStatusUnset TraceSpanStatus = iota
    TraceStatusOK
    TraceStatusError
)
```

## Headers

The package uses standard W3C headers:

| Header | Description |
|--------|-------------|
| `traceparent` | Primary trace context propagation |
| `tracestate` | Vendor-specific trace state |
| `baggage` | Application-defined context |

## Best Practices

1. **Always propagate context**: Extract from incoming requests, inject into outgoing
2. **Use child spans**: Create child spans for sub-operations to maintain trace hierarchy
3. **Set meaningful attributes**: Include relevant context like user IDs, request IDs
4. **Record events**: Add events for significant occurrences within a span
5. **Set status**: Mark spans as OK or Error based on outcome

## Thread Safety

All trace context operations are thread-safe. Span operations should be performed by a single goroutine or synchronized externally.

## See Also

- [W3C Trace Context Specification](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [functional package](../functional/README.md) - Result type for error handling
