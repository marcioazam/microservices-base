# Fault

Fault tolerance patterns for distributed systems with `functional.Result[T]` integration and generic type safety.

## Features

- **Generic Executor** - Type-safe `ResilienceExecutor[T]` interface for composing patterns
- Circuit Breaker - Fail-fast with automatic recovery
- Bulkhead - Concurrency isolation
- Rate Limiting - Token bucket rate control
- Retry - Exponential backoff with jitter
- Timeout - Operation timeouts with context support
- Graceful shutdown
- Health aggregation
- **Execution Metrics** - Shared metrics type for observability

## Generic Executor Interface

The `ResilienceExecutor[T]` interface provides type-safe resilience pattern composition:

```go
import (
    "context"
    "github.com/authcorp/libs/go/src/fault"
    "github.com/authcorp/libs/go/src/functional"
)

// ResilienceExecutor applies resilience patterns to operations with type safety
type ResilienceExecutor[T any] interface {
    Execute(ctx context.Context, policyName string, op func() error) error
    ExecuteWithResult(ctx context.Context, policyName string, op func() (T, error)) functional.Result[T]
    RegisterPolicy(policy PolicyConfig) error
    UnregisterPolicy(policyName string)
    GetPolicyNames() []string
}
```

### Policy Configuration

Configure resilience patterns using `PolicyConfig`:

```go
policy := fault.PolicyConfig{
    Name: "payment-service",
    CircuitBreaker: &fault.CircuitBreakerPolicyConfig{
        FailureThreshold: 5,
        SuccessThreshold: 2,
        Timeout:          30 * time.Second,
        HalfOpenMaxCalls: 3,
    },
    Retry: &fault.RetryPolicyConfig{
        MaxAttempts:     3,
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     10 * time.Second,
        Multiplier:      2.0,
    },
    Timeout: &fault.TimeoutPolicyConfig{
        Timeout: 5 * time.Second,
        Max:     30 * time.Second,
    },
    RateLimit: &fault.RateLimitPolicyConfig{
        Algorithm: "token_bucket",
        Rate:      1000,
        Window:    time.Minute,
        BurstSize: 100,
    },
    Bulkhead: &fault.BulkheadPolicyConfig{
        MaxConcurrent: 10,
        QueueSize:     100,
        QueueTimeout:  5 * time.Second,
    },
}

executor.RegisterPolicy(policy)
```

### Type-Safe Execution

```go
// Execute with typed result
result := executor.ExecuteWithResult(ctx, "payment-service", func() (PaymentResult, error) {
    return processPayment(ctx, request)
})

if result.IsOk() {
    payment := result.Unwrap()
    // Use typed payment result
} else {
    err := result.UnwrapErr()
    // Handle error
}

// Or use helper functions
result := fault.ExecuteFunc(ctx, executor, "payment-service", processPayment)
err := fault.ExecuteSimple(ctx, executor, "health-check", checkHealth)
```

### Executor Configuration

```go
config := fault.DefaultExecutorConfig()
// Returns:
// - DefaultTimeout: 30s
// - MetricsEnabled: true
// - TracingEnabled: true
// - LoggerEnabled: true
```

## Execution Metrics

The `ExecutionMetrics` type captures resilience execution statistics:

```go
metrics := fault.NewExecutionMetrics("payment-service", 150*time.Millisecond, true).
    WithCircuitState("closed").
    WithRetryAttempts(2).
    WithCorrelationID("req-123").
    WithTraceID("trace-456")

// Check metrics
if metrics.WasRetried() {
    log.Info("operation was retried", "attempts", metrics.RetryAttempts)
}
```

### MetricsRecorder Interface

```go
type MetricsRecorder interface {
    RecordExecution(ctx context.Context, metrics ExecutionMetrics)
    RecordCircuitState(ctx context.Context, policyName string, state string)
    RecordRetryAttempt(ctx context.Context, policyName string, attempt int)
    RecordRateLimit(ctx context.Context, policyName string, limited bool)
    RecordBulkheadQueue(ctx context.Context, policyName string, queued bool)
}

// NoOpMetricsRecorder for testing or when metrics are disabled
recorder := fault.NoOpMetricsRecorder{}
```

## Usage

### Retry with Result

