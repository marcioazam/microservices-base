//! Fallback Handler for local encryption
//!
//! Provides local AES-256-GCM encryption when crypto-service is unavailable.

use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use rand::RngCore;
use serde::{Deserialize, Serialize};
use std::collections::VecDeque;
use std::sync::Arc;
use std::time::Instant;
use tokio::sync::RwLock;
use tracing::{info, warn};

use crate::crypto::error::CryptoError;
use crate::crypto::key_manager::KeyId;

/// Encrypted data structure for serialization
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptedData {
    /// Ciphertext bytes
    pub ciphertext: Vec<u8>,
    /// Initialization vector (12 bytes for AES-GCM)
    pub iv: Vec<u8>,
    /// Authentication tag (16 bytes for AES-GCM)
    pub tag: Vec<u8>,
    /// Key ID used for encryption
    pub key_id: KeyId,
    /// Algorithm identifier
    pub algorithm: String,
}

impl EncryptedData {
    /// Creates a new EncryptedData for local fallback
    #[must_use]
    pub fn new_local(ciphertext: Vec<u8>, iv: Vec<u8>, tag: Vec<u8>, key_version: u32) -> Self {
        Self {
            ciphertext,
            iv,
            tag,
            key_id: KeyId::new("local-fallback", "dek", key_version),
            algorithm: "AES-256-GCM".to_string(),
        }
    }

    /// Checks if this was encrypted with local fallback
    #[must_use]
    pub fn is_local_fallback(&self) -> bool {
        self.key_id.namespace == "local-fallback"
    }
}

/// Pending operation for retry when service recovers
#[derive(Debug)]
pub enum PendingOperation {
    /// Key rotation request
    KeyRotation {
        /// Correlation ID for tracing
        correlation_id: String,
        /// When the request was made
        requested_at: Instant,
    },
}

/// Handles fallback encryption when crypto-service is unavailable
pub struct FallbackHandler {
    /// AES-256-GCM cipher
    cipher: Aes256Gcm,
    /// Pending operations queue
    pending_ops: Arc<RwLock<VecDeque<PendingOperation>>>,
    /// Maximum pending operations
    max_pending: usize,
    /// Current key version
    key_version: u32,
}

impl FallbackHandler {
    /// Creates a new FallbackHandler with the given DEK
    ///
    /// # Errors
    ///
    /// Returns error if DEK is invalid (not 32 bytes)
    pub fn new(dek: &[u8], key_version: u32) -> Result<Self, CryptoError> {
        if dek.len() != 32 {
            return Err(CryptoError::encryption_failed(
                "DEK must be 32 bytes for AES-256",
            ));
        }

        let cipher = Aes256Gcm::new_from_slice(dek)
            .map_err(|_| CryptoError::encryption_failed("Invalid DEK"))?;

        Ok(Self {
            cipher,
            pending_ops: Arc::new(RwLock::new(VecDeque::new())),
            max_pending: 100,
            key_version,
        })
    }

    /// Encrypts data locally using AES-256-GCM
    ///
    /// # Errors
    ///
    /// Returns error if encryption fails
    pub fn encrypt(&self, plaintext: &[u8], aad: Option<&[u8]>) -> Result<EncryptedData, CryptoError> {
        // Generate random nonce (12 bytes)
        let mut nonce_bytes = [0u8; 12];
        rand::thread_rng().fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        // Encrypt with AAD if provided
        let ciphertext = if let Some(aad_bytes) = aad {
            use aes_gcm::aead::Payload;
            self.cipher
                .encrypt(nonce, Payload { msg: plaintext, aad: aad_bytes })
                .map_err(|e| CryptoError::encryption_failed(format!("AES-GCM encrypt failed: {e}")))?
        } else {
            self.cipher
                .encrypt(nonce, plaintext)
                .map_err(|e| CryptoError::encryption_failed(format!("AES-GCM encrypt failed: {e}")))?
        };

        // AES-GCM appends the tag to ciphertext, split it
        let tag_start = ciphertext.len().saturating_sub(16);
        let (ct, tag) = ciphertext.split_at(tag_start);

        Ok(EncryptedData::new_local(
            ct.to_vec(),
            nonce_bytes.to_vec(),
            tag.to_vec(),
            self.key_version,
        ))
    }

