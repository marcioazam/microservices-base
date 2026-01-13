//! Encrypted cache storage using CryptoEncryptor.
//!
//! Wraps CacheStorage with optional encryption via Crypto Service.

use crate::crypto::{CryptoEncryptor, KeyId};
use crate::error::TokenError;
use crate::refresh::family::TokenFamily;
use super::cache::CacheStorage;
use std::sync::Arc;
use std::time::Duration;
use tracing::{instrument, warn};

/// Cache storage with optional Crypto Service encryption.
pub struct EncryptedCacheStorage {
    /// Underlying cache storage
    cache: CacheStorage,
    /// Optional encryptor for token families
    encryptor: Option<Arc<CryptoEncryptor>>,
    /// Whether encryption is enabled
    encryption_enabled: bool,
}

impl EncryptedCacheStorage {
    /// Create new encrypted cache storage.
    pub fn new(cache: CacheStorage, encryptor: Option<Arc<CryptoEncryptor>>) -> Self {
        let encryption_enabled = encryptor.is_some();
        Self {
            cache,
            encryptor,
            encryption_enabled,
        }
    }

    /// Create without encryption (fallback mode).
    pub fn without_encryption(cache: CacheStorage) -> Self {
        Self {
            cache,
            encryptor: None,
            encryption_enabled: false,
        }
    }

    /// Check if encryption is enabled.
    #[must_use]
    pub fn is_encryption_enabled(&self) -> bool {
        self.encryption_enabled
    }

    /// Store a token family with optional encryption.
    #[instrument(skip(self, family), fields(family_id = %family.family_id))]
    pub async fn store_token_family(
        &self,
        family: &TokenFamily,
        ttl: Option<Duration>,
    ) -> Result<(), TokenError> {
        if let Some(ref encryptor) = self.encryptor {
            self.store_encrypted(family, ttl, encryptor).await
        } else {
            self.cache.store_token_family(family, ttl).await
        }
    }

    /// Store encrypted token family.
    async fn store_encrypted(
        &self,
        family: &TokenFamily,
        ttl: Option<Duration>,
        encryptor: &CryptoEncryptor,
    ) -> Result<(), TokenError> {
        let encrypted = encryptor.encrypt_token_family(family).await?;
        
        // Store encrypted data
        let key = format!("family:{}", family.family_id);
        self.cache
            .cache_client()
            .set(&key, &encrypted, ttl)
            .await
            .map_err(|e| TokenError::cache(e.to_string()))?;

        // Index by token hash
        let hash_key = format!("hash:{}", family.current_token_hash);
        self.cache
            .cache_client()
            .set(hash_key.as_str(), family.family_id.as_bytes(), ttl)
            .await
            .map_err(|e| TokenError::cache(e.to_string()))?;

        Ok(())
    }

    /// Get a token family by ID with optional decryption.
    #[instrument(skip(self), fields(family_id = %family_id))]
    pub async fn get_token_family(&self, family_id: &str) -> Result<Option<TokenFamily>, TokenError> {
        if let Some(ref encryptor) = self.encryptor {
            self.get_decrypted(family_id, encryptor).await
        } else {
            self.cache.get_token_family(family_id).await
        }
    }

    /// Get and decrypt token family.
    async fn get_decrypted(
        &self,
        family_id: &str,
        encryptor: &CryptoEncryptor,
    ) -> Result<Option<TokenFamily>, TokenError> {
        let key = format!("family:{}", family_id);

        match self.cache.cache_client().get(&key).await {
            Ok(Some(encrypted)) => {
                let family = encryptor.decrypt_token_family(&encrypted, family_id).await?;
                Ok(Some(family))
            }
            Ok(None) => Ok(None),
            Err(e) => Err(TokenError::cache(e.to_string())),
        }
    }

    /// Find token family by token hash.
    pub async fn find_family_by_token_hash(
        &self,
        token_hash: &str,
    ) -> Result<Option<TokenFamily>, TokenError> {
        let hash_key = format!("hash:{}", token_hash);

        match self.cache.cache_client().get(&hash_key).await {
            Ok(Some(data)) => {
                let family_id = String::from_utf8(data)
                    .map_err(|e| TokenError::internal(format!("Invalid family ID: {}", e)))?;
                self.get_token_family(&family_id).await
            }
            Ok(None) => Ok(None),
            Err(e) => Err(TokenError::cache(e.to_string())),
        }
    }

    /// Delegate to underlying cache for non-encrypted operations.
    pub async fn add_to_revocation_list(&self, jti: &str, ttl: Duration) -> Result<(), TokenError> {
        self.cache.add_to_revocation_list(jti, ttl).await
    }

    /// Check if token is revoked.
    pub async fn is_token_revoked(&self, jti: &str) -> Result<bool, TokenError> {
        self.cache.is_token_revoked(jti).await
    }

    /// Check and store DPoP JTI.
    pub async fn check_and_store_dpop_jti(&self, jti: &str, ttl: Duration) -> Result<bool, TokenError> {
        self.cache.check_and_store_dpop_jti(jti, ttl).await
    }

    /// Get underlying cache storage.
    #[must_use]
    pub fn inner(&self) -> &CacheStorage {
        &self.cache
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rust_common::CacheClientConfig;

    #[tokio::test]
    async fn test_without_encryption() {
        let config = CacheClientConfig::default().with_namespace("test-enc");
        let cache = CacheStorage::new(config).await.unwrap();
        let storage = EncryptedCacheStorage::without_encryption(cache);

        assert!(!storage.is_encryption_enabled());
    }

    #[tokio::test]
    async fn test_store_and_get_unencrypted() {
        let config = CacheClientConfig::default().with_namespace("test-enc-store");
        let cache = CacheStorage::new(config).await.unwrap();
        let storage = EncryptedCacheStorage::without_encryption(cache);

        let family = TokenFamily::new(
            "family-enc-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-enc-1".to_string(),
        );

        storage.store_token_family(&family, None).await.unwrap();
        let retrieved = storage.get_token_family("family-enc-1").await.unwrap();

        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().family_id, "family-enc-1");
    }
}
