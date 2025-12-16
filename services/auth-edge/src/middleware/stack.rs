//! Service Stack Builder
//!
//! Composes middleware layers in the correct order.

use std::time::Duration;

use tower::ServiceBuilder;

use crate::circuit_breaker::CircuitBreakerLayer;
use crate::config::Config;
use crate::middleware::rate_limiter::RateLimiterLayer;
use crate::middleware::timeout::TimeoutLayer;
use crate::middleware::tracing::TracingLayer;
use crate::rate_limiter::RateLimitConfig;

/// Builds the complete service stack with all middleware layers
/// 
/// Layer order (outermost to innermost):
/// 1. Tracing - captures all requests and errors
/// 2. Timeout - enforces request timeout
/// 3. RateLimit - prevents abuse
/// 4. CircuitBreaker - protects downstream services
/// 5. Inner Service - actual request handler
pub fn build_service_stack<S>(
    inner: S,
    config: &Config,
) -> impl tower::Service<
    tonic::Request<()>,
    Response = tonic::Response<()>,
    Error = crate::error::AuthEdgeError,
>
where
    S: tower::Service<tonic::Request<()>, Response = tonic::Response<()>> + Clone + Send + 'static,
    S::Error: Into<crate::error::AuthEdgeError> + Send + 'static,
    S::Future: Send + 'static,
{
    ServiceBuilder::new()
        .layer(TracingLayer::new("auth-edge-service"))
        .layer(TimeoutLayer::from_secs(config.timeout_secs()))
        .layer(RateLimiterLayer::new(RateLimitConfig::default()))
        .layer(CircuitBreakerLayer::<5, 3, 30>::new("downstream"))
        .service(inner)
}

/// Configuration extension for middleware
impl Config {
    /// Gets the timeout in seconds
    pub fn timeout_secs(&self) -> u64 {
        30 // Default timeout
    }
}
