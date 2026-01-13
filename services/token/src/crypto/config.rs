//! Configuration for CryptoClient.

use rust_common::CircuitBreakerConfig;
use std::time::Duration;

/// Configuration for CryptoClient.
#[derive(Debug, Clone)]
pub struct CryptoClientConfig {
    /// Crypto Service gRPC address
    pub address: String,
    /// Key namespace for isolation
    pub namespace: String,
    /// Enable signing via Crypto Service
    pub signing_enabled: bool,
    /// Enable encryption via Crypto Service
    pub encryption_enabled: bool,
    /// Enable fallback to local operations
    pub fallback_enabled: bool,
    /// Circuit breaker configuration
    pub circuit_breaker: CircuitBreakerConfig,
    /// Rate limit (requests per second)
    pub rate_limit: u32,
    /// Connection timeout
    pub connect_timeout: Duration,
    /// Request timeout
    pub request_timeout: Duration,
    /// Key metadata cache TTL
    pub metadata_cache_ttl: Duration,
    /// Maximum cache size
    pub metadata_cache_size: usize,
}

impl Default for CryptoClientConfig {
    fn default() -> Self {
        Self {
            address: "http://localhost:50051".to_string(),
            namespace: "token".to_string(),
            signing_enabled: true,
            encryption_enabled: true,
            fallback_enabled: true,
            circuit_breaker: CircuitBreakerConfig::default(),
            rate_limit: 1000,
            connect_timeout: Duration::from_secs(5),
            request_timeout: Duration::from_secs(30),
            metadata_cache_ttl: Duration::from_secs(300),
            metadata_cache_size: 100,
        }
    }
}

impl CryptoClientConfig {
    /// Create config from environment variables.
    #[must_use]
    pub fn from_env() -> Self {
        let mut config = Self::default();

        if let Ok(addr) = std::env::var("CRYPTO_SERVICE_ADDRESS") {
            config.address = addr;
        }

        if let Ok(ns) = std::env::var("CRYPTO_KEY_NAMESPACE") {
            config.namespace = ns;
        }

        if let Ok(val) = std::env::var("CRYPTO_SIGNING_ENABLED") {
            config.signing_enabled = val.parse().unwrap_or(true);
        }

        if let Ok(val) = std::env::var("CRYPTO_ENCRYPTION_ENABLED") {
            config.encryption_enabled = val.parse().unwrap_or(true);
        }

        if let Ok(val) = std::env::var("CRYPTO_FALLBACK_ENABLED") {
            config.fallback_enabled = val.parse().unwrap_or(true);
        }

        if let Ok(val) = std::env::var("CRYPTO_RATE_LIMIT") {
            config.rate_limit = val.parse().unwrap_or(1000);
        }

        config
    }

    /// Validate configuration.
    ///
    /// # Errors
    ///
    /// Returns error if required configuration is missing.
    pub fn validate(&self) -> Result<(), ConfigValidationError> {
        if (self.signing_enabled || self.encryption_enabled) && self.address.is_empty() {
            return Err(ConfigValidationError::MissingAddress);
        }

        if self.namespace.is_empty() {
            return Err(ConfigValidationError::MissingNamespace);
        }

        if self.rate_limit == 0 {
            return Err(ConfigValidationError::InvalidRateLimit);
        }

        Ok(())
    }

    /// Set address.
    #[must_use]
    pub fn with_address(mut self, address: impl Into<String>) -> Self {
        self.address = address.into();
        self
    }

    /// Set namespace.
    #[must_use]
    pub fn with_namespace(mut self, namespace: impl Into<String>) -> Self {
        self.namespace = namespace.into();
        self
    }

    /// Set signing enabled.
    #[must_use]
    pub const fn with_signing_enabled(mut self, enabled: bool) -> Self {
        self.signing_enabled = enabled;
        self
    }

    /// Set encryption enabled.
    #[must_use]
    pub const fn with_encryption_enabled(mut self, enabled: bool) -> Self {
        self.encryption_enabled = enabled;
        self
    }

    /// Set fallback enabled.
    #[must_use]
    pub const fn with_fallback_enabled(mut self, enabled: bool) -> Self {
        self.fallback_enabled = enabled;
        self
    }

    /// Set rate limit.
    #[must_use]
    pub const fn with_rate_limit(mut self, rate_limit: u32) -> Self {
        self.rate_limit = rate_limit;
        self
    }
}

/// Configuration validation errors.
#[derive(Debug, thiserror::Error)]
pub enum ConfigValidationError {
    #[error("CRYPTO_SERVICE_ADDRESS is required when signing or encryption is enabled")]
    MissingAddress,

    #[error("CRYPTO_KEY_NAMESPACE is required")]
    MissingNamespace,

    #[error("CRYPTO_RATE_LIMIT must be greater than 0")]
    InvalidRateLimit,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = CryptoClientConfig::default();
        assert!(config.signing_enabled);
        assert!(config.encryption_enabled);
        assert!(config.fallback_enabled);
        assert_eq!(config.rate_limit, 1000);
    }

    #[test]
    fn test_validate_missing_address() {
        let config = CryptoClientConfig::default()
            .with_address("")
            .with_signing_enabled(true);
        
        let result = config.validate();
        assert!(matches!(result, Err(ConfigValidationError::MissingAddress)));
    }

    #[test]
    fn test_validate_missing_namespace() {
        let config = CryptoClientConfig::default()
            .with_namespace("");
        
        let result = config.validate();
        assert!(matches!(result, Err(ConfigValidationError::MissingNamespace)));
    }

    #[test]
    fn test_validate_invalid_rate_limit() {
        let config = CryptoClientConfig::default()
            .with_rate_limit(0);
        
        let result = config.validate();
        assert!(matches!(result, Err(ConfigValidationError::InvalidRateLimit)));
    }

    #[test]
    fn test_validate_success() {
        let config = CryptoClientConfig::default();
        assert!(config.validate().is_ok());
    }

    #[test]
    fn test_disabled_services_no_address_required() {
        let config = CryptoClientConfig::default()
            .with_address("")
            .with_signing_enabled(false)
            .with_encryption_enabled(false);
        
        assert!(config.validate().is_ok());
    }
}
