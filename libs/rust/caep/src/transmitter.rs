//! CAEP Transmitter implementation.
//!
//! This module provides the transmitter for emitting CAEP events using native async traits.

use crate::{CaepError, CaepEvent, CaepResult, SecurityEventToken, Stream, StreamConfig, StreamStatus};
use jsonwebtoken::{Algorithm, EncodingKey};
use rust_common::{LoggingClient, LoggingClientConfig, LogEntry, LogLevel};
use std::future::Future;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{error, info, instrument};

/// Default signing algorithm (ES256).
pub const DEFAULT_ALGORITHM: Algorithm = Algorithm::ES256;

/// CAEP Transmitter trait for emitting security events.
///
/// Uses native async traits (Rust 2024).
pub trait CaepTransmitter: Send + Sync {
    /// Emit a security event to all registered streams.
    fn emit(&self, event: CaepEvent) -> impl Future<Output = CaepResult<EmitResult>> + Send;

    /// Register a new stream receiver.
    fn register_stream(&self, config: StreamConfig) -> impl Future<Output = CaepResult<String>> + Send;

    /// Remove a stream.
    fn remove_stream(&self, stream_id: &str) -> impl Future<Output = CaepResult<()>> + Send;

    /// Get stream status.
    fn stream_status(&self, stream_id: &str) -> impl Future<Output = CaepResult<StreamStatus>> + Send;

    /// List all streams.
    fn list_streams(&self) -> impl Future<Output = CaepResult<Vec<Stream>>> + Send;
}

/// Result of emitting an event.
#[derive(Debug, Clone)]
pub struct EmitResult {
    /// Unique event ID
    pub event_id: String,
    /// Number of streams successfully notified
    pub streams_notified: usize,
    /// Number of streams that failed
    pub streams_failed: usize,
    /// Delivery times in milliseconds
    pub delivery_times_ms: Vec<u64>,
}

impl EmitResult {
    /// Check if all deliveries succeeded.
    #[must_use]
    pub const fn all_succeeded(&self) -> bool {
        self.streams_failed == 0
    }

    /// Get the average delivery time.
    #[must_use]
    pub fn avg_delivery_time_ms(&self) -> f64 {
        if self.delivery_times_ms.is_empty() {
            0.0
        } else {
            self.delivery_times_ms.iter().sum::<u64>() as f64 / self.delivery_times_ms.len() as f64
        }
    }
}

/// Default CAEP Transmitter implementation.
pub struct DefaultCaepTransmitter {
    issuer: String,
    signing_key: EncodingKey,
    algorithm: Algorithm,
    streams: Arc<RwLock<Vec<Stream>>>,
    http_client: reqwest::Client,
    logging_client: Option<Arc<LoggingClient>>,
}

impl DefaultCaepTransmitter {
    /// Create a new transmitter.
    #[must_use]
    pub fn new(issuer: impl Into<String>, signing_key: EncodingKey) -> Self {
        Self {
            issuer: issuer.into(),
            signing_key,
            algorithm: DEFAULT_ALGORITHM,
            streams: Arc::new(RwLock::new(Vec::new())),
            http_client: reqwest::Client::new(),
            logging_client: None,
        }
    }

    /// Set a custom signing algorithm.
    #[must_use]
    pub const fn with_algorithm(mut self, algorithm: Algorithm) -> Self {
        self.algorithm = algorithm;
        self
    }

    /// Set a logging client for structured logging.
    #[must_use]
    pub fn with_logging_client(mut self, client: Arc<LoggingClient>) -> Self {
        self.logging_client = Some(client);
        self
    }

    /// Create a transmitter with logging enabled.
    pub async fn with_logging(mut self, config: LoggingClientConfig) -> CaepResult<Self> {
        let client = LoggingClient::new(config).await?;
        self.logging_client = Some(Arc::new(client));
        Ok(self)
    }

    /// Deliver a SET to a stream.
    #[instrument(skip(self, set))]
    async fn deliver_to_stream(&self, stream: &Stream, set: &str) -> CaepResult<u64> {
        let start = std::time::Instant::now();

        match &stream.config.delivery {
            crate::DeliveryMethod::Push { endpoint_url } => {
                let response = self
                    .http_client
                    .post(endpoint_url)
                    .header("Content-Type", "application/secevent+jwt")
                    .body(set.to_string())
                    .send()
                    .await
                    .map_err(|e| CaepError::delivery_failed(e.to_string()))?;

                if !response.status().is_success() {
                    return Err(CaepError::delivery_failed(format!(
                        "HTTP {}",
                        response.status()
                    )));
                }
            }
            crate::DeliveryMethod::Poll => {
                // For poll delivery, we just store the event
                // The receiver will poll for it
            }
        }

        Ok(start.elapsed().as_millis() as u64)
    }

    /// Log a message using the logging client if available.
    async fn log(&self, level: LogLevel, message: &str, correlation_id: Option<&str>) {
        if let Some(ref client) = self.logging_client {
            let mut entry = LogEntry::new(level, message, "caep-transmitter");
            if let Some(id) = correlation_id {
                entry = entry.with_correlation_id(id);
            }
            client.log(entry).await;
        }
    }
}

