use rand::Rng;
use sha2::{Sha256, Digest};
use base64::{Engine as _, engine::general_purpose::URL_SAFE_NO_PAD};

pub struct RefreshTokenGenerator;

impl RefreshTokenGenerator {
    pub fn generate() -> String {
        let mut rng = rand::thread_rng();
        let random_bytes: [u8; 32] = rng.gen();
        URL_SAFE_NO_PAD.encode(random_bytes)
    }

    pub fn hash(token: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(token.as_bytes());
        let result = hasher.finalize();
        URL_SAFE_NO_PAD.encode(result)
    }

    pub fn generate_family_id() -> String {
        uuid::Uuid::new_v4().to_string()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_unique_tokens() {
        let token1 = RefreshTokenGenerator::generate();
        let token2 = RefreshTokenGenerator::generate();
        assert_ne!(token1, token2);
        assert_eq!(token1.len(), 43); // Base64 encoded 32 bytes
    }

    #[test]
    fn test_hash_deterministic() {
        let token = "test-token";
        let hash1 = RefreshTokenGenerator::hash(token);
        let hash2 = RefreshTokenGenerator::hash(token);
        assert_eq!(hash1, hash2);
    }

    #[test]
    fn test_hash_different_for_different_tokens() {
        let hash1 = RefreshTokenGenerator::hash("token1");
        let hash2 = RefreshTokenGenerator::hash("token2");
        assert_ne!(hash1, hash2);
    }
}
