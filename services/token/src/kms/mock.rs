//! Mock KMS implementation for testing and development.

use crate::error::TokenError;
use crate::kms::KmsSigner;
use async_trait::async_trait;
use jsonwebtoken::EncodingKey;
use ring::hmac;

/// Mock KMS for testing and development.
pub struct MockKms {
    key_id: String,
    secret: Vec<u8>,
    algorithm: String,
}

impl MockKms {
    /// Create a new mock KMS with default secret.
    #[must_use]
    pub fn new(key_id: impl Into<String>) -> Self {
        Self {
            key_id: key_id.into(),
            secret: b"mock-kms-secret-key-for-testing-purposes-only!".to_vec(),
            algorithm: "HS256".to_string(),
        }
    }

    /// Set a custom secret.
    #[must_use]
    pub fn with_secret(mut self, secret: Vec<u8>) -> Self {
        self.secret = secret;
        self
    }

    /// Set the algorithm.
    #[must_use]
    pub fn with_algorithm(mut self, algorithm: impl Into<String>) -> Self {
        self.algorithm = algorithm.into();
        self
    }
}

#[async_trait]
impl KmsSigner for MockKms {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        let key = hmac::Key::new(hmac::HMAC_SHA256, &self.secret);
        let signature = hmac::sign(&key, data);
        Ok(signature.as_ref().to_vec())
    }

    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError> {
        Ok(EncodingKey::from_secret(&self.secret))
    }

    fn key_id(&self) -> &str {
        &self.key_id
    }

    fn algorithm(&self) -> &str {
        &self.algorithm
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_mock_kms_sign() {
        let kms = MockKms::new("test-key");
        let data = b"test data";

        let sig1 = kms.sign(data).await.unwrap();
        let sig2 = kms.sign(data).await.unwrap();

        assert_eq!(sig1, sig2, "Same data should produce same signature");
    }

    #[tokio::test]
    async fn test_mock_kms_different_data() {
        let kms = MockKms::new("test-key");

        let sig1 = kms.sign(b"data1").await.unwrap();
        let sig2 = kms.sign(b"data2").await.unwrap();

        assert_ne!(sig1, sig2, "Different data should produce different signatures");
    }

    #[test]
    fn test_mock_kms_encoding_key() {
        let kms = MockKms::new("test-key");
        let key = kms.get_encoding_key();
        assert!(key.is_ok());
    }

    #[test]
    fn test_mock_kms_metadata() {
        let kms = MockKms::new("my-key").with_algorithm("HS384");

        assert_eq!(kms.key_id(), "my-key");
        assert_eq!(kms.algorithm(), "HS384");
    }
}
