//! CAEP Stream configuration and management.
//!
//! This module provides stream management for CAEP event delivery.

use crate::CaepEventType;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Delivery method for CAEP events.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(tag = "method", rename_all = "snake_case")]
pub enum DeliveryMethod {
    /// Push delivery via webhook
    Push {
        /// Endpoint URL for webhook delivery
        endpoint_url: String,
    },
    /// Poll delivery (receiver polls for events)
    Poll,
}

impl DeliveryMethod {
    /// Create a push delivery method.
    #[must_use]
    pub fn push(endpoint_url: impl Into<String>) -> Self {
        Self::Push {
            endpoint_url: endpoint_url.into(),
        }
    }

    /// Create a poll delivery method.
    #[must_use]
    pub const fn poll() -> Self {
        Self::Poll
    }

    /// Check if this is a push delivery method.
    #[must_use]
    pub const fn is_push(&self) -> bool {
        matches!(self, Self::Push { .. })
    }

    /// Get the endpoint URL if this is a push delivery.
    #[must_use]
    pub fn endpoint_url(&self) -> Option<&str> {
        match self {
            Self::Push { endpoint_url } => Some(endpoint_url),
            Self::Poll => None,
        }
    }
}

/// Stream configuration.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct StreamConfig {
    /// Audience for the stream (receiver identifier)
    pub audience: String,
    /// Delivery method
    pub delivery: DeliveryMethod,
    /// Event types to receive
    pub events_requested: Vec<CaepEventType>,
    /// Subject format preference
    #[serde(default = "default_format")]
    pub format: String,
}

fn default_format() -> String {
    "iss_sub".to_string()
}

impl StreamConfig {
    /// Create a new stream configuration.
    #[must_use]
    pub fn new(audience: impl Into<String>, delivery: DeliveryMethod) -> Self {
        Self {
            audience: audience.into(),
            delivery,
            events_requested: Vec::new(),
            format: default_format(),
        }
    }

    /// Add an event type to request.
    #[must_use]
    pub fn with_event_type(mut self, event_type: CaepEventType) -> Self {
        self.events_requested.push(event_type);
        self
    }

    /// Set the subject format.
    #[must_use]
    pub fn with_format(mut self, format: impl Into<String>) -> Self {
        self.format = format.into();
        self
    }

    /// Request all event types.
    #[must_use]
    pub fn with_all_events(mut self) -> Self {
        self.events_requested = CaepEventType::all().to_vec();
        self
    }
}

/// Stream status.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum StreamStatus {
    /// Stream is active and delivering events
    Active,
    /// Stream is paused
    Paused,
    /// Stream has failed due to delivery errors
    Failed,
    /// Stream is disabled
    Disabled,
}

impl StreamStatus {
    /// Check if the stream is operational.
    #[must_use]
    pub const fn is_operational(&self) -> bool {
        matches!(self, Self::Active)
    }
}

/// Stream health metrics.
#[derive(Debug, Clone, Serialize, Deserialize, Default, PartialEq)]
pub struct StreamHealth {
    /// Total events delivered successfully
    pub events_delivered: u64,
    /// Total events that failed delivery
    pub events_failed: u64,
    /// Timestamp of last successful delivery
    pub last_delivery_at: Option<DateTime<Utc>>,
    /// Last error message
    pub last_error: Option<String>,
    /// Average latency in milliseconds
    pub avg_latency_ms: f64,
    /// P99 latency in milliseconds
    pub p99_latency_ms: f64,
}

impl StreamHealth {
    /// Calculate the success rate.
    #[must_use]
    pub fn success_rate(&self) -> f64 {
        let total = self.events_delivered + self.events_failed;
        if total == 0 {
            1.0
        } else {
            self.events_delivered as f64 / total as f64
        }
    }

    /// Check if the stream is healthy (>95% success rate).
    #[must_use]
    pub fn is_healthy(&self) -> bool {
        self.success_rate() > 0.95
    }
}

/// CAEP Stream.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Stream {
    /// Unique stream identifier
    pub id: String,
    /// Stream configuration
    pub config: StreamConfig,
    /// Current status
    pub status: StreamStatus,
    /// Health metrics
    pub health: StreamHealth,
    /// Creation timestamp
    pub created_at: DateTime<Utc>,
    /// Last update timestamp
    pub updated_at: DateTime<Utc>,
}

