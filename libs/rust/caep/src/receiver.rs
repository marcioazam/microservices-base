//! CAEP Receiver implementation.
//!
//! This module provides the receiver for processing incoming CAEP events using native async traits.

use crate::{CaepError, CaepEvent, CaepEventType, CaepResult, SecurityEventToken, SubjectIdentifier};
use jsonwebtoken::{decode, DecodingKey, Validation};
use rust_common::{CacheClient, CacheClientConfig};
use std::collections::HashMap;
use std::future::Future;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{error, info, instrument, warn};

/// CAEP Receiver trait for processing incoming security events.
///
/// Uses native async traits (Rust 2024).
pub trait CaepReceiver: Send + Sync {
    /// Process an incoming SET (JWT string).
    fn process_set(&self, set_jwt: &str) -> impl Future<Output = CaepResult<ProcessResult>> + Send;

    /// Validate SET signature against transmitter's JWKS.
    fn validate_signature(
        &self,
        set_jwt: &str,
    ) -> impl Future<Output = CaepResult<SecurityEventToken>> + Send;
}

/// Result of processing a SET.
#[derive(Debug, Clone)]
pub struct ProcessResult {
    /// Event ID from the SET
    pub event_id: String,
    /// Event type that was processed
    pub event_type: CaepEventType,
    /// Whether the event was successfully processed
    pub processed: bool,
    /// Processing time in milliseconds
    pub processing_time_ms: u64,
}

/// Callback trait for event handlers.
pub trait EventCallback: Send + Sync {
    /// Handle an event.
    fn on_event(&self, event: &CaepEvent) -> impl Future<Output = CaepResult<()>> + Send;
}

/// JWKS cache for signature validation.
pub struct JwksCache {
    keys: Arc<RwLock<HashMap<String, DecodingKey>>>,
    http_client: reqwest::Client,
    cache_client: Option<Arc<CacheClient>>,
}

impl JwksCache {
    /// Create a new JWKS cache.
    #[must_use]
    pub fn new() -> Self {
        Self {
            keys: Arc::new(RwLock::new(HashMap::new())),
            http_client: reqwest::Client::new(),
            cache_client: None,
        }
    }

    /// Create a JWKS cache with Cache_Service integration.
    pub async fn with_cache_service(config: CacheClientConfig) -> CaepResult<Self> {
        let cache_client = CacheClient::new(config).await?;
        Ok(Self {
            keys: Arc::new(RwLock::new(HashMap::new())),
            http_client: reqwest::Client::new(),
            cache_client: Some(Arc::new(cache_client)),
        })
    }

    /// Get a key from the cache.
    pub async fn get_key(&self, jwks_uri: &str, kid: &str) -> CaepResult<DecodingKey> {
        // Check local cache first
        {
            let keys = self.keys.read().await;
            if let Some(key) = keys.get(kid) {
                return Ok(key.clone());
            }
        }

        // Check distributed cache if available
        if let Some(ref cache) = self.cache_client {
            let cache_key = format!("jwks:{}:{}", jwks_uri, kid);
            if let Ok(Some(data)) = cache.get(&cache_key).await {
                // Deserialize and return (simplified)
                info!(kid = kid, "JWKS key found in distributed cache");
            }
        }

        // Fetch JWKS
        self.refresh_jwks(jwks_uri).await?;

        // Try again
        let keys = self.keys.read().await;
        keys.get(kid)
            .cloned()
            .ok_or_else(|| CaepError::JwksFetchError(format!("Key {} not found", kid)))
    }

    /// Refresh JWKS from the remote endpoint.
    async fn refresh_jwks(&self, jwks_uri: &str) -> CaepResult<()> {
        let response = self
            .http_client
            .get(jwks_uri)
            .send()
            .await
            .map_err(|e| CaepError::JwksFetchError(e.to_string()))?;

        let jwks: serde_json::Value = response
            .json()
            .await
            .map_err(|e| CaepError::JwksFetchError(e.to_string()))?;

        let mut keys = self.keys.write().await;

        if let Some(key_array) = jwks["keys"].as_array() {
            for key in key_array {
                if let (Some(kid), Some(kty)) = (key["kid"].as_str(), key["kty"].as_str()) {
                    let decoding_key = match kty {
                        "EC" => {
                            if let (Some(x), Some(y)) = (key["x"].as_str(), key["y"].as_str()) {
                                DecodingKey::from_ec_components(x, y)
                                    .map_err(|e| CaepError::JwksFetchError(e.to_string()))?
                            } else {
                                continue;
                            }
                        }
                        "RSA" => {
                            if let (Some(n), Some(e)) = (key["n"].as_str(), key["e"].as_str()) {
                                DecodingKey::from_rsa_components(n, e)
                                    .map_err(|e| CaepError::JwksFetchError(e.to_string()))?
                            } else {
                                continue;
                            }
                        }
                        _ => continue,
                    };
                    keys.insert(kid.to_string(), decoding_key);
                }
            }
        }

        Ok(())
    }

