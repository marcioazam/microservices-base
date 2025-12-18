# Resilience Library

Shared resilience primitives for distributed systems. This library provides domain primitives, event handling, and configuration types used across multiple services.

## Features

- **Event ID Generation**: Unique, timestamp-based event identifiers
- **Correlation Functions**: Context-aware correlation ID propagation
- **Time Serialization**: RFC3339Nano time format helpers
- **Event Types**: Resilience event and audit event structures
- **Configuration Types**: Circuit breaker, retry, timeout, rate limit, and bulkhead configs
- **Event Emitters**: Nil-safe event emission with channel-based implementation

## Installation

```go
import "github.com/auth-platform/libs/go/resilience"
```

## Usage

### Event ID Generation

```go
// Generate a unique event ID
id := resilience.GenerateEventID()
// Output: "20241217150405-a1b2c3d4"

// Generate with custom prefix
id = resilience.GenerateEventIDWithPrefix("cb")
// Output: "cb-20241217150405-a1b2c3d4"
```

### Correlation Functions

```go
// Use default correlation function
fn := resilience.EnsureCorrelationFunc(nil)
id := fn() // Returns ""

// Use context-based correlation
ctx := resilience.ContextWithCorrelationID(ctx, "request-123")
id := resilience.CorrelationIDFromContext(ctx) // Returns "request-123"
```

### Time Serialization

```go
// Marshal time to string
s := resilience.MarshalTime(time.Now())

// Unmarshal string to time
t, err := resilience.UnmarshalTime(s)
```

### Events

```go
// Create and emit events
event := resilience.NewEvent(resilience.EventCircuitStateChange, "my-service").
    WithCorrelationID("corr-123").
    WithMetadata("state", "OPEN")

resilience.EmitEvent(emitter, *event)
```

### Configuration

```go
// Use default configurations
cbConfig := resilience.DefaultCircuitBreakerConfig()
retryConfig := resilience.DefaultRetryConfig()

// Validate configurations
if err := cbConfig.Validate(); err != nil {
    log.Fatal(err)
}
```

## Dependencies

- Standard library only for core primitives
- `github.com/auth-platform/libs/go/error` for error types in config validation
