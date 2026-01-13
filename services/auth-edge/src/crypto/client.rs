//! CryptoClient for centralized cryptographic operations
//!
//! Provides integration with crypto-service via gRPC with fallback support.

use std::sync::Arc;
use std::time::{Duration, Instant};

use rust_common::CircuitBreaker;
use tonic::transport::Channel;
use tracing::{info, instrument, warn};

use crate::crypto::config::CryptoClientConfig;
use crate::crypto::error::CryptoError;
use crate::crypto::fallback::{EncryptedData, FallbackHandler, PendingOperation};
use crate::crypto::key_manager::{KeyId, KeyManager, KeyMetadata};
use crate::crypto::metrics::CryptoMetrics;
use crate::crypto::proto::{
    crypto_service_client::CryptoServiceClient, DecryptRequest, EncryptRequest,
    GetKeyMetadataRequest, RotateKeyRequest,
};

/// CryptoClient for centralized cryptographic operations
pub struct CryptoClient {
    /// gRPC client for crypto-service
    grpc_client: CryptoServiceClient<Channel>,
    /// Circuit breaker for resilience
    circuit_breaker: Arc<CircuitBreaker>,
    /// Key manager for KEK/DEK handling
    key_manager: Arc<KeyManager>,
    /// Fallback handler for degraded mode
    fallback: Option<FallbackHandler>,
    /// Metrics collector
    metrics: Arc<CryptoMetrics>,
    /// Configuration
    config: CryptoClientConfig,
}

impl CryptoClient {
    /// Creates a new CryptoClient with the given configuration
    ///
    /// # Errors
    ///
    /// Returns error if:
    /// - Configuration is invalid
    /// - gRPC channel cannot be created
    pub async fn new(config: CryptoClientConfig) -> Result<Self, CryptoError> {
        config.validate()?;

        let channel = Channel::from_shared(config.service_url.to_string())
            .map_err(|e| CryptoError::invalid_config(format!("Invalid URL: {e}")))?
            .timeout(config.timeout)
            .connect_lazy();

        let grpc_client = CryptoServiceClient::new(channel);
        let circuit_breaker = Arc::new(CircuitBreaker::new(config.circuit_breaker.clone()));
        let key_manager = Arc::new(KeyManager::new(
            &config.key_namespace,
            Duration::from_secs(3600), // 1 hour rotation window
        ));
        let metrics = Arc::new(CryptoMetrics::new());

        Ok(Self {
            grpc_client,
            circuit_breaker,
            key_manager,
            fallback: None,
            metrics,
            config,
        })
    }

    /// Initializes the client by setting up keys with crypto-service
    ///
    /// # Errors
    ///
    /// Returns error if key initialization fails
    pub async fn initialize(&mut self, correlation_id: &str) -> Result<(), CryptoError> {
        let mut client = self.grpc_client.clone();
        let key_id = self.key_manager.initialize(&mut client, correlation_id).await?;

        info!(key_id = %key_id, "CryptoClient initialized");

        // Initialize fallback if enabled
        if self.config.fallback_enabled {
            // In production, we would fetch the DEK from crypto-service
            // For now, generate a local DEK for fallback
            let mut dek = [0u8; 32];
            rand::RngCore::fill_bytes(&mut rand::thread_rng(), &mut dek);

            self.fallback = Some(FallbackHandler::new(&dek, key_id.version)?);
            self.key_manager.cache_dek(dek.to_vec(), key_id.version).await?;

            info!("Fallback handler initialized");
        }

        Ok(())
    }

