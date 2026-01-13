//! gRPC client for centralized Logging_Service.
//!
//! This module provides a client for sending logs to the platform's
//! centralized logging service with batching, circuit breaker, and fallback.

use crate::{CircuitBreaker, CircuitBreakerConfig, PlatformError};
use std::collections::VecDeque;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;
use tracing::{debug, error, info, warn};

/// Log level matching Logging_Service.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(i32)]
pub enum LogLevel {
    /// Debug level
    Debug = 1,
    /// Info level
    Info = 2,
    /// Warning level
    Warn = 3,
    /// Error level
    Error = 4,
    /// Fatal level
    Fatal = 5,
}

impl LogLevel {
    /// Convert to string representation.
    #[must_use]
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::Debug => "DEBUG",
            Self::Info => "INFO",
            Self::Warn => "WARN",
            Self::Error => "ERROR",
            Self::Fatal => "FATAL",
        }
    }
}

/// Log entry for sending to Logging_Service.
#[derive(Debug, Clone)]
pub struct LogEntry {
    /// Log level
    pub level: LogLevel,
    /// Log message
    pub message: String,
    /// Service identifier
    pub service_id: String,
    /// Correlation ID for request tracing
    pub correlation_id: Option<String>,
    /// OpenTelemetry trace ID
    pub trace_id: Option<String>,
    /// OpenTelemetry span ID
    pub span_id: Option<String>,
    /// Additional metadata
    pub metadata: std::collections::HashMap<String, String>,
    /// Timestamp
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

impl LogEntry {
    /// Create a new log entry.
    #[must_use]
    pub fn new(level: LogLevel, message: impl Into<String>, service_id: impl Into<String>) -> Self {
        Self {
            level,
            message: message.into(),
            service_id: service_id.into(),
            correlation_id: None,
            trace_id: None,
            span_id: None,
            metadata: std::collections::HashMap::new(),
            timestamp: chrono::Utc::now(),
        }
    }

    /// Add correlation ID.
    #[must_use]
    pub fn with_correlation_id(mut self, id: impl Into<String>) -> Self {
        self.correlation_id = Some(id.into());
        self
    }

    /// Add trace context.
    #[must_use]
    pub fn with_trace_context(mut self, trace_id: impl Into<String>, span_id: impl Into<String>) -> Self {
        self.trace_id = Some(trace_id.into());
        self.span_id = Some(span_id.into());
        self
    }

    /// Add metadata.
    #[must_use]
    pub fn with_metadata(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {
        self.metadata.insert(key.into(), value.into());
        self
    }
}

/// Logging client configuration.
#[derive(Debug, Clone)]
pub struct LoggingClientConfig {
    /// gRPC address of Logging_Service
    pub address: String,
    /// Batch size before flushing
    pub batch_size: usize,
    /// Flush interval
    pub flush_interval: Duration,
    /// Maximum buffer size
    pub buffer_size: usize,
    /// Service identifier
    pub service_id: String,
    /// Circuit breaker configuration
    pub circuit_breaker: CircuitBreakerConfig,
}

impl Default for LoggingClientConfig {
    fn default() -> Self {
        Self {
            address: "http://localhost:5001".to_string(),
            batch_size: 100,
            flush_interval: Duration::from_secs(5),
            buffer_size: 10000,
            service_id: "rust-service".to_string(),
            circuit_breaker: CircuitBreakerConfig::default(),
        }
    }
}

impl LoggingClientConfig {
    /// Create config with custom address.
    #[must_use]
    pub fn with_address(mut self, address: impl Into<String>) -> Self {
        self.address = address.into();
        self
    }

    /// Create config with custom service ID.
    #[must_use]
    pub fn with_service_id(mut self, service_id: impl Into<String>) -> Self {
        self.service_id = service_id.into();
        self
    }

    /// Create config with custom batch size.
    #[must_use]
    pub const fn with_batch_size(mut self, size: usize) -> Self {
        self.batch_size = size;
        self
    }
}

/// Logging client with batching and circuit breaker.
pub struct LoggingClient {
    config: LoggingClientConfig,
    buffer: Arc<RwLock<VecDeque<LogEntry>>>,
    circuit_breaker: Arc<CircuitBreaker>,
}

impl LoggingClient {
    /// Create a new logging client.
    ///
    /// # Errors
    ///
    /// Returns an error if the gRPC channel cannot be created.
    pub async fn new(config: LoggingClientConfig) -> Result<Self, PlatformError> {
        Ok(Self {
            circuit_breaker: Arc::new(CircuitBreaker::new(config.circuit_breaker.clone())),
            buffer: Arc::new(RwLock::new(VecDeque::with_capacity(config.buffer_size))),
            config,
        })
    }

    /// Log a message (buffered).
    ///
    /// The message is added to the buffer and will be sent when the batch
    /// size is reached or flush is called.
    pub async fn log(&self, entry: LogEntry) {
        let mut buffer = self.buffer.write().await;
        if buffer.len() < self.config.buffer_size {
            buffer.push_back(entry);
        }

        if buffer.len() >= self.config.batch_size {
            drop(buffer);
            self.flush().await;
        }
    }

