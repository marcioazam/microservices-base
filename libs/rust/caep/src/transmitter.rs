//! CAEP Transmitter implementation.

use crate::{CaepError, CaepEvent, SecurityEventToken, Stream, StreamStatus};
use async_trait::async_trait;
use jsonwebtoken::{Algorithm, EncodingKey};
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{error, info, instrument};

/// CAEP Transmitter trait for emitting security events
#[async_trait]
pub trait CaepTransmitter: Send + Sync {
    /// Emit a security event to all registered streams
    async fn emit(&self, event: CaepEvent) -> Result<EmitResult, CaepError>;

    /// Register a new stream receiver
    async fn register_stream(&self, config: crate::StreamConfig) -> Result<String, CaepError>;

    /// Remove a stream
    async fn remove_stream(&self, stream_id: &str) -> Result<(), CaepError>;

    /// Get stream status
    async fn stream_status(&self, stream_id: &str) -> Result<StreamStatus, CaepError>;

    /// List all streams
    async fn list_streams(&self) -> Result<Vec<Stream>, CaepError>;
}

/// Result of emitting an event
#[derive(Debug)]
pub struct EmitResult {
    pub event_id: String,
    pub streams_notified: usize,
    pub streams_failed: usize,
    pub delivery_times_ms: Vec<u64>,
}

/// Default CAEP Transmitter implementation
pub struct DefaultCaepTransmitter {
    issuer: String,
    signing_key: EncodingKey,
    algorithm: Algorithm,
    streams: Arc<RwLock<Vec<Stream>>>,
    http_client: reqwest::Client,
}

impl DefaultCaepTransmitter {
    pub fn new(issuer: String, signing_key: EncodingKey) -> Self {
        Self {
            issuer,
            signing_key,
            algorithm: Algorithm::ES256,
            streams: Arc::new(RwLock::new(Vec::new())),
            http_client: reqwest::Client::new(),
        }
    }

    pub fn with_algorithm(mut self, algorithm: Algorithm) -> Self {
        self.algorithm = algorithm;
        self
    }

    #[instrument(skip(self, set))]
    async fn deliver_to_stream(&self, stream: &Stream, set: &str) -> Result<u64, CaepError> {
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
                    .map_err(|e| CaepError::DeliveryFailed(e.to_string()))?;

                if !response.status().is_success() {
                    return Err(CaepError::DeliveryFailed(format!(
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
}

#[async_trait]
impl CaepTransmitter for DefaultCaepTransmitter {
    #[instrument(skip(self))]
    async fn emit(&self, event: CaepEvent) -> Result<EmitResult, CaepError> {
        let streams = self.streams.read().await;
        let active_streams: Vec<_> = streams
            .iter()
            .filter(|s| s.status == StreamStatus::Active)
            .filter(|s| s.config.events_requested.contains(&event.event_type))
            .collect();

        if active_streams.is_empty() {
            info!("No active streams for event type {:?}", event.event_type);
            return Ok(EmitResult {
                event_id: uuid::Uuid::new_v4().to_string(),
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
            let signed_set = set.sign(&self.signing_key, self.algorithm)?;

            match self.deliver_to_stream(stream, &signed_set).await {
                Ok(time_ms) => {
                    streams_notified += 1;
                    delivery_times.push(time_ms);
                    info!(
                        stream_id = %stream.id,
                        delivery_time_ms = time_ms,
                        "Event delivered successfully"
                    );
                }
                Err(e) => {
                    streams_failed += 1;
                    error!(
                        stream_id = %stream.id,
                        error = %e,
                        "Failed to deliver event"
                    );
                }
            }
        }

        Ok(EmitResult {
            event_id: uuid::Uuid::new_v4().to_string(),
            streams_notified,
            streams_failed,
            delivery_times_ms: delivery_times,
        })
    }

    async fn register_stream(&self, config: crate::StreamConfig) -> Result<String, CaepError> {
        let stream = Stream::new(config);
        let id = stream.id.clone();

        let mut streams = self.streams.write().await;
        streams.push(stream);

        info!(stream_id = %id, "Stream registered");
        Ok(id)
    }

    async fn remove_stream(&self, stream_id: &str) -> Result<(), CaepError> {
        let mut streams = self.streams.write().await;
        let initial_len = streams.len();
        streams.retain(|s| s.id != stream_id);

        if streams.len() == initial_len {
            return Err(CaepError::StreamNotFound(stream_id.to_string()));
        }

        info!(stream_id = %stream_id, "Stream removed");
        Ok(())
    }

    async fn stream_status(&self, stream_id: &str) -> Result<StreamStatus, CaepError> {
        let streams = self.streams.read().await;
        streams
            .iter()
            .find(|s| s.id == stream_id)
            .map(|s| s.status.clone())
            .ok_or_else(|| CaepError::StreamNotFound(stream_id.to_string()))
    }

    async fn list_streams(&self) -> Result<Vec<Stream>, CaepError> {
        let streams = self.streams.read().await;
        Ok(streams.clone())
    }
}
