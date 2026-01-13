//! CryptoClient implementation for Crypto Service integration.

use super::config::CryptoClientConfig;
use super::error::CryptoError;
use super::fallback::FallbackHandler;
use super::metrics::CryptoMetrics;
use super::models::{
    EncryptResult, EncryptedData, KeyAlgorithm, KeyId, KeyMetadata, KeyRotationResult, SignResult,
};
use super::proto::{
    crypto_service_client::CryptoServiceClient, DecryptRequest, EncryptRequest,
    GenerateKeyRequest, GetKeyMetadataRequest, HashAlgorithm, RotateKeyRequest, SignRequest,
    VerifyRequest,
};
use async_trait::async_trait;
use governor::{Quota, RateLimiter as GovRateLimiter};
use lru::LruCache;
use rust_common::CircuitBreaker;
use std::num::NonZeroU32;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::Instant;
use tokio::sync::RwLock;
use tonic::transport::Channel;
use tracing::{error, info, instrument, warn};
use uuid::Uuid;

/// CryptoClient trait for cryptographic operations.
#[async_trait]
pub trait CryptoClient: Send + Sync {
    async fn sign(&self, data: &[u8], key_id: &KeyId) -> Result<SignResult, CryptoError>;
    async fn verify(&self, data: &[u8], signature: &[u8], key_id: &KeyId) -> Result<bool, CryptoError>;
    async fn encrypt(&self, plaintext: &[u8], key_id: &KeyId, aad: Option<&[u8]>) -> Result<EncryptResult, CryptoError>;
    async fn decrypt(&self, encrypted: &EncryptedData, key_id: &KeyId, aad: Option<&[u8]>) -> Result<Vec<u8>, CryptoError>;
    async fn generate_key(&self, algorithm: KeyAlgorithm, namespace: &str) -> Result<KeyId, CryptoError>;
    async fn rotate_key(&self, key_id: &KeyId) -> Result<KeyRotationResult, CryptoError>;
    async fn get_key_metadata(&self, key_id: &KeyId) -> Result<KeyMetadata, CryptoError>;
}

struct CachedMetadata { metadata: KeyMetadata, cached_at: Instant }

type RateLimiter = GovRateLimiter<governor::state::NotKeyed, governor::state::InMemoryState, governor::clock::DefaultClock>;

/// CryptoClient implementation with circuit breaker and fallback.
pub struct CryptoClientCore {
    grpc_client: RwLock<Option<CryptoServiceClient<Channel>>>,
    circuit_breaker: Arc<CircuitBreaker>,
    rate_limiter: Arc<RateLimiter>,
    metadata_cache: Arc<RwLock<LruCache<String, CachedMetadata>>>,
    fallback: Arc<FallbackHandler>,
    config: CryptoClientConfig,
    metrics: Arc<CryptoMetrics>,
    request_counter: AtomicU64,
}

impl CryptoClientCore {
    pub async fn new(config: CryptoClientConfig, fallback: FallbackHandler) -> Result<Self, CryptoError> {
        config.validate().map_err(|e| CryptoError::internal(e.to_string()))?;
        let circuit_breaker = Arc::new(CircuitBreaker::new(config.circuit_breaker.clone()));
        let quota = Quota::per_second(NonZeroU32::new(config.rate_limit).unwrap_or(NonZeroU32::new(1000).unwrap()));
        let rate_limiter = Arc::new(GovRateLimiter::direct(quota));
        let metadata_cache = Arc::new(RwLock::new(LruCache::new(
            std::num::NonZeroUsize::new(config.metadata_cache_size).unwrap_or(std::num::NonZeroUsize::new(100).unwrap()),
        )));
        Ok(Self {
            grpc_client: RwLock::new(None), circuit_breaker, rate_limiter, metadata_cache,
            fallback: Arc::new(fallback), config, metrics: Arc::new(CryptoMetrics::new()),
            request_counter: AtomicU64::new(0),
        })
    }

    async fn connect(&self) -> Result<CryptoServiceClient<Channel>, CryptoError> {
        let mut client_guard = self.grpc_client.write().await;
        if let Some(ref client) = *client_guard { return Ok(client.clone()); }
        let channel = Channel::from_shared(self.config.address.clone())
            .map_err(|e| CryptoError::connection(e.to_string()))?
            .connect_timeout(self.config.connect_timeout)
            .timeout(self.config.request_timeout)
            .connect().await.map_err(|e| CryptoError::connection(e.to_string()))?;
        let client = CryptoServiceClient::new(channel);
        *client_guard = Some(client.clone());
        info!("Connected to Crypto Service at {}", self.config.address);
        Ok(client)
    }

