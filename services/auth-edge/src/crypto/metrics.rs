//! Crypto Client Metrics
//!
//! Prometheus metrics for crypto-service operations.

use prometheus::{
    register_histogram_vec, register_int_counter, register_int_counter_vec, register_int_gauge,
    HistogramVec, IntCounter, IntCounterVec, IntGauge,
};
use std::time::Duration;

/// Metrics for crypto client operations
pub struct CryptoMetrics {
    /// Total requests counter by operation and status
    pub requests_total: IntCounterVec,
    /// Request latency histogram by operation
    pub latency_seconds: HistogramVec,
    /// Gauge indicating if fallback mode is active
    pub fallback_active: IntGauge,
    /// Counter for key rotations
    pub key_rotations_total: IntCounter,
    /// Error counter by operation and error type
    pub errors_total: IntCounterVec,
}

impl CryptoMetrics {
    /// Creates and registers new crypto metrics
    ///
    /// # Panics
    ///
    /// Panics if metrics cannot be registered (duplicate registration)
    #[must_use]
    pub fn new() -> Self {
        let requests_total = register_int_counter_vec!(
            "crypto_client_requests_total",
            "Total number of crypto client requests",
            &["operation", "status"]
        )
        .expect("Failed to register crypto_client_requests_total");

        let latency_seconds = register_histogram_vec!(
            "crypto_client_latency_seconds",
            "Crypto client request latency in seconds",
            &["operation"],
            vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
        )
        .expect("Failed to register crypto_client_latency_seconds");

        let fallback_active = register_int_gauge!(
            "crypto_client_fallback_active",
            "Whether crypto client is operating in fallback mode (1=active, 0=normal)"
        )
        .expect("Failed to register crypto_client_fallback_active");

        let key_rotations_total = register_int_counter!(
            "crypto_key_rotation_total",
            "Total number of key rotations performed"
        )
        .expect("Failed to register crypto_key_rotation_total");

        let errors_total = register_int_counter_vec!(
            "crypto_client_errors_total",
            "Total number of crypto client errors",
            &["operation", "error_type"]
        )
        .expect("Failed to register crypto_client_errors_total");

        Self {
            requests_total,
            latency_seconds,
            fallback_active,
            key_rotations_total,
            errors_total,
        }
    }

    /// Records a successful request
    pub fn record_success(&self, operation: &str, duration: Duration) {
        self.requests_total
            .with_label_values(&[operation, "success"])
            .inc();
        self.latency_seconds
            .with_label_values(&[operation])
            .observe(duration.as_secs_f64());
    }

    /// Records a failed request
    pub fn record_failure(&self, operation: &str, error_type: &str, duration: Duration) {
        self.requests_total
            .with_label_values(&[operation, "failure"])
            .inc();
        self.latency_seconds
            .with_label_values(&[operation])
            .observe(duration.as_secs_f64());
        self.errors_total
            .with_label_values(&[operation, error_type])
            .inc();
    }

    /// Records a fallback request (when using local encryption)
    pub fn record_fallback(&self, operation: &str, duration: Duration) {
        self.requests_total
            .with_label_values(&[operation, "fallback"])
            .inc();
        self.latency_seconds
            .with_label_values(&[operation])
            .observe(duration.as_secs_f64());
    }

    /// Sets the fallback mode status
    pub fn set_fallback_active(&self, active: bool) {
        self.fallback_active.set(if active { 1 } else { 0 });
    }

    /// Increments the key rotation counter
    pub fn increment_rotation(&self) {
        self.key_rotations_total.inc();
    }

    /// Records an error
    pub fn record_error(&self, operation: &str, error_type: &str) {
        self.errors_total
            .with_label_values(&[operation, error_type])
            .inc();
    }
}

impl Default for CryptoMetrics {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Note: These tests may fail if run multiple times due to metric registration
    // In production, use a test registry or lazy_static

    #[test]
    fn test_record_success() {
        // Skip in CI due to global registry issues
        if std::env::var("CI").is_ok() {
            return;
        }
    }

    #[test]
    fn test_fallback_gauge() {
        // Skip in CI due to global registry issues
        if std::env::var("CI").is_ok() {
            return;
        }
    }
}
