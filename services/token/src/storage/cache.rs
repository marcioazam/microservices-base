//! Cache-based storage using platform CacheClient.
//!
//! Replaces direct Redis access with rust-common::CacheClient for
//! namespace isolation, encryption, and circuit breaker integration.

use crate::error::TokenError;
use crate::refresh::family::TokenFamily;
use rust_common::{CacheClient, CacheClientConfig};
use std::sync::Arc;
use std::time::Duration;

/// Storage implementation using platform CacheClient.
pub struct CacheStorage {
    cache: Arc<CacheClient>,
    default_ttl: Duration,
}

impl CacheStorage {
    /// Create new cache storage.
    ///
    /// # Errors
    ///
    /// Returns error if CacheClient initialization fails.
    pub async fn new(config: CacheClientConfig) -> Result<Self, TokenError> {
        let default_ttl = config.default_ttl;
        let cache = CacheClient::new(config)
            .await
            .map_err(|e| TokenError::cache(e.to_string()))?;

        Ok(Self {
            cache: Arc::new(cache),
            default_ttl,
        })
    }

    /// Store a token family.
    pub async fn store_token_family(
        &self,
        family: &TokenFamily,
        ttl: Option<Duration>,
    ) -> Result<(), TokenError> {
        let ttl = ttl.unwrap_or(self.default_ttl);
        let key = format!("family:{}", family.family_id);
        let value = serde_json::to_vec(family)
            .map_err(|e| TokenError::internal(format!("Serialization failed: {}", e)))?;

        self.cache
            .set(&key, &value, Some(ttl))
            .await
            .map_err(|e| TokenError::cache(e.to_string()))?;

        // Index by token hash for lookup
        let hash_key = format!("hash:{}", family.current_token_hash);
        self.cache
            .set(hash_key.as_str(), family.family_id.as_bytes(), Some(ttl))
            .await
            .map_err(|e| TokenError::cache(e.to_string()))?;

        // Index by user for revocation queries
        self.add_to_user_families(&family.user_id, &family.family_id, ttl)
            .await?;

        Ok(())
    }

