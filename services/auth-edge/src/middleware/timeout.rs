//! Timeout Tower Layer
//!
//! Configurable timeout layer for request processing.

use std::future::Future;
use std::pin::Pin;
use std::task::{Context, Poll};
use std::time::Duration;

use futures::future::BoxFuture;
use tower::{Layer, Service};
use tokio::time::timeout;

use crate::error::AuthEdgeError;

/// Timeout layer for Tower
pub struct TimeoutLayer {
    duration: Duration,
}

impl TimeoutLayer {
    /// Creates a new timeout layer with the given duration
    pub fn new(duration: Duration) -> Self {
        Self { duration }
    }

    /// Creates a new timeout layer from seconds
    pub fn from_secs(secs: u64) -> Self {
        Self::new(Duration::from_secs(secs))
    }
}

impl<S> Layer<S> for TimeoutLayer {
    type Service = TimeoutService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        TimeoutService {
            inner,
            duration: self.duration,
        }
    }
}

/// Timeout service wrapper
pub struct TimeoutService<S> {
    inner: S,
    duration: Duration,
}

impl<S: Clone> Clone for TimeoutService<S> {
    fn clone(&self) -> Self {
        Self {
            inner: self.inner.clone(),
            duration: self.duration,
        }
    }
}

impl<S, Req> Service<Req> for TimeoutService<S>
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
        let duration = self.duration;
        let mut inner = self.inner.clone();

        Box::pin(async move {
            match timeout(duration, inner.call(req)).await {
                Ok(result) => result.map_err(Into::into),
                Err(_) => Err(AuthEdgeError::Timeout { duration }),
            }
        })
    }
}
