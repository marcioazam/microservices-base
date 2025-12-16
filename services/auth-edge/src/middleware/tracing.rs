//! Tracing Tower Layer with OpenTelemetry Integration
//!
//! Provides W3C trace context propagation and structured error recording.

use std::future::Future;
use std::pin::Pin;
use std::task::{Context, Poll};

use futures::future::BoxFuture;
use tower::{Layer, Service};
use tracing::{info_span, instrument, Instrument, Span};
use uuid::Uuid;

use crate::error::AuthEdgeError;

/// Tracing layer for Tower with OpenTelemetry integration
pub struct TracingLayer {
    service_name: String,
}

impl TracingLayer {
    /// Creates a new tracing layer
    pub fn new(service_name: impl Into<String>) -> Self {
        Self {
            service_name: service_name.into(),
        }
    }
}

impl<S> Layer<S> for TracingLayer {
    type Service = TracingService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        TracingService {
            inner,
            service_name: self.service_name.clone(),
        }
    }
}

/// Tracing service wrapper
pub struct TracingService<S> {
    inner: S,
    service_name: String,
}

impl<S: Clone> Clone for TracingService<S> {
    fn clone(&self) -> Self {
        Self {
            inner: self.inner.clone(),
            service_name: self.service_name.clone(),
        }
    }
}

impl<S, Req> Service<Req> for TracingService<S>
where
    S: Service<Req> + Clone + Send + 'static,
    S::Response: Send + 'static,
    S::Error: Into<AuthEdgeError> + std::fmt::Debug + Send + 'static,
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
        let correlation_id = Uuid::new_v4();
        let service_name = self.service_name.clone();
        let mut inner = self.inner.clone();

        let span = info_span!(
            "request",
            service = %service_name,
            correlation_id = %correlation_id,
            otel.kind = "server"
        );

        Box::pin(
            async move {
                let result = inner.call(req).await;

                match &result {
                    Ok(_) => {
                        tracing::info!(
                            correlation_id = %correlation_id,
                            "Request completed successfully"
                        );
                    }
                    Err(err) => {
                        tracing::error!(
                            correlation_id = %correlation_id,
                            error = ?err,
                            error_type = std::any::type_name::<S::Error>(),
                            "Request failed"
                        );
                    }
                }

                result.map_err(Into::into)
            }
            .instrument(span),
        )
    }
}

/// Error event attributes for structured logging
#[derive(Debug, Clone)]
pub struct ErrorEventAttributes {
    pub correlation_id: Uuid,
    pub error_type: String,
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub service_name: String,
}

impl ErrorEventAttributes {
    /// Creates new error event attributes
    pub fn new(correlation_id: Uuid, error_type: impl Into<String>, service_name: impl Into<String>) -> Self {
        Self {
            correlation_id,
            error_type: error_type.into(),
            timestamp: chrono::Utc::now(),
            service_name: service_name.into(),
        }
    }

    /// Records the error event to the current span
    pub fn record(&self) {
        tracing::error!(
            correlation_id = %self.correlation_id,
            error_type = %self.error_type,
            timestamp = %self.timestamp.to_rfc3339(),
            service = %self.service_name,
            "Error event recorded"
        );
    }
}
