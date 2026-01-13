//! Centralized configuration for Token Service.
//!
//! All configuration is loaded from environment variables and validated
//! at startup. Platform library configurations are included.

use crate::error::TokenError;
use rust_common::{CacheClientConfig, CircuitBreakerConfig, LoggingClientConfig};
use std::env;
use std::time::Duration;

/// JWT signing algorithm.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum JwtAlgorithm {
    /// RSA with SHA-256
    RS256,
    /// RSA-PSS with SHA-256
    PS256,
    /// ECDSA with P-256 and SHA-256
    ES256,
}

impl JwtAlgorithm {
    /// Parse algorithm from string.
    pub fn from_str(s: &str) -> Result<Self, TokenError> {
        match s.to_uppercase().as_str() {
            "RS256" => Ok(Self::RS256),
            "PS256" => Ok(Self::PS256),
            "ES256" => Ok(Self::ES256),
            _ => Err(TokenError::config(format!("Invalid JWT algorithm: {}", s))),
        }
    }

    /// Get algorithm name for JWT header.
    #[must_use]
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::RS256 => "RS256",
            Self::PS256 => "PS256",
            Self::ES256 => "ES256",
        }
    }
}

/// KMS provider configuration.
#[derive(Debug, Clone)]
pub enum KmsProvider {
    /// AWS KMS
    Aws {
        /// AWS region
        region: String,
    },
    /// Mock KMS for testing
    Mock,
}

/// Token Service configuration.
#[derive(Debug, Clone)]
pub struct Config {
    // Server settings
    /// Host to bind to
    pub host: String,
    /// Port to listen on
    pub port: u16,

    // JWT settings
    /// JWT issuer claim
    pub jwt_issuer: String,
    /// JWT signing algorithm
    pub jwt_algorithm: JwtAlgorithm,
    /// Access token TTL
    pub access_token_ttl: Duration,
    /// Refresh token TTL
    pub refresh_token_ttl: Duration,

    // KMS settings
    /// KMS provider
    pub kms_provider: KmsProvider,
    /// KMS key ID
    pub kms_key_id: String,
    /// Enable fallback signing when KMS unavailable
    pub kms_fallback_enabled: bool,
    /// Fallback timeout duration
    pub kms_fallback_timeout: Duration,

    // DPoP settings
    /// Maximum clock skew for DPoP validation
    pub dpop_clock_skew: Duration,
    /// DPoP JTI cache TTL
    pub dpop_jti_ttl: Duration,

    // Platform integration
    /// Cache client configuration
    pub cache: CacheClientConfig,
    /// Logging client configuration
    pub logging: LoggingClientConfig,
    /// Circuit breaker configuration
    pub circuit_breaker: CircuitBreakerConfig,

    // Security
    /// Encryption key for cached data (32 bytes for AES-256)
    pub encryption_key: [u8; 32],
}

