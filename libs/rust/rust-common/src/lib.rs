//! Shared library for cross-cutting concerns in auth-platform Rust services.
//!
//! This crate provides centralized implementations for:
//! - Error types with retryability classification
//! - HTTP client configuration and building
//! - Retry policies with exponential backoff
//! - Circuit breaker pattern for resilience
//! - Logging service gRPC client
//! - Cache service gRPC client
//! - OpenTelemetry tracing integration
//! - Prometheus metrics helpers

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod error;
pub mod http;
pub mod retry;
pub mod circuit_breaker;
pub mod logging_client;
pub mod cache_client;
pub mod tracing_config;
pub mod metrics;

pub use error::PlatformError;
pub use http::{HttpConfig, build_http_client};
pub use retry::{RetryPolicy, RetryConfig};
pub use circuit_breaker::{CircuitBreaker, CircuitBreakerConfig, CircuitState};
pub use logging_client::{LoggingClient, LoggingClientConfig, LogEntry, LogLevel};
pub use cache_client::{CacheClient, CacheClientConfig};
