//! CacheClient integration with CryptoClient
//!
//! Provides encrypted cache operations using the centralized crypto-service.

use std::sync::Arc;
use std::time::Duration;

use rust_common::{CacheClient, CacheClientConfig, PlatformError};
use tracing::{instrument, warn};

use crate::crypto::client::CryptoClient;
use crate::crypto::error::CryptoError;
use crate::crypto::fallback::EncryptedData;

/// Cache client wrapper that uses CryptoClient for encryption.
pub struct EncryptedCacheClient {
    /// Underlying cache client (without local encryption)
    cache: CacheClient,
    /// Crypto client for centralized encryption
    crypto: Arc<CryptoClient>,
    /// Namespace for AAD construction
    namespace: String,
}

impl EncryptedCacheClient {
    /// Creates a new EncryptedCacheClient.
    ///
    /// # Errors
    ///
    /// Returns error if cache client creation fails.
    pub async fn new(
        cache_config: CacheClientConfig,
        crypto: Arc<CryptoClient>,
    ) -> Result<Self, PlatformError> {
        let namespace = cache_config.namespace.clone();
        
        // Create cache without local encryption (crypto-service handles it)
        let config_no_enc = CacheClientConfig {
            encryption_key: None,
            ..cache_config
        };
        
        let cache = CacheClient::new(config_no_enc).await?;
        
        Ok(Self {
            cache,
            crypto,
            namespace,
        })
    }

    /// Gets a value from cache and decrypts it.
    ///
    /// # Errors
    ///
    /// Returns error if cache read or decryption fails.
    #[instrument(skip(self), fields(namespace = %self.namespace))]
    pub async fn get(&self, key: &str, correlation_id: &str) -> Result<Option<Vec<u8>>, CryptoError> {
        let cached = self.cache.get(key).await.map_err(|e| {
            CryptoError::service_unavailable(format!("Cache read failed: {e}"))
        })?;

        match cached {
            Some(data) => {
                let encrypted = self.deserialize_encrypted(&data)?;
                let aad = self.crypto.build_aad(key);
                let plaintext = self.crypto.decrypt(&encrypted, Some(&aad), correlation_id).await?;
                Ok(Some(plaintext))
            }
            None => Ok(None),
        }
    }

    /// Encrypts and stores a value in cache.
    ///
    /// # Errors
    ///
    /// Returns error if encryption or cache write fails.
    #[instrument(skip(self, value), fields(namespace = %self.namespace))]
    pub async fn set(
        &self,
        key: &str,
        value: &[u8],
        ttl: Option<Duration>,
        correlation_id: &str,
    ) -> Result<(), CryptoError> {
        let aad = self.crypto.build_aad(key);
        let encrypted = self.crypto.encrypt(value, Some(&aad), correlation_id).await?;
        let serialized = self.serialize_encrypted(&encrypted)?;

        self.cache.set(key, &serialized, ttl).await.map_err(|e| {
            CryptoError::service_unavailable(format!("Cache write failed: {e}"))
        })?;

        Ok(())
    }

    /// Deletes a value from cache.
    ///
    /// # Errors
    ///
    /// Returns error if cache delete fails.
    pub async fn delete(&self, key: &str) -> Result<(), CryptoError> {
        self.cache.delete(key).await.map_err(|e| {
            CryptoError::service_unavailable(format!("Cache delete failed: {e}"))
        })?;
        Ok(())
    }

    /// Checks if a key exists in cache.
    ///
    /// # Errors
    ///
    /// Returns error if cache check fails.
    pub async fn exists(&self, key: &str) -> Result<bool, CryptoError> {
        self.cache.exists(key).await.map_err(|e| {
            CryptoError::service_unavailable(format!("Cache exists check failed: {e}"))
        })
    }

    /// Gets the namespace.
    #[must_use]
    pub fn namespace(&self) -> &str {
        &self.namespace
    }

    /// Gets the underlying crypto client.
    #[must_use]
    pub fn crypto_client(&self) -> &CryptoClient {
        &self.crypto
    }

    /// Serializes encrypted data for storage.
    fn serialize_encrypted(&self, data: &EncryptedData) -> Result<Vec<u8>, CryptoError> {
        serde_json::to_vec(data).map_err(|e| {
            CryptoError::encryption_failed(format!("Serialization failed: {e}"))
        })
    }

    /// Deserializes encrypted data from storage.
    fn deserialize_encrypted(&self, data: &[u8]) -> Result<EncryptedData, CryptoError> {
        serde_json::from_slice(data).map_err(|e| {
            CryptoError::decryption_failed(format!("Deserialization failed: {e}"))
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Integration tests would require a running crypto-service
    // Unit tests focus on serialization/deserialization

    #[test]
    fn test_encrypted_data_serialization() {
        use crate::crypto::key_manager::KeyId;

        let data = EncryptedData {
            ciphertext: vec![1, 2, 3, 4],
            iv: vec![0; 12],
            tag: vec![0; 16],
            key_id: KeyId::new("test", "key", 1),
            algorithm: "AES-256-GCM".to_string(),
        };

        let serialized = serde_json::to_vec(&data).unwrap();
        let deserialized: EncryptedData = serde_json::from_slice(&serialized).unwrap();

        assert_eq!(data.ciphertext, deserialized.ciphertext);
        assert_eq!(data.iv, deserialized.iv);
        assert_eq!(data.tag, deserialized.tag);
        assert_eq!(data.key_id, deserialized.key_id);
    }
}
