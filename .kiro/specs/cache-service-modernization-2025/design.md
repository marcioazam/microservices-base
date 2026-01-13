# Design Document: Cache Service Modernization 2025

## Overview

This design document details the modernization of the cache-service microservice to December 2025 state-of-the-art standards. The primary goals are:

1. **Centralized Logging** - Replace local zap logger with gRPC client to logging-service
2. **Zero Redundancy** - Eliminate all duplicated code and centralize logic
3. **Modern Dependencies** - Upgrade to Go 1.24+, rabbitmq/amqp091-go, latest OpenTelemetry
4. **Clean Architecture** - Separate source from tests, consolidate observability

## Architecture

### Current Architecture (Before)

```
platform/cache-service/
├── cmd/cache-service/main.go
├── internal/
│   ├── auth/           # JWT validation
│   ├── broker/         # RabbitMQ/Kafka (streadway/amqp - DEPRECATED)
│   ├── cache/          # Core cache service
│   ├── circuitbreaker/ # Circuit breaker pattern
│   ├── config/         # Configuration loading
│   ├── crypto/         # AES encryption
│   ├── grpc/           # gRPC server (manual proto types - REDUNDANT)
│   ├── http/           # HTTP handlers
│   ├── localcache/     # In-memory LRU cache
│   ├── logging/        # Local zap logger (TO BE REPLACED)
│   ├── metrics/        # Prometheus metrics
│   ├── redis/          # Redis client
│   └── tracing/        # OpenTelemetry tracing
└── tests/
    ├── integration/
    └── property/
```

### Target Architecture (After)

```
platform/cache-service/
├── cmd/cache-service/main.go
├── internal/
│   ├── auth/           # JWT validation (unchanged)
│   ├── broker/         # RabbitMQ/Kafka (rabbitmq/amqp091-go)
│   ├── cache/          # Core cache service (centralized errors/TTL)
│   ├── config/         # Configuration (env/v11, logging-service config)
│   ├── crypto/         # AES encryption (unchanged)
│   ├── grpc/           # gRPC server (generated proto, otelgrpc)
│   ├── http/           # HTTP handlers (centralized errors)
│   ├── localcache/     # In-memory LRU cache (unchanged)
│   ├── loggingclient/  # NEW: gRPC client for logging-service
│   ├── observability/  # MERGED: metrics + tracing + context
│   └── redis/          # Redis client (protected client)
├── api/proto/
│   └── cache/v1/cache.proto
└── tests/
    ├── integration/
    ├── property/
    └── unit/
```

## Components and Interfaces

### 1. Logging Client (NEW)

The logging client replaces the local zap logger with a gRPC client that sends all logs to the centralized logging-service.

```go
// internal/loggingclient/client.go
package loggingclient

import (
    "context"
    "sync"
    "time"
    
    loggingpb "github.com/auth-platform/logging-service/api/grpc"
    "google.golang.org/grpc"
)

// Client sends logs to the centralized logging-service via gRPC.
type Client struct {
    conn        *grpc.ClientConn
    client      loggingpb.LoggingServiceClient
    serviceID   string
    batchSize   int
    flushInterval time.Duration
    
    mu          sync.Mutex
    buffer      []*loggingpb.LogEntryMessage
    flushTimer  *time.Timer
    
    circuitOpen bool
}

// Config holds logging client configuration.
type Config struct {
    Address       string        // logging-service gRPC address
    ServiceID     string        // this service's identifier
    BatchSize     int           // max entries before flush (default: 100)
    FlushInterval time.Duration // max time before flush (default: 5s)
    BufferSize    int           // max buffer size (default: 10000)
}

// New creates a new logging client.
func New(cfg Config) (*Client, error)

// Log sends a log entry (buffered for batching).
func (c *Client) Log(ctx context.Context, level Level, msg string, fields ...Field) error

// Debug logs at debug level.
func (c *Client) Debug(ctx context.Context, msg string, fields ...Field)

// Info logs at info level.
func (c *Client) Info(ctx context.Context, msg string, fields ...Field)

// Warn logs at warn level.
func (c *Client) Warn(ctx context.Context, msg string, fields ...Field)

// Error logs at error level.
func (c *Client) Error(ctx context.Context, msg string, fields ...Field)

// Close flushes remaining logs and closes the connection.
func (c *Client) Close() error
```

### 2. Observability Package (MERGED)

Consolidates logging context, metrics, and tracing into a single package.

