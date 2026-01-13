//! KMS (Key Management Service) module.
//!
//! Provides trait-based abstraction for signing operations with
//! AWS KMS integration, Crypto Service integration, and mock implementation.

pub mod aws;
pub mod mock;

pub use aws::{AwsKmsConfig, AwsKmsSigner};
pub use mock::MockKms;

use crate::crypto::{CryptoClient, CryptoClientConfig, CryptoClientFactory, CryptoSigner, KeyId};
use crate::error::TokenError;
use async_trait::async_trait;
use jsonwebtoken::EncodingKey;
use std::sync::Arc;

/// KMS signer trait with async support.
#[async_trait]
pub trait KmsSigner: Send + Sync {
    /// Sign data and return signature bytes.
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError>;

    /// Get the encoding key for JWT serialization.
    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError>;

    /// Get the key ID for JWT header.
    fn key_id(&self) -> &str;

    /// Get the algorithm name for JWT header.
    fn algorithm(&self) -> &str;
}

/// KMS provider factory.
pub struct KmsFactory;

impl KmsFactory {
    /// Create a KMS signer based on configuration.
    pub fn create(provider: &crate::config::KmsProvider, key_id: &str) -> Box<dyn KmsSigner> {
        match provider {
            crate::config::KmsProvider::Aws { region } => {
                let config = AwsKmsConfig {
                    key_id: key_id.to_string(),
                    region: region.clone(),
                    ..Default::default()
                };
                Box::new(AwsKmsSigner::new(config))
            }
            crate::config::KmsProvider::Mock => Box::new(MockKms::new(key_id)),
        }
    }

    /// Create a CryptoSigner using Crypto Service.
    ///
    /// # Arguments
    /// * `client` - CryptoClient instance
    /// * `namespace` - Key namespace
    /// * `key_name` - Key identifier
    /// * `version` - Key version
    /// * `algorithm` - JWT algorithm (PS256, ES256, etc.)
    pub fn create_crypto_signer(
        client: Arc<dyn CryptoClient>,
        namespace: &str,
        key_name: &str,
        version: u32,
        algorithm: &str,
    ) -> CryptoSigner {
        let key_id = KeyId::new(namespace, key_name, version);
        CryptoClientFactory::create_signer(client, key_id, algorithm)
    }
}
