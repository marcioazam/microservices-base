# Migration Guide: Cache Client

## Overview

This guide helps migrate from local cache implementations to the centralized cache client that uses the cache-service microservice.

## Before Migration

If you were using:
- `libs/go/src/collections/lru.go` - Local LRU cache
- Custom in-memory caching solutions

## After Migration

Use `libs/go/src/cache` which provides:
- Remote caching via cache-service (gRPC)
- Built-in circuit breaker for fault tolerance
- Local fallback cache for resilience
- Namespace isolation for multi-tenant scenarios

## Migration Steps

### 1. Update Imports

```go
// Before
import "github.com/authcorp/libs/go/src/collections"

// After
import "github.com/authcorp/libs/go/src/cache"
```

### 2. Replace Cache Initialization

```go
// Before
lru := collections.NewLRU[string, []byte](1000)

// After
client, err := cache.NewClient(cache.ClientConfig{
    Address:   "cache-service:50051",
    Namespace: "my-service",
    LocalFallback: true,
})
```

### 3. Update Operations

```go
// Before
lru.Put("key", value)
val, ok := lru.Get("key")

// After
ctx := context.Background()
err := client.Set(ctx, "key", value, time.Hour)
result := client.Get(ctx, "key")
if result.IsOk() {
    val := result.Unwrap()
}
```

### 4. Handle Errors

The new client uses `functional.Result` for error handling:

```go
result := client.Get(ctx, "key")
result.Match(
    func(val []byte) { /* success */ },
    func(err error) { /* handle error */ },
)
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Address | cache-service gRPC address | localhost:50051 |
| Namespace | Key namespace prefix | "" |
| Timeout | Operation timeout | 5s |
| LocalFallback | Enable local cache fallback | false |
| LocalCacheSize | Local cache max entries | 1000 |
| CircuitBreaker | Enable circuit breaker | true |

## Benefits

1. **Centralized caching** - Single source of truth
2. **Fault tolerance** - Circuit breaker + local fallback
3. **Observability** - Integrated metrics and tracing
4. **Scalability** - Distributed cache via Redis
5. **Consistency** - Cache invalidation across services