    /// Encrypts data using the crypto-service
    ///
    /// # Errors
    ///
    /// Returns error if encryption fails and fallback is unavailable
    #[instrument(skip(self, plaintext, aad), fields(correlation_id = %correlation_id))]
    pub async fn encrypt(
        &self,
        plaintext: &[u8],
        aad: Option<&[u8]>,
        correlation_id: &str,
    ) -> Result<EncryptedData, CryptoError> {
        let start = Instant::now();

        // Check circuit breaker
        if !self.circuit_breaker.allow_request().await {
            return self.encrypt_fallback(plaintext, aad, start);
        }

        let key_id = self.key_manager.active_key();
        let request = EncryptRequest {
            plaintext: plaintext.to_vec(),
            key_id: Some(key_id.to_proto()),
            aad: aad.map(|a| a.to_vec()).unwrap_or_default(),
            correlation_id: correlation_id.to_string(),
        };

        let mut client = self.grpc_client.clone();
        match client.encrypt(request).await {
            Ok(response) => {
                self.circuit_breaker.record_success().await;
                let inner = response.into_inner();

                let encrypted = EncryptedData {
                    ciphertext: inner.ciphertext,
                    iv: inner.iv,
                    tag: inner.tag,
                    key_id: inner
                        .key_id
                        .map(|k| KeyId::from_proto(&k))
                        .unwrap_or(key_id),
                    algorithm: inner.algorithm,
                };

                self.metrics.record_success("encrypt", start.elapsed());
                self.metrics.set_fallback_active(false);

                Ok(encrypted)
            }
            Err(status) => {
                self.circuit_breaker.record_failure().await;
                let error = CryptoError::from(status);

                if error.is_retryable() {
                    warn!(error = %error, "Crypto-service unavailable, using fallback");
                    self.encrypt_fallback(plaintext, aad, start)
                } else {
                    self.metrics.record_failure("encrypt", "service_error", start.elapsed());
                    Err(error)
                }
            }
        }
    }

    /// Encrypts using local fallback
    fn encrypt_fallback(
        &self,
        plaintext: &[u8],
        aad: Option<&[u8]>,
        start: Instant,
    ) -> Result<EncryptedData, CryptoError> {
        let fallback = self.fallback.as_ref().ok_or(CryptoError::FallbackUnavailable)?;

        let result = fallback.encrypt(plaintext, aad)?;
        self.metrics.record_fallback("encrypt", start.elapsed());
        self.metrics.set_fallback_active(true);

        Ok(result)
    }

    /// Decrypts data using the crypto-service
    ///
    /// # Errors
    ///
    /// Returns error if decryption fails
    #[instrument(skip(self, encrypted, aad), fields(correlation_id = %correlation_id))]
    pub async fn decrypt(
        &self,
        encrypted: &EncryptedData,
        aad: Option<&[u8]>,
        correlation_id: &str,
    ) -> Result<Vec<u8>, CryptoError> {
        let start = Instant::now();

        // If encrypted with local fallback, decrypt locally
        if encrypted.is_local_fallback() {
            return self.decrypt_fallback(encrypted, aad, start);
        }

        // Check circuit breaker
        if !self.circuit_breaker.allow_request().await {
            // Can't decrypt remote-encrypted data locally
            self.metrics.record_failure("decrypt", "circuit_open", start.elapsed());
            return Err(CryptoError::CircuitOpen);
        }

        let request = DecryptRequest {
            ciphertext: encrypted.ciphertext.clone(),
            iv: encrypted.iv.clone(),
            tag: encrypted.tag.clone(),
            key_id: Some(encrypted.key_id.to_proto()),
            aad: aad.map(|a| a.to_vec()).unwrap_or_default(),
            correlation_id: correlation_id.to_string(),
        };

        let mut client = self.grpc_client.clone();
        match client.decrypt(request).await {
            Ok(response) => {
                self.circuit_breaker.record_success().await;
                self.metrics.record_success("decrypt", start.elapsed());
                self.metrics.set_fallback_active(false);

                Ok(response.into_inner().plaintext)
            }
            Err(status) => {
                self.circuit_breaker.record_failure().await;
                let error = CryptoError::from(status);
                self.metrics.record_failure("decrypt", "service_error", start.elapsed());
                Err(error)
            }
        }
    }

    /// Decrypts using local fallback
    fn decrypt_fallback(
        &self,
        encrypted: &EncryptedData,
        aad: Option<&[u8]>,
        start: Instant,
    ) -> Result<Vec<u8>, CryptoError> {
        let fallback = self.fallback.as_ref().ok_or(CryptoError::FallbackUnavailable)?;

        let result = fallback.decrypt(encrypted, aad)?;
        self.metrics.record_fallback("decrypt", start.elapsed());

        Ok(result)
    }

