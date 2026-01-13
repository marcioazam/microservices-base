//! Fallback handler for local cryptographic operations.

use super::error::CryptoError;
use super::models::{EncryptResult, EncryptedData, KeyId};
use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use rand::RngCore;
use ring::hmac;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use tracing::warn;
use zeroize::Zeroizing;

/// Fallback handler for local cryptographic operations.
pub struct FallbackHandler {
    /// Local signing key (HMAC)
    local_signing_key: Option<Zeroizing<Vec<u8>>>,
    /// Local encryption key (AES-256)
    local_encryption_key: Option<Zeroizing<[u8; 32]>>,
    /// Fallback enabled flag
    enabled: bool,
    /// Fallback activation counter
    activation_count: AtomicU64,
}

impl FallbackHandler {
    /// Create a new fallback handler with keys.
    #[must_use]
    pub fn new(signing_key: Option<Vec<u8>>, encryption_key: Option<[u8; 32]>) -> Self {
        Self {
            local_signing_key: signing_key.map(Zeroizing::new),
            local_encryption_key: encryption_key.map(Zeroizing::new),
            enabled: true,
            activation_count: AtomicU64::new(0),
        }
    }

    /// Create a disabled fallback handler.
    #[must_use]
    pub fn new_disabled() -> Self {
        Self {
            local_signing_key: None,
            local_encryption_key: None,
            enabled: false,
            activation_count: AtomicU64::new(0),
        }
    }

    /// Create fallback handler from environment.
    #[must_use]
    pub fn from_env() -> Self {
        let signing_key = std::env::var("FALLBACK_SIGNING_KEY")
            .ok()
            .and_then(|s| base64::decode(&s).ok());

        let encryption_key = std::env::var("FALLBACK_ENCRYPTION_KEY")
            .ok()
            .and_then(|s| base64::decode(&s).ok())
            .and_then(|v| {
                if v.len() == 32 {
                    let mut arr = [0u8; 32];
                    arr.copy_from_slice(&v);
                    Some(arr)
                } else {
                    None
                }
            });

        let enabled = std::env::var("CRYPTO_FALLBACK_ENABLED")
            .map(|v| v.parse().unwrap_or(true))
            .unwrap_or(true);

        Self {
            local_signing_key: signing_key.map(Zeroizing::new),
            local_encryption_key: encryption_key.map(Zeroizing::new),
            enabled,
            activation_count: AtomicU64::new(0),
        }
    }

    /// Check if fallback is enabled.
    #[must_use]
    pub fn is_enabled(&self) -> bool {
        self.enabled
    }

    /// Get fallback activation count.
    #[must_use]
    pub fn activation_count(&self) -> u64 {
        self.activation_count.load(Ordering::Relaxed)
    }

    /// Increment activation count.
    fn record_activation(&self) {
        self.activation_count.fetch_add(1, Ordering::Relaxed);
    }

    /// Sign data locally using HMAC-SHA256.
    pub async fn sign_local(&self, data: &[u8], key_id: &KeyId) -> Result<super::models::SignResult, CryptoError> {
        if !self.enabled {
            return Err(CryptoError::internal("Fallback is disabled"));
        }

        let key = self
            .local_signing_key
            .as_ref()
            .ok_or_else(|| CryptoError::internal("No fallback signing key configured"))?;

        self.record_activation();
        warn!("Using fallback signing - Crypto Service unavailable");

        let signing_key = hmac::Key::new(hmac::HMAC_SHA256, key);
        let signature = hmac::sign(&signing_key, data);

        Ok(super::models::SignResult {
            signature: signature.as_ref().to_vec(),
            key_id: key_id.clone(),
            algorithm: "HS256".to_string(),
        })
    }

    /// Verify signature locally using HMAC-SHA256.
    pub async fn verify_local(
        &self,
        data: &[u8],
        signature: &[u8],
        _key_id: &KeyId,
    ) -> Result<bool, CryptoError> {
        if !self.enabled {
            return Err(CryptoError::internal("Fallback is disabled"));
        }

        let key = self
            .local_signing_key
            .as_ref()
            .ok_or_else(|| CryptoError::internal("No fallback signing key configured"))?;

        self.record_activation();
        warn!("Using fallback verification - Crypto Service unavailable");

        let signing_key = hmac::Key::new(hmac::HMAC_SHA256, key);
        Ok(hmac::verify(&signing_key, data, signature).is_ok())
    }

    /// Encrypt data locally using AES-256-GCM.
    pub async fn encrypt_local(
        &self,
        plaintext: &[u8],
        aad: Option<&[u8]>,
    ) -> Result<EncryptResult, CryptoError> {
        if !self.enabled {
            return Err(CryptoError::internal("Fallback is disabled"));
        }

        let key = self
            .local_encryption_key
            .as_ref()
            .ok_or_else(|| CryptoError::internal("No fallback encryption key configured"))?;

        self.record_activation();
        warn!("Using fallback encryption - Crypto Service unavailable");

        let cipher = Aes256Gcm::new_from_slice(key.as_ref())
            .map_err(|e| CryptoError::encryption(e.to_string()))?;

        // Generate random nonce
        let mut nonce_bytes = [0u8; 12];
        rand::thread_rng().fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        // Encrypt with optional AAD
        let ciphertext = if let Some(aad_data) = aad {
            use aes_gcm::aead::Payload;
            cipher
                .encrypt(nonce, Payload { msg: plaintext, aad: aad_data })
                .map_err(|e| CryptoError::encryption(e.to_string()))?
        } else {
            cipher
                .encrypt(nonce, plaintext)
                .map_err(|e| CryptoError::encryption(e.to_string()))?
        };

        // Split ciphertext and tag (last 16 bytes is tag)
        let tag_start = ciphertext.len().saturating_sub(16);
        let (ct, tag) = ciphertext.split_at(tag_start);

        Ok(EncryptResult {
            ciphertext: ct.to_vec(),
            iv: nonce_bytes.to_vec(),
            tag: tag.to_vec(),
            key_id: KeyId::new("fallback", "local-aes-key", 1),
            algorithm: "AES-256-GCM".to_string(),
        })
    }