    /// Decrypts data locally using AES-256-GCM
    ///
    /// # Errors
    ///
    /// Returns error if decryption fails
    pub fn decrypt(&self, encrypted: &EncryptedData, aad: Option<&[u8]>) -> Result<Vec<u8>, CryptoError> {
        if encrypted.iv.len() != 12 {
            return Err(CryptoError::decryption_failed("Invalid IV length"));
        }

        if encrypted.tag.len() != 16 {
            return Err(CryptoError::decryption_failed("Invalid tag length"));
        }

        let nonce = Nonce::from_slice(&encrypted.iv);

        // Reconstruct ciphertext with tag appended
        let mut ciphertext_with_tag = encrypted.ciphertext.clone();
        ciphertext_with_tag.extend_from_slice(&encrypted.tag);

        // Decrypt with AAD if provided
        let plaintext = if let Some(aad_bytes) = aad {
            use aes_gcm::aead::Payload;
            self.cipher
                .decrypt(
                    nonce,
                    Payload {
                        msg: &ciphertext_with_tag,
                        aad: aad_bytes,
                    },
                )
                .map_err(|_| CryptoError::decryption_failed("AES-GCM decrypt failed: authentication failed"))?
        } else {
            self.cipher
                .decrypt(nonce, ciphertext_with_tag.as_slice())
                .map_err(|_| CryptoError::decryption_failed("AES-GCM decrypt failed: authentication failed"))?
        };

        Ok(plaintext)
    }

    /// Queues a pending operation for retry
    ///
    /// # Errors
    ///
    /// Returns error if queue is full
    pub async fn queue_operation(&self, op: PendingOperation) -> Result<(), CryptoError> {
        let mut queue = self.pending_ops.write().await;

        if queue.len() >= self.max_pending {
            warn!("Pending operations queue full, dropping oldest");
            queue.pop_front();
        }

        queue.push_back(op);
        Ok(())
    }

    /// Gets the number of pending operations
    pub async fn pending_count(&self) -> usize {
        self.pending_ops.read().await.len()
    }

    /// Drains pending operations for processing
    pub async fn drain_pending(&self) -> Vec<PendingOperation> {
        let mut queue = self.pending_ops.write().await;
        queue.drain(..).collect()
    }

    /// Gets the key version
    #[must_use]
    pub const fn key_version(&self) -> u32 {
        self.key_version
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_dek() -> [u8; 32] {
        [
            0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
            0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b,
            0x1c, 0x1d, 0x1e, 0x1f,
        ]
    }

    #[test]
    fn test_encrypt_decrypt_round_trip() {
        let handler = FallbackHandler::new(&test_dek(), 1).unwrap();
        let plaintext = b"Hello, World!";

        let encrypted = handler.encrypt(plaintext, None).unwrap();
        let decrypted = handler.decrypt(&encrypted, None).unwrap();

        assert_eq!(decrypted, plaintext);
    }

    #[test]
    fn test_encrypt_decrypt_with_aad() {
        let handler = FallbackHandler::new(&test_dek(), 1).unwrap();
        let plaintext = b"Secret data";
        let aad = b"auth-edge:cache:key1";

        let encrypted = handler.encrypt(plaintext, Some(aad)).unwrap();
        let decrypted = handler.decrypt(&encrypted, Some(aad)).unwrap();

        assert_eq!(decrypted, plaintext);
    }

    #[test]
    fn test_decrypt_fails_with_wrong_aad() {
        let handler = FallbackHandler::new(&test_dek(), 1).unwrap();
        let plaintext = b"Secret data";
        let aad = b"auth-edge:cache:key1";
        let wrong_aad = b"auth-edge:cache:key2";

        let encrypted = handler.encrypt(plaintext, Some(aad)).unwrap();
        let result = handler.decrypt(&encrypted, Some(wrong_aad));

        assert!(result.is_err());
    }

    #[test]
    fn test_invalid_dek_length() {
        let short_dek = [0u8; 16];
        let result = FallbackHandler::new(&short_dek, 1);
        assert!(result.is_err());
    }

    #[test]
    fn test_encrypted_data_is_local_fallback() {
        let handler = FallbackHandler::new(&test_dek(), 1).unwrap();
        let encrypted = handler.encrypt(b"test", None).unwrap();

        assert!(encrypted.is_local_fallback());
        assert_eq!(encrypted.algorithm, "AES-256-GCM");
    }

    #[tokio::test]
    async fn test_pending_operations_queue() {
        let handler = FallbackHandler::new(&test_dek(), 1).unwrap();

        handler
            .queue_operation(PendingOperation::KeyRotation {
                correlation_id: "test-1".to_string(),
                requested_at: Instant::now(),
            })
            .await
            .unwrap();

        assert_eq!(handler.pending_count().await, 1);

        let pending = handler.drain_pending().await;
        assert_eq!(pending.len(), 1);
        assert_eq!(handler.pending_count().await, 0);
    }
}
