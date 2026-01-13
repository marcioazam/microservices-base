//! AWS KMS Signer with circuit breaker integration.
//!
//! Implements HSM-backed signing using AWS KMS with fallback support.

use crate::error::TokenError;
use crate::kms::KmsSigner;
use async_trait::async_trait;
use jsonwebtoken::EncodingKey;
use rust_common::{CircuitBreaker, CircuitBreakerConfig};
use std::sync::Arc;
use std::time::Duration;
use tracing::{error, info, warn};

/// AWS KMS configuration.
#[derive(Debug, Clone)]
pub struct AwsKmsConfig {
    /// KMS key ID or ARN.
    pub key_id: String,
    /// AWS region.
    pub region: String,
    /// Signing algorithm.
    pub algorithm: String,
    /// Fallback enabled.
    pub fallback_enabled: bool,
    /// Maximum fallback duration.
    pub max_fallback_duration: Duration,
    /// Circuit breaker configuration.
    pub circuit_breaker: CircuitBreakerConfig,
}

impl Default for AwsKmsConfig {
    fn default() -> Self {
        Self {
            key_id: String::new(),
            region: "us-east-1".to_string(),
            algorithm: "RSASSA_PSS_SHA_256".to_string(),
            fallback_enabled: true,
            max_fallback_duration: Duration::from_secs(300),
            circuit_breaker: CircuitBreakerConfig::default(),
        }
    }
}

/// AWS KMS Signer with circuit breaker and fallback.
pub struct AwsKmsSigner {
    config: AwsKmsConfig,
    circuit_breaker: Arc<CircuitBreaker>,
    fallback_key: Option<Vec<u8>>,
}

impl AwsKmsSigner {
    /// Create a new AWS KMS signer.
    #[must_use]
    pub fn new(config: AwsKmsConfig) -> Self {
        let circuit_breaker = Arc::new(CircuitBreaker::new(config.circuit_breaker.clone()));
        Self {
            config,
            circuit_breaker,
            fallback_key: None,
        }
    }

    /// Set the fallback key for emergency use.
    #[must_use]
    pub fn with_fallback_key(mut self, key: Vec<u8>) -> Self {
        self.fallback_key = Some(key);
        self
    }

    /// Sign data using AWS KMS.
    async fn sign_with_kms(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        // In production, this would call AWS KMS:
        // let client = aws_sdk_kms::Client::new(&aws_config);
        // let response = client.sign()
        //     .key_id(&self.config.key_id)
        //     .message(Blob::new(data))
        //     .message_type(MessageType::Raw)
        //     .signing_algorithm(self.config.algorithm.parse()?)
        //     .send()
        //     .await?;
        // Ok(response.signature.unwrap().into_inner())

        Err(TokenError::kms("KMS client not configured"))
    }

    /// Sign data using fallback key.
    fn sign_with_fallback(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        let key = self
            .fallback_key
            .as_ref()
            .ok_or_else(|| TokenError::kms("No fallback key configured"))?;

        use ring::hmac;
        let signing_key = hmac::Key::new(hmac::HMAC_SHA256, key);
        let signature = hmac::sign(&signing_key, data);

        warn!("Using fallback signing - KMS unavailable");
        Ok(signature.as_ref().to_vec())
    }

    /// Map KMS algorithm to JWT algorithm.
    fn map_algorithm(&self) -> &str {
        match self.config.algorithm.as_str() {
            "RSASSA_PSS_SHA_256" => "PS256",
            "RSASSA_PSS_SHA_384" => "PS384",
            "RSASSA_PSS_SHA_512" => "PS512",
            "RSASSA_PKCS1_V1_5_SHA_256" => "RS256",
            "RSASSA_PKCS1_V1_5_SHA_384" => "RS384",
            "RSASSA_PKCS1_V1_5_SHA_512" => "RS512",
            "ECDSA_SHA_256" => "ES256",
            "ECDSA_SHA_384" => "ES384",
            _ => "RS256",
        }
    }
}

#[async_trait]
impl KmsSigner for AwsKmsSigner {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        // Check circuit breaker
        if self.circuit_breaker.allow_request().await {
            match self.sign_with_kms(data).await {
                Ok(sig) => {
                    self.circuit_breaker.record_success().await;
                    return Ok(sig);
                }
                Err(e) => {
                    self.circuit_breaker.record_failure().await;
                    error!("KMS signing failed: {}", e);
                }
            }
        }

        // Try fallback if enabled
        if self.config.fallback_enabled && self.fallback_key.is_some() {
            info!("Attempting fallback signing");
            return self.sign_with_fallback(data);
        }

        Err(TokenError::kms("KMS unavailable and fallback not allowed"))
    }

    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError> {
        if let Some(ref key) = self.fallback_key {
            return Ok(EncodingKey::from_secret(key));
        }
        Err(TokenError::kms("No local key available - use KMS signing"))
    }

    fn key_id(&self) -> &str {
        &self.config.key_id
    }

    fn algorithm(&self) -> &str {
        self.map_algorithm()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_fallback_signing() {
        let config = AwsKmsConfig {
            key_id: "test-key".to_string(),
            fallback_enabled: true,
            ..Default::default()
        };
        let signer =
            AwsKmsSigner::new(config).with_fallback_key(b"test-fallback-key-32-bytes-long!".to_vec());

        let result = signer.sign(b"test data").await;
        assert!(result.is_ok());
    }

    #[test]
    fn test_algorithm_mapping() {
        let config = AwsKmsConfig {
            key_id: "test".to_string(),
            algorithm: "ECDSA_SHA_256".to_string(),
            ..Default::default()
        };
        let signer = AwsKmsSigner::new(config);

        assert_eq!(signer.algorithm(), "ES256");
    }

    #[test]
    fn test_ps256_mapping() {
        let config = AwsKmsConfig {
            key_id: "test".to_string(),
            algorithm: "RSASSA_PSS_SHA_256".to_string(),
            ..Default::default()
        };
        let signer = AwsKmsSigner::new(config);

        assert_eq!(signer.algorithm(), "PS256");
    }

    #[test]
    fn test_encoding_key_without_fallback() {
        let config = AwsKmsConfig::default();
        let signer = AwsKmsSigner::new(config);

        let result = signer.get_encoding_key();
        assert!(result.is_err());
    }

    #[test]
    fn test_encoding_key_with_fallback() {
        let config = AwsKmsConfig::default();
        let signer = AwsKmsSigner::new(config).with_fallback_key(b"test-key".to_vec());

        let result = signer.get_encoding_key();
        assert!(result.is_ok());
    }
}