    /// Decrypt data locally using AES-256-GCM.
    pub async fn decrypt_local(
        &self,
        encrypted: &EncryptedData,
        aad: Option<&[u8]>,
    ) -> Result<Vec<u8>, CryptoError> {
        if !self.enabled {
            return Err(CryptoError::internal("Fallback is disabled"));
        }

        let key = self
            .local_encryption_key
            .as_ref()
            .ok_or_else(|| CryptoError::internal("No fallback encryption key configured"))?;

        self.record_activation();
        warn!("Using fallback decryption - Crypto Service unavailable");

        let cipher = Aes256Gcm::new_from_slice(key.as_ref())
            .map_err(|e| CryptoError::decryption(e.to_string()))?;

        if encrypted.iv.len() != 12 {
            return Err(CryptoError::decryption("Invalid IV length"));
        }

        let nonce = Nonce::from_slice(&encrypted.iv);

        // Combine ciphertext and tag
        let mut ciphertext_with_tag = encrypted.ciphertext.clone();
        ciphertext_with_tag.extend_from_slice(&encrypted.tag);

        // Decrypt with optional AAD
        let plaintext = if let Some(aad_data) = aad {
            use aes_gcm::aead::Payload;
            cipher
                .decrypt(nonce, Payload { msg: &ciphertext_with_tag, aad: aad_data })
                .map_err(|e| CryptoError::decryption(e.to_string()))?
        } else {
            cipher
                .decrypt(nonce, ciphertext_with_tag.as_slice())
                .map_err(|e| CryptoError::decryption(e.to_string()))?
        };

        Ok(plaintext)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_keys() -> (Vec<u8>, [u8; 32]) {
        let signing_key = b"test-signing-key-for-hmac-256!!".to_vec();
        let encryption_key = [0u8; 32];
        (signing_key, encryption_key)
    }

    #[tokio::test]
    async fn test_sign_local() {
        let (signing_key, encryption_key) = test_keys();
        let handler = FallbackHandler::new(Some(signing_key), Some(encryption_key));

        let key_id = KeyId::new("test", "key", 1);
        let data = b"test data to sign";

        let result = handler.sign_local(data, &key_id).await;
        assert!(result.is_ok());

        let sign_result = result.unwrap();
        assert!(!sign_result.signature.is_empty());
        assert_eq!(sign_result.algorithm, "HS256");
    }

    #[tokio::test]
    async fn test_verify_local() {
        let (signing_key, encryption_key) = test_keys();
        let handler = FallbackHandler::new(Some(signing_key), Some(encryption_key));

        let key_id = KeyId::new("test", "key", 1);
        let data = b"test data to sign";

        let sign_result = handler.sign_local(data, &key_id).await.unwrap();
        let valid = handler
            .verify_local(data, &sign_result.signature, &key_id)
            .await
            .unwrap();

        assert!(valid);
    }

    #[tokio::test]
    async fn test_verify_local_invalid() {
        let (signing_key, encryption_key) = test_keys();
        let handler = FallbackHandler::new(Some(signing_key), Some(encryption_key));

        let key_id = KeyId::new("test", "key", 1);
        let data = b"test data";
        let wrong_signature = b"wrong signature";

        let valid = handler
            .verify_local(data, wrong_signature, &key_id)
            .await
            .unwrap();

        assert!(!valid);
    }

    #[tokio::test]
    async fn test_encrypt_decrypt_local() {
        let (signing_key, encryption_key) = test_keys();
        let handler = FallbackHandler::new(Some(signing_key), Some(encryption_key));

        let plaintext = b"sensitive data to encrypt";
        let aad = b"additional authenticated data";

        let encrypt_result = handler.encrypt_local(plaintext, Some(aad)).await.unwrap();
        assert!(!encrypt_result.ciphertext.is_empty());
        assert_eq!(encrypt_result.iv.len(), 12);

        let encrypted = EncryptedData::from_result(&encrypt_result);
        let decrypted = handler.decrypt_local(&encrypted, Some(aad)).await.unwrap();

        assert_eq!(decrypted, plaintext);
    }

    #[tokio::test]
    async fn test_encrypt_decrypt_without_aad() {
        let (signing_key, encryption_key) = test_keys();
        let handler = FallbackHandler::new(Some(signing_key), Some(encryption_key));

        let plaintext = b"data without aad";

        let encrypt_result = handler.encrypt_local(plaintext, None).await.unwrap();
        let encrypted = EncryptedData::from_result(&encrypt_result);
        let decrypted = handler.decrypt_local(&encrypted, None).await.unwrap();

        assert_eq!(decrypted, plaintext);
    }

    #[tokio::test]
    async fn test_disabled_fallback() {
        let handler = FallbackHandler::new_disabled();
        let key_id = KeyId::new("test", "key", 1);

        let result = handler.sign_local(b"data", &key_id).await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_activation_count() {
        let (signing_key, encryption_key) = test_keys();
        let handler = FallbackHandler::new(Some(signing_key), Some(encryption_key));

        assert_eq!(handler.activation_count(), 0);

        let key_id = KeyId::new("test", "key", 1);
        handler.sign_local(b"data", &key_id).await.unwrap();

        assert_eq!(handler.activation_count(), 1);

        handler.encrypt_local(b"data", None).await.unwrap();
        assert_eq!(handler.activation_count(), 2);
    }
}
