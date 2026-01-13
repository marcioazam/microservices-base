//! Key Manager for KEK/DEK lifecycle management
//!
//! Manages encryption keys with support for rotation and fallback.

use arc_swap::ArcSwap;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use tracing::{info, warn};

use crate::crypto::error::CryptoError;
use crate::crypto::proto::{
    crypto_service_client::CryptoServiceClient, GenerateKeyRequest, GetKeyMetadataRequest,
    KeyAlgorithm,
};

/// Key identifier matching crypto-service proto
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct KeyId {
    /// Namespace for key isolation
    pub namespace: String,
    /// Unique key identifier
    pub id: String,
    /// Key version (increments on rotation)
    pub version: u32,
}

impl KeyId {
    /// Creates a new KeyId
    #[must_use]
    pub fn new(namespace: impl Into<String>, id: impl Into<String>, version: u32) -> Self {
        Self {
            namespace: namespace.into(),
            id: id.into(),
            version,
        }
    }

    /// Converts to proto KeyId
    #[must_use]
    pub fn to_proto(&self) -> crate::crypto::proto::KeyId {
        crate::crypto::proto::KeyId {
            namespace: self.namespace.clone(),
            id: self.id.clone(),
            version: self.version,
        }
    }

    /// Creates from proto KeyId
    #[must_use]
    pub fn from_proto(proto: &crate::crypto::proto::KeyId) -> Self {
        Self {
            namespace: proto.namespace.clone(),
            id: proto.id.clone(),
            version: proto.version,
        }
    }
}

impl std::fmt::Display for KeyId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}:{}:v{}", self.namespace, self.id, self.version)
    }
}

/// Key metadata from crypto-service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyMetadata {
    /// Key identifier
    pub id: KeyId,
    /// Algorithm used
    pub algorithm: String,
    /// Key state (active, deprecated, etc.)
    pub state: String,
    /// Creation timestamp (Unix)
    pub created_at: i64,
    /// Expiration timestamp (Unix)
    pub expires_at: i64,
    /// Last rotation timestamp (Unix)
    pub rotated_at: Option<i64>,
    /// Previous key version (if rotated)
    pub previous_version: Option<KeyId>,
}

/// Cached DEK for fallback mode
struct CachedDek {
    /// Encrypted DEK (encrypted with local master key)
    encrypted_dek: Vec<u8>,
    /// When the DEK was cached
    cached_at: Instant,
    /// Key version this DEK belongs to
    key_version: u32,
}

/// Manages encryption key lifecycle
pub struct KeyManager {
    /// Current active key ID
    active_key: ArcSwap<KeyId>,
    /// Previous key IDs within rotation window
    previous_keys: Arc<RwLock<Vec<KeyId>>>,
    /// Cached DEK for fallback mode
    cached_dek: Arc<RwLock<Option<CachedDek>>>,
    /// Rotation window duration
    rotation_window: Duration,
    /// Key namespace
    namespace: String,
}

impl KeyManager {
    /// Creates a new KeyManager (without initialization)
    #[must_use]
    pub fn new(namespace: impl Into<String>, rotation_window: Duration) -> Self {
        let ns = namespace.into();
        Self {
            active_key: ArcSwap::new(Arc::new(KeyId::new(&ns, "uninitialized", 0))),
            previous_keys: Arc::new(RwLock::new(Vec::new())),
            cached_dek: Arc::new(RwLock::new(None)),
            rotation_window,
            namespace: ns,
        }
    }

    /// Initializes the KeyManager by requesting/creating a KEK from crypto-service
    ///
    /// # Errors
    ///
    /// Returns error if key creation fails
    pub async fn initialize<T>(
        &self,
        client: &mut CryptoServiceClient<T>,
        correlation_id: &str,
    ) -> Result<KeyId, CryptoError>
    where
        T: tonic::client::GrpcService<tonic::body::BoxBody> + Clone + Send + 'static,
        T::ResponseBody: tonic::codegen::Body<Data = tonic::codegen::Bytes> + Send + 'static,
        <T::ResponseBody as tonic::codegen::Body>::Error:
            Into<tonic::codegen::StdError> + Send,
        T::Future: Send,
    {
        info!(
            namespace = %self.namespace,
            correlation_id = %correlation_id,
            "Initializing KeyManager"
        );

        // Try to get existing key metadata first
        let key_id = KeyId::new(&self.namespace, "cache-kek", 1);
        let get_request = GetKeyMetadataRequest {
            key_id: Some(key_id.to_proto()),
            correlation_id: correlation_id.to_string(),
        };

        match client.get_key_metadata(get_request).await {
            Ok(response) => {
                if let Some(metadata) = response.into_inner().metadata {
                    if let Some(proto_id) = metadata.id {
                        let existing_key = KeyId::from_proto(&proto_id);
                        info!(key_id = %existing_key, "Using existing KEK");
                        self.active_key.store(Arc::new(existing_key.clone()));
                        return Ok(existing_key);
                    }
                }
            }
            Err(status) if status.code() == tonic::Code::NotFound => {
                // Key doesn't exist, create it
                info!(namespace = %self.namespace, "KEK not found, creating new one");
            }
            Err(e) => {
                return Err(CryptoError::from(e));
            }
        }

        // Create new key
        let create_request = GenerateKeyRequest {
            algorithm: KeyAlgorithm::Aes256Gcm as i32,
            namespace: self.namespace.clone(),
            metadata: std::collections::HashMap::from([
                ("purpose".to_string(), "cache-encryption".to_string()),
                ("service".to_string(), "auth-edge".to_string()),
            ]),
            correlation_id: correlation_id.to_string(),
        };

        let response = client
            .generate_key(create_request)
            .await
            .map_err(CryptoError::from)?;

        let inner = response.into_inner();
        let new_key = inner
            .key_id
            .map(|k| KeyId::from_proto(&k))
            .ok_or_else(|| CryptoError::encryption_failed("No key ID in response"))?;

        info!(key_id = %new_key, "Created new KEK");
        self.active_key.store(Arc::new(new_key.clone()));

        Ok(new_key)
    }