    fn generate_correlation_id(&self) -> String {
        format!("token-{}-{}", Uuid::new_v4(), self.request_counter.fetch_add(1, Ordering::Relaxed))
    }

    pub(crate) fn check_rate_limit(&self) -> Result<(), CryptoError> {
        if self.rate_limiter.check().is_err() { self.metrics.record_rate_limited(); return Err(CryptoError::RateLimited); }
        Ok(())
    }

    pub(crate) async fn check_circuit_breaker(&self) -> Result<(), CryptoError> {
        if !self.circuit_breaker.allow_request().await { self.metrics.record_circuit_breaker_open(); return Err(CryptoError::CircuitBreakerOpen); }
        Ok(())
    }

    async fn get_cached_metadata(&self, key_id: &KeyId) -> Option<KeyMetadata> {
        let cache_key = format!("{}:{}:{}", key_id.namespace, key_id.id, key_id.version);
        let cache = self.metadata_cache.read().await;
        if let Some(cached) = cache.peek(&cache_key) {
            if cached.cached_at.elapsed() < self.config.metadata_cache_ttl { return Some(cached.metadata.clone()); }
        }
        None
    }

    async fn cache_metadata(&self, key_id: &KeyId, metadata: KeyMetadata) {
        let cache_key = format!("{}:{}:{}", key_id.namespace, key_id.id, key_id.version);
        let mut cache = self.metadata_cache.write().await;
        cache.put(cache_key, CachedMetadata { metadata, cached_at: Instant::now() });
    }

    async fn validate_key_for_signing(&self, key_id: &KeyId) -> Result<(), CryptoError> {
        let metadata = self.get_key_metadata(key_id).await?;
        if !metadata.state.can_sign() {
            return Err(CryptoError::InvalidKeyState { state: metadata.state, operation: "sign".to_string() });
        }
        Ok(())
    }
}

#[async_trait]
impl CryptoClient for CryptoClientCore {
    #[instrument(skip(self, data), fields(key_id = %key_id.id))]
    async fn sign(&self, data: &[u8], key_id: &KeyId) -> Result<SignResult, CryptoError> {
        if !self.config.signing_enabled { return self.fallback.sign_local(data, key_id).await; }
        self.check_rate_limit()?;
        self.check_circuit_breaker().await?;
        self.validate_key_for_signing(key_id).await?;
        let correlation_id = self.generate_correlation_id();
        let start = Instant::now();
        let result = async {
            let mut client = self.connect().await?;
            let request = SignRequest { data: data.to_vec(), key_id: Some(key_id.to_proto()), hash_algorithm: HashAlgorithm::Sha256 as i32, correlation_id: correlation_id.clone() };
            let response = client.sign(request).await.map_err(CryptoError::from)?.into_inner();
            let result_key_id = response.key_id.map(|k| KeyId::from_proto(&k)).unwrap_or_else(|| key_id.clone());
            Ok(SignResult { signature: response.signature, key_id: result_key_id, algorithm: response.algorithm })
        }.await;
        self.metrics.record_operation("sign", result.is_ok(), start.elapsed());
        match result {
            Ok(r) => { self.circuit_breaker.record_success().await; Ok(r) }
            Err(e) if e.is_transient() && self.config.fallback_enabled => {
                self.circuit_breaker.record_failure().await; warn!(error = %e, "Sign failed, using fallback");
                self.metrics.record_fallback_activation("sign"); self.fallback.sign_local(data, key_id).await
            }
            Err(e) => { self.circuit_breaker.record_failure().await; error!(error = %e, correlation_id = %correlation_id, "Sign failed"); Err(e) }
        }
    }

