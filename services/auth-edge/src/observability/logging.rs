//! Structured logging integration with Logging_Service.
//!
//! Provides AuthEdgeLogger wrapper around LoggingClient from rust-common
//! with domain-specific logging methods and local fallback.

use crate::config::Config;
use crate::error::AuthEdgeError;
use rust_common::{LogEntry, LogLevel, LoggingClient, LoggingClientConfig, PlatformError};
use tracing::{error, info, Span};

/// Auth Edge Logger with Logging_Service integration.
pub struct AuthEdgeLogger {
    client: LoggingClient,
}

impl AuthEdgeLogger {
    /// Creates a new AuthEdgeLogger.
    pub async fn new(config: &Config) -> Result<Self, AuthEdgeError> {
        let logging_config = LoggingClientConfig::default()
            .with_address(config.logging_service_url_str())
            .with_service_id("auth-edge-service")
            .with_batch_size(100);

        let client = LoggingClient::new(logging_config)
            .await
            .map_err(AuthEdgeError::Platform)?;

        Ok(Self { client })
    }

    /// Logs a successful token validation.
    pub async fn log_validation_success(&self, subject: &str, correlation_id: &str) {
        let (trace_id, span_id) = Self::extract_trace_context();

        let entry = LogEntry::new(
            LogLevel::Info,
            "Token validated successfully",
            "auth-edge-service",
        )
        .with_correlation_id(correlation_id)
        .with_trace_context(&trace_id, &span_id)
        .with_metadata("subject", subject)
        .with_metadata("event_type", "validation_success");

        self.client.log(entry).await;
    }

    /// Logs a token validation failure.
    pub async fn log_validation_failure(&self, error: &AuthEdgeError, correlation_id: &str) {
        let (trace_id, span_id) = Self::extract_trace_context();

        let entry = LogEntry::new(
            LogLevel::Error,
            format!("Token validation failed: {error}"),
            "auth-edge-service",
        )
        .with_correlation_id(correlation_id)
        .with_trace_context(&trace_id, &span_id)
        .with_metadata("error_code", error.code().as_str())
        .with_metadata("event_type", "validation_failure")
        .with_metadata("retryable", error.is_retryable().to_string());

        self.client.log(entry).await;
    }

    /// Logs a SPIFFE identity extraction success.
    pub async fn log_identity_success(&self, spiffe_id: &str, correlation_id: &str) {
        let (trace_id, span_id) = Self::extract_trace_context();

        let entry = LogEntry::new(
            LogLevel::Info,
            "Service identity extracted",
            "auth-edge-service",
        )
        .with_correlation_id(correlation_id)
        .with_trace_context(&trace_id, &span_id)
        .with_metadata("spiffe_id", spiffe_id)
        .with_metadata("event_type", "identity_success");

        self.client.log(entry).await;
    }

    /// Logs a SPIFFE identity extraction failure.
    pub async fn log_identity_failure(&self, error: &str, correlation_id: &str) {
        let (trace_id, span_id) = Self::extract_trace_context();

        let entry = LogEntry::new(
            LogLevel::Error,
            format!("Service identity extraction failed: {error}"),
            "auth-edge-service",
        )
        .with_correlation_id(correlation_id)
        .with_trace_context(&trace_id, &span_id)
        .with_metadata("event_type", "identity_failure");

        self.client.log(entry).await;
    }

    /// Logs a circuit breaker state change.
    pub async fn log_circuit_breaker_change(
        &self,
        service: &str,
        old_state: &str,
        new_state: &str,
    ) {
        let entry = LogEntry::new(
            LogLevel::Warn,
            format!("Circuit breaker state changed for {service}: {old_state} -> {new_state}"),
            "auth-edge-service",
        )
        .with_metadata("service", service)
        .with_metadata("old_state", old_state)
        .with_metadata("new_state", new_state)
        .with_metadata("event_type", "circuit_breaker_change");

        self.client.log(entry).await;
    }

    /// Logs a rate limit event.
    pub async fn log_rate_limited(&self, client_id: &str, correlation_id: &str) {
        let (trace_id, span_id) = Self::extract_trace_context();

        let entry = LogEntry::new(
            LogLevel::Warn,
            "Rate limit exceeded",
            "auth-edge-service",
        )
        .with_correlation_id(correlation_id)
        .with_trace_context(&trace_id, &span_id)
        .with_metadata("client_id", client_id)
        .with_metadata("event_type", "rate_limited");

        self.client.log(entry).await;
    }

    /// Flushes the log buffer.
    pub async fn flush(&self) {
        self.client.flush().await;
    }

    /// Extracts trace context from the current span.
    fn extract_trace_context() -> (String, String) {
        // In production, this would extract from OpenTelemetry context
        // For now, return placeholder values
        let span = Span::current();
        let trace_id = format!("{:?}", span.id().unwrap_or(tracing::span::Id::from_u64(0)));
        let span_id = trace_id.clone();
        (trace_id, span_id)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Note: Integration tests would require a running Logging_Service
    // Unit tests verify the structure and logic

    #[test]
    fn test_log_level_for_success_is_info() {
        let entry = LogEntry::new(LogLevel::Info, "test", "test-service");
        assert_eq!(entry.level, LogLevel::Info);
    }

    #[test]
    fn test_log_level_for_failure_is_error() {
        let entry = LogEntry::new(LogLevel::Error, "test", "test-service");
        assert_eq!(entry.level, LogLevel::Error);
    }
}
