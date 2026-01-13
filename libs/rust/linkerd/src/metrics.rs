//! Linkerd metrics types.

use serde::{Deserialize, Serialize};

/// Linkerd proxy metrics.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LinkerdMetrics {
    /// Total requests
    pub request_total: u64,
    /// Successful requests
    pub success_total: u64,
    /// Failed requests
    pub failure_total: u64,
    /// p50 latency in milliseconds
    pub latency_p50_ms: f64,
    /// p95 latency in milliseconds
    pub latency_p95_ms: f64,
    /// p99 latency in milliseconds
    pub latency_p99_ms: f64,
}

impl LinkerdMetrics {
    /// Calculate success rate (0.0 to 1.0).
    #[must_use]
    pub fn success_rate(&self) -> f64 {
        if self.request_total == 0 {
            1.0
        } else {
            self.success_total as f64 / self.request_total as f64
        }
    }

    /// Calculate error rate (0.0 to 1.0).
    #[must_use]
    pub fn error_rate(&self) -> f64 {
        1.0 - self.success_rate()
    }

    /// Check if error rate exceeds threshold.
    #[must_use]
    pub fn should_alert(&self, threshold: f64) -> bool {
        self.error_rate() > threshold
    }

    /// Check if latency overhead is within bounds.
    /// Linkerd overhead should be: p50 <= 1ms, p95 <= 1.5ms, p99 <= 2ms
    #[must_use]
    pub fn latency_within_bounds(&self) -> bool {
        self.latency_p50_ms <= 1.0
            && self.latency_p95_ms <= 1.5
            && self.latency_p99_ms <= 2.0
    }
}

impl Default for LinkerdMetrics {
    fn default() -> Self {
        Self {
            request_total: 0,
            success_total: 0,
            failure_total: 0,
            latency_p50_ms: 0.0,
            latency_p95_ms: 0.0,
            latency_p99_ms: 0.0,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_success_rate() {
        let metrics = LinkerdMetrics {
            request_total: 1000,
            success_total: 990,
            failure_total: 10,
            ..Default::default()
        };

        assert!((metrics.success_rate() - 0.99).abs() < 0.001);
        assert!((metrics.error_rate() - 0.01).abs() < 0.001);
    }

    #[test]
    fn test_empty_metrics() {
        let metrics = LinkerdMetrics::default();
        assert!((metrics.success_rate() - 1.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_alert_threshold() {
        let metrics = LinkerdMetrics {
            request_total: 100,
            success_total: 98,
            failure_total: 2,
            ..Default::default()
        };

        assert!(metrics.should_alert(0.01)); // 2% > 1%
        assert!(!metrics.should_alert(0.05)); // 2% < 5%
    }

    #[test]
    fn test_latency_bounds() {
        let good = LinkerdMetrics {
            latency_p50_ms: 0.5,
            latency_p95_ms: 1.0,
            latency_p99_ms: 1.5,
            ..Default::default()
        };
        assert!(good.latency_within_bounds());

        let bad = LinkerdMetrics {
            latency_p50_ms: 2.0,
            latency_p95_ms: 3.0,
            latency_p99_ms: 5.0,
            ..Default::default()
        };
        assert!(!bad.latency_within_bounds());
    }
}