    #[instrument(skip(self, data, signature), fields(key_id = %key_id.id))]
    async fn verify(&self, data: &[u8], signature: &[u8], key_id: &KeyId) -> Result<bool, CryptoError> {
        if !self.config.signing_enabled { return self.fallback.verify_local(data, signature, key_id).await; }
        self.check_rate_limit()?;
        self.check_circuit_breaker().await?;
        let correlation_id = self.generate_correlation_id();
        let start = Instant::now();
        let result = async {
            let mut client = self.connect().await?;
            let request = VerifyRequest { data: data.to_vec(), signature: signature.to_vec(), key_id: Some(key_id.to_proto()), hash_algorithm: HashAlgorithm::Sha256 as i32, correlation_id: correlation_id.clone() };
            Ok(client.verify(request).await.map_err(CryptoError::from)?.into_inner().valid)
        }.await;
        self.metrics.record_operation("verify", result.is_ok(), start.elapsed());
        match result {
            Ok(v) => { self.circuit_breaker.record_success().await; Ok(v) }
            Err(e) if e.is_transient() && self.config.fallback_enabled => {
                self.circuit_breaker.record_failure().await; warn!(error = %e, "Verify failed, using fallback");
                self.metrics.record_fallback_activation("verify"); self.fallback.verify_local(data, signature, key_id).await
            }
            Err(e) => { self.circuit_breaker.record_failure().await; error!(error = %e, correlation_id = %correlation_id, "Verify failed"); Err(e) }
        }
    }

    #[instrument(skip(self, plaintext, aad), fields(key_id = %key_id.id))]
    async fn encrypt(&self, plaintext: &[u8], key_id: &KeyId, aad: Option<&[u8]>) -> Result<EncryptResult, CryptoError> {
        if !self.config.encryption_enabled { return self.fallback.encrypt_local(plaintext, aad).await; }
        self.check_rate_limit()?;
        self.check_circuit_breaker().await?;
        let correlation_id = self.generate_correlation_id();
        let start = Instant::now();
        let result = async {
            let mut client = self.connect().await?;
            let request = EncryptRequest { plaintext: plaintext.to_vec(), key_id: Some(key_id.to_proto()), aad: aad.map(|a| a.to_vec()).unwrap_or_default(), correlation_id: correlation_id.clone() };
            let response = client.encrypt(request).await.map_err(CryptoError::from)?.into_inner();
            let result_key_id = response.key_id.map(|k| KeyId::from_proto(&k)).unwrap_or_else(|| key_id.clone());
            Ok(EncryptResult { ciphertext: response.ciphertext, iv: response.iv, tag: response.tag, key_id: result_key_id, algorithm: response.algorithm })
        }.await;
        self.metrics.record_operation("encrypt", result.is_ok(), start.elapsed());
        match result {
            Ok(r) => { self.circuit_breaker.record_success().await; Ok(r) }
            Err(e) if e.is_transient() && self.config.fallback_enabled => {
                self.circuit_breaker.record_failure().await; warn!(error = %e, "Encrypt failed, using fallback");
                self.metrics.record_fallback_activation("encrypt"); self.fallback.encrypt_local(plaintext, aad).await
            }
            Err(e) => { self.circuit_breaker.record_failure().await; error!(error = %e, correlation_id = %correlation_id, "Encrypt failed"); Err(e) }
        }
    }

    #[instrument(skip(self, encrypted, aad), fields(key_id = %key_id.id))]
    async fn decrypt(&self, encrypted: &EncryptedData, key_id: &KeyId, aad: Option<&[u8]>) -> Result<Vec<u8>, CryptoError> {
        if !self.config.encryption_enabled { return self.fallback.decrypt_local(encrypted, aad).await; }
        self.check_rate_limit()?;
        self.check_circuit_breaker().await?;
        let correlation_id = self.generate_correlation_id();
        let start = Instant::now();
        let result = async {
            let mut client = self.connect().await?;
            let request = DecryptRequest { ciphertext: encrypted.ciphertext.clone(), iv: encrypted.iv.clone(), tag: encrypted.tag.clone(), key_id: Some(key_id.to_proto()), aad: aad.map(|a| a.to_vec()).unwrap_or_default(), correlation_id: correlation_id.clone() };
            Ok(client.decrypt(request).await.map_err(CryptoError::from)?.into_inner().plaintext)
        }.await;
        self.metrics.record_operation("decrypt", result.is_ok(), start.elapsed());
        match result {
            Ok(p) => { self.circuit_breaker.record_success().await; Ok(p) }
            Err(e) if e.is_transient() && self.config.fallback_enabled => {
                self.circuit_breaker.record_failure().await; warn!(error = %e, "Decrypt failed, using fallback");
                self.metrics.record_fallback_activation("decrypt"); self.fallback.decrypt_local(encrypted, aad).await
            }
            Err(e) => { self.circuit_breaker.record_failure().await; error!(error = %e, correlation_id = %correlation_id, "Decrypt failed"); Err(e) }
        }
    }

