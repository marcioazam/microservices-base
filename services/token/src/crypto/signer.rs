//! CryptoSigner - JWT signer using Crypto Service.

use super::client::CryptoClient;
use super::error::CryptoError;
use super::models::{KeyId, KeyMetadata, KeyState};
use crate::error::TokenError;
use crate::kms::KmsSigner;
use async_trait::async_trait;
use jsonwebtoken::EncodingKey;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use tracing::{info, warn};

/// Cached key metadata with timestamp.
struct CachedKeyMetadata {
    metadata: KeyMetadata,
    cached_at: Instant,
}

/// Crypto Service based JWT signer.
pub struct CryptoSigner {
    /// CryptoClient for signing operations
    client: Arc<dyn CryptoClient>,
    /// Signing key ID
    key_id: KeyId,
    /// Algorithm (PS256, ES256, etc.)
    algorithm: String,
    /// Cached key metadata
    cached_metadata: RwLock<Option<CachedKeyMetadata>>,
    /// Cache TTL
    cache_ttl: Duration,
}

impl CryptoSigner {
    /// Create a new CryptoSigner.
    #[must_use]
    pub fn new(client: Arc<dyn CryptoClient>, key_id: KeyId, algorithm: impl Into<String>) -> Self {
        Self {
            client,
            key_id,
            algorithm: algorithm.into(),
            cached_metadata: RwLock::new(None),
            cache_ttl: Duration::from_secs(300),
        }
    }

    /// Set cache TTL.
    #[must_use]
    pub fn with_cache_ttl(mut self, ttl: Duration) -> Self {
        self.cache_ttl = ttl;
        self
    }

    /// Verify key state before signing.
    async fn verify_key_state(&self) -> Result<(), TokenError> {
        let metadata = self.get_metadata().await?;

        if !metadata.state.can_sign() {
            return Err(TokenError::kms(format!(
                "Key {} is in state {:?} and cannot be used for signing",
                self.key_id.id, metadata.state
            )));
        }

        Ok(())
    }

    /// Get key metadata (cached).
    async fn get_metadata(&self) -> Result<KeyMetadata, TokenError> {
        // Check cache
        {
            let cache = self.cached_metadata.read().await;
            if let Some(ref cached) = *cache {
                if cached.cached_at.elapsed() < self.cache_ttl {
                    return Ok(cached.metadata.clone());
                }
            }
        }

        // Fetch from Crypto Service
        let metadata = self
            .client
            .get_key_metadata(&self.key_id)
            .await
            .map_err(|e| TokenError::kms(e.to_string()))?;

        // Update cache
        {
            let mut cache = self.cached_metadata.write().await;
            *cache = Some(CachedKeyMetadata {
                metadata: metadata.clone(),
                cached_at: Instant::now(),
            });
        }

        Ok(metadata)
    }

    /// Invalidate cached metadata.
    pub async fn invalidate_cache(&self) {
        let mut cache = self.cached_metadata.write().await;
        *cache = None;
    }

    /// Handle key rotation.
    pub async fn handle_key_rotation(&self, new_key_id: KeyId) -> Result<Self, TokenError> {
        info!(
            old_key = %self.key_id.id,
            new_key = %new_key_id.id,
            "Handling key rotation"
        );

        // Create new signer with new key
        let new_signer = Self::new(
            Arc::clone(&self.client),
            new_key_id,
            self.algorithm.clone(),
        )
        .with_cache_ttl(self.cache_ttl);

        // Verify new key is active
        let metadata = new_signer.get_metadata().await?;
        if metadata.state != KeyState::Active {
            return Err(TokenError::kms(format!(
                "New key is not active: {:?}",
                metadata.state
            )));
        }

        Ok(new_signer)
    }
}

#[async_trait]
impl KmsSigner for CryptoSigner {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        // Verify key state before signing
        self.verify_key_state().await?;

        let result = self
            .client
            .sign(data, &self.key_id)
            .await
            .map_err(|e| TokenError::signing(e.to_string()))?;

