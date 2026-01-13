//! CryptoClient Configuration
//!
//! Type-safe configuration for the crypto-service client with validation.

use std::time::Duration;
use url::Url;

use crate::crypto::error::CryptoError;
use rust_common::CircuitBreakerConfig;

/// Configuration for CryptoClient
#[derive(Debug, Clone)]
pub struct CryptoClientConfig {
    /// gRPC endpoint URL for crypto-service
    pub service_url: Url,
    /// Key namespace for isolation (e.g., "auth-edge")
    pub key_namespace: String,
    /// Enable local fallback when crypto-service is unavailable
    pub fallback_enabled: bool,
    /// Request timeout for crypto operations
    pub timeout: Duration,
    /// Circuit breaker configuration
    pub circuit_breaker: CircuitBreakerConfig,
}

impl Default for CryptoClientConfig {
    fn default() -> Self {
        Self {
            service_url: Url::parse("http://localhost:50051").expect("valid default URL"),
            key_namespace: "auth-edge".to_string(),
            fallback_enabled: true,
            timeout: Duration::from_secs(5),
            circuit_breaker: CircuitBreakerConfig::default(),
        }
    }
}

impl CryptoClientConfig {
    /// Creates a new config with the given service URL
    #[must_use]
    pub fn with_service_url(mut self, url: Url) -> Self {
        self.service_url = url;
        self
    }

    /// Creates a new config with the given key namespace
    #[must_use]
    pub fn with_key_namespace(mut self, namespace: impl Into<String>) -> Self {
        self.key_namespace = namespace.into();
        self
    }

    /// Creates a new config with fallback enabled/disabled
    #[must_use]
    pub const fn with_fallback_enabled(mut self, enabled: bool) -> Self {
        self.fallback_enabled = enabled;
        self
    }

    /// Creates a new config with the given timeout
    #[must_use]
    pub const fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Creates a new config with the given circuit breaker config
    #[must_use]
    pub fn with_circuit_breaker(mut self, config: CircuitBreakerConfig) -> Self {
        self.circuit_breaker = config;
        self
    }

    /// Validates the configuration
    ///
    /// # Errors
    ///
    /// Returns `CryptoError::InvalidConfig` if:
    /// - Service URL scheme is not http or https
    /// - Key namespace is empty
    /// - Timeout is zero
    pub fn validate(&self) -> Result<(), CryptoError> {
        // Validate URL scheme
        let scheme = self.service_url.scheme();
        if scheme != "http" && scheme != "https" {
            return Err(CryptoError::InvalidConfig {
                reason: format!("Invalid URL scheme '{}': must be http or https", scheme),
            });
        }

        // Validate namespace
        if self.key_namespace.is_empty() {
            return Err(CryptoError::InvalidConfig {
                reason: "Key namespace cannot be empty".to_string(),
            });
        }

        if self.key_namespace.len() > 64 {
            return Err(CryptoError::InvalidConfig {
                reason: "Key namespace cannot exceed 64 characters".to_string(),
            });
        }

        // Validate timeout
        if self.timeout.is_zero() {
            return Err(CryptoError::InvalidConfig {
                reason: "Timeout must be greater than zero".to_string(),
            });
        }

        Ok(())
    }

    /// Returns the service URL as a string
    #[must_use]
    pub fn service_url_str(&self) -> &str {
        self.service_url.as_str()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config_is_valid() {
        let config = CryptoClientConfig::default();
        assert!(config.validate().is_ok());
    }

    #[test]
    fn test_empty_namespace_invalid() {
        let config = CryptoClientConfig::default().with_key_namespace("");
        let result = config.validate();
        assert!(matches!(result, Err(CryptoError::InvalidConfig { .. })));
    }

    #[test]
    fn test_zero_timeout_invalid() {
        let config = CryptoClientConfig::default().with_timeout(Duration::ZERO);
        let result = config.validate();
        assert!(matches!(result, Err(CryptoError::InvalidConfig { .. })));
    }

    #[test]
    fn test_builder_pattern() {
        let url = Url::parse("https://crypto.example.com:50051").unwrap();
        let config = CryptoClientConfig::default()
            .with_service_url(url.clone())
            .with_key_namespace("test-namespace")
            .with_fallback_enabled(false)
            .with_timeout(Duration::from_secs(10));

        assert_eq!(config.service_url, url);
        assert_eq!(config.key_namespace, "test-namespace");
        assert!(!config.fallback_enabled);
        assert_eq!(config.timeout, Duration::from_secs(10));
    }
}
