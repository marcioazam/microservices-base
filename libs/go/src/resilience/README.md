# Resilience Module

Unified resilience patterns with centralized error handling and configuration.

## Error Types

All resilience errors extend `ResilienceError` with:
- `Code`: Error type identifier
- `Service`: Service name
- `Message`: Human-readable message
- `CorrelationID`: Request correlation ID
- `Timestamp`: When error occurred

### Error Checking

```go
if resilience.IsCircuitOpen(err) {
    circuitErr, _ := resilience.AsCircuitOpenError(err)
    log.Printf("Circuit open until: %v", circuitErr.ResetAfter)
}

code, ok := resilience.GetErrorCode(err)
```

## Circuit Breaker

```go
config := resilience.NewCircuitBreakerConfig("my-service",
    resilience.WithFailureThreshold(5),
    resilience.WithCircuitTimeout(30*time.Second),
)

cb, _ := resilience.NewCircuitBreaker(config)

err := cb.Execute(ctx, func(ctx context.Context) error {
    return callExternalService(ctx)
})
```

## Retry

```go
config := resilience.NewRetryConfig(
    resilience.WithMaxAttempts(3),
    resilience.WithInitialInterval(100*time.Millisecond),
    resilience.WithJitterStrategy(resilience.FullJitter),
)

result := resilience.RetryWithResult(ctx, config, func(ctx context.Context) (string, error) {
    return fetchData(ctx)
})
```

## Rate Limiting

```go
config := resilience.RateLimitConfig{
    Rate:   100,
    Window: time.Second,
    BurstSize: 10,
}

limiter, _ := resilience.NewRateLimiter(config)

if limiter.Allow() {
    // Process request
}
```

## Bulkhead

```go
config := resilience.BulkheadConfig{
    MaxConcurrent: 10,
    QueueSize:     100,
    MaxWait:       time.Second,
}

bulkhead, _ := resilience.NewBulkhead(config)

err := bulkhead.Execute(ctx, func(ctx context.Context) error {
    return processRequest(ctx)
})
```

## Timeout

```go
config := resilience.TimeoutConfig{
    Timeout: 5 * time.Second,
}

result := resilience.TimeoutWithResult(ctx, config, func(ctx context.Context) (string, error) {
    return slowOperation(ctx)
})
```