    /// Get a token family by ID.
    pub async fn get_token_family(&self, family_id: &str) -> Result<Option<TokenFamily>, TokenError> {
        let key = format!("family:{}", family_id);

        match self.cache.get(&key).await {
            Ok(Some(data)) => {
                let family: TokenFamily = serde_json::from_slice(&data)
                    .map_err(|e| TokenError::internal(format!("Deserialization failed: {}", e)))?;
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

        match self.cache.get(&hash_key).await {
            Ok(Some(data)) => {
                let family_id = String::from_utf8(data)
                    .map_err(|e| TokenError::internal(format!("Invalid family ID: {}", e)))?;
                self.get_token_family(&family_id).await
            }
            Ok(None) => Ok(None),
            Err(e) => Err(TokenError::cache(e.to_string())),
        }
    }

    /// Get all token families for a user.
    pub async fn get_user_token_families(
        &self,
        user_id: &str,
    ) -> Result<Vec<TokenFamily>, TokenError> {
        let key = format!("user_families:{}", user_id);

        match self.cache.get(&key).await {
            Ok(Some(data)) => {
                let family_ids: Vec<String> = serde_json::from_slice(&data)
                    .map_err(|e| TokenError::internal(format!("Deserialization failed: {}", e)))?;

                let mut families = Vec::with_capacity(family_ids.len());
                for id in family_ids {
                    if let Some(family) = self.get_token_family(&id).await? {
                        families.push(family);
                    }
                }
                Ok(families)
            }
            Ok(None) => Ok(Vec::new()),
            Err(e) => Err(TokenError::cache(e.to_string())),
        }
    }

    /// Add JTI to revocation list.
    pub async fn add_to_revocation_list(
        &self,
        jti: &str,
        ttl: Duration,
    ) -> Result<(), TokenError> {
        let key = format!("revoked:{}", jti);
        self.cache
            .set(&key, b"1", Some(ttl))
            .await
            .map_err(|e| TokenError::cache(e.to_string()))
    }

    /// Check if token is revoked.
    pub async fn is_token_revoked(&self, jti: &str) -> Result<bool, TokenError> {
        let key = format!("revoked:{}", jti);
        self.cache
            .exists(&key)
            .await
            .map_err(|e| TokenError::cache(e.to_string()))
    }

    /// Check and store DPoP JTI for replay prevention.
    ///
    /// Returns true if JTI is new, false if already seen (replay).
    pub async fn check_and_store_dpop_jti(
        &self,
        jti: &str,
        ttl: Duration,
    ) -> Result<bool, TokenError> {
        let key = format!("dpop_jti:{}", jti);

        // Check if exists first
        let exists = self.cache
            .exists(&key)
            .await
            .map_err(|e| TokenError::cache(e.to_string()))?;

        if exists {
            return Ok(false); // Replay detected
        }

        // Store the JTI
        self.cache
            .set(&key, b"1", Some(ttl))
            .await
            .map_err(|e| TokenError::cache(e.to_string()))?;

        Ok(true)
    }

    /// Delete a key from cache.
    pub async fn delete(&self, key: &str) -> Result<(), TokenError> {
        self.cache
            .delete(key)
            .await
            .map_err(|e| TokenError::cache(e.to_string()))
    }

    /// Get the underlying cache client for advanced operations.
    #[must_use]
    pub fn cache_client(&self) -> &CacheClient {
        &self.cache
    }

    /// Add family ID to user's family list.
    async fn add_to_user_families(
        &self,
        user_id: &str,
        family_id: &str,
        ttl: Duration,
    ) -> Result<(), TokenError> {
        let key = format!("user_families:{}", user_id);

        // Get existing list or create new
        let mut family_ids: Vec<String> = match self.cache.get(&key).await {
            Ok(Some(data)) => serde_json::from_slice(&data).unwrap_or_default(),
            _ => Vec::new(),
        };

        // Add if not present
        if !family_ids.contains(&family_id.to_string()) {
            family_ids.push(family_id.to_string());
        }

        let value = serde_json::to_vec(&family_ids)
            .map_err(|e| TokenError::internal(format!("Serialization failed: {}", e)))?;

        self.cache
            .set(&key, &value, Some(ttl))
            .await
            .map_err(|e| TokenError::cache(e.to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_cache_storage() {
        let config = CacheClientConfig::default()
            .with_namespace("token-test");
        let storage = CacheStorage::new(config).await;
        assert!(storage.is_ok());
    }

    #[tokio::test]
    async fn test_store_and_get_family() {
        let config = CacheClientConfig::default()
            .with_namespace("token-test");
        let storage = CacheStorage::new(config).await.unwrap();

        let family = TokenFamily::new(
            "family-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-1".to_string(),
        );

        storage.store_token_family(&family, None).await.unwrap();

        let retrieved = storage.get_token_family("family-1").await.unwrap();
        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().family_id, "family-1");
    }

    #[tokio::test]
    async fn test_find_by_token_hash() {
        let config = CacheClientConfig::default()
            .with_namespace("token-test-hash");
        let storage = CacheStorage::new(config).await.unwrap();

        let family = TokenFamily::new(
            "family-2".to_string(),
            "user-2".to_string(),
            "session-2".to_string(),
            "unique-hash".to_string(),
        );

        storage.store_token_family(&family, None).await.unwrap();

        let found = storage.find_family_by_token_hash("unique-hash").await.unwrap();
        assert!(found.is_some());
        assert_eq!(found.unwrap().family_id, "family-2");
    }

    #[tokio::test]
    async fn test_dpop_jti_replay_detection() {
        let config = CacheClientConfig::default()
            .with_namespace("token-test-dpop");
        let storage = CacheStorage::new(config).await.unwrap();

        let jti = "test-jti-123";
        let ttl = Duration::from_secs(300);

        // First check should succeed
        let first = storage.check_and_store_dpop_jti(jti, ttl).await.unwrap();
        assert!(first);

        // Second check should fail (replay)
        let second = storage.check_and_store_dpop_jti(jti, ttl).await.unwrap();
        assert!(!second);
    }

    #[tokio::test]
    async fn test_revocation_list() {
        let config = CacheClientConfig::default()
            .with_namespace("token-test-revoke");
        let storage = CacheStorage::new(config).await.unwrap();

        let jti = "revoked-token-123";

        assert!(!storage.is_token_revoked(jti).await.unwrap());

        storage.add_to_revocation_list(jti, Duration::from_secs(3600)).await.unwrap();

        assert!(storage.is_token_revoked(jti).await.unwrap());
    }
}