impl CaepTransmitter for DefaultCaepTransmitter {
    #[instrument(skip(self))]
    async fn emit(&self, event: CaepEvent) -> CaepResult<EmitResult> {
        let event_id = uuid::Uuid::new_v4().to_string();

        let streams = self.streams.read().await;
        let active_streams: Vec<_> = streams
            .iter()
            .filter(|s| s.status == StreamStatus::Active)
            .filter(|s| s.config.events_requested.contains(&event.event_type))
            .collect();

        if active_streams.is_empty() {
            info!("No active streams for event type {:?}", event.event_type);
            self.log(
                LogLevel::Info,
                &format!("No active streams for event type {:?}", event.event_type),
                Some(&event_id),
            )
            .await;

            return Ok(EmitResult {
                event_id,
                streams_notified: 0,
                streams_failed: 0,
                delivery_times_ms: vec![],
            });
        }

        let mut streams_notified = 0;
        let mut streams_failed = 0;
        let mut delivery_times = Vec::new();

        for stream in active_streams {
            let set = SecurityEventToken::from_event(&event, &self.issuer, &stream.config.audience);
            let signed_set = set.sign(&self.signing_key)?;

            match self.deliver_to_stream(stream, &signed_set).await {
                Ok(time_ms) => {
                    streams_notified += 1;
                    delivery_times.push(time_ms);
                    info!(
                        stream_id = %stream.id,
                        delivery_time_ms = time_ms,
                        "Event delivered successfully"
                    );
                    self.log(
                        LogLevel::Info,
                        &format!("Event delivered to stream {} in {}ms", stream.id, time_ms),
                        Some(&event_id),
                    )
                    .await;
                }
                Err(e) => {
                    streams_failed += 1;
                    error!(
                        stream_id = %stream.id,
                        error = %e,
                        "Failed to deliver event"
                    );
                    self.log(
                        LogLevel::Error,
                        &format!("Failed to deliver to stream {}: {}", stream.id, e),
                        Some(&event_id),
                    )
                    .await;
                }
            }
        }

        Ok(EmitResult {
            event_id,
            streams_notified,
            streams_failed,
            delivery_times_ms: delivery_times,
        })
    }

    async fn register_stream(&self, config: StreamConfig) -> CaepResult<String> {
        let stream = Stream::new(config);
        let id = stream.id.clone();

        let mut streams = self.streams.write().await;
        streams.push(stream);

        info!(stream_id = %id, "Stream registered");
        self.log(LogLevel::Info, &format!("Stream registered: {}", id), None)
            .await;

        Ok(id)
    }

    async fn remove_stream(&self, stream_id: &str) -> CaepResult<()> {
        let mut streams = self.streams.write().await;
        let initial_len = streams.len();
        streams.retain(|s| s.id != stream_id);

        if streams.len() == initial_len {
            return Err(CaepError::stream_not_found(stream_id));
        }

        info!(stream_id = %stream_id, "Stream removed");
        self.log(
            LogLevel::Info,
            &format!("Stream removed: {}", stream_id),
            None,
        )
        .await;

        Ok(())
    }

    async fn stream_status(&self, stream_id: &str) -> CaepResult<StreamStatus> {
        let streams = self.streams.read().await;
        streams
            .iter()
            .find(|s| s.id == stream_id)
            .map(|s| s.status.clone())
            .ok_or_else(|| CaepError::stream_not_found(stream_id))
    }

    async fn list_streams(&self) -> CaepResult<Vec<Stream>> {
        let streams = self.streams.read().await;
        Ok(streams.clone())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::event::SubjectIdentifier;

    fn test_signing_key() -> EncodingKey {
        // Use a test key (in production, use proper EC key)
        EncodingKey::from_secret(b"test-secret-key-for-testing-only")
    }

    #[tokio::test]
    async fn test_transmitter_creation() {
        let transmitter = DefaultCaepTransmitter::new("https://issuer.com", test_signing_key());
        assert_eq!(transmitter.algorithm, Algorithm::ES256);
    }

    #[tokio::test]
    async fn test_emit_no_streams() {
        let transmitter = DefaultCaepTransmitter::new("https://issuer.com", test_signing_key());

        let subject = SubjectIdentifier::email("user@example.com");
        let event = CaepEvent::session_revoked(subject, None);

        let result = transmitter.emit(event).await.unwrap();
        assert_eq!(result.streams_notified, 0);
        assert_eq!(result.streams_failed, 0);
    }

    #[tokio::test]
    async fn test_register_and_list_streams() {
        let transmitter = DefaultCaepTransmitter::new("https://issuer.com", test_signing_key());

        let config = StreamConfig::new("https://receiver.com", crate::DeliveryMethod::poll())
            .with_event_type(crate::CaepEventType::SessionRevoked);

        let stream_id = transmitter.register_stream(config).await.unwrap();
        assert!(!stream_id.is_empty());

        let streams = transmitter.list_streams().await.unwrap();
        assert_eq!(streams.len(), 1);
        assert_eq!(streams[0].id, stream_id);
    }

    #[tokio::test]
    async fn test_remove_stream() {
        let transmitter = DefaultCaepTransmitter::new("https://issuer.com", test_signing_key());

        let config = StreamConfig::new("https://receiver.com", crate::DeliveryMethod::poll());
        let stream_id = transmitter.register_stream(config).await.unwrap();

        transmitter.remove_stream(&stream_id).await.unwrap();

        let streams = transmitter.list_streams().await.unwrap();
        assert!(streams.is_empty());
    }

    #[tokio::test]
    async fn test_remove_nonexistent_stream() {
        let transmitter = DefaultCaepTransmitter::new("https://issuer.com", test_signing_key());

        let result = transmitter.remove_stream("nonexistent").await;
        assert!(result.is_err());
    }

    #[test]
    fn test_emit_result() {
        let result = EmitResult {
            event_id: "test".to_string(),
            streams_notified: 3,
            streams_failed: 0,
            delivery_times_ms: vec![100, 200, 300],
        };

        assert!(result.all_succeeded());
        assert_eq!(result.avg_delivery_time_ms(), 200.0);
    }
}
