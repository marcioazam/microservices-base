//! JWK Thumbprint calculation per RFC 7638
//!
//! Used for DPoP token binding (cnf.jkt claim)

use crate::dpop::proof::Jwk;
use sha2::{Sha256, Digest};
use serde_json::json;

/// Calculates the JWK thumbprint per RFC 7638
pub struct JwkThumbprint;

impl JwkThumbprint {
    /// Computes the SHA-256 thumbprint of a JWK
    /// 
    /// Per RFC 7638, the thumbprint is computed over the required
    /// members of the JWK in lexicographic order.
    pub fn compute(jwk: &Jwk) -> String {
        let canonical = Self::canonical_json(jwk);
        let hash = Sha256::digest(canonical.as_bytes());
        base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, hash)
    }

    /// Creates the canonical JSON representation for thumbprint calculation
    fn canonical_json(jwk: &Jwk) -> String {
        match jwk.kty.as_str() {
            "EC" => {
                // For EC keys: crv, kty, x, y (lexicographic order)
                json!({
                    "crv": jwk.crv.as_ref().unwrap_or(&"".to_string()),
                    "kty": "EC",
                    "x": jwk.x.as_ref().unwrap_or(&"".to_string()),
                    "y": jwk.y.as_ref().unwrap_or(&"".to_string())
                }).to_string()
            }
            "RSA" => {
                // For RSA keys: e, kty, n (lexicographic order)
                json!({
                    "e": jwk.e.as_ref().unwrap_or(&"".to_string()),
                    "kty": "RSA",
                    "n": jwk.n.as_ref().unwrap_or(&"".to_string())
                }).to_string()
            }
            _ => {
                // Fallback: include all non-null fields
                serde_json::to_string(jwk).unwrap_or_default()
            }
        }
    }

    /// Verifies that a thumbprint matches a JWK
    pub fn verify(jwk: &Jwk, expected_thumbprint: &str) -> bool {
        let computed = Self::compute(jwk);
        // Constant-time comparison
        Self::constant_time_eq(computed.as_bytes(), expected_thumbprint.as_bytes())
    }

    /// Constant-time comparison to prevent timing attacks
    fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
        if a.len() != b.len() {
            return false;
        }
        
        let mut result = 0u8;
        for (x, y) in a.iter().zip(b.iter()) {
            result |= x ^ y;
        }
        result == 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ec_thumbprint() {
        let jwk = Jwk {
            kty: "EC".to_string(),
            crv: Some("P-256".to_string()),
            x: Some("WbbXwVQpNcx4JpLfTo0qjQLwpHA4cb9YNQKM7VjPMns".to_string()),
            y: Some("6Pbt6dwxAeS7yHp7YV4GHKaGMPaY2dSzfb0V4L5Vooo".to_string()),
            n: None,
            e: None,
        };

        let thumbprint = JwkThumbprint::compute(&jwk);
        assert!(!thumbprint.is_empty());
        
        // Verify the thumbprint matches
        assert!(JwkThumbprint::verify(&jwk, &thumbprint));
    }

    #[test]
    fn test_rsa_thumbprint() {
        let jwk = Jwk {
            kty: "RSA".to_string(),
            crv: None,
            x: None,
            y: None,
            n: Some("0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw".to_string()),
            e: Some("AQAB".to_string()),
        };

        let thumbprint = JwkThumbprint::compute(&jwk);
        assert!(!thumbprint.is_empty());
        
        // Verify the thumbprint matches
        assert!(JwkThumbprint::verify(&jwk, &thumbprint));
    }

    #[test]
    fn test_thumbprint_mismatch() {
        let jwk = Jwk {
            kty: "EC".to_string(),
            crv: Some("P-256".to_string()),
            x: Some("test-x".to_string()),
            y: Some("test-y".to_string()),
            n: None,
            e: None,
        };

        assert!(!JwkThumbprint::verify(&jwk, "wrong-thumbprint"));
    }
}
