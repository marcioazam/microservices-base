//! Metrics for CryptoClient operations.

use once_cell::sync::Lazy;
use prometheus::{
    register_counter_vec, register_gauge, register_histogram_vec, CounterVec, Gauge, HistogramVec,
};
use std::time::Duration;

/// Prometheus metrics for Crypto Service operations.
static CRYPTO_OPERATIONS: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_crypto_operations_total",
        "Total Crypto Service operations",
        &["operation", "status"]
    )
    .expect("Failed to register crypto_operations metric")
});

static CRYPTO_LATENCY: Lazy<HistogramVec> = Lazy::new(|| {
    register_histogram_vec!(
        "token_service_crypto_latency_seconds",
        "Crypto Service operation latency",
        &["operation"],
        vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
    )
    .expect("Failed to register crypto_latency metric")
});

static CRYPTO_FALLBACK: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_crypto_fallback_total",
        "Crypto Service fallback activations",
        &["operation"]
    )
    .expect("Failed to register crypto_fallback metric")
});

static CRYPTO_CACHE: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_crypto_cache_total",
        "Crypto Service cache operations",
        &["cache_type", "result"]
    )
    .expect("Failed to register crypto_cache metric")
});

static CRYPTO_RATE_LIMITED: Lazy<prometheus::Counter> = Lazy::new(|| {
    prometheus::register_counter!(
        "token_service_crypto_rate_limited_total",
        "Total rate limited requests"
    )
    .expect("Failed to register crypto_rate_limited metric")
});

static CRYPTO_CIRCUIT_BREAKER: Lazy<Gauge> = Lazy::new(|| {
    register_gauge!(
        "token_service_crypto_circuit_breaker_open",
        "Circuit breaker state (1 = open, 0 = closed)"
    )
    .expect("Failed to register crypto_circuit_breaker metric")
});

static CRYPTO_SECURITY_EVENTS: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_crypto_security_events_total",
        "Security events from Crypto Service",
        &["event_type"]
    )
    .expect("Failed to register crypto_security_events metric")
});

/// Metrics collector for CryptoClient.
pub struct CryptoMetrics;

impl CryptoMetrics {
    /// Create a new metrics collector.
    #[must_use]
    pub fn new() -> Self {
        // Initialize lazy statics
        Lazy::force(&CRYPTO_OPERATIONS);
        Lazy::force(&CRYPTO_LATENCY);
        Lazy::force(&CRYPTO_FALLBACK);
        Lazy::force(&CRYPTO_CACHE);
        Lazy::force(&CRYPTO_RATE_LIMITED);
        Lazy::force(&CRYPTO_CIRCUIT_BREAKER);
        Lazy::force(&CRYPTO_SECURITY_EVENTS);
        Self
    }

    /// Record an operation.
    pub fn record_operation(&self, operation: &str, success: bool, latency: Duration) {
        let status = if success { "success" } else { "failure" };
        CRYPTO_OPERATIONS
            .with_label_values(&[operation, status])
            .inc();
        CRYPTO_LATENCY
            .with_label_values(&[operation])
            .observe(latency.as_secs_f64());
    }

    /// Record fallback activation.
    pub fn record_fallback_activation(&self, operation: &str) {
        CRYPTO_FALLBACK.with_label_values(&[operation]).inc();
    }

    /// Record cache hit.
    pub fn record_cache_hit(&self, cache_type: &str) {
        CRYPTO_CACHE.with_label_values(&[cache_type, "hit"]).inc();
    }

    /// Record cache miss.
    pub fn record_cache_miss(&self, cache_type: &str) {
        CRYPTO_CACHE.with_label_values(&[cache_type, "miss"]).inc();
    }

    /// Record rate limited request.
    pub fn record_rate_limited(&self) {
        CRYPTO_RATE_LIMITED.inc();
    }

    /// Record circuit breaker open.
    pub fn record_circuit_breaker_open(&self) {
        CRYPTO_CIRCUIT_BREAKER.set(1.0);
    }

    /// Record circuit breaker closed.
    pub fn record_circuit_breaker_closed(&self) {
        CRYPTO_CIRCUIT_BREAKER.set(0.0);
    }

    /// Record security event.
    pub fn record_security_event(&self, event_type: &str) {
        CRYPTO_SECURITY_EVENTS
            .with_label_values(&[event_type])
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

    #[test]
    fn test_metrics_creation() {
        let metrics = CryptoMetrics::new();
        metrics.record_operation("sign", true, Duration::from_millis(10));
        metrics.record_fallback_activation("sign");
        metrics.record_cache_hit("metadata");
        metrics.record_cache_miss("metadata");
        metrics.record_rate_limited();
        metrics.record_circuit_breaker_open();
        metrics.record_security_event("invalid_algorithm");
    }
}
