# Resilience Patterns

A collection of resilience patterns for building fault-tolerant microservices.

## Category: Resilience Patterns

## Packages

### Legacy Packages (Deprecated)

| Package | Description |
|---------|-------------|
| [circuitbreaker](./circuitbreaker/) | Prevents cascading failures |
| [ratelimit](./ratelimit/) | Controls request throughput |
| [retry](./retry/) | Retries with exponential backoff |
| [bulkhead](./bulkhead/) | Isolates concurrent operations |
| [timeout](./timeout/) | Enforces operation timeouts |
| [domain](./domain/) | Shared domain types |
| [errors](./errors/) | Resilience-specific errors |

### Consolidated Module (Recommended)

The new unified `src/resilience` module consolidates all resilience patterns into a single package with:
- Centralized error types (`ResilienceError`, `CircuitOpenError`, `RateLimitError`, etc.)
- Unified configuration pattern with functional options
- Integration with `functional.Result[T]` type

```go
import "github.com/authcorp/libs/go/src/resilience"
```

## Quick Start

### Circuit Breaker (Consolidated Module)

```go
import (
    "github.com/authcorp/libs/go/src/resilience"
)

// Create with functional options
config := resilience.NewCircuitBreakerConfig("my-service",
    resilience.WithFailureThreshold(5),
    resilience.WithSuccessThreshold(2),
    resilience.WithCircuitTimeout(30*time.Second),
)

cb, err := resilience.NewCircuitBreaker(config)
if err != nil {
    return err
}

// Execute operation
err = cb.Execute(ctx, func(ctx context.Context) error {
    return callExternalService(ctx)
})

// Or use with Result[T] for typed returns
result := resilience.ExecuteWithResult(cb, ctx, func(ctx context.Context) (string, error) {
    return callExternalService(ctx)
})
if result.IsOk() {
    fmt.Println(result.Unwrap())
}

// Check circuit state
if cb.State() == resilience.StateOpen {
    // Circuit is open, requests will fail fast
}
```

### Circuit Breaker (Legacy)

```go
import "github.com/auth-platform/libs/go/resilience/circuitbreaker"

cb := circuitbreaker.New[string]("my-service",
    circuitbreaker.WithFailureThreshold(5),
    circuitbreaker.WithTimeout(30*time.Second),
)

result, err := cb.Execute(ctx, func() (string, error) {
    return callExternalService()
})
```

### Rate Limiter

```go
import "github.com/authcorp/libs/go/src/resilience"

config := resilience.RateLimitConfig{
    Rate:      100,
    Window:    time.Second,
    BurstSize: 10,
}

limiter, err := resilience.NewRateLimiter(config)
if err != nil {
    return err
}

if limiter.Allow() {
    // Process request
}

// Or execute with automatic rate limiting
err = limiter.Execute(ctx, func(ctx context.Context) error {
    return processRequest(ctx)
})
```

### Retry

```go
import "github.com/authcorp/libs/go/src/resilience"

config := resilience.NewRetryConfig(
    resilience.WithMaxAttempts(3),
    resilience.WithInitialInterval(100*time.Millisecond),
    resilience.WithMaxInterval(10*time.Second),
    resilience.WithJitterStrategy(resilience.FullJitter),
)

err := resilience.Retry(ctx, config, func(ctx context.Context) error {
    return callService(ctx)
})

// Or with typed result
result := resilience.RetryWithResult(ctx, config, func(ctx context.Context) (string, error) {
    return callService(ctx)
})
```

### Bulkhead (Legacy)

```go
import "github.com/auth-platform/libs/go/resilience/bulkhead"

bh := bulkhead.New[string]("partition",
    bulkhead.WithMaxConcurrent(10),
    bulkhead.WithMaxQueue(100),
)

result, err := bh.Execute(ctx, func() (string, error) {
    return processRequest()
})
```

## Error Handling

The consolidated module provides centralized error types:

```go
import "github.com/authcorp/libs/go/src/resilience"

err := cb.Execute(ctx, operation)

// Check specific error types
if resilience.IsCircuitOpen(err) {
    circuitErr, _ := resilience.AsCircuitOpenError(err)
    log.Printf("Circuit open, retry after: %v", circuitErr.ResetAfter)
}

if resilience.IsRateLimited(err) {
    rateErr, _ := resilience.AsRateLimitError(err)
    log.Printf("Rate limited, retry after: %v", rateErr.RetryAfter)
}

if resilience.IsTimeout(err) {
    // Handle timeout
}

if resilience.IsBulkheadFull(err) {
    // Handle bulkhead rejection
}

// Get error code for logging/metrics
if code, ok := resilience.GetErrorCode(err); ok {
    metrics.IncrementError(string(code))
}
```

## Configuration

All components use a unified configuration pattern with validation:

```go
// Default configs are always valid
config := resilience.DefaultCircuitBreakerConfig()

// Validate custom configs
config := resilience.CircuitBreakerConfig{
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:          30 * time.Second,
    HalfOpenRequests: 1,
}

if err := config.Validate(); err != nil {
    if resilience.IsInvalidPolicy(err) {
        // Handle invalid configuration
    }
}
```

## Architecture

```
resilience/
├── circuitbreaker/  # (Legacy) State machine: Closed → Open → Half-Open
├── ratelimit/       # (Legacy) Token bucket and sliding window algorithms
├── retry/           # (Legacy) Exponential backoff with jitter
├── bulkhead/        # (Legacy) Semaphore-based isolation
├── timeout/         # (Legacy) Context-based timeout enforcement
├── domain/          # (Legacy) Shared configuration types
└── errors/          # (Legacy) Typed error handling

src/resilience/      # (Recommended) Consolidated module
├── circuitbreaker.go  # Circuit breaker with Result[T] integration
├── ratelimit.go       # Token bucket rate limiter
├── retry.go           # Retry with exponential backoff and jitter
├── bulkhead.go        # Semaphore-based isolation
├── config.go          # Unified configuration with functional options
├── errors.go          # Centralized error types
└── errors_check.go    # Error type checking functions
```

## Circuit Breaker States

```
     ┌─────────────────────────────────────────┐
     │                                         │
     ▼                                         │
┌─────────┐  failure threshold  ┌──────────┐  │
│ CLOSED  │ ─────────────────▶  │   OPEN   │  │
└─────────┘                     └──────────┘  │
     ▲                               │        │
     │                               │ timeout│
     │  success threshold            ▼        │
     │                          ┌──────────┐  │
     └───────────────────────── │HALF-OPEN │ ─┘
                                └──────────┘
                                     │ failure
                                     └────────┘
```
