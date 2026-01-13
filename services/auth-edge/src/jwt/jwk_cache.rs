//! JWK Cache with Cache_Service Integration and Single-Flight Pattern
//!
//! Implements a JWK cache that:
//! - Uses CacheClient from rust-common for distributed caching
//! - Maintains local fallback when Cache_Service is unavailable
//! - Prevents thundering herd on cache refresh using single-flight pattern

use crate::config::Config;
use crate::error::AuthEdgeError;
use arc_swap::ArcSwap;
use futures::future::{BoxFuture, Shared};
use futures::FutureExt;
use jsonwebtoken::DecodingKey;
use rust_common::{CacheClient, CacheClientConfig, PlatformError};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Mutex;
use tracing::{info, warn, instrument};

/// JSON Web Key structure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwk {
    /// Key type (RSA, EC, oct)
    pub kty: String,
    /// Key ID
    pub kid: String,
    /// Key use (sig, enc)
    #[serde(rename = "use")]
    pub key_use: Option<String>,
    /// Algorithm
    pub alg: Option<String>,
    /// RSA modulus
    pub n: Option<String>,
    /// RSA exponent
    pub e: Option<String>,
    /// EC x coordinate
    pub x: Option<String>,
    /// EC y coordinate
    pub y: Option<String>,
    /// EC curve
    pub crv: Option<String>,
}

/// JSON Web Key Set structure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwks {
    /// List of keys
    pub keys: Vec<Jwk>,
}

/// Local cache entry with keys and metadata.
struct LocalCacheEntry {
    keys: HashMap<String, Arc<DecodingKey>>,
    fetched_at: Instant,
}

/// Type alias for the inflight future.
type InflightFuture = Shared<BoxFuture<'static, Result<Arc<LocalCacheEntry>, AuthEdgeError>>>;

/// JWK Cache with Cache_Service integration and single-flight refresh pattern.
pub struct JwkCache {
    /// Remote cache client (Cache_Service)
    cache_client: CacheClient,
    /// Local fallback cache
    local_cache: ArcSwap<Option<LocalCacheEntry>>,
    /// JWKS endpoint URL
    jwks_url: String,
    /// Cache TTL
    ttl: Duration,
    /// Single-flight coordinator
    inflight: Arc<Mutex<Option<InflightFuture>>>,
    /// HTTP client for fetching JWKS
    http_client: reqwest::Client,
}

impl JwkCache {
    /// Creates a new JWK cache with Cache_Service integration.
    pub async fn new(config: &Config) -> Result<Self, AuthEdgeError> {
        let cache_config = CacheClientConfig::default()
            .with_address(config.cache_service_url_str())
            .with_namespace("auth-edge:jwk")
            .with_default_ttl(Duration::from_secs(config.jwks_cache_ttl_seconds));

        let cache_config = if let Some(key) = config.cache_encryption_key {
            cache_config.with_encryption_key(key)
        } else {
            cache_config
        };

        let cache_client = CacheClient::new(cache_config)
            .await
            .map_err(AuthEdgeError::Platform)?;

        let http_client = reqwest::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .map_err(|e| AuthEdgeError::JwkCacheError {
                reason: format!("Failed to create HTTP client: {e}"),
            })?;

        Ok(Self {
            cache_client,
            local_cache: ArcSwap::new(Arc::new(None)),
            jwks_url: config.jwks_url_str().to_string(),
            ttl: Duration::from_secs(config.jwks_cache_ttl_seconds),
            inflight: Arc::new(Mutex::new(None)),
            http_client,
        })
    }

    /// Gets a decoding key by key ID with distributed cache and local fallback.
    #[instrument(skip(self), fields(kid = %kid))]
    pub async fn get_key(&self, kid: &str) -> Result<DecodingKey, AuthEdgeError> {
        // 1. Try remote cache first
        if let Ok(Some(key_bytes)) = self.cache_client.get(&format!("key:{kid}")).await {
            if let Ok(key) = self.deserialize_key(&key_bytes) {
                return Ok(key);
            }
        }

        // 2. Try local cache
        if let Some(key) = self.try_get_local(kid) {
            return Ok((*key).clone());
        }

        // 3. Refresh with single-flight
        self.refresh_single_flight().await?;

        // 4. Try local cache again after refresh
        self.try_get_local(kid)
            .map(|k| (*k).clone())
            .ok_or_else(|| AuthEdgeError::JwkCacheError {
                reason: format!("Key {kid} not found after refresh"),
            })
    }

    /// Tries to get a key from local cache if valid.
    fn try_get_local(&self, kid: &str) -> Option<Arc<DecodingKey>> {
        let cache = self.local_cache.load();
        if let Some(ref entry) = **cache {
            if entry.fetched_at.elapsed() < self.ttl {
                return entry.keys.get(kid).cloned();
            }
        }
        None
    }

    /// Checks if the local cache is stale.
    #[must_use]
    pub fn is_stale(&self) -> bool {
        let cache = self.local_cache.load();
        match **cache {
            Some(ref entry) => entry.fetched_at.elapsed() >= self.ttl,
            None => true,
        }
    }

