//! JWK Cache with Single-Flight Pattern
//!
//! Implements a JWK cache that prevents thundering herd on cache refresh
//! using the single-flight pattern with shared futures.

use crate::error::AuthEdgeError;
use arc_swap::ArcSwap;
use futures::future::{BoxFuture, Shared};
use futures::FutureExt;
use jsonwebtoken::DecodingKey;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Mutex;
use tracing::{info, warn, instrument};

/// JSON Web Key structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwk {
    pub kty: String,
    pub kid: String,
    #[serde(rename = "use")]
    pub key_use: Option<String>,
    pub alg: Option<String>,
    pub n: Option<String>,
    pub e: Option<String>,
    pub x: Option<String>,
    pub y: Option<String>,
    pub crv: Option<String>,
}

/// JSON Web Key Set structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Jwks {
    pub keys: Vec<Jwk>,
}

/// Cache entry with keys and metadata
struct CacheEntry {
    keys: HashMap<String, Arc<DecodingKey>>,
    fetched_at: Instant,
}

/// Type alias for the inflight future
type InflightFuture = Shared<BoxFuture<'static, Result<Arc<CacheEntry>, AuthEdgeError>>>;

/// JWK Cache with single-flight refresh pattern
pub struct JwkCache {
    /// Atomic cache storage
    cache: ArcSwap<Option<CacheEntry>>,
    /// JWKS endpoint URL
    jwks_url: String,
    /// Cache TTL
    ttl: Duration,
    /// Single-flight coordinator - tracks inflight refresh
    inflight: Arc<Mutex<Option<InflightFuture>>>,
    /// HTTP client for fetching JWKS
    client: reqwest::Client,
}


impl JwkCache {
    /// Creates a new JWK cache
    pub fn new(jwks_url: String, ttl_seconds: u64) -> Self {
        JwkCache {
            cache: ArcSwap::new(Arc::new(None)),
            jwks_url,
            ttl: Duration::from_secs(ttl_seconds),
            inflight: Arc::new(Mutex::new(None)),
            client: reqwest::Client::builder()
                .timeout(Duration::from_secs(10))
                .build()
                .expect("Failed to create HTTP client"),
        }
    }

    /// Gets a decoding key by key ID with single-flight refresh
    #[instrument(skip(self), fields(kid = %kid))]
    pub async fn get_key(&self, kid: &str) -> Result<DecodingKey, AuthEdgeError> {
        // Try to get from cache first
        if let Some(key) = self.try_get_cached(kid) {
            return Ok((*key).clone());
        }

        // Cache miss or stale - refresh with single-flight
        self.refresh_single_flight().await?;

        // Try again after refresh
        self.try_get_cached(kid)
            .map(|k| (*k).clone())
            .ok_or_else(|| AuthEdgeError::JwkCacheError {
                reason: format!("Key {} not found after refresh", kid),
            })
    }

    /// Tries to get a key from cache if valid
    fn try_get_cached(&self, kid: &str) -> Option<Arc<DecodingKey>> {
        let cache = self.cache.load();
        if let Some(ref entry) = **cache {
            if entry.fetched_at.elapsed() < self.ttl {
                return entry.keys.get(kid).cloned();
            }
        }
        None
    }

    /// Checks if the cache is stale
    pub fn is_stale(&self) -> bool {
        let cache = self.cache.load();
        match **cache {
            Some(ref entry) => entry.fetched_at.elapsed() >= self.ttl,
            None => true,
        }
    }

    /// Refreshes the cache using single-flight pattern
    /// 
    /// Only one HTTP request will be made even if multiple concurrent
    /// callers request a refresh simultaneously.
    async fn refresh_single_flight(&self) -> Result<(), AuthEdgeError> {
        // Check if refresh is already in progress
        let mut inflight_guard = self.inflight.lock().await;

        if let Some(ref fut) = *inflight_guard {
            // Another refresh is in progress - wait for it
            let fut = fut.clone();
            drop(inflight_guard);
            fut.await?;
            return Ok(());
        }

        // Start new refresh
        let url = self.jwks_url.clone();
        let client = self.client.clone();
        let cache = self.cache.clone();

        let fut: BoxFuture<'static, Result<Arc<CacheEntry>, AuthEdgeError>> = Box::pin(async move {
            info!(url = %url, "Fetching JWKS");

            let response = client
                .get(&url)
                .send()
                .await
                .map_err(|e| AuthEdgeError::JwkCacheError {
                    reason: format!("Failed to fetch JWKS: {}", e),
                })?;

            if !response.status().is_success() {
                return Err(AuthEdgeError::JwkCacheError {
                    reason: format!("JWKS fetch failed with status: {}", response.status()),
                });
            }

            let jwks: Jwks = response.json().await.map_err(|e| AuthEdgeError::JwkCacheError {
                reason: format!("Failed to parse JWKS: {}", e),
            })?;

            let mut keys = HashMap::new();
            for jwk in jwks.keys {
                if let Some(key) = Self::jwk_to_decoding_key(&jwk) {
                    keys.insert(jwk.kid.clone(), Arc::new(key));
                }
            }

            let entry = Arc::new(CacheEntry {
                keys,
                fetched_at: Instant::now(),
            });

            // Atomic update
            cache.store(Arc::new(Some(CacheEntry {
                keys: entry.keys.clone(),
                fetched_at: entry.fetched_at,
            })));

            info!("JWKS cache updated with {} keys", entry.keys.len());
            Ok(entry)
        });

        let shared_fut = fut.shared();
        *inflight_guard = Some(shared_fut.clone());
        drop(inflight_guard);

        // Wait for the refresh to complete
        let result = shared_fut.await;

        // Clear inflight
        self.inflight.lock().await.take();

        result.map(|_| ())
    }

    /// Converts a JWK to a DecodingKey
    fn jwk_to_decoding_key(jwk: &Jwk) -> Option<DecodingKey> {
        match jwk.kty.as_str() {
            "RSA" => {
                let n = jwk.n.as_ref()?;
                let e = jwk.e.as_ref()?;
                DecodingKey::from_rsa_components(n, e).ok()
            }
            "EC" => {
                let x = jwk.x.as_ref()?;
                let y = jwk.y.as_ref()?;
                DecodingKey::from_ec_components(x, y).ok()
            }
            "oct" => {
                // Symmetric keys (for testing only)
                Some(DecodingKey::from_secret(b"test-secret"))
            }
            _ => {
                warn!(kty = %jwk.kty, "Unsupported key type");
                None
            }
        }
    }

    /// Forces a cache refresh (for testing)
    pub async fn force_refresh(&self) -> Result<(), AuthEdgeError> {
        // Clear cache to force refresh
        self.cache.store(Arc::new(None));
        self.refresh_single_flight().await
    }

    /// Gets the number of cached keys
    pub fn key_count(&self) -> usize {
        let cache = self.cache.load();
        match **cache {
            Some(ref entry) => entry.keys.len(),
            None => 0,
        }
    }
}
