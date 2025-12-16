//! OpenTelemetry Observability Module
//!
//! Provides tracing, metrics, and structured logging with OpenTelemetry integration.

pub mod telemetry;
pub mod metrics;

pub use telemetry::{init_telemetry, TelemetryConfig};
pub use metrics::CircuitBreakerMetrics;