impl Config {
    /// Load configuration from environment variables.
    ///
    /// # Errors
    ///
    /// Returns an error if required variables are missing or invalid.
    pub fn from_env() -> Result<Self, TokenError> {
        dotenvy::dotenv().ok();

        let host = env::var("HOST").unwrap_or_else(|_| "0.0.0.0".to_string());
        let port = parse_env("PORT", 50051)?;

        let jwt_issuer = env::var("JWT_ISSUER").unwrap_or_else(|_| "auth-platform".to_string());
        let jwt_algorithm = JwtAlgorithm::from_str(
            &env::var("JWT_ALGORITHM").unwrap_or_else(|_| "RS256".to_string()),
        )?;
        let access_token_ttl = Duration::from_secs(parse_env("ACCESS_TOKEN_TTL", 900)?);
        let refresh_token_ttl = Duration::from_secs(parse_env("REFRESH_TOKEN_TTL", 604800)?);

        let kms_provider = match env::var("KMS_PROVIDER")
            .unwrap_or_else(|_| "mock".to_string())
            .to_lowercase()
            .as_str()
        {
            "aws" => KmsProvider::Aws {
                region: env::var("AWS_REGION").unwrap_or_else(|_| "us-east-1".to_string()),
            },
            _ => KmsProvider::Mock,
        };
        let kms_key_id = env::var("KMS_KEY_ID").unwrap_or_else(|_| "default-key".to_string());
        let kms_fallback_enabled = parse_env("KMS_FALLBACK_ENABLED", false)?;
        let kms_fallback_timeout = Duration::from_secs(parse_env("KMS_FALLBACK_TIMEOUT", 300)?);

        let dpop_clock_skew = Duration::from_secs(parse_env("DPOP_CLOCK_SKEW", 60)?);
        let dpop_jti_ttl = Duration::from_secs(parse_env("DPOP_JTI_TTL", 300)?);

        let cache_address =
            env::var("CACHE_SERVICE_ADDRESS").unwrap_or_else(|_| "http://localhost:50051".to_string());
        let logging_address =
            env::var("LOGGING_SERVICE_ADDRESS").unwrap_or_else(|_| "http://localhost:5001".to_string());

        let encryption_key = parse_encryption_key()?;

        let cache = CacheClientConfig::default()
            .with_address(cache_address)
            .with_namespace("token")
            .with_default_ttl(refresh_token_ttl)
            .with_encryption_key(encryption_key);

        let logging = LoggingClientConfig::default()
            .with_address(logging_address)
            .with_service_id("token-service");

        let circuit_breaker = CircuitBreakerConfig::default()
            .with_failure_threshold(parse_env("CB_FAILURE_THRESHOLD", 5)?)
            .with_success_threshold(parse_env("CB_SUCCESS_THRESHOLD", 2)?)
            .with_timeout(Duration::from_secs(parse_env("CB_TIMEOUT", 30)?));

        Ok(Self {
            host,
            port,
            jwt_issuer,
            jwt_algorithm,
            access_token_ttl,
            refresh_token_ttl,
            kms_provider,
            kms_key_id,
            kms_fallback_enabled,
            kms_fallback_timeout,
            dpop_clock_skew,
            dpop_jti_ttl,
            cache,
            logging,
            circuit_breaker,
            encryption_key,
        })
    }
}

/// Parse environment variable with default value.
fn parse_env<T: std::str::FromStr>(name: &str, default: T) -> Result<T, TokenError>
where
    T::Err: std::fmt::Display,
{
    match env::var(name) {
        Ok(val) => val
            .parse()
            .map_err(|e| TokenError::config(format!("Invalid {}: {}", name, e))),
        Err(_) => Ok(default),
    }
}

/// Parse encryption key from environment.
fn parse_encryption_key() -> Result<[u8; 32], TokenError> {
    match env::var("ENCRYPTION_KEY") {
        Ok(key) => {
            let bytes = base64::Engine::decode(
                &base64::engine::general_purpose::STANDARD,
                &key,
            )
            .map_err(|e| TokenError::config(format!("Invalid ENCRYPTION_KEY: {}", e)))?;

            if bytes.len() != 32 {
                return Err(TokenError::config(format!(
                    "ENCRYPTION_KEY must be 32 bytes, got {}",
                    bytes.len()
                )));
            }

            let mut arr = [0u8; 32];
            arr.copy_from_slice(&bytes);
            Ok(arr)
        }
        Err(_) => {
            // Generate random key for development
            use rand::RngCore;
            let mut key = [0u8; 32];
            rand::thread_rng().fill_bytes(&mut key);
            Ok(key)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_jwt_algorithm_parsing() {
        assert_eq!(JwtAlgorithm::from_str("RS256").unwrap(), JwtAlgorithm::RS256);
        assert_eq!(JwtAlgorithm::from_str("rs256").unwrap(), JwtAlgorithm::RS256);
        assert_eq!(JwtAlgorithm::from_str("PS256").unwrap(), JwtAlgorithm::PS256);
        assert_eq!(JwtAlgorithm::from_str("ES256").unwrap(), JwtAlgorithm::ES256);
        assert!(JwtAlgorithm::from_str("invalid").is_err());
    }

    #[test]
    fn test_jwt_algorithm_as_str() {
        assert_eq!(JwtAlgorithm::RS256.as_str(), "RS256");
        assert_eq!(JwtAlgorithm::PS256.as_str(), "PS256");
        assert_eq!(JwtAlgorithm::ES256.as_str(), "ES256");
    }

    #[test]
    fn test_config_from_env_defaults() {
        // Clear any existing env vars
        env::remove_var("HOST");
        env::remove_var("PORT");
        env::remove_var("JWT_ISSUER");

        let config = Config::from_env().unwrap();

        assert_eq!(config.host, "0.0.0.0");
        assert_eq!(config.port, 50051);
        assert_eq!(config.jwt_issuer, "auth-platform");
        assert_eq!(config.jwt_algorithm, JwtAlgorithm::RS256);
    }
}
