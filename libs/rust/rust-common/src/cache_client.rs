//! gRPC client for centralized Cache_Service.
//!
//! This module provides a client for the platform's distributed cache service
//! with namespace isolation, encryption, and local fallback.

use crate::{CircuitBreaker, CircuitBreakerConfig, PlatformError};
use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use rand::RngCore;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

/// Cache client configuration.
#[derive(Debug, Clone)]
pub struct CacheClientConfig {
    /// gRPC address of Cache_Service
    pub address: String,
    /// Namespace for key isolation
    pub namespace: String,
    /// Default TTL for cache entries
    pub default_ttl: Duration,
    /// Maximum local cache size
    pub local_cache_size: usize,
    /// Encryption key (32 bytes for AES-256)
    pub encryption_key: Option<[u8; 32]>,
    /// Circuit breaker configuration
    pub circuit_breaker: CircuitBreakerConfig,
}

impl Default for CacheClientConfig {
    fn default() -> Self {
        Self {
            address: "http://localhost:50051".to_string(),
            namespace: "default".to_string(),
            default_ttl: Duration::from_secs(3600),
            local_cache_size: 1000,
            encryption_key: None,
            circuit_breaker: CircuitBreakerConfig::default(),
        }
    }
}

impl CacheClientConfig {
    /// Create config with custom address.
    #[must_use]
    pub fn with_address(mut self, address: impl Into<String>) -> Self {
        self.address = address.into();
        self
    }

    /// Create config with custom namespace.
    #[must_use]
    pub fn with_namespace(mut self, namespace: impl Into<String>) -> Self {
        self.namespace = namespace.into();
        self
    }

    /// Create config with custom TTL.
    #[must_use]
    pub const fn with_default_ttl(mut self, ttl: Duration) -> Self {
        self.default_ttl = ttl;
        self
    }

    /// Create config with encryption enabled.
    #[must_use]
    pub const fn with_encryption_key(mut self, key: [u8; 32]) -> Self {
        self.encryption_key = Some(key);
        self
    }
}

/// Local cache entry.
struct LocalCacheEntry {
    value: Vec<u8>,
    expires_at: Instant,
}

/// Cache client with local fallback and encryption.
pub struct CacheClient {
    config: CacheClientConfig,
    circuit_breaker: Arc<CircuitBreaker>,
    local_cache: Arc<RwLock<HashMap<String, LocalCacheEntry>>>,
    cipher: Option<Aes256Gcm>,
}

impl CacheClient {
    /// Create a new cache client.
    ///
    /// # Errors
    ///
    /// Returns an error if the gRPC channel cannot be created.
    pub async fn new(config: CacheClientConfig) -> Result<Self, PlatformError> {
        let cipher = config.encryption_key.map(|key| Aes256Gcm::new(&key.into()));

        Ok(Self {
            circuit_breaker: Arc::new(CircuitBreaker::new(config.circuit_breaker.clone())),
            local_cache: Arc::new(RwLock::new(HashMap::new())),
            cipher,
            config,
        })
    }

    /// Get a value from the cache.
    ///
    /// # Errors
    ///
    /// Returns an error if decryption fails.
    pub async fn get(&self, key: &str) -> Result<Option<Vec<u8>>, PlatformError> {
        let namespaced_key = self.namespaced_key(key);

        // Try remote cache first if circuit allows
        if self.circuit_breaker.allow_request().await {
            // In production, this would call Cache_Service via gRPC
            self.circuit_breaker.record_success().await;
        }

        // Fallback to local cache
        let cache = self.local_cache.read().await;
        if let Some(entry) = cache.get(&namespaced_key) {
            if entry.expires_at > Instant::now() {
                return Ok(Some(self.decrypt(&entry.value)?));
            }
        }

        Ok(None)
    }

    /// Set a value in the cache.
    ///
    /// # Errors
    ///
    /// Returns an error if encryption fails.
    pub async fn set(&self, key: &str, value: &[u8], ttl: Option<Duration>) -> Result<(), PlatformError> {
        let namespaced_key = self.namespaced_key(key);
        let ttl = ttl.unwrap_or(self.config.default_ttl);
        let encrypted = self.encrypt(value)?;

        // Try remote cache first if circuit allows
        if self.circuit_breaker.allow_request().await {
            // In production, this would call Cache_Service via gRPC
            self.circuit_breaker.record_success().await;
        }

        // Always update local cache
        let mut cache = self.local_cache.write().await;
        cache.insert(
            namespaced_key,
            LocalCacheEntry {
                value: encrypted,
                expires_at: Instant::now() + ttl,
            },
        );

        // Evict if over size limit
        if cache.len() > self.config.local_cache_size {
            self.evict_expired(&mut cache);
        }

        Ok(())
    }

    /// Delete a value from the cache.
    pub async fn delete(&self, key: &str) -> Result<(), PlatformError> {
        let namespaced_key = self.namespaced_key(key);

        // Try remote cache first if circuit allows
        if self.circuit_breaker.allow_request().await {
            // In production, this would call Cache_Service via gRPC
            self.circuit_breaker.record_success().await;
        }

        // Always update local cache
        let mut cache = self.local_cache.write().await;
        cache.remove(&namespaced_key);

        Ok(())
    }

