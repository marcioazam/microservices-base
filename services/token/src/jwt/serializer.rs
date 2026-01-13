//! JWT serialization and deserialization.

use crate::error::TokenError;
use crate::jwt::claims::Claims;
use jsonwebtoken::{decode, encode, Algorithm, DecodingKey, EncodingKey, Header, Validation};

/// JWT serializer with configurable algorithm.
pub struct JwtSerializer {
    algorithm: Algorithm,
}

impl JwtSerializer {
    /// Create a new serializer with the specified algorithm.
    #[must_use]
    pub const fn new(algorithm: Algorithm) -> Self {
        Self { algorithm }
    }

    /// Create serializer from algorithm string.
    #[must_use]
    pub fn from_str(algorithm: &str) -> Self {
        let alg = match algorithm.to_uppercase().as_str() {
            "RS256" => Algorithm::RS256,
            "RS384" => Algorithm::RS384,
            "RS512" => Algorithm::RS512,
            "PS256" => Algorithm::PS256,
            "PS384" => Algorithm::PS384,
            "PS512" => Algorithm::PS512,
            "ES256" => Algorithm::ES256,
            "ES384" => Algorithm::ES384,
            "HS256" => Algorithm::HS256,
            "HS384" => Algorithm::HS384,
            "HS512" => Algorithm::HS512,
            _ => Algorithm::RS256,
        };
        Self { algorithm: alg }
    }

    /// Serialize claims to a JWT string.
    pub fn serialize(
        &self,
        claims: &Claims,
        key: &EncodingKey,
        key_id: Option<&str>,
    ) -> Result<String, TokenError> {
        let mut header = Header::new(self.algorithm);
        if let Some(kid) = key_id {
            header.kid = Some(kid.to_string());
        }

        encode(&header, claims, key).map_err(|e| TokenError::jwt_encoding(e.to_string()))
    }

    /// Deserialize and verify a JWT string.
    pub fn deserialize(&self, token: &str, key: &DecodingKey) -> Result<Claims, TokenError> {
        let mut validation = Validation::new(self.algorithm);
        validation.validate_exp = true;
        validation.validate_nbf = true;
        // Disable audience validation - we'll validate manually if needed
        validation.validate_aud = false;

        let token_data = decode::<Claims>(token, key, &validation)
            .map_err(|e| TokenError::jwt_decoding(e.to_string()))?;

        Ok(token_data.claims)
    }

    /// Deserialize without signature verification (for inspection only).
    ///
    /// # Security Warning
    ///
    /// This method does NOT verify the signature. Only use for
    /// extracting claims when you need to look up the key.
    pub fn deserialize_unverified(&self, token: &str) -> Result<Claims, TokenError> {
        let parts: Vec<&str> = token.split('.').collect();
        if parts.len() != 3 {
            return Err(TokenError::jwt_decoding("Invalid token format"));
        }

        let payload = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            parts[1],
        )
        .map_err(|e| TokenError::jwt_decoding(e.to_string()))?;

        serde_json::from_slice(&payload).map_err(|e| TokenError::jwt_decoding(e.to_string()))
    }

    /// Get the algorithm.
    #[must_use]
    pub const fn algorithm(&self) -> Algorithm {
        self.algorithm
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::jwt::builder::JwtBuilder;

    fn generate_test_keys() -> (EncodingKey, DecodingKey) {
        let secret = b"test-secret-key-for-testing-only";
        (
            EncodingKey::from_secret(secret),
            DecodingKey::from_secret(secret),
        )
    }

    #[test]
    fn test_round_trip_hs256() {
        let serializer = JwtSerializer::new(Algorithm::HS256);
        let (encoding_key, decoding_key) = generate_test_keys();

        let claims = JwtBuilder::new("test-issuer".to_string())
            .subject("user-123".to_string())
            .audience(vec!["api".to_string()])
            .ttl_seconds(3600)
            .build()
            .unwrap();

        let token = serializer
            .serialize(&claims, &encoding_key, Some("key-1"))
            .unwrap();
        let decoded = serializer.deserialize(&token, &decoding_key).unwrap();

        assert_eq!(claims.iss, decoded.iss);
        assert_eq!(claims.sub, decoded.sub);
        assert_eq!(claims.aud, decoded.aud);
        assert_eq!(claims.jti, decoded.jti);
    }

    #[test]
    fn test_deserialize_unverified() {
        let serializer = JwtSerializer::new(Algorithm::HS256);
        let (encoding_key, _) = generate_test_keys();

        let claims = JwtBuilder::new("test-issuer".to_string())
            .subject("user-123".to_string())
            .audience(vec!["api".to_string()])
            .build()
            .unwrap();

        let token = serializer.serialize(&claims, &encoding_key, None).unwrap();
        let decoded = serializer.deserialize_unverified(&token).unwrap();

        assert_eq!(claims.sub, decoded.sub);
    }

    #[test]
    fn test_from_str() {
        assert_eq!(JwtSerializer::from_str("RS256").algorithm(), Algorithm::RS256);
        assert_eq!(JwtSerializer::from_str("es256").algorithm(), Algorithm::ES256);
        assert_eq!(JwtSerializer::from_str("HS256").algorithm(), Algorithm::HS256);
    }

    #[test]
    fn test_invalid_token_format() {
        let serializer = JwtSerializer::new(Algorithm::HS256);
        let result = serializer.deserialize_unverified("invalid");
        assert!(result.is_err());
    }
}
