//! CAEP Stream configuration and management.

use crate::CaepEventType;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Delivery method for CAEP events
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "method", rename_all = "snake_case")]
pub enum DeliveryMethod {
    /// Push delivery via webhook
    Push { endpoint_url: String },
    /// Poll delivery (receiver polls for events)
    Poll,
}

/// Stream configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
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

/// Stream status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum StreamStatus {
    Active,
    Paused,
    Failed,
    Disabled,
}

/// Stream health metrics
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct StreamHealth {
    pub events_delivered: u64,
    pub events_failed: u64,
    pub last_delivery_at: Option<DateTime<Utc>>,
    pub last_error: Option<String>,
    pub avg_latency_ms: f64,
    pub p99_latency_ms: f64,
}

/// CAEP Stream
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Stream {
    pub id: String,
    pub config: StreamConfig,
    pub status: StreamStatus,
    pub health: StreamHealth,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl Stream {
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

    pub fn record_success(&mut self, latency_ms: u64) {
        self.health.events_delivered += 1;
        self.health.last_delivery_at = Some(Utc::now());
        self.health.last_error = None;
        self.updated_at = Utc::now();
        
        // Update average latency (simple moving average)
        let n = self.health.events_delivered as f64;
        self.health.avg_latency_ms = 
            (self.health.avg_latency_ms * (n - 1.0) + latency_ms as f64) / n;
    }

    pub fn record_failure(&mut self, error: String) {
        self.health.events_failed += 1;
        self.health.last_error = Some(error);
        self.updated_at = Utc::now();

        // Auto-disable after too many failures
        if self.health.events_failed > 10 && self.consecutive_failures() > 5 {
            self.status = StreamStatus::Failed;
        }
    }

    fn consecutive_failures(&self) -> u64 {
        // Simplified: if last_delivery is None or old, assume consecutive
        match self.health.last_delivery_at {
            Some(last) if Utc::now().signed_duration_since(last).num_minutes() < 5 => 0,
            _ => self.health.events_failed.min(10),
        }
    }

    pub fn success_rate(&self) -> f64 {
        let total = self.health.events_delivered + self.health.events_failed;
        if total == 0 {
            1.0
        } else {
            self.health.events_delivered as f64 / total as f64
        }
    }
}