    /// Invalidate a key from the cache.
    pub async fn invalidate(&self, kid: &str) {
        let mut keys = self.keys.write().await;
        keys.remove(kid);
    }

    /// Clear all cached keys.
    pub async fn clear(&self) {
        let mut keys = self.keys.write().await;
        keys.clear();
    }
}

impl Default for JwksCache {
    fn default() -> Self {
        Self::new()
    }
}

/// Retry configuration for failed event processing.
#[derive(Debug, Clone)]
pub struct RetryConfig {
    /// Maximum number of retries
    pub max_retries: u32,
    /// Initial delay in milliseconds
    pub initial_delay_ms: u64,
    /// Maximum delay in milliseconds
    pub max_delay_ms: u64,
}

impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            max_retries: 3,
            initial_delay_ms: 100,
            max_delay_ms: 5000,
        }
    }
}

/// Boxed event callback for dynamic dispatch.
pub type BoxedCallback = Box<dyn DynEventCallback + Send + Sync>;

/// Dynamic event callback trait for type erasure.
pub trait DynEventCallback: Send + Sync {
    /// Handle an event.
    fn on_event_dyn(
        &self,
        event: &CaepEvent,
    ) -> std::pin::Pin<Box<dyn Future<Output = CaepResult<()>> + Send + '_>>;
}

/// Default CAEP Receiver implementation.
pub struct DefaultCaepReceiver {
    jwks_cache: JwksCache,
    jwks_uri: String,
    expected_issuer: String,
    expected_audience: String,
    handlers: HashMap<CaepEventType, Vec<BoxedCallback>>,
    retry_config: RetryConfig,
}

impl DefaultCaepReceiver {
    /// Create a new receiver.
    #[must_use]
    pub fn new(
        jwks_uri: impl Into<String>,
        expected_issuer: impl Into<String>,
        expected_audience: impl Into<String>,
    ) -> Self {
        Self {
            jwks_cache: JwksCache::new(),
            jwks_uri: jwks_uri.into(),
            expected_issuer: expected_issuer.into(),
            expected_audience: expected_audience.into(),
            handlers: HashMap::new(),
            retry_config: RetryConfig::default(),
        }
    }

    /// Set retry configuration.
    #[must_use]
    pub fn with_retry_config(mut self, config: RetryConfig) -> Self {
        self.retry_config = config;
        self
    }

    /// Set JWKS cache with Cache_Service integration.
    #[must_use]
    pub fn with_jwks_cache(mut self, cache: JwksCache) -> Self {
        self.jwks_cache = cache;
        self
    }

    /// Register a handler for an event type.
    pub fn register_handler(&mut self, event_type: CaepEventType, handler: BoxedCallback) {
        self.handlers
            .entry(event_type)
            .or_default()
            .push(handler);
    }

    /// Parse an event from a SET.
    fn parse_event_from_set(&self, set: &SecurityEventToken) -> CaepResult<CaepEvent> {
        for (event_uri, event_data) in &set.events {
            let event_type = match event_uri.as_str() {
                "https://schemas.openid.net/secevent/caep/event-type/session-revoked" => {
                    CaepEventType::SessionRevoked
                }
                "https://schemas.openid.net/secevent/caep/event-type/credential-change" => {
                    CaepEventType::CredentialChange
                }
                "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change" => {
                    CaepEventType::AssuranceLevelChange
                }
                "https://schemas.openid.net/secevent/caep/event-type/token-claims-change" => {
                    CaepEventType::TokenClaimsChange
                }
                "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change" => {
                    CaepEventType::DeviceComplianceChange
                }
                uri => return Err(CaepError::UnknownEventType(uri.to_string())),
            };

            let subject: SubjectIdentifier =
                serde_json::from_value(event_data["subject"].clone())
                    .map_err(|e| CaepError::invalid_set(e.to_string()))?;

            let event_timestamp = chrono::DateTime::from_timestamp(
                event_data["event_timestamp"].as_i64().unwrap_or(set.iat),
                0,
            )
            .unwrap_or_else(chrono::Utc::now);

            return Ok(CaepEvent {
                event_type,
                subject,
                event_timestamp,
                reason_admin: None,
                extra: event_data.clone(),
            });
        }

        Err(CaepError::invalid_set("No events in SET"))
    }