```go
// internal/observability/observability.go
package observability

import (
    "context"
    
    "github.com/auth-platform/cache-service/internal/loggingclient"
    "github.com/prometheus/client_golang/prometheus"
    "go.opentelemetry.io/otel/trace"
)

// Provider holds all observability components.
type Provider struct {
    Logger  *loggingclient.Client
    Metrics *Metrics
    Tracer  trace.Tracer
}

// Config holds observability configuration.
type Config struct {
    ServiceName     string
    LoggingConfig   loggingclient.Config
    MetricsEnabled  bool
    TracingEnabled  bool
    TracingEndpoint string
}

// New creates a new observability provider.
func New(cfg Config) (*Provider, error)

// Shutdown gracefully shuts down all observability components.
func (p *Provider) Shutdown(ctx context.Context) error

// Context keys (centralized - removes duplicates from logging/tracing)
type contextKey string

const (
    CorrelationIDKey contextKey = "correlation_id"
    RequestIDKey     contextKey = "request_id"
    TraceIDKey       contextKey = "trace_id"
    SpanIDKey        contextKey = "span_id"
)

// WithCorrelationID adds correlation ID to context.
func WithCorrelationID(ctx context.Context, id string) context.Context

// GetCorrelationID extracts correlation ID from context.
func GetCorrelationID(ctx context.Context) string
```

### 3. Centralized Error Handling

All error types and handling consolidated in cache/errors.go with HTTP/gRPC mapping.

```go
// internal/cache/errors.go (ENHANCED)
package cache

import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "net/http"
)

// ErrorCode represents cache-specific error codes.
type ErrorCode int

const (
    ErrUnknown ErrorCode = iota
    ErrKeyNotFound
    ErrInvalidKey
    ErrInvalidValue
    ErrTTLInvalid
    ErrRedisUnavailable
    ErrCircuitOpen
    ErrEncryptionFailed
    ErrDecryptionFailed
    ErrUnauthorized
    ErrForbidden
    ErrNamespaceInvalid
    ErrBrokerUnavailable
)

// Error represents a cache-specific error.
type Error struct {
    Code    ErrorCode
    Message string
    Cause   error
}

// ToHTTPStatus maps error code to HTTP status.
func (e *Error) ToHTTPStatus() int {
    switch e.Code {
    case ErrKeyNotFound:
        return http.StatusNotFound
    case ErrInvalidKey, ErrInvalidValue, ErrTTLInvalid, ErrNamespaceInvalid:
        return http.StatusBadRequest
    case ErrUnauthorized:
        return http.StatusUnauthorized
    case ErrForbidden:
        return http.StatusForbidden
    case ErrRedisUnavailable, ErrCircuitOpen, ErrBrokerUnavailable:
        return http.StatusServiceUnavailable
    default:
        return http.StatusInternalServerError
    }
}

// ToGRPCStatus maps error code to gRPC status.
func (e *Error) ToGRPCStatus() *status.Status {
    switch e.Code {
    case ErrKeyNotFound:
        return status.New(codes.NotFound, e.Message)
    case ErrInvalidKey, ErrInvalidValue, ErrTTLInvalid, ErrNamespaceInvalid:
        return status.New(codes.InvalidArgument, e.Message)
    case ErrUnauthorized:
        return status.New(codes.Unauthenticated, e.Message)
    case ErrForbidden:
        return status.New(codes.PermissionDenied, e.Message)
    case ErrRedisUnavailable, ErrCircuitOpen, ErrBrokerUnavailable:
        return status.New(codes.Unavailable, e.Message)
    default:
        return status.New(codes.Internal, e.Message)
    }
}
```

### 4. HTTP Error Response (CENTRALIZED)

```go
// internal/http/errors.go (NEW - centralized)
package http

import (
    "encoding/json"
    "net/http"
    
    "github.com/auth-platform/cache-service/internal/cache"
)

// ErrorResponse represents a JSON error response.
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
    Message string `json:"message,omitempty"`
}

// WriteError writes a standardized error response.
func WriteError(w http.ResponseWriter, err error) {
    var cacheErr *cache.Error
    if errors.As(err, &cacheErr) {
        writeJSON(w, cacheErr.ToHTTPStatus(), ErrorResponse{
            Error:   cacheErr.Code.String(),
            Code:    cacheErr.Code.String(),
            Message: cacheErr.Message,
        })
        return
    }
    writeJSON(w, http.StatusInternalServerError, ErrorResponse{
        Error: "internal_error",
    })
}
```

