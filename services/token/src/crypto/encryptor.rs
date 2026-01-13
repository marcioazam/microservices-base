//! CryptoEncryptor - Cache encryption using Crypto Service.

use super::client::CryptoClient;
use super::error::CryptoError;
use super::models::{EncryptedData, KeyId};
use crate::error::TokenError;
use crate::refresh::family::TokenFamily;
use std::sync::Arc;
use tracing::instrument;

/// Crypto Service based encryptor for cache data.
pub struct CryptoEncryptor {
    /// CryptoClient for encryption operations
    client: Arc<dyn CryptoClient>,
    /// Encryption key ID
    key_id: KeyId,
    /// Namespace for cache keys
    namespace: String,
}

impl CryptoEncryptor {
    /// Create a new CryptoEncryptor.
    #[must_use]
    pub fn new(client: Arc<dyn CryptoClient>, key_id: KeyId, namespace: impl Into<String>) -> Self {
        Self {
            client,
            key_id,
            namespace: namespace.into(),
        }
    }

    /// Get the namespace.
    #[must_use]
    pub fn namespace(&self) -> &str {
        &self.namespace
    }

    /// Get the key ID.
    #[must_use]
    pub fn key_id(&self) -> &KeyId {
        &self.key_id
    }

    /// Encrypt token family data for cache storage.
    #[instrument(skip(self, family), fields(family_id = %family.family_id))]
    pub async fn encrypt_token_family(&self, family: &TokenFamily) -> Result<Vec<u8>, TokenError> {
        let plaintext = serde_json::to_vec(family)
            .map_err(|e| TokenError::internal(format!("Serialization failed: {}", e)))?;

        // Use family_id as AAD for additional authentication
        let aad = family.family_id.as_bytes();

        let result = self
            .client
            .encrypt(&plaintext, &self.key_id, Some(aad))
            .await
            .map_err(|e| TokenError::encryption(e.to_string()))?;

        // Serialize encrypted result for storage
        result.to_bytes().map_err(|e| TokenError::internal(e.to_string()))
    }

    /// Decrypt token family data from cache.
    #[instrument(skip(self, encrypted), fields(family_id = %family_id))]
    pub async fn decrypt_token_family(
        &self,
        encrypted: &[u8],
        family_id: &str,
    ) -> Result<TokenFamily, TokenError> {
        // Deserialize encrypted data
        let encrypt_result: super::models::EncryptResult = serde_json::from_slice(encrypted)
            .map_err(|e| TokenError::internal(format!("Deserialization failed: {}", e)))?;

        let encrypted_data = EncryptedData::from_result(&encrypt_result);

        // Use family_id as AAD for verification
        let aad = family_id.as_bytes();

        let plaintext = self
            .client
            .decrypt(&encrypted_data, &self.key_id, Some(aad))
            .await
            .map_err(|e| TokenError::decryption(e.to_string()))?;

        serde_json::from_slice(&plaintext)
            .map_err(|e| TokenError::internal(format!("Deserialization failed: {}", e)))
    }

    /// Encrypt arbitrary data.
    pub async fn encrypt(&self, data: &[u8], aad: Option<&[u8]>) -> Result<Vec<u8>, TokenError> {
        let result = self
            .client
            .encrypt(data, &self.key_id, aad)
            .await
            .map_err(|e| TokenError::encryption(e.to_string()))?;

        result.to_bytes().map_err(|e| TokenError::internal(e.to_string()))
    }

