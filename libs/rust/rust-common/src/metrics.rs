//! Prometheus metrics helpers.
//!
//! This module provides utilities for exposing metrics in Prometheus format.

use std::sync::atomic::{AtomicU64, Ordering};

/// A simple counter metric.
#[derive(Debug)]
pub struct Counter {
    name: String,
    help: String,
    value: AtomicU64,
}

impl Counter {
    /// Create a new counter.
    #[must_use]
    pub fn new(name: impl Into<String>, help: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            help: help.into(),
            value: AtomicU64::new(0),
        }
    }

    /// Increment the counter by 1.
    pub fn inc(&self) {
        self.value.fetch_add(1, Ordering::Relaxed);
    }

    /// Increment the counter by a specific amount.
    pub fn inc_by(&self, amount: u64) {
        self.value.fetch_add(amount, Ordering::Relaxed);
    }

    /// Get the current value.
    #[must_use]
    pub fn get(&self) -> u64 {
        self.value.load(Ordering::Relaxed)
    }

    /// Get the metric name.
    #[must_use]
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Format as Prometheus text.
    #[must_use]
    pub fn to_prometheus(&self) -> String {
        format!(
            "# HELP {} {}\n# TYPE {} counter\n{} {}\n",
            self.name, self.help, self.name, self.name, self.get()
        )
    }
}

/// A simple gauge metric.
#[derive(Debug)]
pub struct Gauge {
    name: String,
    help: String,
    value: AtomicU64,
}

impl Gauge {
    /// Create a new gauge.
    #[must_use]
    pub fn new(name: impl Into<String>, help: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            help: help.into(),
            value: AtomicU64::new(0),
        }
    }

    /// Set the gauge value.
    pub fn set(&self, value: u64) {
        self.value.store(value, Ordering::Relaxed);
    }

    /// Increment the gauge by 1.
    pub fn inc(&self) {
        self.value.fetch_add(1, Ordering::Relaxed);
    }

    /// Decrement the gauge by 1.
    pub fn dec(&self) {
        self.value.fetch_sub(1, Ordering::Relaxed);
    }

    /// Get the current value.
    #[must_use]
    pub fn get(&self) -> u64 {
        self.value.load(Ordering::Relaxed)
    }

    /// Get the metric name.
    #[must_use]
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Format as Prometheus text.
    #[must_use]
    pub fn to_prometheus(&self) -> String {
        format!(
            "# HELP {} {}\n# TYPE {} gauge\n{} {}\n",
            self.name, self.help, self.name, self.name, self.get()
        )
    }
}

/// Cache metrics.
#[derive(Debug)]
pub struct CacheMetrics {
    /// Cache hits
    pub hits: Counter,
    /// Cache misses
    pub misses: Counter,
    /// Current cache size
    pub size: Gauge,
}

impl CacheMetrics {
    /// Create new cache metrics with the given prefix.
    #[must_use]
    pub fn new(prefix: &str) -> Self {
        Self {
            hits: Counter::new(
                format!("{}_cache_hits_total", prefix),
                "Total number of cache hits",
            ),
            misses: Counter::new(
                format!("{}_cache_misses_total", prefix),
                "Total number of cache misses",
            ),
            size: Gauge::new(
                format!("{}_cache_size", prefix),
                "Current number of items in cache",
            ),
        }
    }

    /// Record a cache hit.
    pub fn record_hit(&self) {
        self.hits.inc();
    }

    /// Record a cache miss.
    pub fn record_miss(&self) {
        self.misses.inc();
    }

    /// Update cache size.
    pub fn update_size(&self, size: u64) {
        self.size.set(size);
    }

    /// Format all metrics as Prometheus text.
    #[must_use]
    pub fn to_prometheus(&self) -> String {
        format!(
            "{}{}{}",
            self.hits.to_prometheus(),
            self.misses.to_prometheus(),
            self.size.to_prometheus()
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_counter() {
        let counter = Counter::new("test_counter", "A test counter");
        assert_eq!(counter.get(), 0);

        counter.inc();
        assert_eq!(counter.get(), 1);

        counter.inc_by(5);
        assert_eq!(counter.get(), 6);
    }

    #[test]
    fn test_gauge() {
        let gauge = Gauge::new("test_gauge", "A test gauge");
        assert_eq!(gauge.get(), 0);

        gauge.set(10);
        assert_eq!(gauge.get(), 10);

        gauge.inc();
        assert_eq!(gauge.get(), 11);

        gauge.dec();
        assert_eq!(gauge.get(), 10);
    }

    #[test]
    fn test_cache_metrics() {
        let metrics = CacheMetrics::new("vault");

        metrics.record_hit();
        metrics.record_hit();
        metrics.record_miss();
        metrics.update_size(100);

        assert_eq!(metrics.hits.get(), 2);
        assert_eq!(metrics.misses.get(), 1);
        assert_eq!(metrics.size.get(), 100);
    }

    #[test]
    fn test_prometheus_format() {
        let counter = Counter::new("requests_total", "Total requests");
        counter.inc_by(42);

        let output = counter.to_prometheus();
        assert!(output.contains("# HELP requests_total Total requests"));
        assert!(output.contains("# TYPE requests_total counter"));
        assert!(output.contains("requests_total 42"));
    }
}
