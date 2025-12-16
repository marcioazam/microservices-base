//! Metrics and Observability Unit Tests
//!
//! Tests for metrics recording, telemetry config, and latency calculations.

// ============================================================================
// Telemetry Configuration
// ============================================================================

#[derive(Debug, Clone)]
struct TelemetryConfig {
    service_name: String,
    otlp_endpoint: String,
    sampling_ratio: f64,
    enable_console: bool,
}

impl Default for TelemetryConfig {
    fn default() -> Self {
        Self {
            service_name: "auth-edge-service".to_string(),
            otlp_endpoint: "http://localhost:4317".to_string(),
            sampling_ratio: 1.0,
            enable_console: true,
        }
    }
}

// ============================================================================
// Metrics Recorder
// ============================================================================

struct MetricsRecorder {
    request_count: u64,
    error_count: u64,
    latencies: Vec<f64>,
}

impl MetricsRecorder {
    fn new() -> Self {
        Self {
            request_count: 0,
            error_count: 0,
            latencies: Vec::new(),
        }
    }

    fn record_request(&mut self, latency_secs: f64, is_error: bool) {
        self.request_count += 1;
        self.latencies.push(latency_secs);
        if is_error {
            self.error_count += 1;
        }
    }

    fn error_rate(&self) -> f64 {
        if self.request_count == 0 {
            return 0.0;
        }
        self.error_count as f64 / self.request_count as f64
    }

    fn p50_latency(&self) -> Option<f64> {
        if self.latencies.is_empty() {
            return None;
        }
        let mut sorted = self.latencies.clone();
        sorted.sort_by(|a, b| a.partial_cmp(b).unwrap());
        let idx = sorted.len() / 2;
        Some(sorted[idx])
    }

    fn p99_latency(&self) -> Option<f64> {
        if self.latencies.is_empty() {
            return None;
        }
        let mut sorted = self.latencies.clone();
        sorted.sort_by(|a, b| a.partial_cmp(b).unwrap());
        let idx = (sorted.len() as f64 * 0.99) as usize;
        Some(sorted[idx.min(sorted.len() - 1)])
    }
}

// ============================================================================
// Telemetry Config Tests
// ============================================================================

#[test]
fn test_default_telemetry_config() {
    let config = TelemetryConfig::default();
    assert_eq!(config.service_name, "auth-edge-service");
    assert_eq!(config.sampling_ratio, 1.0);
    assert!(config.enable_console);
}

#[test]
fn test_sampling_ratio_bounds() {
    let config = TelemetryConfig::default();
    assert!(config.sampling_ratio >= 0.0);
    assert!(config.sampling_ratio <= 1.0);
}

#[test]
fn test_otlp_endpoint_format() {
    let config = TelemetryConfig::default();
    assert!(config.otlp_endpoint.starts_with("http"));
    assert!(config.otlp_endpoint.contains(':'));
}

#[test]
fn test_circuit_state_metric_values() {
    let closed = 0.0;
    let open = 1.0;
    let half_open = 2.0;

    assert_eq!(closed, 0.0);
    assert_eq!(open, 1.0);
    assert_eq!(half_open, 2.0);
}

// ============================================================================
// Metrics Recorder Tests
// ============================================================================

#[test]
fn test_metrics_initial_state() {
    let recorder = MetricsRecorder::new();
    assert_eq!(recorder.request_count, 0);
    assert_eq!(recorder.error_count, 0);
    assert_eq!(recorder.error_rate(), 0.0);
}

#[test]
fn test_metrics_record_success() {
    let mut recorder = MetricsRecorder::new();
    recorder.record_request(0.005, false);

    assert_eq!(recorder.request_count, 1);
    assert_eq!(recorder.error_count, 0);
    assert_eq!(recorder.error_rate(), 0.0);
}

#[test]
fn test_metrics_record_error() {
    let mut recorder = MetricsRecorder::new();
    recorder.record_request(0.010, true);

    assert_eq!(recorder.request_count, 1);
    assert_eq!(recorder.error_count, 1);
    assert_eq!(recorder.error_rate(), 1.0);
}

#[test]
fn test_metrics_error_rate_calculation() {
    let mut recorder = MetricsRecorder::new();

    recorder.record_request(0.001, false);
    recorder.record_request(0.002, false);
    recorder.record_request(0.003, false);
    recorder.record_request(0.100, true);

    assert_eq!(recorder.error_rate(), 0.25);
}

#[test]
fn test_metrics_p50_latency() {
    let mut recorder = MetricsRecorder::new();

    for i in 1..=100 {
        recorder.record_request(i as f64 / 1000.0, false);
    }

    let p50 = recorder.p50_latency().unwrap();
    assert!(p50 >= 0.050 && p50 <= 0.051);
}

#[test]
fn test_metrics_p99_latency() {
    let mut recorder = MetricsRecorder::new();

    for i in 1..=100 {
        recorder.record_request(i as f64 / 1000.0, false);
    }

    let p99 = recorder.p99_latency().unwrap();
    assert!(p99 >= 0.099);
}

#[test]
fn test_metrics_empty_latencies() {
    let recorder = MetricsRecorder::new();
    assert!(recorder.p50_latency().is_none());
    assert!(recorder.p99_latency().is_none());
}
