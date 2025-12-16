use crate::error::TokenError;
use crate::jwt::signer::JwtSigner;
use async_trait::async_trait;
use jsonwebtoken::EncodingKey;
use ring::hmac;

pub struct MockKms {
    key_id: String,
    secret: Vec<u8>,
}

impl MockKms {
    pub fn new(key_id: String) -> Self {
        MockKms {
            key_id,
            secret: b"mock-kms-secret-key-for-testing-purposes-only!".to_vec(),
        }
    }

    pub fn with_secret(mut self, secret: Vec<u8>) -> Self {
        self.secret = secret;
        self
    }
}

#[async_trait]
impl JwtSigner for MockKms {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        let key = hmac::Key::new(hmac::HMAC_SHA256, &self.secret);
        let signature = hmac::sign(&key, data);
        Ok(signature.as_ref().to_vec())
    }

    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError> {
        Ok(EncodingKey::from_secret(&self.secret))
    }

    fn get_key_id(&self) -> &str {
        &self.key_id
    }

    fn get_algorithm(&self) -> &str {
        "HS256"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_mock_kms_sign() {
        let kms = MockKms::new("test-key".to_string());
        let data = b"test data";
        
        let sig1 = kms.sign(data).await.unwrap();
        let sig2 = kms.sign(data).await.unwrap();
        
        // Same data should produce same signature
        assert_eq!(sig1, sig2);
    }

    #[tokio::test]
    async fn test_mock_kms_different_data() {
        let kms = MockKms::new("test-key".to_string());
        
        let sig1 = kms.sign(b"data1").await.unwrap();
        let sig2 = kms.sign(b"data2").await.unwrap();
        
        // Different data should produce different signatures
        assert_ne!(sig1, sig2);
    }
}
