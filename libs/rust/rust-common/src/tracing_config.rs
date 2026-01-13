//! OpenTelemetry tracing integration.
//!
//! This module provides configuration for distributed tracing using OpenTelemetry.

use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

/// Tracing configuration.
#[derive(Debug, Clone)]
pub struct TracingConfig {
    /// Service name for tracing
    pub service_name: String,
    /// Log level filter
    pub log_level: String,
    /// Whether to output JSON format
    pub json_output: bool,
}

impl Default for TracingConfig {
    fn default() -> Self {
        Self {
            service_name: "rust-service".to_string(),
            log_level: "info".to_string(),
            json_output: false,
        }
    }
}

impl TracingConfig {
    /// Create config with custom service name.
    #[must_use]
    pub fn with_service_name(mut self, name: impl Into<String>) -> Self {
        self.service_name = name.into();
        self
    }

    /// Create config with custom log level.
    #[must_use]
    pub fn with_log_level(mut self, level: impl Into<String>) -> Self {
        self.log_level = level.into();
        self
    }

    /// Enable JSON output.
    #[must_use]
    pub const fn with_json_output(mut self) -> Self {
        self.json_output = true;
        self
    }
}

/// Initialize tracing with the given configuration.
///
/// This sets up the global tracing subscriber with the specified configuration.
/// Should be called once at application startup.
pub fn init_tracing(config: &TracingConfig) {
    let filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&config.log_level));

    if config.json_output {
        tracing_subscriber::registry()
            .with(filter)
            .with(tracing_subscriber::fmt::layer().json())
            .init();
    } else {
        tracing_subscriber::registry()
            .with(filter)
            .with(tracing_subscriber::fmt::layer())
            .init();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = TracingConfig::default();
        assert_eq!(config.service_name, "rust-service");
        assert_eq!(config.log_level, "info");
        assert!(!config.json_output);
    }

    #[test]
    fn test_config_builder() {
        let config = TracingConfig::default()
            .with_service_name("my-service")
            .with_log_level("debug")
            .with_json_output();

        assert_eq!(config.service_name, "my-service");
        assert_eq!(config.log_level, "debug");
        assert!(config.json_output);
    }
}