    /// Triggers key rotation
    ///
    /// # Errors
    ///
    /// Returns error if rotation fails
    #[instrument(skip(self), fields(correlation_id = %correlation_id))]
    pub async fn rotate_key(&self, correlation_id: &str) -> Result<KeyId, CryptoError> {
        let start = Instant::now();

        if !self.circuit_breaker.allow_request().await {
            // Queue for later
            if let Some(ref fallback) = self.fallback {
                fallback
                    .queue_operation(PendingOperation::KeyRotation {
                        correlation_id: correlation_id.to_string(),
                        requested_at: Instant::now(),
                    })
                    .await?;
            }
            return Err(CryptoError::CircuitOpen);
        }

        let current_key = self.key_manager.active_key();
        let request = RotateKeyRequest {
            key_id: Some(current_key.to_proto()),
            correlation_id: correlation_id.to_string(),
        };

        let mut client = self.grpc_client.clone();
        match client.rotate_key(request).await {
            Ok(response) => {
                self.circuit_breaker.record_success().await;
                let inner = response.into_inner();

                let new_key = inner
                    .new_key_id
                    .map(|k| KeyId::from_proto(&k))
                    .ok_or_else(|| CryptoError::rotation_failed("No new key ID in response"))?;

                self.key_manager.rotate(new_key.clone()).await?;
                self.metrics.increment_rotation();
                self.metrics.record_success("rotate_key", start.elapsed());

                info!(new_key = %new_key, "Key rotation completed");
                Ok(new_key)
            }
            Err(status) => {
                self.circuit_breaker.record_failure().await;
                let error = CryptoError::from(status);
                self.metrics.record_failure("rotate_key", "service_error", start.elapsed());
                Err(error)
            }
        }
    }

    /// Gets current key metadata
    ///
    /// # Errors
    ///
    /// Returns error if metadata retrieval fails
    pub async fn get_key_metadata(&self, correlation_id: &str) -> Result<KeyMetadata, CryptoError> {
        let key_id = self.key_manager.active_key();
        let request = GetKeyMetadataRequest {
            key_id: Some(key_id.to_proto()),
            correlation_id: correlation_id.to_string(),
        };

        let mut client = self.grpc_client.clone();
        let response = client.get_key_metadata(request).await?;
        let inner = response.into_inner();

        let metadata = inner
            .metadata
            .ok_or_else(|| CryptoError::key_not_found(key_id.to_string()))?;

        Ok(KeyMetadata {
            id: metadata.id.map(|k| KeyId::from_proto(&k)).unwrap_or(key_id),
            algorithm: format!("{:?}", metadata.algorithm),
            state: format!("{:?}", metadata.state),
            created_at: metadata.created_at,
            expires_at: metadata.expires_at,
            rotated_at: if metadata.rotated_at > 0 {
                Some(metadata.rotated_at)
            } else {
                None
            },
            previous_version: metadata.previous_version.map(|k| KeyId::from_proto(&k)),
        })
    }

    /// Checks if operating in fallback mode
    #[must_use]
    pub fn is_fallback_active(&self) -> bool {
        !self.circuit_breaker.is_closed()
    }

    /// Gets the key manager
    #[must_use]
    pub fn key_manager(&self) -> &KeyManager {
        &self.key_manager
    }

    /// Gets the metrics
    #[must_use]
    pub fn metrics(&self) -> &CryptoMetrics {
        &self.metrics
    }

    /// Builds AAD from namespace and key name
    #[must_use]
    pub fn build_aad(&self, key_name: &str) -> Vec<u8> {
        format!("{}:{}", self.config.key_namespace, key_name).into_bytes()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use url::Url;

    #[tokio::test]
    async fn test_build_aad() {
        let config = CryptoClientConfig::default()
            .with_key_namespace("test-ns");
        let client = CryptoClient::new(config).await.unwrap();

        let aad = client.build_aad("my-key");
        assert_eq!(aad, b"test-ns:my-key");
    }

    #[tokio::test]
    async fn test_invalid_config_rejected() {
        let config = CryptoClientConfig::default()
            .with_key_namespace("");

        let result = CryptoClient::new(config).await;
        assert!(matches!(result, Err(CryptoError::InvalidConfig { .. })));
    }
}
