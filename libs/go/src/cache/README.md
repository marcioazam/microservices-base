# Cache Client Library

Distributed cache client for communication with `cache-service` via gRPC.

## Features

- gRPC client for cache-service
- Circuit breaker integration for resilience
- Local fallback cache when remote is unavailable
- Namespace isolation for multi-tenant scenarios
- Configurable timeouts and retries

## Usage

```go
package main

import (
    "context"
    "time"
    
    "github.com/authcorp/libs/go/src/cache"
)

func main() {
    // Create client with default config
    config := cache.DefaultConfig()
    config.Address = "cache-service:50051"
    config.Namespace = "my-service"
    
    client, err := cache.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    ctx := context.Background()
    
    // Set a value
    err = client.Set(ctx, "user:123", []byte(`{"name":"John"}`), 5*time.Minute)
    if err != nil {
        panic(err)
    }
    
    // Get a value
    result := client.Get(ctx, "user:123")
    if result.IsOk() {
        entry := result.Unwrap()
        fmt.Printf("Value: %s, Source: %v\n", entry.Value, entry.Source)
    }
    
    // Delete a value
    err = client.Delete(ctx, "user:123")
    if err != nil {
        panic(err)
    }
}
```

## Testing

For unit tests, use `LocalOnly` to create a client without network:

```go
client := cache.LocalOnly(1000)
defer client.Close()

// Use client in tests
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| Address | localhost:50051 | cache-service gRPC address |
| Namespace | default | Key namespace for isolation |
| Timeout | 5s | Operation timeout |
| MaxRetries | 3 | Max retry attempts |
| LocalFallback | true | Enable local cache fallback |
| LocalCacheSize | 10000 | Max local cache entries |
