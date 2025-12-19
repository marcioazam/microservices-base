# Migration Guide

This guide helps migrate from the old micro-module structure to the new consolidated modules.

## Import Path Changes

| Old Path | New Path |
|----------|----------|
| `libs/go/functional/option` | `libs/go/src/functional` |
| `libs/go/functional/result` | `libs/go/src/functional` |
| `libs/go/functional/either` | `libs/go/src/functional` |
| `libs/go/functional/iterator` | `libs/go/src/functional` |
| `libs/go/functional/lazy` | `libs/go/src/functional` |
| `libs/go/functional/pipeline` | `libs/go/src/functional` |
| `libs/go/functional/stream` | `libs/go/src/functional` |
| `libs/go/functional/tuple` | `libs/go/src/functional` |
| `libs/go/collections/lru` | `libs/go/src/collections` |
| `libs/go/collections/set` | `libs/go/src/collections` |
| `libs/go/collections/queue` | `libs/go/src/collections` |
| `libs/go/resilience/circuitbreaker` | `libs/go/src/resilience` |
| `libs/go/resilience/retry` | `libs/go/src/resilience` |
| `libs/go/resilience/ratelimit` | `libs/go/src/resilience` |
| `libs/go/resilience/bulkhead` | `libs/go/src/resilience` |
| `libs/go/resilience/timeout` | `libs/go/src/resilience` |
| `libs/go/async/future` | `libs/go/src/concurrency` |
| `libs/go/async/pool` | `libs/go/src/concurrency` |
| `libs/go/events/eventbus` | `libs/go/src/events` |
| `libs/go/events/pubsub` | `libs/go/src/events` |

## Code Changes

### Option

```go
// Old
import "libs/go/functional/option"
opt := option.Some(42)
opt.IsSome()

// New
import "libs/go/src/functional"
opt := functional.Some(42)
opt.IsSome()
```

### Result

```go
// Old
import "libs/go/functional/result"
res := result.Ok(42)

// New
import "libs/go/src/functional"
res := functional.Ok(42)
```

### Circuit Breaker

```go
// Old
import "libs/go/resilience/circuitbreaker"
cb := circuitbreaker.New(circuitbreaker.Config{
    FailureThreshold: 5,
})

// New
import "libs/go/src/resilience"
config := resilience.NewCircuitBreakerConfig("service",
    resilience.WithFailureThreshold(5),
)
cb, _ := resilience.NewCircuitBreaker(config)
```

### Error Handling

```go
// Old - scattered error types
if err, ok := err.(*circuitbreaker.OpenError); ok {
    // handle
}

// New - unified error checking
if resilience.IsCircuitOpen(err) {
    circuitErr, _ := resilience.AsCircuitOpenError(err)
    // handle with full error details
}
```

### Cache

```go
// Old
import "libs/go/collections/lru"
cache := lru.New[string, int](100)

// New
import "libs/go/src/collections"
cache := collections.NewLRUCache[string, int](100).
    WithTTL(5 * time.Minute).
    WithEvictCallback(func(k string, v int) {
        log.Printf("Evicted: %s", k)
    })
```

## Breaking Changes

1. **Unified Error Types**: All resilience errors now extend `ResilienceError`
2. **Functional Options**: Configuration uses functional options pattern
3. **Result Integration**: Resilience operations return `Result[T]`
4. **Context Support**: All async operations support context cancellation
5. **Generic Types**: Collections use Go generics instead of interface{}

## Deprecation Timeline

- v1.0: Old packages marked deprecated
- v1.1: Deprecation warnings in logs
- v2.0: Old packages removed
