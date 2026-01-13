//! OpenTelemetry Observability Module
//!
//! Provides tracing, metrics, and structured logging with OpenTelemetry integration.

#[cfg(feature = "otel")]
pub mod telemetry;
pub mod metrics;
pub mod logging;

#[cfg(feature = "otel")]
pub use telemetry::{init_telemetry, TelemetryConfig, shutdown_telemetry};
pub use metrics::CircuitBreakerMetrics;
pub use logging::AuthEdgeLogger;
