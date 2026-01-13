//! Centralized HTTP client configuration and building.
//!
//! This module provides a standardized way to create HTTP clients with
//! consistent configuration across all auth-platform services.

use reqwest::{Client, ClientBuilder};
use std::time::Duration;

/// HTTP client configuration.
///
/// Provides sensible defaults for production use with connection pooling,
/// timeouts, and TLS configuration.
#[derive(Debug, Clone)]
pub struct HttpConfig {
    /// Request timeout (default: 30s)
    pub timeout: Duration,
    /// Connection timeout (default: 10s)
    pub connect_timeout: Duration,
    /// Pool idle timeout (default: 90s)
    pub pool_idle_timeout: Duration,
    /// Maximum idle connections per host (default: 10)
    pub pool_max_idle_per_host: usize,
    /// User agent string
    pub user_agent: String,
}

impl Default for HttpConfig {
    fn default() -> Self {
        Self {
            timeout: Duration::from_secs(30),
            connect_timeout: Duration::from_secs(10),
            pool_idle_timeout: Duration::from_secs(90),
            pool_max_idle_per_host: 10,
            user_agent: "auth-platform-rust/1.0".to_string(),
        }
    }
}

impl HttpConfig {
    /// Create a new HTTP config with custom timeout.
    #[must_use]
    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Create a new HTTP config with custom connect timeout.
    #[must_use]
    pub fn with_connect_timeout(mut self, timeout: Duration) -> Self {
        self.connect_timeout = timeout;
        self
    }

    /// Create a new HTTP config with custom user agent.
    #[must_use]
    pub fn with_user_agent(mut self, user_agent: impl Into<String>) -> Self {
        self.user_agent = user_agent.into();
        self
    }

    /// Create a new HTTP config with custom pool settings.
    #[must_use]
    pub fn with_pool_config(mut self, idle_timeout: Duration, max_idle: usize) -> Self {
        self.pool_idle_timeout = idle_timeout;
        self.pool_max_idle_per_host = max_idle;
        self
    }
}

/// Build a configured HTTP client.
///
/// Creates a reqwest client with rustls TLS, connection pooling, and
/// the specified configuration.
///
/// # Errors
///
/// Returns an error if the client cannot be built (e.g., TLS initialization fails).
///
/// # Examples
///
/// ```
/// use rust_common::{HttpConfig, build_http_client};
/// use std::time::Duration;
///
/// let config = HttpConfig::default()
///     .with_timeout(Duration::from_secs(60));
/// let client = build_http_client(&config).expect("Failed to build client");
/// ```
pub fn build_http_client(config: &HttpConfig) -> Result<Client, reqwest::Error> {
    ClientBuilder::new()
        .timeout(config.timeout)
        .connect_timeout(config.connect_timeout)
        .pool_idle_timeout(config.pool_idle_timeout)
        .pool_max_idle_per_host(config.pool_max_idle_per_host)
        .user_agent(&config.user_agent)
        .use_rustls_tls()
        .build()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = HttpConfig::default();
        assert_eq!(config.timeout, Duration::from_secs(30));
        assert_eq!(config.connect_timeout, Duration::from_secs(10));
        assert_eq!(config.pool_max_idle_per_host, 10);
    }

    #[test]
    fn test_config_builder() {
        let config = HttpConfig::default()
            .with_timeout(Duration::from_secs(60))
            .with_user_agent("test-agent");

        assert_eq!(config.timeout, Duration::from_secs(60));
        assert_eq!(config.user_agent, "test-agent");
    }

    #[test]
    fn test_build_client() {
        let config = HttpConfig::default();
        let result = build_http_client(&config);
        assert!(result.is_ok());
    }
}
