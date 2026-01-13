//! JWK Thumbprint calculation per RFC 7638.
//!
//! Used for DPoP token binding (cnf.jkt claim).
//! Uses constant-time comparison for security.

use crate::dpop::proof::Jwk;
use sha2::{Digest, Sha256};
use subtle::ConstantTimeEq;

/// Calculates the JWK thumbprint per RFC 7638.
pub struct JwkThumbprint;

impl JwkThumbprint {
    /// Computes the SHA-256 thumbprint of a JWK.
    ///
    /// Per RFC 7638, the thumbprint is computed over the required
    /// members of the JWK in lexicographic order.
    #[must_use]
    pub fn compute(jwk: &Jwk) -> String {
        let canonical = Self::canonical_json(jwk);
        let hash = Sha256::digest(canonical.as_bytes());
        base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, hash)
    }

    /// Creates the canonical JSON representation for thumbprint calculation.
    ///
    /// Per RFC 7638, members must be in lexicographic order with no whitespace.
    fn canonical_json(jwk: &Jwk) -> String {
        match jwk.kty.as_str() {
            "EC" => {
                // For EC keys: crv, kty, x, y (lexicographic order)
                format!(
                    r#"{{"crv":"{}","kty":"EC","x":"{}","y":"{}"}}"#,
                    jwk.crv.as_deref().unwrap_or(""),
                    jwk.x.as_deref().unwrap_or(""),
                    jwk.y.as_deref().unwrap_or("")
                )
            }
            "RSA" => {
                // For RSA keys: e, kty, n (lexicographic order)
                format!(
                    r#"{{"e":"{}","kty":"RSA","n":"{}"}}"#,
                    jwk.e.as_deref().unwrap_or(""),
                    jwk.n.as_deref().unwrap_or("")
                )
            }
            _ => {
                // Fallback: include all non-null fields
                serde_json::to_string(jwk).unwrap_or_default()
            }
        }
    }

    /// Verifies that a thumbprint matches a JWK using constant-time comparison.
    ///
    /// Uses `subtle::ConstantTimeEq` to prevent timing attacks.
    #[must_use]
    pub fn verify(jwk: &Jwk, expected_thumbprint: &str) -> bool {
        let computed = Self::compute(jwk);
        let computed_bytes = computed.as_bytes();
        let expected_bytes = expected_thumbprint.as_bytes();

        // Length check first (this leaks length but that's acceptable)
        if computed_bytes.len() != expected_bytes.len() {
            return false;
        }

        // Constant-time comparison
        computed_bytes.ct_eq(expected_bytes).into()
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
        assert!(JwkThumbprint::verify(&jwk, &thumbprint));
    }

    #[test]
    fn test_rsa_thumbprint() {
        let jwk = Jwk {
            kty: "RSA".to_string(),
            crv: None,
            x: None,
            y: None,
            n: Some("0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx".to_string()),
            e: Some("AQAB".to_string()),
        };

        let thumbprint = JwkThumbprint::compute(&jwk);
        assert!(!thumbprint.is_empty());
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

    #[test]
    fn test_thumbprint_deterministic() {
        let jwk = Jwk {
            kty: "EC".to_string(),
            crv: Some("P-256".to_string()),
            x: Some("test-x".to_string()),
            y: Some("test-y".to_string()),
            n: None,
            e: None,
        };

        let t1 = JwkThumbprint::compute(&jwk);
        let t2 = JwkThumbprint::compute(&jwk);
        assert_eq!(t1, t2);
    }

    #[test]
    fn test_canonical_json_ec() {
        let jwk = Jwk {
            kty: "EC".to_string(),
            crv: Some("P-256".to_string()),
            x: Some("x-value".to_string()),
            y: Some("y-value".to_string()),
            n: None,
            e: None,
        };

        let canonical = JwkThumbprint::canonical_json(&jwk);
        // Verify lexicographic order: crv, kty, x, y
        assert!(canonical.find("crv") < canonical.find("kty"));
        assert!(canonical.find("kty") < canonical.find("\"x\""));
        assert!(canonical.find("\"x\"") < canonical.find("\"y\""));
    }
}
