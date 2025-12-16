//! OpenTelemetry Telemetry Configuration
//!
//! Sets up OTLP exporter and tracing subscriber with W3C trace context propagation.

use opentelemetry::trace::TracerProvider;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{
    runtime,
    trace::{Config, Sampler},
    Resource,
};
use opentelemetry::KeyValue;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

/// Telemetry configuration
#[derive(Debug, Clone)]
pub struct TelemetryConfig {
    /// Service name for traces
    pub service_name: String,
    /// OTLP endpoint URL
    pub otlp_endpoint: String,
    /// Sampling ratio (0.0 to 1.0)
    pub sampling_ratio: f64,
    /// Enable console output
    pub enable_console: bool,
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

/// Initializes OpenTelemetry tracing with OTLP exporter
pub fn init_telemetry(config: &TelemetryConfig) -> Result<(), Box<dyn std::error::Error>> {
    // Create OTLP exporter
    let exporter = opentelemetry_otlp::new_exporter()
        .tonic()
        .with_endpoint(&config.otlp_endpoint);

    // Create tracer provider with sampling
    let sampler = if config.sampling_ratio >= 1.0 {
        Sampler::AlwaysOn
    } else if config.sampling_ratio <= 0.0 {
        Sampler::AlwaysOff
    } else {
        Sampler::TraceIdRatioBased(config.sampling_ratio)
    };

    let tracer_provider = opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(exporter)
        .with_trace_config(
            Config::default()
                .with_sampler(sampler)
                .with_resource(Resource::new(vec![
                    KeyValue::new("service.name", config.service_name.clone()),
                    KeyValue::new("service.version", env!("CARGO_PKG_VERSION")),
                ])),
        )
        .install_batch(runtime::Tokio)?;

    // Create OpenTelemetry tracing layer
    let tracer = tracer_provider.tracer("auth-edge-service");
    let otel_layer = tracing_opentelemetry::layer().with_tracer(tracer);

    // Create subscriber with layers
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    let subscriber = tracing_subscriber::registry()
        .with(env_filter)
        .with(otel_layer);

    if config.enable_console {
        let fmt_layer = tracing_subscriber::fmt::layer()
            .json()
            .with_target(true)
            .with_thread_ids(true)
            .with_file(true)
            .with_line_number(true);
        
        subscriber.with(fmt_layer).init();
    } else {
        subscriber.init();
    }

    Ok(())
}

/// Shuts down OpenTelemetry gracefully
pub fn shutdown_telemetry() {
    opentelemetry::global::shutdown_tracer_provider();
}

/// Records an error event with structured attributes
pub fn record_error_event(
    correlation_id: &str,
    error_type: &str,
    error_message: &str,
) {
    tracing::error!(
        correlation_id = %correlation_id,
        error_type = %error_type,
        error_message = %error_message,
        timestamp = %chrono::Utc::now().to_rfc3339(),
        "Error event recorded"
    );
}