    /// Gets the current active key ID
    #[must_use]
    pub fn active_key(&self) -> KeyId {
        (**self.active_key.load()).clone()
    }

    /// Rotates to a new key version
    ///
    /// # Errors
    ///
    /// Returns error if rotation fails
    pub async fn rotate(&self, new_key: KeyId) -> Result<(), CryptoError> {
        let old_key = self.active_key();

        // Add old key to previous keys
        {
            let mut previous = self.previous_keys.write().await;
            previous.push(old_key.clone());

            // Clean up keys outside rotation window
            // In production, this would check timestamps
            if previous.len() > 5 {
                previous.remove(0);
            }
        }

        // Update active key
        self.active_key.store(Arc::new(new_key.clone()));

        info!(
            old_key = %old_key,
            new_key = %new_key,
            "Key rotated successfully"
        );

        Ok(())
    }

    /// Checks if a key ID is valid (current or within rotation window)
    pub async fn is_valid_key(&self, key_id: &KeyId) -> bool {
        // Check if it's the active key
        if *key_id == self.active_key() {
            return true;
        }

        // Check if it's in the rotation window
        let previous = self.previous_keys.read().await;
        previous.contains(key_id)
    }

    /// Gets the cached DEK for fallback mode
    pub async fn get_fallback_dek(&self) -> Option<Vec<u8>> {
        let cached = self.cached_dek.read().await;
        cached.as_ref().map(|c| c.encrypted_dek.clone())
    }

    /// Caches a DEK for fallback mode
    ///
    /// # Errors
    ///
    /// Returns error if caching fails
    pub async fn cache_dek(&self, encrypted_dek: Vec<u8>, key_version: u32) -> Result<(), CryptoError> {
        let mut cached = self.cached_dek.write().await;
        *cached = Some(CachedDek {
            encrypted_dek,
            cached_at: Instant::now(),
            key_version,
        });

        info!(key_version = key_version, "DEK cached for fallback mode");
        Ok(())
    }

    /// Checks if the cached DEK is still valid
    pub async fn is_dek_cache_valid(&self, max_age: Duration) -> bool {
        let cached = self.cached_dek.read().await;
        match cached.as_ref() {
            Some(c) => c.cached_at.elapsed() < max_age,
            None => false,
        }
    }

    /// Gets the namespace
    #[must_use]
    pub fn namespace(&self) -> &str {
        &self.namespace
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_key_id_display() {
        let key = KeyId::new("auth-edge", "cache-kek", 1);
        assert_eq!(key.to_string(), "auth-edge:cache-kek:v1");
    }

    #[test]
    fn test_key_id_equality() {
        let key1 = KeyId::new("ns", "id", 1);
        let key2 = KeyId::new("ns", "id", 1);
        let key3 = KeyId::new("ns", "id", 2);

        assert_eq!(key1, key2);
        assert_ne!(key1, key3);
    }

    #[tokio::test]
    async fn test_key_manager_rotation() {
        let manager = KeyManager::new("test", Duration::from_secs(3600));

        let key1 = KeyId::new("test", "key", 1);
        let key2 = KeyId::new("test", "key", 2);

        // Manually set active key
        manager.active_key.store(Arc::new(key1.clone()));

        // Rotate
        manager.rotate(key2.clone()).await.unwrap();

        // Check active key changed
        assert_eq!(manager.active_key(), key2);

        // Check old key is still valid
        assert!(manager.is_valid_key(&key1).await);
        assert!(manager.is_valid_key(&key2).await);
    }

    #[tokio::test]
    async fn test_dek_caching() {
        let manager = KeyManager::new("test", Duration::from_secs(3600));

        // Initially no DEK
        assert!(manager.get_fallback_dek().await.is_none());

        // Cache DEK
        let dek = vec![1, 2, 3, 4];
        manager.cache_dek(dek.clone(), 1).await.unwrap();

        // Retrieve DEK
        let cached = manager.get_fallback_dek().await;
        assert_eq!(cached, Some(dek));

        // Check validity
        assert!(manager.is_dek_cache_valid(Duration::from_secs(60)).await);
    }
}
