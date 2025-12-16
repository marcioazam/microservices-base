//! AWS KMS Signer Implementation
//!
//! Implements HSM-backed signing using AWS KMS for token signing operations.

use crate::error::TokenError;
use crate::jwt::signer::JwtSigner;
use async_trait::async_trait;
use jsonwebtoken::EncodingKey;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use tracing::{info, warn, error};

/// AWS KMS configuration
#[derive(Debug, Clone)]
pub struct AwsKmsConfig {
    /// KMS key ID or ARN
    pub key_id: String,
    /// AWS region
    pub region: String,
    /// Signing algorithm (RSASSA_PSS_SHA_256, ECDSA_SHA_256, etc.)
    pub algorithm: String,
    /// Cache TTL for public key
    pub cache_ttl: Duration,
    /// Fallback enabled
    pub fallback_enabled: bool,
    /// Maximum fallback duration
    pub max_fallback_duration: Duration,
}

impl Default for AwsKmsConfig {
    fn default() -> Self {
        AwsKmsConfig {
            key_id: String::new(),
            region: "us-east-1".to_string(),
            algorithm: "RSASSA_PSS_SHA_256".to_string(),
            cache_ttl: Duration::from_secs(3600),
            fallback_enabled: true,
            max_fallback_duration: Duration::from_secs(300), // 5 minutes
        }
    }
}

/// Cached public key for validation
struct CachedKey {
    public_key: Vec<u8>,
    cached_at: Instant,
}

/// Circuit breaker state for KMS
#[derive(Debug, Clone, Copy, PartialEq)]
enum KmsCircuitState {
    Closed,
    Open,
    HalfOpen,
}

/// AWS KMS Signer with fallback support
pub struct AwsKmsSigner {
    config: AwsKmsConfig,
    // In production, this would be aws_sdk_kms::Client
    // Using placeholder for compilation
    cached_key: Arc<RwLock<Option<CachedKey>>>,
    circuit_state: Arc<RwLock<KmsCircuitState>>,
    failure_count: Arc<RwLock<u32>>,
    last_failure: Arc<RwLock<Option<Instant>>>,
    fallback_key: Option<Vec<u8>>,
}

impl AwsKmsSigner {
    pub fn new(config: AwsKmsConfig) -> Self {
        AwsKmsSigner {
            config,
            cached_key: Arc::new(RwLock::new(None)),
            circuit_state: Arc::new(RwLock::new(KmsCircuitState::Closed)),
            failure_count: Arc::new(RwLock::new(0)),
            last_failure: Arc::new(RwLock::new(None)),
            fallback_key: None,
        }
    }

    /// Sets the fallback key for emergency use
    pub fn with_fallback_key(mut self, key: Vec<u8>) -> Self {
        self.fallback_key = Some(key);
        self
    }

    /// Checks if KMS is available (circuit breaker)
    async fn is_kms_available(&self) -> bool {
        let state = *self.circuit_state.read().await;
        
        match state {
            KmsCircuitState::Closed => true,
            KmsCircuitState::Open => {
                // Check if timeout has passed
                if let Some(last) = *self.last_failure.read().await {
                    if last.elapsed() > Duration::from_secs(30) {
                        *self.circuit_state.write().await = KmsCircuitState::HalfOpen;
                        return true;
                    }
                }
                false
            }
            KmsCircuitState::HalfOpen => true,
        }
    }

    /// Records a KMS failure
    async fn record_failure(&self) {
        let mut count = self.failure_count.write().await;
        *count += 1;
        
        if *count >= 3 {
            *self.circuit_state.write().await = KmsCircuitState::Open;
            *self.last_failure.write().await = Some(Instant::now());
            warn!("KMS circuit breaker opened after {} failures", *count);
        }
    }

    /// Records a KMS success
    async fn record_success(&self) {
        let state = *self.circuit_state.read().await;
        
        if state == KmsCircuitState::HalfOpen {
            *self.circuit_state.write().await = KmsCircuitState::Closed;
            *self.failure_count.write().await = 0;
            info!("KMS circuit breaker closed");
        }
    }

    /// Signs data using KMS
    async fn sign_with_kms(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        // In production, this would call AWS KMS:
        // let response = self.client.sign()
        //     .key_id(&self.config.key_id)
        //     .message(Blob::new(data))
        //     .message_type(MessageType::Raw)
        //     .signing_algorithm(self.config.algorithm.parse()?)
        //     .send()
        //     .await?;
        // Ok(response.signature.unwrap().into_inner())

        // Placeholder implementation
        Err(TokenError::KmsError("KMS client not configured".to_string()))
    }

    /// Signs data using fallback key
    fn sign_with_fallback(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        let key = self.fallback_key.as_ref()
            .ok_or_else(|| TokenError::KmsError("No fallback key configured".to_string()))?;

        use ring::hmac;
        let signing_key = hmac::Key::new(hmac::HMAC_SHA256, key);
        let signature = hmac::sign(&signing_key, data);
        
        warn!("Using fallback signing - KMS unavailable");
        Ok(signature.as_ref().to_vec())
    }

    /// Checks if fallback is allowed
    async fn is_fallback_allowed(&self) -> bool {
        if !self.config.fallback_enabled {
            return false;
        }

        if let Some(last) = *self.last_failure.read().await {
            return last.elapsed() < self.config.max_fallback_duration;
        }

        false
    }
}

#[async_trait]
impl JwtSigner for AwsKmsSigner {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        // Check circuit breaker
        if self.is_kms_available().await {
            match self.sign_with_kms(data).await {
                Ok(sig) => {
                    self.record_success().await;
                    return Ok(sig);
                }
                Err(e) => {
                    self.record_failure().await;
                    error!("KMS signing failed: {}", e);
                }
            }
        }

        // Try fallback if allowed
        if self.is_fallback_allowed().await {
            return self.sign_with_fallback(data);
        }

        Err(TokenError::KmsError("KMS unavailable and fallback not allowed".to_string()))
    }

    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError> {
        // For KMS, we don't have direct access to the private key
        // This is used for fallback only
        if let Some(ref key) = self.fallback_key {
            return Ok(EncodingKey::from_secret(key));
        }
        Err(TokenError::KmsError("No local key available - use KMS signing".to_string()))
    }

    fn get_key_id(&self) -> &str {
        &self.config.key_id
    }

    fn get_algorithm(&self) -> &str {
        match self.config.algorithm.as_str() {
            "RSASSA_PSS_SHA_256" => "PS256",
            "RSASSA_PKCS1_V1_5_SHA_256" => "RS256",
            "ECDSA_SHA_256" => "ES256",
            "ECDSA_SHA_384" => "ES384",
            _ => "RS256",
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_circuit_breaker_opens_on_failures() {
        let config = AwsKmsConfig {
            key_id: "test-key".to_string(),
            ..Default::default()
        };
        let signer = AwsKmsSigner::new(config);

        // Record 3 failures
        for _ in 0..3 {
            signer.record_failure().await;
        }

        // Circuit should be open
        assert!(!signer.is_kms_available().await);
    }

    #[tokio::test]
    async fn test_fallback_signing() {
        let config = AwsKmsConfig {
            key_id: "test-key".to_string(),
            fallback_enabled: true,
            ..Default::default()
        };
        let signer = AwsKmsSigner::new(config)
            .with_fallback_key(b"test-fallback-key-32-bytes-long!".to_vec());

        // Trigger fallback by opening circuit
        for _ in 0..3 {
            signer.record_failure().await;
        }

        // Fallback should work
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

        assert_eq!(signer.get_algorithm(), "ES256");
    }
}