### 5. Broker Modernization (rabbitmq/amqp091-go)

```go
// internal/broker/rabbitmq.go (UPDATED)
package broker

import (
    "context"
    "sync"
    "time"
    
    amqp "github.com/rabbitmq/amqp091-go" // UPDATED from streadway/amqp
    "github.com/auth-platform/cache-service/internal/cache"
)

// RabbitMQBroker implements the Broker interface using RabbitMQ.
type RabbitMQBroker struct {
    mu          sync.RWMutex
    conn        *amqp.Connection
    channel     *amqp.Channel
    url         string
    retryConfig RetryConfig
    healthy     bool
    closed      bool
    
    // Connection recovery (amqp091-go feature)
    notifyClose chan *amqp.Error
}

// NewRabbitMQBroker creates a new RabbitMQ broker with auto-recovery.
func NewRabbitMQBroker(url string, retryConfig RetryConfig) (*RabbitMQBroker, error) {
    b := &RabbitMQBroker{
        url:         url,
        retryConfig: retryConfig,
        notifyClose: make(chan *amqp.Error),
    }
    
    if err := b.connect(); err != nil {
        return nil, err
    }
    
    // Start connection recovery goroutine
    go b.handleReconnect()
    
    return b, nil
}
```

### 6. Configuration Updates

```go
// internal/config/config.go (UPDATED)
package config

import (
    "time"
    
    "github.com/caarlos0/env/v11" // UPDATED from v10
)

// Config holds all service configuration.
type Config struct {
    Server   ServerConfig
    Redis    RedisConfig
    Broker   BrokerConfig
    Auth     AuthConfig
    Cache    CacheConfig
    Metrics  MetricsConfig
    Logging  LoggingConfig // NEW
}

// LoggingConfig holds logging-service client configuration.
type LoggingConfig struct {
    ServiceAddress string        `env:"LOGGING_SERVICE_ADDRESS" envDefault:"localhost:50052"`
    BatchSize      int           `env:"LOGGING_BATCH_SIZE" envDefault:"100"`
    FlushInterval  time.Duration `env:"LOGGING_FLUSH_INTERVAL" envDefault:"5s"`
    BufferSize     int           `env:"LOGGING_BUFFER_SIZE" envDefault:"10000"`
    Enabled        bool          `env:"LOGGING_ENABLED" envDefault:"true"`
}
```

## Data Models

### Log Entry (for Logging Service)

```go
// Matches logging-service protobuf schema
type LogEntry struct {
    ID            string            // UUID
    Timestamp     time.Time         // ISO 8601 UTC
    CorrelationID string            // Request correlation
    ServiceID     string            // "cache-service"
    Level         LogLevel          // Debug/Info/Warn/Error/Fatal
    Message       string            // Log message
    TraceID       string            // OpenTelemetry trace ID
    SpanID        string            // OpenTelemetry span ID
    Metadata      map[string]string // Additional fields
}
```

### Redundancy Elimination Map

| Location | Type | Current State | Target State | Strategy |
|----------|------|---------------|--------------|----------|
| cache/types.go | CircuitState | Duplicate of circuitbreaker.State | Remove | Use circuitbreaker.State only |
| broker/invalidation.go | Logger interface | Duplicate logging abstraction | Remove | Use loggingclient.Client |
| logging/logger.go | contextKey | Duplicate of tracing context keys | Remove | Centralize in observability |
| grpc/server.go | Proto types | Manual definitions | Remove | Use generated protobuf |
| http/handlers.go | writeError | Duplicate error handling | Remove | Use http/errors.go |

## Correctness Properties

This section defines the testable correctness properties that ensure the modernized cache-service behaves correctly under all conditions.

### Property 1: Logging Client Delivery Guarantee

**Property:** All log entries submitted to the logging client are eventually delivered to the logging-service or written to stderr fallback.

**Formal Definition:**
```
∀ log_entry L submitted to LoggingClient:
  eventually(delivered_to_logging_service(L) ∨ written_to_stderr(L))
```

**Test Strategy:** Property-based test with gopter generating random log entries, simulating network failures, and verifying delivery or fallback.

### Property 2: Batch Flush Invariant

**Property:** Log batches are flushed when either batch size is reached OR flush interval expires, whichever comes first.

**Formal Definition:**
```
∀ batch B in LoggingClient buffer:
  flush(B) ⟺ (|B| ≥ batch_size ∨ age(B) ≥ flush_interval)
```

