//! Prometheus metrics for Token Service.
//!
//! Provides counters, histograms, and gauges for observability.

use once_cell::sync::Lazy;
use prometheus::{
    register_counter_vec, register_histogram_vec, CounterVec, HistogramVec,
};

/// Tokens issued counter.
pub static TOKENS_ISSUED: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_tokens_issued_total",
        "Total number of tokens issued",
        &["token_type", "algorithm"]
    )
    .expect("Failed to register tokens_issued metric")
});

/// Tokens refreshed counter.
pub static TOKENS_REFRESHED: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_tokens_refreshed_total",
        "Total number of tokens refreshed",
        &["status"]
    )
    .expect("Failed to register tokens_refreshed metric")
});

/// Tokens revoked counter.
pub static TOKENS_REVOKED: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_tokens_revoked_total",
        "Total number of tokens revoked",
        &["reason"]
    )
    .expect("Failed to register tokens_revoked metric")
});

/// DPoP validations counter.
pub static DPOP_VALIDATIONS: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_dpop_validations_total",
        "Total number of DPoP validations",
        &["status", "error_type"]
    )
    .expect("Failed to register dpop_validations metric")
});

/// gRPC method latency histogram.
pub static GRPC_LATENCY: Lazy<HistogramVec> = Lazy::new(|| {
    register_histogram_vec!(
        "token_service_grpc_latency_seconds",
        "gRPC method latency in seconds",
        &["method"],
        vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
    )
    .expect("Failed to register grpc_latency metric")
});

/// KMS operations counter.
pub static KMS_OPERATIONS: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_kms_operations_total",
        "Total number of KMS operations",
        &["operation", "status"]
    )
    .expect("Failed to register kms_operations metric")
});

/// Cache operations counter.
pub static CACHE_OPERATIONS: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_cache_operations_total",
        "Total number of cache operations",
        &["operation", "status"]
    )
    .expect("Failed to register cache_operations metric")
});

/// Security events counter.
pub static SECURITY_EVENTS: Lazy<CounterVec> = Lazy::new(|| {
    register_counter_vec!(
        "token_service_security_events_total",
        "Total number of security events",
        &["event_type"]
    )
    .expect("Failed to register security_events metric")
});

/// Record a token issuance.
pub fn record_token_issued(token_type: &str, algorithm: &str) {
    TOKENS_ISSUED
        .with_label_values(&[token_type, algorithm])
        .inc();
}

/// Record a token refresh.
pub fn record_token_refreshed(status: &str) {
    TOKENS_REFRESHED.with_label_values(&[status]).inc();
}

/// Record a token revocation.
pub fn record_token_revoked(reason: &str) {
    TOKENS_REVOKED.with_label_values(&[reason]).inc();
}

/// Record a DPoP validation.
pub fn record_dpop_validation(status: &str, error_type: &str) {
    DPOP_VALIDATIONS
        .with_label_values(&[status, error_type])
        .inc();
}

/// Record gRPC method latency.
pub fn record_grpc_latency(method: &str, duration_secs: f64) {
    GRPC_LATENCY.with_label_values(&[method]).observe(duration_secs);
}

/// Record a KMS operation.
pub fn record_kms_operation(operation: &str, status: &str) {
    KMS_OPERATIONS
        .with_label_values(&[operation, status])
        .inc();
}

/// Record a cache operation.
pub fn record_cache_operation(operation: &str, status: &str) {
    CACHE_OPERATIONS
        .with_label_values(&[operation, status])
        .inc();
}

/// Record a security event.
pub fn record_security_event(event_type: &str) {
    SECURITY_EVENTS.with_label_values(&[event_type]).inc();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_record_token_issued() {
        record_token_issued("access", "RS256");
        let value = TOKENS_ISSUED
            .with_label_values(&["access", "RS256"])
            .get();
        assert!(value > 0.0);
    }

    #[test]
    fn test_record_grpc_latency() {
        record_grpc_latency("IssueTokenPair", 0.05);
        // Histogram observation doesn't have a simple getter
    }

    #[test]
    fn test_record_security_event() {
        record_security_event("REPLAY_ATTACK");
        let value = SECURITY_EVENTS
            .with_label_values(&["REPLAY_ATTACK"])
            .get();
        assert!(value > 0.0);
    }
}