```go
import (
    "context"
    "github.com/authcorp/libs/go/src/fault"
)

config := fault.RetryConfig{
    MaxAttempts:     3,
    InitialInterval: 100 * time.Millisecond,
    MaxInterval:     1 * time.Second,
    Multiplier:      2.0,
    RetryIf:         func(err error) bool { return true },
}

result := fault.RetryWithResult(ctx, config, func(ctx context.Context) (int, error) {
    return fetchData(ctx)
})

if result.IsOk() {
    data := result.Unwrap()
} else {
    err := result.UnwrapErr()
    // Handle RetryExhaustedError, context.Canceled, etc.
}
```

### Circuit Breaker with Result

```go
config := fault.CircuitBreakerConfig{
    Name:             "my-service",
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:          30 * time.Second,
}

cb, _ := fault.NewCircuitBreaker(config)

result := fault.ExecuteWithResult(cb, ctx, func(ctx context.Context) (int, error) {
    return callService(ctx)
})

if result.IsErr() {
    var circuitErr *fault.CircuitOpenError
    if errors.As(result.UnwrapErr(), &circuitErr) {
        // Circuit is open, fail fast
    }
}
```

## Error Types

All resilience errors extend the base `ResilienceError` type, which embeds `AppError` for consistent error handling across the application.

| Error Type | Code | HTTP Status | Description |
|------------|------|-------------|-------------|
| `CircuitOpenError` | `CIRCUIT_OPEN` | 503 | Circuit breaker is open |
| `RateLimitError` | `RATE_LIMITED` | 429 | Rate limit exceeded |
| `TimeoutError` | `TIMEOUT` | 504 | Operation timed out |
| `BulkheadFullError` | `BULKHEAD_FULL` | 503 | Bulkhead capacity exceeded |
| `RetryExhaustedError` | `RETRY_EXHAUSTED` | 502 | All retry attempts failed |

### Base Error Structure

```go
// ResilienceError extends AppError for unified error handling
type ResilienceError struct {
    *apperrors.AppError           // Embedded base error
    Code          ErrorCode       // Resilience-specific code
    Service       string          // Service name
    Pattern       string          // Pattern: "circuit_breaker", "retry", etc.
    CorrelationID string          // Request correlation ID
    Timestamp     time.Time       // Error timestamp
    Cause         error           // Underlying cause
}

// HTTPStatus returns the appropriate HTTP status code
func (e *ResilienceError) HTTPStatus() int

// Is implements errors.Is for error matching by code
func (e *ResilienceError) Is(target error) bool
```

### Specific Error Types

```go
type CircuitOpenError struct {
    *ResilienceError
    OpenedAt    time.Time
    ResetAfter  time.Duration
    FailureRate float64
}

type RetryExhaustedError struct {
    ResilienceError
    Attempts   int
    TotalTime  time.Duration
    LastErrors []error
}
```

### Creating Errors

```go
// Using constructor
err := fault.NewResilienceError(
    fault.ErrCodeCircuitOpen,
    "payment-service",
    "circuit breaker is open",
    "circuit_breaker",
)

// Specific error constructors
circuitErr := fault.NewCircuitOpenError(service, correlationID, openedAt, resetAfter, failureRate)
retryErr := fault.NewRetryExhaustedError(service, correlationID, attempts, totalTime, lastErrors)
```

### HTTP Status Mapping

```go
err := fault.NewCircuitOpenError(...)
status := err.HTTPStatus() // Returns 503 Service Unavailable
```

## Context Support

All operations respect `context.Context`:

- Cancelled context returns `context.Canceled`
- Timed out context returns `context.DeadlineExceeded`
- Operations check `ctx.Done()` between retries

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result := fault.RetryWithResult(ctx, config, operation)
if errors.Is(result.UnwrapErr(), context.DeadlineExceeded) {
    // Handle timeout
}
```

## Error Codes

```go
const (
    ErrCodeCircuitOpen    ErrorCode = "CIRCUIT_OPEN"
    ErrCodeRateLimited    ErrorCode = "RATE_LIMITED"
    ErrCodeTimeout        ErrorCode = "TIMEOUT"
    ErrCodeBulkheadFull   ErrorCode = "BULKHEAD_FULL"
    ErrCodeRetryExhausted ErrorCode = "RETRY_EXHAUSTED"
    ErrCodeInvalidPolicy  ErrorCode = "INVALID_POLICY"
)
```

## Error Matching

Resilience errors support `errors.Is()` for matching by error code:

```go
err := fault.NewCircuitOpenError(...)
target := &fault.ResilienceError{Code: fault.ErrCodeCircuitOpen}

if errors.Is(err, target) {
    // Matches by error code
}

// Also works with AppError matching
if errors.Is(err, &apperrors.AppError{Code: apperrors.ErrCodeUnavailable}) {
    // Matches embedded AppError
}
```
