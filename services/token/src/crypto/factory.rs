//! CryptoClient factory for creating appropriate client instances.

use super::client::{CryptoClient, CryptoClientCore};
use super::config::CryptoClientConfig;
use super::encryptor::CryptoEncryptor;
use super::error::CryptoError;
use super::fallback::FallbackHandler;
use super::models::KeyId;
use super::signer::CryptoSigner;
use std::sync::Arc;

/// Factory for creating CryptoClient instances.
pub struct CryptoClientFactory;

impl CryptoClientFactory {
    /// Create a CryptoClient based on configuration.
    ///
    /// # Errors
    ///
    /// Returns error if client creation fails.
    pub async fn create(
        config: CryptoClientConfig,
        signing_key: Option<Vec<u8>>,
        encryption_key: Option<[u8; 32]>,
    ) -> Result<Arc<dyn CryptoClient>, CryptoError> {
        let fallback = FallbackHandler::new(signing_key, encryption_key);
        let client = CryptoClientCore::new(config, fallback).await?;
        Ok(Arc::new(client))
    }

    /// Create a CryptoClient with disabled fallback (for testing).
    ///
    /// # Errors
    ///
    /// Returns error if client creation fails.
    pub async fn create_without_fallback(
        config: CryptoClientConfig,
    ) -> Result<Arc<dyn CryptoClient>, CryptoError> {
        let fallback = FallbackHandler::new_disabled();
        let client = CryptoClientCore::new(config, fallback).await?;
        Ok(Arc::new(client))
    }

    /// Create a CryptoSigner for JWT signing.
    pub fn create_signer(
        client: Arc<dyn CryptoClient>,
        key_id: KeyId,
        algorithm: &str,
    ) -> CryptoSigner {
        CryptoSigner::new(client, key_id, algorithm)
    }

    /// Create a CryptoEncryptor for cache encryption.
    pub fn create_encryptor(
        client: Arc<dyn CryptoClient>,
        key_id: KeyId,
        namespace: &str,
    ) -> CryptoEncryptor {
        CryptoEncryptor::new(client, key_id, namespace.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_client() {
        let config = CryptoClientConfig::default();
        let signing_key = Some(b"test-signing-key-for-hmac-256!!".to_vec());
        let encryption_key = Some([0u8; 32]);

        let result = CryptoClientFactory::create(config, signing_key, encryption_key).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_create_without_fallback() {
        let config = CryptoClientConfig::default();
        let result = CryptoClientFactory::create_without_fallback(config).await;
        assert!(result.is_ok());
    }
}
