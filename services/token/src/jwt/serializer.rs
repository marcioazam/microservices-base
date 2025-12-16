use crate::error::TokenError;
use crate::jwt::claims::Claims;
use jsonwebtoken::{decode, encode, Algorithm, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};

pub struct JwtSerializer {
    algorithm: Algorithm,
}

impl JwtSerializer {
    pub fn new(algorithm: &str) -> Self {
        let alg = match algorithm {
            "RS256" => Algorithm::RS256,
            "RS384" => Algorithm::RS384,
            "RS512" => Algorithm::RS512,
            "ES256" => Algorithm::ES256,
            "ES384" => Algorithm::ES384,
            _ => Algorithm::RS256,
        };
        JwtSerializer { algorithm: alg }
    }

    pub fn serialize(&self, claims: &Claims, key: &EncodingKey, key_id: Option<&str>) -> Result<String, TokenError> {
        let mut header = Header::new(self.algorithm);
        if let Some(kid) = key_id {
            header.kid = Some(kid.to_string());
        }

        encode(&header, claims, key).map_err(|e| TokenError::JwtEncodingError(e.to_string()))
    }

    pub fn deserialize(&self, token: &str, key: &DecodingKey) -> Result<Claims, TokenError> {
        let mut validation = Validation::new(self.algorithm);
        validation.validate_exp = true;
        validation.validate_nbf = true;

        let token_data = decode::<Claims>(token, key, &validation)
            .map_err(|e| TokenError::JwtDecodingError(e.to_string()))?;

        Ok(token_data.claims)
    }

    pub fn deserialize_unverified(&self, token: &str) -> Result<Claims, TokenError> {
        let parts: Vec<&str> = token.split('.').collect();
        if parts.len() != 3 {
            return Err(TokenError::JwtDecodingError("Invalid token format".to_string()));
        }

        let payload = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            parts[1],
        )
        .map_err(|e| TokenError::JwtDecodingError(e.to_string()))?;

        serde_json::from_slice(&payload)
            .map_err(|e| TokenError::JwtDecodingError(e.to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::jwt::builder::JwtBuilder;
    use ring::signature::{Ed25519KeyPair, KeyPair};

    fn generate_test_keys() -> (EncodingKey, DecodingKey) {
        // For testing, we use a simple secret key with HS256
        let secret = b"test-secret-key-for-testing-only";
        (
            EncodingKey::from_secret(secret),
            DecodingKey::from_secret(secret),
        )
    }

    #[test]
    fn test_round_trip_hs256() {
        let serializer = JwtSerializer { algorithm: Algorithm::HS256 };
        let (encoding_key, decoding_key) = generate_test_keys();

        let claims = JwtBuilder::new("test-issuer".to_string())
            .subject("user-123".to_string())
            .audience(vec!["api".to_string()])
            .ttl_seconds(3600)
            .build()
            .unwrap();

        let token = serializer.serialize(&claims, &encoding_key, Some("key-1")).unwrap();
        let decoded = serializer.deserialize(&token, &decoding_key).unwrap();

        assert_eq!(claims.iss, decoded.iss);
        assert_eq!(claims.sub, decoded.sub);
        assert_eq!(claims.aud, decoded.aud);
        assert_eq!(claims.jti, decoded.jti);
    }
}