**Test Strategy:** Property-based test varying batch sizes and timing to verify flush triggers.

### Property 3: Circuit Breaker State Transitions

**Property:** Circuit breaker transitions follow the state machine: Closed → Open → HalfOpen → (Closed | Open).

**Formal Definition:**
```
∀ state transition T:
  valid(T) ⟺ T ∈ {
    (Closed, Open),      // on failure threshold
    (Open, HalfOpen),    // on timeout
    (HalfOpen, Closed),  // on success
    (HalfOpen, Open)     // on failure
  }
```

**Test Strategy:** Property-based test generating random failure/success sequences and verifying state machine invariants.

### Property 4: Error Code Mapping Consistency

**Property:** Every ErrorCode maps to exactly one HTTP status and one gRPC code, and the mapping is bijective within error categories.

**Formal Definition:**
```
∀ ErrorCode E:
  ∃! http_status H, grpc_code G:
    ToHTTPStatus(E) = H ∧ ToGRPCStatus(E).Code() = G
```

**Test Strategy:** Exhaustive test of all ErrorCode values verifying consistent mapping.

### Property 5: TTL Normalization Idempotence

**Property:** Normalizing a TTL value twice produces the same result as normalizing once.

**Formal Definition:**
```
∀ ttl T:
  normalize(normalize(T)) = normalize(T)
```

**Test Strategy:** Property-based test with random TTL values.

### Property 6: Cache Entry Expiration Monotonicity

**Property:** Once a cache entry is expired, it remains expired (time moves forward).

**Formal Definition:**
```
∀ entry E, time t1 < t2:
  expired(E, t1) ⟹ expired(E, t2)
```

**Test Strategy:** Property-based test with random entries and time progressions.

### Property 7: Correlation ID Propagation

**Property:** Correlation ID set in context is preserved through all observability operations.

**Formal Definition:**
```
∀ context C with correlation_id ID:
  GetCorrelationID(WithCorrelationID(C, ID)) = ID
```

**Test Strategy:** Property-based test with random correlation IDs.

### Property 8: Buffer Overflow Protection

**Property:** Logging client buffer never exceeds configured maximum size.

**Formal Definition:**
```
∀ time t:
  |buffer(t)| ≤ max_buffer_size
```

**Test Strategy:** Property-based test flooding the logging client and verifying buffer bounds.

### Property 9: Graceful Shutdown Completeness

**Property:** On shutdown, all buffered logs are flushed before connection close.

**Formal Definition:**
```
∀ shutdown S:
  pre(S).buffer_count = 0 ∨ post(S).flushed_count = pre(S).buffer_count
```

**Test Strategy:** Integration test with buffered logs verifying flush on Close().

### Property 10: Redis Operation Context Propagation

**Property:** All Redis operations propagate context for tracing and cancellation.

**Formal Definition:**
```
∀ Redis operation O with context C:
  cancelled(C) ⟹ cancelled(O) ∨ completed(O)
```

**Test Strategy:** Integration test with context cancellation during Redis operations.

### Property 11: Message Broker Reconnection Resilience

**Property:** Broker reconnects automatically after transient failures without message loss for published messages.

**Formal Definition:**
```
∀ message M published during reconnection:
  eventually(delivered(M) ∨ error_returned(M))
```

**Test Strategy:** Integration test with testcontainers simulating broker restarts.

### Property 12: Configuration Validation Completeness

**Property:** Invalid configuration is rejected at startup with descriptive error.

**Formal Definition:**
```
∀ config C:
  ¬valid(C) ⟹ startup_fails(C) ∧ error_message_describes_issue(C)
```

**Test Strategy:** Table-driven test with invalid configurations.

### Property 13: gRPC Interceptor Chain Order

**Property:** gRPC interceptors execute in defined order: auth → tracing → logging → handler.

**Formal Definition:**
```
∀ request R:
  execution_order(R) = [auth, tracing, logging, handler]
```

**Test Strategy:** Integration test with interceptor execution tracking.

### Property 14: HTTP Middleware Chain Order

**Property:** HTTP middleware executes in defined order: requestID → tracing → logging → auth → handler.

**Formal Definition:**
```
∀ request R:
  execution_order(R) = [requestID, tracing, logging, auth, handler]
```

**Test Strategy:** Integration test with middleware execution tracking.

### Property 15: Local Cache Fallback Activation

