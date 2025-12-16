//! Rate Limiter Tower Layer
//!
//! Implements rate limiting as a Tower Layer with HTTP headers.

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use std::task::{Context, Poll};
use std::time::Duration;

use futures::future::BoxFuture;
use tower::{Layer, Service};

use crate::error::AuthEdgeError;
use crate::rate_limiter::{AdaptiveRateLimiter, RateLimitConfig, RateLimitDecision};

/// Rate limiter layer for Tower
pub struct RateLimiterLayer {
    limiter: Arc<AdaptiveRateLimiter>,
}

impl RateLimiterLayer {
    /// Creates a new rate limiter layer with the given configuration
    pub fn new(config: RateLimitConfig) -> Self {
        Self {
            limiter: Arc::new(AdaptiveRateLimiter::new(config)),
        }
    }

    /// Creates a new rate limiter layer with default configuration
    pub fn with_defaults() -> Self {
        Self::new(RateLimitConfig::default())
    }
}

impl<S> Layer<S> for RateLimiterLayer {
    type Service = RateLimiterService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        RateLimiterService {
            inner,
            limiter: self.limiter.clone(),
        }
    }
}

/// Rate limiter service wrapper
pub struct RateLimiterService<S> {
    inner: S,
    limiter: Arc<AdaptiveRateLimiter>,
}

impl<S: Clone> Clone for RateLimiterService<S> {
    fn clone(&self) -> Self {
        Self {
            inner: self.inner.clone(),
            limiter: self.limiter.clone(),
        }
    }
}

/// Response wrapper that includes rate limit headers
#[derive(Debug)]
pub struct RateLimitedResponse<T> {
    pub response: T,
    pub remaining: u32,
    pub reset_at: Duration,
}

impl<S, Req> Service<Req> for RateLimiterService<S>
where
    S: Service<Req> + Clone + Send + 'static,
    S::Response: Send + 'static,
    S::Error: Into<AuthEdgeError> + Send + 'static,
    S::Future: Send + 'static,
    Req: Send + 'static,
{
    type Response = S::Response;
    type Error = AuthEdgeError;
    type Future = BoxFuture<'static, Result<Self::Response, Self::Error>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx).map_err(Into::into)
    }

    fn call(&mut self, req: Req) -> Self::Future {
        let limiter = self.limiter.clone();
        let mut inner = self.inner.clone();

        Box::pin(async move {
            // Use a default client ID for now - in production this would come from the request
            let client_id = "default";

            match limiter.check(client_id).await {
                RateLimitDecision::Allowed => {
                    let result = inner.call(req).await;
                    
                    // Record outcome for adaptive rate limiting
                    limiter.record_outcome(client_id, result.is_ok()).await;
                    
                    result.map_err(Into::into)
                }
                RateLimitDecision::Denied { retry_after } => {
                    Err(AuthEdgeError::RateLimited { retry_after })
                }
            }
        })
    }
}

/// Extracts rate limit headers from a response
pub struct RateLimitHeaders {
    pub remaining: u32,
    pub limit: u32,
    pub reset: u64,
}

impl RateLimitHeaders {
    /// Creates headers from rate limit info
    pub fn new(remaining: u32, limit: u32, reset_secs: u64) -> Self {
        Self {
            remaining,
            limit,
            reset: reset_secs,
        }
    }

    /// Returns the X-RateLimit-Remaining header value
    pub fn remaining_header(&self) -> String {
        self.remaining.to_string()
    }

    /// Returns the X-RateLimit-Limit header value
    pub fn limit_header(&self) -> String {
        self.limit.to_string()
    }

    /// Returns the X-RateLimit-Reset header value
    pub fn reset_header(&self) -> String {
        self.reset.to_string()
    }
}
