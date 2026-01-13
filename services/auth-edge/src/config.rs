//! Type-Safe Configuration with Validation
//!
//! Provides type-safe configuration with URL validation and environment variable support.

use serde::Deserialize;
use std::env;
use thiserror::Error;
use url::Url;

/// Configuration errors.
#[derive(Error, Debug)]
pub enum ConfigError {
    /// Invalid URL format
    #[error("Invalid URL for {field}: {reason}")]
    InvalidUrl { field: String, reason: String },

    /// Invalid port number
    #[error("Invalid port: must be between 1 and 65535")]
    InvalidPort,

    /// Invalid TTL value
    #[error("Invalid TTL: must be greater than 0")]
    InvalidTtl,

    /// Invalid threshold value
    #[error("Invalid threshold: must be greater than 0")]
    InvalidThreshold,

    /// Missing required field
    #[error("Missing required configuration: {0}")]
    MissingRequired(String),

    /// Environment variable parse error
    #[error("Failed to parse environment variable {name}: {reason}")]
    ParseError { name: String, reason: String },
}

/// Service configuration with validation.
#[derive(Debug, Clone)]
pub struct Config {
    /// Server host address
    pub host: String,
    /// Server port (1-65535)
    pub port: u16,
    /// Token service URL
    pub token_service_url: Url,
    /// Session service URL
    pub session_service_url: Url,
    /// IAM service URL
    pub iam_service_url: Url,
    /// JWKS endpoint URL
    pub jwks_url: Url,
    /// Cache service URL
    pub cache_service_url: Url,
    /// Logging service URL
    pub logging_service_url: Url,
    /// OTLP endpoint URL
    pub otlp_endpoint: Url,
    /// JWKS cache TTL in seconds (must be > 0)
    pub jwks_cache_ttl_seconds: u64,
    /// Circuit breaker failure threshold (must be > 0)
    pub circuit_breaker_failure_threshold: u32,
    /// Circuit breaker timeout in seconds
    pub circuit_breaker_timeout_seconds: u64,
    /// Request timeout in seconds
    pub request_timeout_secs: u64,
    /// Allowed SPIFFE domains
    pub allowed_spiffe_domains: Vec<String>,
    /// Graceful shutdown timeout in seconds
    pub shutdown_timeout_seconds: u64,
    /// Cache encryption key (32 bytes for AES-256) - deprecated, use crypto_service
    pub cache_encryption_key: Option<[u8; 32]>,
    /// Crypto service URL
    pub crypto_service_url: Url,
    /// Crypto key namespace for isolation
    pub crypto_key_namespace: String,
    /// Enable fallback when crypto-service is unavailable
    pub crypto_fallback_enabled: bool,
    /// Crypto service timeout in seconds
    pub crypto_timeout_secs: u64,
}

