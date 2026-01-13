//! JWT signing traits and implementations.
//!
//! Uses native async traits (Rust 2024 edition).

use crate::error::TokenError;
use jsonwebtoken::EncodingKey;
use std::future::Future;

/// JWT signer trait with native async (no async-trait crate).
pub trait JwtSigner: Send + Sync {
    /// Sign data and return signature bytes.
    fn sign(&self, data: &[u8]) -> impl Future<Output = Result<Vec<u8>, TokenError>> + Send;

    /// Get the encoding key for JWT serialization.
    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError>;

    /// Get the key ID for JWT header.
    fn key_id(&self) -> &str;

    /// Get the algorithm name for JWT header.
    fn algorithm(&self) -> &str;
}

/// Mock signer for testing and development.
pub struct MockSigner {
    key_id: String,
    secret: Vec<u8>,
    algorithm: String,
}

impl MockSigner {
    /// Create a new mock signer with default secret.
    #[must_use]
    pub fn new(key_id: impl Into<String>) -> Self {
        Self {
            key_id: key_id.into(),
            secret: b"mock-secret-key-for-testing-only-32bytes!".to_vec(),
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

impl JwtSigner for MockSigner {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        use ring::hmac;
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
    async fn test_mock_signer_sign() {
        let signer = MockSigner::new("test-key");
        let data = b"test data to sign";

        let signature = signer.sign(data).await.unwrap();
        assert!(!signature.is_empty());
    }

    #[tokio::test]
    async fn test_mock_signer_deterministic() {
        let signer = MockSigner::new("test-key");
        let data = b"same data";

        let sig1 = signer.sign(data).await.unwrap();
        let sig2 = signer.sign(data).await.unwrap();

        assert_eq!(sig1, sig2);
    }

    #[test]
    fn test_mock_signer_encoding_key() {
        let signer = MockSigner::new("test-key");
        let key = signer.get_encoding_key();
        assert!(key.is_ok());
    }

    #[test]
    fn test_mock_signer_metadata() {
        let signer = MockSigner::new("my-key").with_algorithm("HS384");

        assert_eq!(signer.key_id(), "my-key");
        assert_eq!(signer.algorithm(), "HS384");
    }
}
