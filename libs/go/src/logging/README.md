# Logging Client Library

Structured logging client for communication with `logging-service` via gRPC.

## Features

- gRPC client for logging-service
- Async batched log shipping for performance
- Context propagation (correlation_id, trace_id, span_id)
- PII redaction before sending
- Local fallback when remote is unavailable
- Configurable buffer size and flush intervals

## Usage

```go
package main

import (
    "context"
    
    "github.com/authcorp/libs/go/src/logging"
    "github.com/authcorp/libs/go/src/observability"
)

func main() {
    // Create client
    config := logging.DefaultConfig()
    config.Address = "logging-service:50052"
    config.ServiceName = "my-service"
    
    logger, err := logging.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    // Create context with correlation ID
    ctx := observability.WithCorrelationID(context.Background(), "req-123")
    
    // Log messages
    logger.Info(ctx, "Processing request",
        logging.String("user_id", "user-456"),
        logging.Int("items", 5),
    )
    
    logger.Error(ctx, "Failed to process",
        logging.Error(err),
    )
    
    // Create logger with additional fields
    userLogger := logger.With(
        logging.String("user_id", "user-456"),
    )
    userLogger.Info(ctx, "User action completed")
}
```

## Testing

For unit tests, use `LocalOnly` to create a client without network:

```go
logger := logging.LocalOnly("test-service")
defer logger.Close()

// Use logger in tests
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| Address | localhost:50052 | logging-service gRPC address |
| ServiceName | unknown | Service identifier |
| BufferSize | 100 | Max logs before flush |
| FlushInterval | 5s | Auto-flush interval |
| Timeout | 5s | Operation timeout |
| LocalFallback | true | Enable stdout fallback |
| MinLevel | INFO | Minimum log level |

## PII Redaction

The client automatically redacts:
- Sensitive field names (password, token, api_key, etc.)
- PII patterns in values (email, phone, SSN, credit card)
