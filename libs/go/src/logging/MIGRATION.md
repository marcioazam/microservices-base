# Migration Guide: Logging Client

## Overview

This guide helps migrate from local logging implementations to the centralized logging client that uses the logging-service microservice.

## Before Migration

If you were using:
- `libs/go/src/observability/logger.go` - Local structured logger
- Standard library `log` package
- Third-party loggers (zap, logrus, etc.)

## After Migration

Use `libs/go/src/logging` which provides:
- Remote logging via logging-service (gRPC)
- Async batching for performance
- PII redaction for compliance
- Context propagation (correlation ID, trace ID)
- Local fallback when service unavailable

## Migration Steps

### 1. Update Imports

```go
// Before
import "github.com/authcorp/libs/go/src/observability"

// After
import "github.com/authcorp/libs/go/src/logging"
```

### 2. Replace Logger Initialization

```go
// Before
logger := observability.NewLogger(observability.LoggerConfig{
    Level: "info",
})

// After
client, err := logging.NewClient(logging.ClientConfig{
    Address:   "logging-service:50052",
    ServiceID: "my-service",
    BatchSize: 100,
})
```

### 3. Update Log Calls

```go
// Before
logger.Info("message", "key", "value")

// After
client.Info(ctx, "message", logging.String("key", "value"))
```

### 4. Context Propagation

The new client automatically extracts context values:

```go
ctx := observability.WithCorrelationID(ctx, "corr-123")
client.Info(ctx, "message") // correlation_id included automatically
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Address | logging-service gRPC address | localhost:50052 |
| ServiceID | Service identifier | "" |
| BatchSize | Logs before flush | 100 |
| FlushInterval | Max time before flush | 5s |
| BufferSize | Max buffer size | 10000 |
| PIIRedaction | Enable PII redaction | true |

## Field Helpers

```go
logging.String("key", "value")
logging.Int("count", 42)
logging.Bool("enabled", true)
logging.Duration("elapsed", time.Second)
logging.Error(err)
logging.Any("data", struct{}{})
```

## PII Redaction

Sensitive fields are automatically redacted:
- email, password, token, secret, key
- ssn, credit_card, phone, address

```go
client.Info(ctx, "user login",
    logging.String("email", "user@example.com"), // redacted
    logging.String("user_id", "123"),            // not redacted
)
```

## Benefits

1. **Centralized logging** - Single log aggregation point
2. **Performance** - Async batching reduces latency
3. **Compliance** - Automatic PII redaction
4. **Observability** - Correlation ID propagation
5. **Resilience** - Local fallback when service down