impl Stream {
    /// Create a new stream.
    #[must_use]
    pub fn new(config: StreamConfig) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4().to_string(),
            config,
            status: StreamStatus::Active,
            health: StreamHealth::default(),
            created_at: now,
            updated_at: now,
        }
    }

    /// Record a successful delivery.
    pub fn record_success(&mut self, latency_ms: u64) {
        self.health.events_delivered += 1;
        self.health.last_delivery_at = Some(Utc::now());
        self.health.last_error = None;
        self.updated_at = Utc::now();

        // Update average latency (simple moving average)
        let n = self.health.events_delivered as f64;
        self.health.avg_latency_ms =
            (self.health.avg_latency_ms * (n - 1.0) + latency_ms as f64) / n;

        // Update p99 (simplified: track max for now)
        if latency_ms as f64 > self.health.p99_latency_ms {
            self.health.p99_latency_ms = latency_ms as f64;
        }
    }

    /// Record a failed delivery.
    pub fn record_failure(&mut self, error: impl Into<String>) {
        self.health.events_failed += 1;
        self.health.last_error = Some(error.into());
        self.updated_at = Utc::now();

        // Auto-disable after too many consecutive failures
        if self.consecutive_failures() > 5 {
            self.status = StreamStatus::Failed;
        }
    }

    /// Estimate consecutive failures.
    fn consecutive_failures(&self) -> u64 {
        match self.health.last_delivery_at {
            Some(last) if Utc::now().signed_duration_since(last).num_minutes() < 5 => 0,
            _ => self.health.events_failed.min(10),
        }
    }

    /// Get the success rate.
    #[must_use]
    pub fn success_rate(&self) -> f64 {
        self.health.success_rate()
    }

    /// Pause the stream.
    pub fn pause(&mut self) {
        self.status = StreamStatus::Paused;
        self.updated_at = Utc::now();
    }

    /// Resume the stream.
    pub fn resume(&mut self) {
        self.status = StreamStatus::Active;
        self.updated_at = Utc::now();
    }

    /// Disable the stream.
    pub fn disable(&mut self) {
        self.status = StreamStatus::Disabled;
        self.updated_at = Utc::now();
    }

    /// Check if the stream is operational.
    #[must_use]
    pub const fn is_operational(&self) -> bool {
        self.status.is_operational()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_delivery_method() {
        let push = DeliveryMethod::push("https://example.com/webhook");
        assert!(push.is_push());
        assert_eq!(push.endpoint_url(), Some("https://example.com/webhook"));

        let poll = DeliveryMethod::poll();
        assert!(!poll.is_push());
        assert_eq!(poll.endpoint_url(), None);
    }

    #[test]
    fn test_stream_config_builder() {
        let config = StreamConfig::new("https://receiver.com", DeliveryMethod::poll())
            .with_event_type(CaepEventType::SessionRevoked)
            .with_event_type(CaepEventType::CredentialChange)
            .with_format("email");

        assert_eq!(config.events_requested.len(), 2);
        assert_eq!(config.format, "email");
    }

    #[test]
    fn test_stream_creation() {
        let config = StreamConfig::new("https://receiver.com", DeliveryMethod::poll());
        let stream = Stream::new(config);

        assert!(!stream.id.is_empty());
        assert_eq!(stream.status, StreamStatus::Active);
        assert!(stream.is_operational());
    }

    #[test]
    fn test_stream_health_tracking() {
        let config = StreamConfig::new("https://receiver.com", DeliveryMethod::poll());
        let mut stream = Stream::new(config);

        stream.record_success(100);
        stream.record_success(200);

        assert_eq!(stream.health.events_delivered, 2);
        assert_eq!(stream.health.avg_latency_ms, 150.0);
        assert_eq!(stream.success_rate(), 1.0);
    }

    #[test]
    fn test_stream_failure_tracking() {
        let config = StreamConfig::new("https://receiver.com", DeliveryMethod::poll());
        let mut stream = Stream::new(config);

        stream.record_success(100);
        stream.record_failure("Connection refused");

        assert_eq!(stream.health.events_delivered, 1);
        assert_eq!(stream.health.events_failed, 1);
        assert_eq!(stream.success_rate(), 0.5);
        assert!(stream.health.last_error.is_some());
    }

    #[test]
    fn test_stream_status_changes() {
        let config = StreamConfig::new("https://receiver.com", DeliveryMethod::poll());
        let mut stream = Stream::new(config);

        assert!(stream.is_operational());

        stream.pause();
        assert!(!stream.is_operational());
        assert_eq!(stream.status, StreamStatus::Paused);

        stream.resume();
        assert!(stream.is_operational());

        stream.disable();
        assert!(!stream.is_operational());
        assert_eq!(stream.status, StreamStatus::Disabled);
    }
}