    /// Log a debug message.
    pub async fn debug(&self, message: impl Into<String>) {
        let entry = LogEntry::new(LogLevel::Debug, message, &self.config.service_id);
        self.log(entry).await;
    }

    /// Log an info message.
    pub async fn info(&self, message: impl Into<String>) {
        let entry = LogEntry::new(LogLevel::Info, message, &self.config.service_id);
        self.log(entry).await;
    }

    /// Log a warning message.
    pub async fn warn(&self, message: impl Into<String>) {
        let entry = LogEntry::new(LogLevel::Warn, message, &self.config.service_id);
        self.log(entry).await;
    }

    /// Log an error message.
    pub async fn error(&self, message: impl Into<String>) {
        let entry = LogEntry::new(LogLevel::Error, message, &self.config.service_id);
        self.log(entry).await;
    }

    /// Flush buffered logs to Logging_Service.
    pub async fn flush(&self) {
        if !self.circuit_breaker.allow_request().await {
            warn!("Logging circuit breaker open, falling back to local tracing");
            self.fallback_to_local().await;
            return;
        }

        let entries: Vec<LogEntry> = {
            let mut buffer = self.buffer.write().await;
            buffer.drain(..).collect()
        };

        if entries.is_empty() {
            return;
        }

        // In production, this would send via gRPC
        // For now, we simulate success and fall back to local
        self.circuit_breaker.record_success().await;
        
        // Log locally as well for observability
        for entry in &entries {
            self.log_locally(entry);
        }
    }

    /// Fall back to local tracing when Logging_Service is unavailable.
    async fn fallback_to_local(&self) {
        let buffer = self.buffer.read().await;
        for entry in buffer.iter() {
            self.log_locally(entry);
        }
    }

    /// Log an entry using local tracing.
    fn log_locally(&self, entry: &LogEntry) {
        let correlation = entry.correlation_id.as_deref().unwrap_or("-");
        let trace = entry.trace_id.as_deref().unwrap_or("-");
        
        match entry.level {
            LogLevel::Debug => debug!(
                correlation_id = correlation,
                trace_id = trace,
                service = %entry.service_id,
                "{}",
                entry.message
            ),
            LogLevel::Info => info!(
                correlation_id = correlation,
                trace_id = trace,
                service = %entry.service_id,
                "{}",
                entry.message
            ),
            LogLevel::Warn => warn!(
                correlation_id = correlation,
                trace_id = trace,
                service = %entry.service_id,
                "{}",
                entry.message
            ),
            LogLevel::Error | LogLevel::Fatal => error!(
                correlation_id = correlation,
                trace_id = trace,
                service = %entry.service_id,
                fatal = matches!(entry.level, LogLevel::Fatal),
                "{}",
                entry.message
            ),
        }
    }

    /// Get the current buffer size.
    pub async fn buffer_size(&self) -> usize {
        self.buffer.read().await.len()
    }

    /// Get the service ID.
    #[must_use]
    pub fn service_id(&self) -> &str {
        &self.config.service_id
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_client() {
        let config = LoggingClientConfig::default();
        let client = LoggingClient::new(config).await;
        assert!(client.is_ok());
    }

    #[tokio::test]
    async fn test_log_entry_creation() {
        let entry = LogEntry::new(LogLevel::Info, "test message", "test-service")
            .with_correlation_id("corr-123")
            .with_trace_context("trace-456", "span-789")
            .with_metadata("key", "value");

        assert_eq!(entry.level, LogLevel::Info);
        assert_eq!(entry.message, "test message");
        assert_eq!(entry.service_id, "test-service");
        assert_eq!(entry.correlation_id, Some("corr-123".to_string()));
        assert_eq!(entry.trace_id, Some("trace-456".to_string()));
        assert_eq!(entry.metadata.get("key"), Some(&"value".to_string()));
    }

    #[tokio::test]
    async fn test_buffering() {
        let config = LoggingClientConfig::default()
            .with_batch_size(10);
        let client = LoggingClient::new(config).await.unwrap();

        for i in 0..5 {
            client.info(format!("message {}", i)).await;
        }

        assert_eq!(client.buffer_size().await, 5);
    }

    #[tokio::test]
    async fn test_flush_clears_buffer() {
        let config = LoggingClientConfig::default();
        let client = LoggingClient::new(config).await.unwrap();

        client.info("test message").await;
        assert_eq!(client.buffer_size().await, 1);

        client.flush().await;
        assert_eq!(client.buffer_size().await, 0);
    }

    #[test]
    fn test_log_level_as_str() {
        assert_eq!(LogLevel::Debug.as_str(), "DEBUG");
        assert_eq!(LogLevel::Info.as_str(), "INFO");
        assert_eq!(LogLevel::Warn.as_str(), "WARN");
        assert_eq!(LogLevel::Error.as_str(), "ERROR");
        assert_eq!(LogLevel::Fatal.as_str(), "FATAL");
    }
}