        Ok(result.signature)
    }

    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError> {
        // For Crypto Service, we don't have local access to private key
        Err(TokenError::kms(
            "Use sign() method for Crypto Service signing - no local key available",
        ))
    }

    fn key_id(&self) -> &str {
        &self.key_id.id
    }

    fn algorithm(&self) -> &str {
        &self.algorithm
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use super::super::models::{KeyAlgorithm, SignResult};
    use chrono::Utc;

    /// Mock CryptoClient for testing.
    struct MockCryptoClient {
        sign_result: Result<SignResult, CryptoError>,
        metadata: KeyMetadata,
    }

    impl MockCryptoClient {
        fn new_success() -> Self {
            Self {
                sign_result: Ok(SignResult {
                    signature: vec![1, 2, 3, 4],
                    key_id: KeyId::new("test", "key", 1),
                    algorithm: "PS256".to_string(),
                }),
                metadata: KeyMetadata {
                    id: KeyId::new("test", "key", 1),
                    algorithm: KeyAlgorithm::Rsa2048,
                    state: KeyState::Active,
                    created_at: Utc::now(),
                    expires_at: None,
                    rotated_at: None,
                    previous_version: None,
                    owner_service: "token-service".to_string(),
                    allowed_operations: vec!["sign".to_string()],
                    usage_count: 0,
                },
            }
        }

        fn with_state(mut self, state: KeyState) -> Self {
            self.metadata.state = state;
            self
        }
    }

    #[async_trait]
    impl CryptoClient for MockCryptoClient {
        async fn sign(&self, _data: &[u8], _key_id: &KeyId) -> Result<SignResult, CryptoError> {
            self.sign_result.clone()
        }

        async fn verify(
            &self,
            _data: &[u8],
            _signature: &[u8],
            _key_id: &KeyId,
        ) -> Result<bool, CryptoError> {
            Ok(true)
        }

        async fn encrypt(
            &self,
            _plaintext: &[u8],
            _key_id: &KeyId,
            _aad: Option<&[u8]>,
        ) -> Result<super::super::models::EncryptResult, CryptoError> {
            unimplemented!()
        }

        async fn decrypt(
            &self,
            _encrypted: &super::super::models::EncryptedData,
            _key_id: &KeyId,
            _aad: Option<&[u8]>,
        ) -> Result<Vec<u8>, CryptoError> {
            unimplemented!()
        }

        async fn generate_key(
            &self,
            _algorithm: KeyAlgorithm,
            _namespace: &str,
        ) -> Result<KeyId, CryptoError> {
            unimplemented!()
        }

        async fn rotate_key(
            &self,
            _key_id: &KeyId,
        ) -> Result<super::super::models::KeyRotationResult, CryptoError> {
            unimplemented!()
        }

        async fn get_key_metadata(&self, _key_id: &KeyId) -> Result<KeyMetadata, CryptoError> {
            Ok(self.metadata.clone())
        }
    }

    #[tokio::test]
    async fn test_sign_success() {
        let client = Arc::new(MockCryptoClient::new_success());
        let key_id = KeyId::new("test", "key", 1);
        let signer = CryptoSigner::new(client, key_id, "PS256");

        let result = signer.sign(b"test data").await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), vec![1, 2, 3, 4]);
    }

    #[tokio::test]
    async fn test_sign_deprecated_key() {
        let client = Arc::new(MockCryptoClient::new_success().with_state(KeyState::Deprecated));
        let key_id = KeyId::new("test", "key", 1);
        let signer = CryptoSigner::new(client, key_id, "PS256");

        let result = signer.sign(b"test data").await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_key_id_and_algorithm() {
        let client = Arc::new(MockCryptoClient::new_success());
        let key_id = KeyId::new("test", "my-key", 1);
        let signer = CryptoSigner::new(client, key_id, "ES256");

        assert_eq!(signer.key_id(), "my-key");
        assert_eq!(signer.algorithm(), "ES256");
    }

    #[tokio::test]
    async fn test_encoding_key_not_available() {
        let client = Arc::new(MockCryptoClient::new_success());
        let key_id = KeyId::new("test", "key", 1);
        let signer = CryptoSigner::new(client, key_id, "PS256");

        let result = signer.get_encoding_key();
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_metadata_caching() {
        let client = Arc::new(MockCryptoClient::new_success());
        let key_id = KeyId::new("test", "key", 1);
        let signer = CryptoSigner::new(client, key_id, "PS256");

        // First call fetches from client
        let metadata1 = signer.get_metadata().await.unwrap();
        assert_eq!(metadata1.state, KeyState::Active);

        // Second call uses cache
        let metadata2 = signer.get_metadata().await.unwrap();
        assert_eq!(metadata2.state, KeyState::Active);
    }

    #[tokio::test]
    async fn test_invalidate_cache() {
        let client = Arc::new(MockCryptoClient::new_success());
        let key_id = KeyId::new("test", "key", 1);
        let signer = CryptoSigner::new(client, key_id, "PS256");

        // Populate cache
        signer.get_metadata().await.unwrap();

        // Invalidate
        signer.invalidate_cache().await;

        // Cache should be empty
        let cache = signer.cached_metadata.read().await;
        assert!(cache.is_none());
    }
}
