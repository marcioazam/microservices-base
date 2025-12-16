use crate::error::TokenError;
use async_trait::async_trait;
use jsonwebtoken::EncodingKey;

#[async_trait]
pub trait JwtSigner: Send + Sync {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError>;
    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError>;
    fn get_key_id(&self) -> &str;
    fn get_algorithm(&self) -> &str;
}

pub struct MockSigner {
    key_id: String,
    secret: Vec<u8>,
    algorithm: String,
}

impl MockSigner {
    pub fn new(key_id: String) -> Self {
        MockSigner {
            key_id,
            secret: b"mock-secret-key-for-testing-only-32bytes!".to_vec(),
            algorithm: "HS256".to_string(),
        }
    }

    pub fn with_secret(mut self, secret: Vec<u8>) -> Self {
        self.secret = secret;
        self
    }
}

#[async_trait]
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

    fn get_key_id(&self) -> &str {
        &self.key_id
    }

    fn get_algorithm(&self) -> &str {
        &self.algorithm
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_mock_signer() {
        let signer = MockSigner::new("test-key".to_string());
        let data = b"test data to sign";
        
        let signature = signer.sign(data).await.unwrap();
        assert!(!signature.is_empty());
    }
}