    /// Process an event with retry logic.
    async fn process_with_retry(&self, event: &CaepEvent) -> CaepResult<()> {
        let handlers = self.handlers.get(&event.event_type);

        if handlers.is_none() || handlers.is_some_and(Vec::is_empty) {
            warn!(event_type = ?event.event_type, "No handlers registered for event type");
            return Ok(());
        }

        let handlers = handlers.unwrap();
        let mut last_error = None;

        for attempt in 0..=self.retry_config.max_retries {
            if attempt > 0 {
                let delay = std::cmp::min(
                    self.retry_config.initial_delay_ms * 2u64.pow(attempt - 1),
                    self.retry_config.max_delay_ms,
                );
                tokio::time::sleep(tokio::time::Duration::from_millis(delay)).await;
            }

            let mut all_succeeded = true;
            for handler in handlers {
                if let Err(e) = handler.on_event_dyn(event).await {
                    error!(attempt = attempt, error = %e, "Handler failed");
                    last_error = Some(e);
                    all_succeeded = false;
                }
            }

            if all_succeeded {
                return Ok(());
            }
        }

        Err(last_error.unwrap_or_else(|| CaepError::processing("Unknown error")))
    }
}

impl CaepReceiver for DefaultCaepReceiver {
    #[instrument(skip(self, set_jwt))]
    async fn process_set(&self, set_jwt: &str) -> CaepResult<ProcessResult> {
        let start = std::time::Instant::now();

        // Validate and decode
        let set = self.validate_signature(set_jwt).await?;
        let event = self.parse_event_from_set(&set)?;

        info!(
            event_type = ?event.event_type,
            jti = %set.jti,
            "Processing CAEP event"
        );

        // Process with retry
        self.process_with_retry(&event).await?;

        Ok(ProcessResult {
            event_id: set.jti,
            event_type: event.event_type,
            processed: true,
            processing_time_ms: start.elapsed().as_millis() as u64,
        })
    }

    async fn validate_signature(&self, set_jwt: &str) -> CaepResult<SecurityEventToken> {
        // Decode header to get kid
        let header = jsonwebtoken::decode_header(set_jwt)
            .map_err(|e| CaepError::verification(e.to_string()))?;

        let kid = header
            .kid
            .ok_or_else(|| CaepError::verification("Missing kid in header"))?;

        // Get key from cache
        let key = self.jwks_cache.get_key(&self.jwks_uri, &kid).await?;

        // Validate
        let mut validation = Validation::new(header.alg);
        validation.set_issuer(&[&self.expected_issuer]);
        validation.set_audience(&[&self.expected_audience]);

        let token_data = decode::<SecurityEventToken>(set_jwt, &key, &validation)
            .map_err(|e| CaepError::verification(e.to_string()))?;

        Ok(token_data.claims)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_retry_config_default() {
        let config = RetryConfig::default();
        assert_eq!(config.max_retries, 3);
        assert_eq!(config.initial_delay_ms, 100);
        assert_eq!(config.max_delay_ms, 5000);
    }

    #[test]
    fn test_jwks_cache_creation() {
        let cache = JwksCache::new();
        assert!(cache.cache_client.is_none());
    }

    #[test]
    fn test_receiver_creation() {
        let receiver = DefaultCaepReceiver::new(
            "https://issuer.com/.well-known/jwks.json",
            "https://issuer.com",
            "https://receiver.com",
        );

        assert_eq!(receiver.jwks_uri, "https://issuer.com/.well-known/jwks.json");
        assert_eq!(receiver.expected_issuer, "https://issuer.com");
        assert_eq!(receiver.expected_audience, "https://receiver.com");
    }

    #[test]
    fn test_process_result() {
        let result = ProcessResult {
            event_id: "test-123".to_string(),
            event_type: CaepEventType::SessionRevoked,
            processed: true,
            processing_time_ms: 50,
        };

        assert!(result.processed);
        assert_eq!(result.processing_time_ms, 50);
    }
}