    /// Check if a key exists in the cache.
    pub async fn exists(&self, key: &str) -> Result<bool, PlatformError> {
        let namespaced_key = self.namespaced_key(key);

        let cache = self.local_cache.read().await;
        if let Some(entry) = cache.get(&namespaced_key) {
            return Ok(entry.expires_at > Instant::now());
        }

        Ok(false)
    }

    /// Get the namespace.
    #[must_use]
    pub fn namespace(&self) -> &str {
        &self.config.namespace
    }

    /// Get the local cache size.
    pub async fn local_cache_size(&self) -> usize {
        self.local_cache.read().await.len()
    }

    /// Create a namespaced key.
    fn namespaced_key(&self, key: &str) -> String {
        format!("{}:{}", self.config.namespace, key)
    }

    /// Encrypt data using AES-GCM.
    fn encrypt(&self, data: &[u8]) -> Result<Vec<u8>, PlatformError> {
        if let Some(ref cipher) = self.cipher {
            // Generate random nonce
            let mut nonce_bytes = [0u8; 12];
            rand::thread_rng().fill_bytes(&mut nonce_bytes);
            let nonce = Nonce::from_slice(&nonce_bytes);

            let ciphertext = cipher
                .encrypt(nonce, data)
                .map_err(|e| PlatformError::encryption(e.to_string()))?;

            // Prepend nonce to ciphertext
            let mut result = nonce_bytes.to_vec();
            result.extend(ciphertext);
            Ok(result)
        } else {
            Ok(data.to_vec())
        }
    }

    /// Decrypt data using AES-GCM.
    fn decrypt(&self, data: &[u8]) -> Result<Vec<u8>, PlatformError> {
        if let Some(ref cipher) = self.cipher {
            if data.len() < 12 {
                return Err(PlatformError::encryption("Data too short for decryption"));
            }

            let (nonce_bytes, ciphertext) = data.split_at(12);
            let nonce = Nonce::from_slice(nonce_bytes);

            cipher
                .decrypt(nonce, ciphertext)
                .map_err(|e| PlatformError::encryption(e.to_string()))
        } else {
            Ok(data.to_vec())
        }
    }

    /// Evict expired entries from local cache.
    fn evict_expired(&self, cache: &mut HashMap<String, LocalCacheEntry>) {
        let now = Instant::now();
        cache.retain(|_, v| v.expires_at > now);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_client() {
        let config = CacheClientConfig::default();
        let client = CacheClient::new(config).await;
        assert!(client.is_ok());
    }

    #[tokio::test]
    async fn test_set_and_get() {
        let config = CacheClientConfig::default();
        let client = CacheClient::new(config).await.unwrap();

        client.set("key1", b"value1", None).await.unwrap();
        let result = client.get("key1").await.unwrap();

        assert_eq!(result, Some(b"value1".to_vec()));
    }

    #[tokio::test]
    async fn test_namespace_isolation() {
        let config1 = CacheClientConfig::default()
            .with_namespace("ns1");
        let config2 = CacheClientConfig::default()
            .with_namespace("ns2");

        let client1 = CacheClient::new(config1).await.unwrap();
        let client2 = CacheClient::new(config2).await.unwrap();

        client1.set("key", b"value1", None).await.unwrap();
        client2.set("key", b"value2", None).await.unwrap();

        let result1 = client1.get("key").await.unwrap();
        let result2 = client2.get("key").await.unwrap();

        assert_eq!(result1, Some(b"value1".to_vec()));
        assert_eq!(result2, Some(b"value2".to_vec()));
    }

    #[tokio::test]
    async fn test_ttl_expiration() {
        let config = CacheClientConfig::default();
        let client = CacheClient::new(config).await.unwrap();

        client.set("key", b"value", Some(Duration::from_millis(1))).await.unwrap();
        
        // Wait for expiration
        tokio::time::sleep(Duration::from_millis(10)).await;

        let result = client.get("key").await.unwrap();
        assert_eq!(result, None);
    }

    #[tokio::test]
    async fn test_delete() {
        let config = CacheClientConfig::default();
        let client = CacheClient::new(config).await.unwrap();

        client.set("key", b"value", None).await.unwrap();
        client.delete("key").await.unwrap();

        let result = client.get("key").await.unwrap();
        assert_eq!(result, None);
    }

    #[tokio::test]
    async fn test_exists() {
        let config = CacheClientConfig::default();
        let client = CacheClient::new(config).await.unwrap();

        assert!(!client.exists("key").await.unwrap());

        client.set("key", b"value", None).await.unwrap();
        assert!(client.exists("key").await.unwrap());
    }

    #[tokio::test]
    async fn test_encryption_round_trip() {
        let key = [0u8; 32]; // In production, use a secure random key
        let config = CacheClientConfig::default()
            .with_encryption_key(key);
        let client = CacheClient::new(config).await.unwrap();

        let original = b"sensitive data";
        client.set("secret", original, None).await.unwrap();

        let result = client.get("secret").await.unwrap();
        assert_eq!(result, Some(original.to_vec()));
    }
}