**Property:** Local cache is used as fallback when Redis is unavailable and circuit breaker is open.

**Formal Definition:**
```
∀ Get operation G when circuit_open ∧ key_in_local_cache:
  result(G).Source = SourceLocal
```

**Test Strategy:** Integration test with Redis unavailable verifying local cache fallback.

## Error Handling Strategy

### Error Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Error Sources                             │
├─────────────┬─────────────┬─────────────┬─────────────┬─────────┤
│   Redis     │   Broker    │   Auth      │   Crypto    │  Config │
└──────┬──────┴──────┬──────┴──────┬──────┴──────┬──────┴────┬────┘
       │             │             │             │           │
       ▼             ▼             ▼             ▼           ▼
┌─────────────────────────────────────────────────────────────────┐
│              cache.WrapError(code, message, cause)               │
│                    Centralized Error Creation                    │
└─────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌──────────────┐
        │   HTTP   │   │   gRPC   │   │   Logging    │
        │ Handler  │   │  Server  │   │   Client     │
        └────┬─────┘   └────┬─────┘   └──────┬───────┘
             │              │                │
             ▼              ▼                ▼
        ToHTTPStatus() ToGRPCStatus()  Log with context
```

### Error Categories and Handling

| Category | ErrorCodes | HTTP Status | gRPC Code | Retry | Log Level |
|----------|------------|-------------|-----------|-------|-----------|
| Client Error | InvalidKey, InvalidValue, TTLInvalid, NamespaceInvalid | 400 | InvalidArgument | No | Warn |
| Not Found | KeyNotFound | 404 | NotFound | No | Debug |
| Auth Error | Unauthorized | 401 | Unauthenticated | No | Warn |
| Permission | Forbidden | 403 | PermissionDenied | No | Warn |
| Transient | RedisUnavailable, CircuitOpen, BrokerUnavailable | 503 | Unavailable | Yes | Error |
| Internal | Unknown, EncryptionFailed, DecryptionFailed | 500 | Internal | No | Error |

### Error Response Format

**HTTP JSON Response:**
```json
{
  "error": "key_not_found",
  "code": "key_not_found",
  "message": "The requested key does not exist in cache",
  "correlation_id": "abc-123-def"
}
```

**gRPC Status Details:**
```go
status.New(codes.NotFound, "key not found").WithDetails(&errdetails.ErrorInfo{
    Reason: "KEY_NOT_FOUND",
    Domain: "cache.service",
    Metadata: map[string]string{
        "key": "user:123",
        "namespace": "sessions",
    },
})
```

## Testing Strategy

### Test Organization

```
platform/cache-service/
└── tests/
    ├── unit/                    # Fast, isolated tests
    │   ├── cache/
    │   │   ├── errors_test.go
    │   │   ├── ttl_test.go
    │   │   └── types_test.go
    │   ├── loggingclient/
    │   │   ├── client_test.go
    │   │   └── batch_test.go
    │   └── observability/
    │       └── context_test.go
    ├── integration/             # Tests with real dependencies
    │   ├── redis_test.go
    │   ├── broker_test.go
    │   ├── loggingclient_test.go
    │   └── grpc_test.go
    ├── property/                # Property-based tests
    │   ├── logging_properties_test.go
    │   ├── circuit_breaker_properties_test.go
    │   ├── ttl_properties_test.go
    │   └── error_mapping_properties_test.go
    └── e2e/                     # End-to-end flows
        ├── cache_flow_test.go
        └── invalidation_flow_test.go
```

### Test Categories

#### Unit Tests (80% of tests)
- Pure function testing
- Mock all external dependencies
- Target: <100ms per test
- Coverage: 80%+ per package

#### Integration Tests (15% of tests)
- Use testcontainers for Redis, RabbitMQ
- Real gRPC connections to mock logging-service
- Target: <5s per test
- Coverage: Critical paths

#### Property-Based Tests (5% of tests)
- Use leanovate/gopter
- Minimum 100 iterations per property
- Focus on invariants and edge cases

### Test Infrastructure

```go
// tests/testutil/containers.go
package testutil