    /// Decrypt arbitrary data.
    pub async fn decrypt(&self, encrypted: &[u8], aad: Option<&[u8]>) -> Result<Vec<u8>, TokenError> {
        let encrypt_result: super::models::EncryptResult = serde_json::from_slice(encrypted)
            .map_err(|e| TokenError::internal(format!("Deserialization failed: {}", e)))?;

        let encrypted_data = EncryptedData::from_result(&encrypt_result);

        self.client
            .decrypt(&encrypted_data, &self.key_id, aad)
            .await
            .map_err(|e| TokenError::decryption(e.to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use super::super::models::{EncryptResult, KeyAlgorithm, KeyMetadata, KeyRotationResult, KeyState, SignResult};
    use async_trait::async_trait;
    use chrono::Utc;

    /// Mock CryptoClient for testing encryption.
    struct MockEncryptClient {
        encrypt_result: EncryptResult,
    }

    impl MockEncryptClient {
        fn new() -> Self {
            Self {
                encrypt_result: EncryptResult {
                    ciphertext: vec![1, 2, 3, 4],
                    iv: vec![0; 12],
                    tag: vec![5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16],
                    key_id: KeyId::new("token-cache", "enc-key", 1),
                    algorithm: "AES-256-GCM".to_string(),
                },
            }
        }
    }

    #[async_trait]
    impl CryptoClient for MockEncryptClient {
        async fn sign(&self, _data: &[u8], _key_id: &KeyId) -> Result<SignResult, CryptoError> {
            unimplemented!()
        }

        async fn verify(
            &self,
            _data: &[u8],
            _signature: &[u8],
            _key_id: &KeyId,
        ) -> Result<bool, CryptoError> {
            unimplemented!()
        }

        async fn encrypt(
            &self,
            _plaintext: &[u8],
            _key_id: &KeyId,
            _aad: Option<&[u8]>,
        ) -> Result<EncryptResult, CryptoError> {
            Ok(self.encrypt_result.clone())
        }

        async fn decrypt(
            &self,
            _encrypted: &EncryptedData,
            _key_id: &KeyId,
            _aad: Option<&[u8]>,
        ) -> Result<Vec<u8>, CryptoError> {
            // Return a valid TokenFamily JSON
            let family = TokenFamily::new(
                "family-1".to_string(),
                "user-1".to_string(),
                "session-1".to_string(),
                "hash-1".to_string(),
            );
            Ok(serde_json::to_vec(&family).unwrap())
        }

        async fn generate_key(
            &self,
            _algorithm: KeyAlgorithm,
            _namespace: &str,
        ) -> Result<KeyId, CryptoError> {
            unimplemented!()
        }

        async fn rotate_key(&self, _key_id: &KeyId) -> Result<KeyRotationResult, CryptoError> {
            unimplemented!()
        }

        async fn get_key_metadata(&self, _key_id: &KeyId) -> Result<KeyMetadata, CryptoError> {
            Ok(KeyMetadata {
                id: KeyId::new("token-cache", "enc-key", 1),
                algorithm: KeyAlgorithm::Aes256Gcm,
                state: KeyState::Active,
                created_at: Utc::now(),
                expires_at: None,
                rotated_at: None,
                previous_version: None,
                owner_service: "token-service".to_string(),
                allowed_operations: vec!["encrypt".to_string(), "decrypt".to_string()],
                usage_count: 0,
            })
        }
    }

    #[tokio::test]
    async fn test_encrypt_token_family() {
        let client = Arc::new(MockEncryptClient::new());
        let key_id = KeyId::new("token-cache", "enc-key", 1);
        let encryptor = CryptoEncryptor::new(client, key_id, "token-cache");

        let family = TokenFamily::new(
            "family-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-1".to_string(),
        );

        let result = encryptor.encrypt_token_family(&family).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_decrypt_token_family() {
        let client = Arc::new(MockEncryptClient::new());
        let key_id = KeyId::new("token-cache", "enc-key", 1);
        let encryptor = CryptoEncryptor::new(client, key_id, "token-cache");

        // Create encrypted data (mock will return valid family)
        let encrypt_result = EncryptResult {
            ciphertext: vec![1, 2, 3, 4],
            iv: vec![0; 12],
            tag: vec![5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16],
            key_id: KeyId::new("token-cache", "enc-key", 1),
            algorithm: "AES-256-GCM".to_string(),
        };
        let encrypted = serde_json::to_vec(&encrypt_result).unwrap();

        let result = encryptor.decrypt_token_family(&encrypted, "family-1").await;
        assert!(result.is_ok());

        let family = result.unwrap();
        assert_eq!(family.family_id, "family-1");
    }

    #[test]
    fn test_encryptor_properties() {
        let client = Arc::new(MockEncryptClient::new());
        let key_id = KeyId::new("token-cache", "enc-key", 1);
        let encryptor = CryptoEncryptor::new(client, key_id.clone(), "token-cache");

        assert_eq!(encryptor.namespace(), "token-cache");
        assert_eq!(encryptor.key_id(), &key_id);
    }
}
