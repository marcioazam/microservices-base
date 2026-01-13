# rust-common

Shared library for cross-cutting concerns in auth-platform Rust services.

## Features

- **Error Types** - Centralized `PlatformError` with retryability classification
- **HTTP Client** - Configured client builder with connection pooling and rustls-tls
- **Retry Policy** - Exponential backoff with configurable jitter
- **Circuit Breaker** - Fail-fast pattern with automatic recovery
- **Logging Client** - gRPC client for Logging_Service with batching
- **Cache Client** - gRPC client for Cache_Service with encryption
- **Tracing** - OpenTelemetry integration
- **Metrics** - Prometheus-compatible counters and gauges

## Usage

```rust
use rust_common::{
    PlatformError,
    HttpConfig, build_http_client,
    RetryPolicy, RetryConfig,
    CircuitBreaker, CircuitBreakerConfig, CircuitState,
    LoggingClient, LoggingClientConfig, LogEntry, LogLevel,
    CacheClient, CacheClientConfig,
};
```

## Modules

### error

Centralized error types with retryability classification:

```rust
use rust_common::PlatformError;

let err = PlatformError::RateLimited;
assert!(err.is_retryable());

let err = PlatformError::NotFound("user".to_string());
assert!(!err.is_retryable());
```

### http

HTTP client configuration with sensible defaults:

```rust
use rust_common::{HttpConfig, build_http_client};
use std::time::Duration;

let config = HttpConfig::default()
    .with_timeout(Duration::from_secs(60))
    .with_connect_timeout(Duration::from_secs(10))
    .with_user_agent("my-service/1.0");

let client = build_http_client(&config)?;
```

### retry

Retry policy with exponential backoff:

```rust
use rust_common::{RetryPolicy, RetryConfig};
use std::time::Duration;

let policy = RetryPolicy::new(
    RetryConfig::default()
        .with_max_retries(3)
        .with_initial_delay(Duration::from_millis(100))
        .with_max_delay(Duration::from_secs(10))
);

let result = policy.execute(|| async {
    // Your operation here
    Ok::<_, rust_common::PlatformError>(42)
}).await?;
```

### circuit_breaker

Circuit breaker for protecting external services:

```rust
use rust_common::{CircuitBreaker, CircuitBreakerConfig, CircuitState};
use std::time::Duration;

let cb = CircuitBreaker::new(
    CircuitBreakerConfig::default()
        .with_failure_threshold(5)
        .with_success_threshold(2)
        .with_timeout(Duration::from_secs(30))
);

if cb.allow_request().await {
    match do_operation().await {
        Ok(_) => cb.record_success().await,
        Err(_) => cb.record_failure().await,
    }
}
```

Configuration options:
| Field | Default | Description |
|-------|---------|-------------|
| `failure_threshold` | 5 | Consecutive failures before opening circuit |
| `success_threshold` | 2 | Consecutive successes in half-open to close |
| `timeout` | 30s | Time before transitioning from open to half-open |
| `half_open_max_requests` | 3 | Maximum requests allowed in half-open state |
```

### logging_client

gRPC client for Logging_Service with batching and fallback:

```rust
use rust_common::{LoggingClient, LoggingClientConfig, LogEntry, LogLevel};

let client = LoggingClient::new(
    LoggingClientConfig::default()
        .with_address("http://logging-service:5001")
        .with_service_id("my-service")
        .with_batch_size(100)
).await?;

// Simple logging
client.info("Operation completed").await;
client.error("Operation failed").await;

// Structured logging with context
let entry = LogEntry::new(LogLevel::Info, "User logged in", "auth-service")
    .with_correlation_id("req-123")
    .with_trace_context("trace-456", "span-789")
    .with_metadata("user_id", "user-123");
client.log(entry).await;

// Flush buffered logs
client.flush().await;
```

### cache_client

gRPC client for Cache_Service with encryption and local fallback:

```rust
use rust_common::{CacheClient, CacheClientConfig};
use std::time::Duration;

let client = CacheClient::new(
    CacheClientConfig::default()
        .with_address("http://cache-service:50051")
        .with_namespace("my-service")
        .with_default_ttl(Duration::from_secs(3600))
        .with_encryption_key([0u8; 32]) // Use secure key in production
).await?;

// Set with default TTL
client.set("key", b"value", None).await?;

// Set with custom TTL
client.set("key", b"value", Some(Duration::from_secs(60))).await?;

// Get
if let Some(value) = client.get("key").await? {
    println!("Got: {:?}", value);
}

// Delete
client.delete("key").await?;
```

### metrics

Prometheus-compatible metrics:

```rust
use rust_common::metrics::{Counter, Gauge, CacheMetrics};

let requests = Counter::new("http_requests_total", "Total HTTP requests");
requests.inc();
requests.inc_by(5);

let connections = Gauge::new("active_connections", "Active connections");
connections.set(10);
connections.inc();
connections.dec();

let cache_metrics = CacheMetrics::new("vault");
cache_metrics.record_hit();
cache_metrics.record_miss();
cache_metrics.update_size(100);

// Export to Prometheus format
println!("{}", requests.to_prometheus());
```

## Testing

```bash
# Run all tests
cargo test -p rust-common

# Run property tests
cargo test -p rust-common property

# Run with verbose output
cargo test -p rust-common -- --nocapture
```

## License

Internal use only.
