//! Circuit Breaker and Service Metrics
//!
//! Provides Prometheus metrics for circuit breaker state changes and service health.

use prometheus::{Counter, CounterVec, Gauge, GaugeVec, Histogram, HistogramVec, Opts, Registry};
use std::sync::Arc;

/// Circuit breaker metrics
pub struct CircuitBreakerMetrics {
    /// State changes counter
    pub state_changes: CounterVec,
    /// Current state gauge (0=closed, 1=open, 2=half-open)
    pub current_state: GaugeVec,
    /// Failure count
    pub failures: CounterVec,
    /// Success count
    pub successes: CounterVec,
}

impl CircuitBreakerMetrics {
    /// Creates new circuit breaker metrics
    pub fn new(registry: &Registry) -> Result<Self, prometheus::Error> {
        let state_changes = CounterVec::new(
            Opts::new("circuit_breaker_state_changes_total", "Total circuit breaker state changes")
                .namespace("auth_edge"),
            &["circuit", "from_state", "to_state"],
        )?;
        registry.register(Box::new(state_changes.clone()))?;

        let current_state = GaugeVec::new(
            Opts::new("circuit_breaker_state", "Current circuit breaker state")
                .namespace("auth_edge"),
            &["circuit"],
        )?;
        registry.register(Box::new(current_state.clone()))?;

        let failures = CounterVec::new(
            Opts::new("circuit_breaker_failures_total", "Total circuit breaker failures")
                .namespace("auth_edge"),
            &["circuit"],
        )?;
        registry.register(Box::new(failures.clone()))?;

        let successes = CounterVec::new(
            Opts::new("circuit_breaker_successes_total", "Total circuit breaker successes")
                .namespace("auth_edge"),
            &["circuit"],
        )?;
        registry.register(Box::new(successes.clone()))?;

        Ok(Self {
            state_changes,
            current_state,
            failures,
            successes,
        })
    }

    /// Records a state change
    pub fn record_state_change(&self, circuit: &str, from: &str, to: &str) {
        self.state_changes
            .with_label_values(&[circuit, from, to])
            .inc();
        
        let state_value = match to {
            "closed" => 0.0,
            "open" => 1.0,
            "half_open" => 2.0,
            _ => -1.0,
        };
        self.current_state
            .with_label_values(&[circuit])
            .set(state_value);
    }

    /// Records a failure
    pub fn record_failure(&self, circuit: &str) {
        self.failures.with_label_values(&[circuit]).inc();
    }

    /// Records a success
    pub fn record_success(&self, circuit: &str) {
        self.successes.with_label_values(&[circuit]).inc();
    }
}

/// Service metrics
pub struct ServiceMetrics {
    /// Request latency histogram
    pub request_latency: HistogramVec,
    /// Request count
    pub request_count: CounterVec,
    /// Error count
    pub error_count: CounterVec,
    /// Active requests gauge
    pub active_requests: Gauge,
}

impl ServiceMetrics {
    /// Creates new service metrics
    pub fn new(registry: &Registry) -> Result<Self, prometheus::Error> {
        let request_latency = HistogramVec::new(
            prometheus::HistogramOpts::new(
                "request_latency_seconds",
                "Request latency in seconds",
            )
            .namespace("auth_edge")
            .buckets(vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0]),
            &["method", "status"],
        )?;
        registry.register(Box::new(request_latency.clone()))?;

        let request_count = CounterVec::new(
            Opts::new("requests_total", "Total requests")
                .namespace("auth_edge"),
            &["method", "status"],
        )?;
        registry.register(Box::new(request_count.clone()))?;

        let error_count = CounterVec::new(
            Opts::new("errors_total", "Total errors")
                .namespace("auth_edge"),
            &["error_type"],
        )?;
        registry.register(Box::new(error_count.clone()))?;

        let active_requests = Gauge::new(
            "active_requests",
            "Number of active requests",
        )?;
        registry.register(Box::new(active_requests.clone()))?;

        Ok(Self {
            request_latency,
            request_count,
            error_count,
            active_requests,
        })
    }

    /// Records a request
    pub fn record_request(&self, method: &str, status: &str, latency_secs: f64) {
        self.request_latency
            .with_label_values(&[method, status])
            .observe(latency_secs);
        self.request_count
            .with_label_values(&[method, status])
            .inc();
    }

    /// Records an error
    pub fn record_error(&self, error_type: &str) {
        self.error_count.with_label_values(&[error_type]).inc();
    }

    /// Increments active requests
    pub fn inc_active(&self) {
        self.active_requests.inc();
    }

    /// Decrements active requests
    pub fn dec_active(&self) {
        self.active_requests.dec();
    }
}