    #[instrument(skip(self), fields(algorithm = ?algorithm, namespace = namespace))]
    async fn generate_key(&self, algorithm: KeyAlgorithm, namespace: &str) -> Result<KeyId, CryptoError> {
        self.check_rate_limit()?;
        self.check_circuit_breaker().await?;
        let correlation_id = self.generate_correlation_id();
        let start = Instant::now();
        let result = async {
            let mut client = self.connect().await?;
            let request = GenerateKeyRequest { algorithm: algorithm.to_proto(), namespace: namespace.to_string(), metadata: std::collections::HashMap::new(), correlation_id: correlation_id.clone() };
            let response = client.generate_key(request).await.map_err(CryptoError::from)?.into_inner();
            let key_id = response.key_id.map(|k| KeyId::from_proto(&k)).ok_or_else(|| CryptoError::internal("No key_id in response"))?;
            if let Some(metadata) = response.metadata { self.cache_metadata(&key_id, KeyMetadata::from_proto(&metadata)).await; }
            Ok(key_id)
        }.await;
        self.metrics.record_operation("generate_key", result.is_ok(), start.elapsed());
        match &result { Ok(_) => self.circuit_breaker.record_success().await, Err(_) => self.circuit_breaker.record_failure().await }
        result
    }

    #[instrument(skip(self), fields(key_id = %key_id.id))]
    async fn rotate_key(&self, key_id: &KeyId) -> Result<KeyRotationResult, CryptoError> {
        self.check_rate_limit()?;
        self.check_circuit_breaker().await?;
        let correlation_id = self.generate_correlation_id();
        let start = Instant::now();
        let result = async {
            let mut client = self.connect().await?;
            let request = RotateKeyRequest { key_id: Some(key_id.to_proto()), correlation_id: correlation_id.clone() };
            let response = client.rotate_key(request).await.map_err(CryptoError::from)?.into_inner();
            let new_key_id = response.new_key_id.map(|k| KeyId::from_proto(&k)).ok_or_else(|| CryptoError::internal("No new_key_id"))?;
            let old_key_id = response.old_key_id.map(|k| KeyId::from_proto(&k)).ok_or_else(|| CryptoError::internal("No old_key_id"))?;
            let metadata = response.metadata.map(|m| KeyMetadata::from_proto(&m)).ok_or_else(|| CryptoError::internal("No metadata"))?;
            self.cache_metadata(&new_key_id, metadata.clone()).await;
            Ok(KeyRotationResult { new_key_id, old_key_id, metadata })
        }.await;
        self.metrics.record_operation("rotate_key", result.is_ok(), start.elapsed());
        match &result { Ok(_) => self.circuit_breaker.record_success().await, Err(_) => self.circuit_breaker.record_failure().await }
        result
    }

    #[instrument(skip(self), fields(key_id = %key_id.id))]
    async fn get_key_metadata(&self, key_id: &KeyId) -> Result<KeyMetadata, CryptoError> {
        if let Some(cached) = self.get_cached_metadata(key_id).await { self.metrics.record_cache_hit("metadata"); return Ok(cached); }
        self.metrics.record_cache_miss("metadata");
        self.check_rate_limit()?;
        self.check_circuit_breaker().await?;
        let correlation_id = self.generate_correlation_id();
        let start = Instant::now();
        let result = async {
            let mut client = self.connect().await?;
            let request = GetKeyMetadataRequest { key_id: Some(key_id.to_proto()), correlation_id: correlation_id.clone() };
            let response = client.get_key_metadata(request).await.map_err(CryptoError::from)?.into_inner();
            let metadata = response.metadata.map(|m| KeyMetadata::from_proto(&m)).ok_or_else(|| CryptoError::internal("No metadata"))?;
            self.cache_metadata(key_id, metadata.clone()).await;
            Ok(metadata)
        }.await;
        self.metrics.record_operation("get_key_metadata", result.is_ok(), start.elapsed());
        match &result { Ok(_) => self.circuit_breaker.record_success().await, Err(_) => self.circuit_breaker.record_failure().await }
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_client() {
        let config = CryptoClientConfig::default();
        let fallback = FallbackHandler::new_disabled();
        assert!(CryptoClientCore::new(config, fallback).await.is_ok());
    }

    #[tokio::test]
    async fn test_rate_limit_check() {
        let config = CryptoClientConfig::default().with_rate_limit(1);
        let fallback = FallbackHandler::new_disabled();
        let client = CryptoClientCore::new(config, fallback).await.unwrap();
        assert!(client.check_rate_limit().is_ok());
    }
}