impl Config {
    /// Loads configuration from environment variables with validation.
    pub fn from_env() -> Result<Self, ConfigError> {
        dotenvy::dotenv().ok();

        let config = Self {
            host: env::var("HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: parse_env("PORT", 50052)?,
            token_service_url: parse_url_env("TOKEN_SERVICE_URL", "http://localhost:50051")?,
            session_service_url: parse_url_env("SESSION_SERVICE_URL", "http://localhost:50053")?,
            iam_service_url: parse_url_env("IAM_SERVICE_URL", "http://localhost:50054")?,
            jwks_url: parse_url_env("JWKS_URL", "http://localhost:50051/.well-known/jwks.json")?,
            cache_service_url: parse_url_env("CACHE_SERVICE_URL", "http://localhost:50060")?,
            logging_service_url: parse_url_env("LOGGING_SERVICE_URL", "http://localhost:50061")?,
            otlp_endpoint: parse_url_env("OTLP_ENDPOINT", "http://localhost:4317")?,
            jwks_cache_ttl_seconds: parse_env("JWKS_CACHE_TTL", 3600)?,
            circuit_breaker_failure_threshold: parse_env("CB_FAILURE_THRESHOLD", 5)?,
            circuit_breaker_timeout_seconds: parse_env("CB_TIMEOUT", 30)?,
            request_timeout_secs: parse_env("REQUEST_TIMEOUT", 30)?,
            allowed_spiffe_domains: parse_list_env("ALLOWED_SPIFFE_DOMAINS"),
            shutdown_timeout_seconds: parse_env("SHUTDOWN_TIMEOUT", 30)?,
            cache_encryption_key: parse_encryption_key_env("CACHE_ENCRYPTION_KEY"),
            crypto_service_url: parse_url_env("CRYPTO_SERVICE_URL", "http://localhost:50051")?,
            crypto_key_namespace: env::var("CRYPTO_KEY_NAMESPACE")
                .unwrap_or_else(|_| "auth-edge".to_string()),
            crypto_fallback_enabled: parse_env("CRYPTO_FALLBACK_ENABLED", true)?,
            crypto_timeout_secs: parse_env("CRYPTO_TIMEOUT", 5)?,
        };

        config.validate()?;
        Ok(config)
    }

    /// Validates the configuration.
    fn validate(&self) -> Result<(), ConfigError> {
        if self.port == 0 {
            return Err(ConfigError::InvalidPort);
        }
        if self.jwks_cache_ttl_seconds == 0 {
            return Err(ConfigError::InvalidTtl);
        }
        if self.circuit_breaker_failure_threshold == 0 {
            return Err(ConfigError::InvalidThreshold);
        }
        if self.crypto_key_namespace.is_empty() {
            return Err(ConfigError::MissingRequired(
                "crypto_key_namespace".to_string(),
            ));
        }
        if self.crypto_timeout_secs == 0 {
            return Err(ConfigError::ParseError {
                name: "CRYPTO_TIMEOUT".to_string(),
                reason: "timeout must be greater than 0".to_string(),
            });
        }
        Ok(())
    }

    /// Gets the crypto service URL as a string.
    #[must_use]
    pub fn crypto_service_url_str(&self) -> &str {
        self.crypto_service_url.as_str()
    }

    /// Creates a CryptoClientConfig from this config.
    #[must_use]
    pub fn crypto_client_config(&self) -> crate::crypto::CryptoClientConfig {
        crate::crypto::CryptoClientConfig::default()
            .with_service_url(self.crypto_service_url.clone())
            .with_key_namespace(&self.crypto_key_namespace)
            .with_fallback_enabled(self.crypto_fallback_enabled)
            .with_timeout(std::time::Duration::from_secs(self.crypto_timeout_secs))
    }

    /// Gets the cache service URL as a string.
    #[must_use]
    pub fn cache_service_url_str(&self) -> &str {
        self.cache_service_url.as_str()
    }

    /// Gets the logging service URL as a string.
    #[must_use]
    pub fn logging_service_url_str(&self) -> &str {
        self.logging_service_url.as_str()
    }

    /// Gets the OTLP endpoint URL as a string.
    #[must_use]
    pub fn otlp_endpoint_str(&self) -> &str {
        self.otlp_endpoint.as_str()
    }

    /// Gets the JWKS URL as a string.
    #[must_use]
    pub fn jwks_url_str(&self) -> &str {
        self.jwks_url.as_str()
    }
}

/// Parse an environment variable with a default value.
fn parse_env<T: std::str::FromStr>(name: &str, default: T) -> Result<T, ConfigError>
where
    T::Err: std::fmt::Display,
{
    match env::var(name) {
        Ok(val) => val.parse().map_err(|e: T::Err| ConfigError::ParseError {
            name: name.to_string(),
            reason: e.to_string(),
        }),
        Err(_) => Ok(default),
    }
}

/// Parse a URL environment variable with a default value.
fn parse_url_env(name: &str, default: &str) -> Result<Url, ConfigError> {
    let url_str = env::var(name).unwrap_or_else(|_| default.to_string());
    Url::parse(&url_str).map_err(|e| ConfigError::InvalidUrl {
        field: name.to_string(),
        reason: e.to_string(),
    })
}

/// Parse a comma-separated list environment variable.
fn parse_list_env(name: &str) -> Vec<String> {
    env::var(name)
        .map(|v| v.split(',').map(|s| s.trim().to_string()).collect())
        .unwrap_or_default()
}

/// Parse an encryption key from hex-encoded environment variable.
fn parse_encryption_key_env(name: &str) -> Option<[u8; 32]> {
    env::var(name).ok().and_then(|hex| {
        let bytes: Vec<u8> = (0..hex.len())
            .step_by(2)
            .filter_map(|i| u8::from_str_radix(&hex[i..i + 2], 16).ok())
            .collect();
        if bytes.len() == 32 {
            let mut arr = [0u8; 32];
            arr.copy_from_slice(&bytes);
            Some(arr)
        } else {
            None
        }
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_config_base() -> Config {
        Config {
            host: "localhost".to_string(),
            port: 8080,
            token_service_url: Url::parse("http://localhost:50051").unwrap(),
            session_service_url: Url::parse("http://localhost:50053").unwrap(),
            iam_service_url: Url::parse("http://localhost:50054").unwrap(),
            jwks_url: Url::parse("http://localhost:50051/.well-known/jwks.json").unwrap(),
            cache_service_url: Url::parse("http://localhost:50060").unwrap(),
            logging_service_url: Url::parse("http://localhost:50061").unwrap(),
            otlp_endpoint: Url::parse("http://localhost:4317").unwrap(),
            jwks_cache_ttl_seconds: 3600,
            circuit_breaker_failure_threshold: 5,
            circuit_breaker_timeout_seconds: 30,
            request_timeout_secs: 30,
            allowed_spiffe_domains: vec![],
            shutdown_timeout_seconds: 30,
            cache_encryption_key: None,
            crypto_service_url: Url::parse("http://localhost:50051").unwrap(),
            crypto_key_namespace: "auth-edge".to_string(),
            crypto_fallback_enabled: true,
            crypto_timeout_secs: 5,
        }
    }

    #[test]
    fn test_config_validation_invalid_port() {
        let mut config = test_config_base();
        config.port = 0;
        assert!(matches!(config.validate(), Err(ConfigError::InvalidPort)));
    }

    #[test]
    fn test_config_validation_invalid_ttl() {
        let mut config = test_config_base();
        config.jwks_cache_ttl_seconds = 0;
        assert!(matches!(config.validate(), Err(ConfigError::InvalidTtl)));
    }

    #[test]
    fn test_config_validation_empty_crypto_namespace() {
        let mut config = test_config_base();
        config.crypto_key_namespace = String::new();
        assert!(matches!(
            config.validate(),
            Err(ConfigError::MissingRequired(_))
        ));
    }

    #[test]
    fn test_config_validation_zero_crypto_timeout() {
        let mut config = test_config_base();
        config.crypto_timeout_secs = 0;
        assert!(matches!(config.validate(), Err(ConfigError::ParseError { .. })));
    }

    #[test]
    fn test_parse_url_env_invalid() {
        let result = parse_url_env("NONEXISTENT_VAR", "not-a-valid-url");
        assert!(result.is_err());
    }

    #[test]
    fn test_crypto_client_config_creation() {
        let config = test_config_base();
        let crypto_config = config.crypto_client_config();
        assert_eq!(crypto_config.key_namespace, "auth-edge");
        assert!(crypto_config.fallback_enabled);
    }
}
