//! CAEP Receiver implementation.

use crate::{CaepError, CaepEvent, CaepEventType, SecurityEventToken, SubjectIdentifier};
use async_trait::async_trait;
use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{error, info, instrument, warn};

/// CAEP Receiver trait for processing incoming security events
#[async_trait]
pub trait CaepReceiver: Send + Sync {
    /// Process an incoming SET (JWT string)
    async fn process_set(&self, set_jwt: &str) -> Result<ProcessResult, CaepError>;

    /// Validate SET signature against transmitter's JWKS
    async fn validate_signature(&self, set_jwt: &str) -> Result<SecurityEventToken, CaepError>;

    /// Register a handler for an event type
    fn register_handler(&mut self, event_type: CaepEventType, handler: Box<dyn EventCallback>);
}

/// Result of processing a SET
#[derive(Debug)]
pub struct ProcessResult {
    pub event_id: String,
    pub event_type: CaepEventType,
    pub processed: bool,
    pub processing_time_ms: u64,
}

/// Callback trait for event handlers
#[async_trait]
pub trait EventCallback: Send + Sync {
    async fn on_event(&self, event: &CaepEvent) -> Result<(), CaepError>;
}

/// JWKS cache for signature validation
pub struct JwksCache {
    keys: Arc<RwLock<HashMap<String, DecodingKey>>>,
    http_client: reqwest::Client,
}

impl JwksCache {
    pub fn new() -> Self {
        Self {
            keys: Arc::new(RwLock::new(HashMap::new())),
            http_client: reqwest::Client::new(),
        }
    }

    pub async fn get_key(&self, jwks_uri: &str, kid: &str) -> Result<DecodingKey, CaepError> {
        // Check cache first
        {
            let keys = self.keys.read().await;
            if let Some(key) = keys.get(kid) {
                return Ok(key.clone());
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

    async fn refresh_jwks(&self, jwks_uri: &str) -> Result<(), CaepError> {
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
                            // Parse EC key
                            if let (Some(x), Some(y)) = (key["x"].as_str(), key["y"].as_str()) {
                                DecodingKey::from_ec_components(x, y)
                                    .map_err(|e| CaepError::JwksFetchError(e.to_string()))?
                            } else {
                                continue;
                            }
                        }
                        "RSA" => {
                            // Parse RSA key
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
}

impl Default for JwksCache {
    fn default() -> Self {
        Self::new()
    }
}

/// Default CAEP Receiver implementation
pub struct DefaultCaepReceiver {
    jwks_cache: JwksCache,
    jwks_uri: String,
    expected_issuer: String,
    expected_audience: String,
    handlers: HashMap<CaepEventType, Vec<Box<dyn EventCallback>>>,
    retry_config: RetryConfig,
}

/// Retry configuration for failed event processing
#[derive(Debug, Clone)]
pub struct RetryConfig {
    pub max_retries: u32,
    pub initial_delay_ms: u64,
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

impl DefaultCaepReceiver {
    pub fn new(jwks_uri: String, expected_issuer: String, expected_audience: String) -> Self {
        Self {
            jwks_cache: JwksCache::new(),
            jwks_uri,
            expected_issuer,
            expected_audience,
            handlers: HashMap::new(),
            retry_config: RetryConfig::default(),
        }
    }

    pub fn with_retry_config(mut self, config: RetryConfig) -> Self {
        self.retry_config = config;
        self
    }

    fn parse_event_from_set(&self, set: &SecurityEventToken) -> Result<CaepEvent, CaepError> {
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
                uri => return Err(CaepError::UnknownEventType(uri.to_string())),
            };

            let subject: SubjectIdentifier = serde_json::from_value(
                event_data["subject"].clone(),
            )
            .map_err(|e| CaepError::InvalidSet(e.to_string()))?;

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

        Err(CaepError::InvalidSet("No events in SET".to_string()))
    }

    async fn process_with_retry(&self, event: &CaepEvent) -> Result<(), CaepError> {
        let handlers = self.handlers.get(&event.event_type);
        
        if handlers.is_none() || handlers.unwrap().is_empty() {
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
                if let Err(e) = handler.on_event(event).await {
                    error!(
                        attempt = attempt,
                        error = %e,
                        "Handler failed"
                    );
                    last_error = Some(e);
                    all_succeeded = false;
                }
            }

            if all_succeeded {
                return Ok(());
            }
        }

        Err(last_error.unwrap_or_else(|| CaepError::ProcessingError("Unknown error".to_string())))
    }
}

#[async_trait]
impl CaepReceiver for DefaultCaepReceiver {
    #[instrument(skip(self, set_jwt))]
    async fn process_set(&self, set_jwt: &str) -> Result<ProcessResult, CaepError> {
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

    async fn validate_signature(&self, set_jwt: &str) -> Result<SecurityEventToken, CaepError> {
        // Decode header to get kid
        let header = jsonwebtoken::decode_header(set_jwt)
            .map_err(|e| CaepError::VerificationError(e.to_string()))?;

        let kid = header
            .kid
            .ok_or_else(|| CaepError::VerificationError("Missing kid in header".to_string()))?;

        // Get key from cache
        let key = self.jwks_cache.get_key(&self.jwks_uri, &kid).await?;

        // Validate
        let mut validation = Validation::new(header.alg);
        validation.set_issuer(&[&self.expected_issuer]);
        validation.set_audience(&[&self.expected_audience]);

        let token_data = decode::<SecurityEventToken>(set_jwt, &key, &validation)
            .map_err(|e| CaepError::VerificationError(e.to_string()))?;

        Ok(token_data.claims)
    }

    fn register_handler(&mut self, event_type: CaepEventType, handler: Box<dyn EventCallback>) {
        self.handlers
            .entry(event_type)
            .or_insert_with(Vec::new)
            .push(handler);
    }
}