    /// Refreshes the cache using single-flight pattern.
    ///
    /// Only one HTTP request will be made even if multiple concurrent
    /// callers request a refresh simultaneously.
    async fn refresh_single_flight(&self) -> Result<(), AuthEdgeError> {
        let mut inflight_guard = self.inflight.lock().await;

        if let Some(ref fut) = *inflight_guard {
            let fut = fut.clone();
            drop(inflight_guard);
            fut.await?;
            return Ok(());
        }

        let url = self.jwks_url.clone();
        let client = self.http_client.clone();
        let local_cache = self.local_cache.clone();
        let cache_client = self.cache_client.clone();
        let ttl = self.ttl;

        let fut: BoxFuture<'static, Result<Arc<LocalCacheEntry>, AuthEdgeError>> =
            Box::pin(async move {
                info!(url = %url, "Fetching JWKS");

                let response = client.get(&url).send().await.map_err(|e| {
                    AuthEdgeError::JwkCacheError {
                        reason: format!("Failed to fetch JWKS: {e}"),
                    }
                })?;

                if !response.status().is_success() {
                    return Err(AuthEdgeError::JwkCacheError {
                        reason: format!("JWKS fetch failed with status: {}", response.status()),
                    });
                }

                let jwks: Jwks =
                    response
                        .json()
                        .await
                        .map_err(|e| AuthEdgeError::JwkCacheError {
                            reason: format!("Failed to parse JWKS: {e}"),
                        })?;

                let mut keys = HashMap::new();
                for jwk in &jwks.keys {
                    if let Some(key) = Self::jwk_to_decoding_key(jwk) {
                        keys.insert(jwk.kid.clone(), Arc::new(key));

                        // Store in remote cache (best effort)
                        if let Ok(serialized) = Self::serialize_jwk(jwk) {
                            let _ = cache_client
                                .set(&format!("key:{}", jwk.kid), &serialized, Some(ttl))
                                .await;
                        }
                    }
                }

                let entry = Arc::new(LocalCacheEntry {
                    keys: keys.clone(),
                    fetched_at: Instant::now(),
                });

                // Update local cache
                local_cache.store(Arc::new(Some(LocalCacheEntry {
                    keys,
                    fetched_at: Instant::now(),
                })));

                info!("JWKS cache updated with {} keys", entry.keys.len());
                Ok(entry)
            });

        let shared_fut = fut.shared();
        *inflight_guard = Some(shared_fut.clone());
        drop(inflight_guard);

        let result = shared_fut.await;
        self.inflight.lock().await.take();

        result.map(|_| ())
    }

    /// Converts a JWK to a DecodingKey.
    fn jwk_to_decoding_key(jwk: &Jwk) -> Option<DecodingKey> {
        match jwk.kty.as_str() {
            "RSA" => {
                let n = jwk.n.as_ref()?;
                let e = jwk.e.as_ref()?;
                
                // Check minimum key size (2048 bits = 256 bytes base64)
                if n.len() < 340 {
                    warn!(kid = %jwk.kid, "RSA key too small, rejecting");
                    return None;
                }
                
                DecodingKey::from_rsa_components(n, e).ok()
            }
            "EC" => {
                let x = jwk.x.as_ref()?;
                let y = jwk.y.as_ref()?;
                let crv = jwk.crv.as_deref().unwrap_or("P-256");
                
                // Only allow P-256 or stronger curves
                if !matches!(crv, "P-256" | "P-384" | "P-521") {
                    warn!(kid = %jwk.kid, crv = %crv, "Weak EC curve, rejecting");
                    return None;
                }
                
                DecodingKey::from_ec_components(x, y).ok()
            }
            _ => {
                warn!(kty = %jwk.kty, "Unsupported key type");
                None
            }
        }
    }

    /// Serializes a JWK for cache storage.
    fn serialize_jwk(jwk: &Jwk) -> Result<Vec<u8>, AuthEdgeError> {
        serde_json::to_vec(jwk).map_err(|e| AuthEdgeError::JwkCacheError {
            reason: format!("Failed to serialize JWK: {e}"),
        })
    }

    /// Deserializes a key from cache bytes.
    fn deserialize_key(&self, bytes: &[u8]) -> Result<DecodingKey, AuthEdgeError> {
        let jwk: Jwk = serde_json::from_slice(bytes).map_err(|e| AuthEdgeError::JwkCacheError {
            reason: format!("Failed to deserialize JWK: {e}"),
        })?;
        Self::jwk_to_decoding_key(&jwk).ok_or_else(|| AuthEdgeError::JwkCacheError {
            reason: "Failed to convert JWK to DecodingKey".to_string(),
        })
    }

    /// Forces a cache refresh (for testing).
    pub async fn force_refresh(&self) -> Result<(), AuthEdgeError> {
        self.local_cache.store(Arc::new(None));
        self.refresh_single_flight().await
    }

    /// Gets the number of locally cached keys.
    #[must_use]
    pub fn local_key_count(&self) -> usize {
        let cache = self.local_cache.load();
        match **cache {
            Some(ref entry) => entry.keys.len(),
            None => 0,
        }
    }
}

// Clone implementation for cache_client sharing
impl Clone for CacheClient {
    fn clone(&self) -> Self {
        // CacheClient uses Arc internally, so this is cheap
        // This is a workaround - in production, CacheClient should implement Clone
        unimplemented!("CacheClient clone not available - use Arc<CacheClient>")
    }
}