import (
    "context"
    "testing"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/redis"
    "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

// RedisContainer provides a Redis instance for testing.
func RedisContainer(t *testing.T) (string, func()) {
    ctx := context.Background()
    container, err := redis.Run(ctx, "redis:7-alpine")
    if err != nil {
        t.Fatalf("failed to start redis: %v", err)
    }
    
    endpoint, _ := container.Endpoint(ctx, "")
    cleanup := func() { container.Terminate(ctx) }
    
    return endpoint, cleanup
}

// RabbitMQContainer provides a RabbitMQ instance for testing.
func RabbitMQContainer(t *testing.T) (string, func()) {
    ctx := context.Background()
    container, err := rabbitmq.Run(ctx, "rabbitmq:3-management-alpine")
    if err != nil {
        t.Fatalf("failed to start rabbitmq: %v", err)
    }
    
    endpoint, _ := container.AmqpURL(ctx)
    cleanup := func() { container.Terminate(ctx) }
    
    return endpoint, cleanup
}
```

### Property Test Examples

```go
// tests/property/logging_properties_test.go
package property

import (
    "testing"
    "time"
    "context"
    
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

func TestLoggingClientBatchFlushProperty(t *testing.T) {
    properties := gopter.NewProperties(gopter.DefaultTestParameters())
    
    properties.Property("batch flushes at size or interval", prop.ForAll(
        func(batchSize int, entries []string) bool {
            client := newTestLoggingClient(batchSize, 100*time.Millisecond)
            defer client.Close()
            
            for _, entry := range entries {
                client.Info(context.Background(), entry)
            }
            
            // Verify flush behavior
            if len(entries) >= batchSize {
                return client.flushCount > 0
            }
            
            time.Sleep(150 * time.Millisecond)
            return client.flushCount > 0 || len(entries) == 0
        },
        gen.IntRange(1, 100),
        gen.SliceOf(gen.AlphaString()),
    ))
    
    properties.TestingRun(t)
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
    properties := gopter.NewProperties(gopter.DefaultTestParameters())
    
    properties.Property("valid state transitions only", prop.ForAll(
        func(events []bool) bool { // true=success, false=failure
            cb := newTestCircuitBreaker(3, 5*time.Second)
            
            for _, success := range events {
                prevState := cb.State()
                if success {
                    cb.RecordSuccess()
                } else {
                    cb.RecordFailure()
                }
                newState := cb.State()
                
                if !isValidTransition(prevState, newState) {
                    return false
                }
            }
            return true
        },
        gen.SliceOf(gen.Bool()),
    ))
    
    properties.TestingRun(t)
}
```

### Coverage Requirements

| Package | Minimum Coverage | Critical Paths |
|---------|------------------|----------------|
| cache | 85% | Get, Set, Delete, TTL validation |
| loggingclient | 90% | Log, Batch, Flush, Fallback |
| observability | 80% | Context propagation |
| redis | 80% | Protected client, circuit breaker |
| broker | 75% | Publish, reconnection |
| http | 80% | Error handling, middleware |
| grpc | 80% | Interceptors, error mapping |

### CI/CD Integration

```yaml
# .github/workflows/test.yml
test:
  runs-on: ubuntu-latest
  services:
    redis:
      image: redis:7-alpine
    rabbitmq:
      image: rabbitmq:3-management-alpine
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'
    - name: Run unit tests
      run: go test -v -race -coverprofile=coverage.out ./...
    - name: Check coverage
      run: |
        go tool cover -func=coverage.out | grep total | awk '{print $3}' | \
        awk -F'%' '{if ($1 < 80) exit 1}'
    - name: Run property tests
      run: go test -v -tags=property ./tests/property/...
```

## Implementation Phases

### Phase 1: Foundation (Days 1-2)
1. Create loggingclient package with gRPC client
2. Create observability package (merge logging/metrics/tracing)
3. Update go.mod with new dependencies
4. Generate protobuf code for logging-service

### Phase 2: Integration (Days 3-4)
1. Replace zap logger with loggingclient throughout codebase
2. Update broker to use rabbitmq/amqp091-go
3. Centralize error handling with HTTP/gRPC mapping
4. Remove duplicate CircuitState from cache/types.go

### Phase 3: Cleanup (Day 5)
1. Remove internal/logging package
2. Remove internal/metrics package (merge to observability)
3. Remove internal/tracing package (merge to observability)
4. Remove duplicate Logger interface from broker
5. Update all imports

### Phase 4: Testing (Days 6-7)
1. Reorganize tests into tests/ directory
2. Add property-based tests for correctness properties
3. Add integration tests for logging-service communication
4. Verify 80%+ coverage

### Phase 5: Validation (Day 8)
1. Run full test suite
2. Security scan (gosec)
3. Lint check (golangci-lint)
4. Documentation update
