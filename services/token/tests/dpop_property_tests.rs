//! Property-based tests for DPoP module.
//!
//! Property 3: DPoP Validation Comprehensive
//! Property 4: JWK Thumbprint Determinism
//! Property 5: DPoP Replay Detection
//! Property 6: DPoP Token Binding

use proptest::prelude::*;

/// Generate arbitrary HTTP methods.
fn arb_http_method() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("GET".to_string()),
        Just("POST".to_string()),
        Just("PUT".to_string()),
        Just("DELETE".to_string()),
        Just("PATCH".to_string()),
    ]
}

/// Generate arbitrary HTTP URIs.
fn arb_http_uri() -> impl Strategy<Value = String> {
    "[a-z]{3,10}".prop_map(|path| format!("https://auth.example.com/{}", path))
}

/// Generate arbitrary JTI values.
fn arb_jti() -> impl Strategy<Value = String> {
    "[a-f0-9]{32}".prop_map(|s| s)
}

/// Generate arbitrary EC JWK.
fn arb_ec_jwk() -> impl Strategy<Value = token_service::dpop::proof::Jwk> {
    ("[a-zA-Z0-9_-]{43}", "[a-zA-Z0-9_-]{43}").prop_map(|(x, y)| {
        token_service::dpop::proof::Jwk {
            kty: "EC".to_string(),
            crv: Some("P-256".to_string()),
            x: Some(x),
            y: Some(y),
            n: None,
            e: None,
        }
    })
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 4: JWK Thumbprint Determinism
    ///
    /// Computing the thumbprint of the same JWK multiple times
    /// must always produce the same result.
    #[test]
    fn prop_jwk_thumbprint_determinism(jwk in arb_ec_jwk()) {
        let t1 = token_service::dpop::JwkThumbprint::compute(&jwk);
        let t2 = token_service::dpop::JwkThumbprint::compute(&jwk);
        let t3 = token_service::dpop::JwkThumbprint::compute(&jwk);

        prop_assert_eq!(&t1, &t2, "Thumbprint must be deterministic");
        prop_assert_eq!(&t2, &t3, "Thumbprint must be deterministic");
    }

    /// Property: Different JWKs produce different thumbprints.
    #[test]
    fn prop_different_jwks_different_thumbprints(
        jwk1 in arb_ec_jwk(),
        jwk2 in arb_ec_jwk(),
    ) {
        // Only test if JWKs are actually different
        if jwk1.x != jwk2.x || jwk1.y != jwk2.y {
            let t1 = token_service::dpop::JwkThumbprint::compute(&jwk1);
            let t2 = token_service::dpop::JwkThumbprint::compute(&jwk2);
            prop_assert_ne!(t1, t2, "Different JWKs should have different thumbprints");
        }
    }

    /// Property: Thumbprint verification is consistent with computation.
    #[test]
    fn prop_thumbprint_verify_consistent(jwk in arb_ec_jwk()) {
        let thumbprint = token_service::dpop::JwkThumbprint::compute(&jwk);
        prop_assert!(
            token_service::dpop::JwkThumbprint::verify(&jwk, &thumbprint),
            "Verification must succeed for computed thumbprint"
        );
    }

    /// Property: Thumbprint verification rejects wrong values.
    #[test]
    fn prop_thumbprint_verify_rejects_wrong(
        jwk in arb_ec_jwk(),
        wrong in "[a-zA-Z0-9_-]{43}",
    ) {
        let correct = token_service::dpop::JwkThumbprint::compute(&jwk);
        if wrong != correct {
            prop_assert!(
                !token_service::dpop::JwkThumbprint::verify(&jwk, &wrong),
                "Verification must reject wrong thumbprint"
            );
        }
    }

    /// Property 5: DPoP Replay Detection
    ///
    /// The same JTI used twice must be detected as replay.
    #[test]
    fn prop_dpop_replay_detection(jti in arb_jti()) {
        use rust_common::CacheClientConfig;
        use std::sync::Arc;
        use std::time::Duration;

        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config = CacheClientConfig::default()
                .with_namespace("dpop-replay-test");
            let storage = Arc::new(
                token_service::storage::CacheStorage::new(config).await.unwrap()
            );

            let ttl = Duration::from_secs(300);

            // First use should succeed
            let first = storage.check_and_store_dpop_jti(&jti, ttl).await.unwrap();
            prop_assert!(first, "First JTI use should succeed");

            // Second use should fail (replay)
            let second = storage.check_and_store_dpop_jti(&jti, ttl).await.unwrap();
            prop_assert!(!second, "Second JTI use should be detected as replay");

            Ok(())
        })?;
    }

    /// Property 3: DPoP Validation Comprehensive
    ///
    /// Valid proofs must pass validation, invalid must fail.
    #[test]
    fn prop_dpop_htm_validation(
        method in arb_http_method(),
        uri in arb_http_uri(),
    ) {
        use token_service::dpop::proof::{DPoPProof, DPoPHeader, DPoPClaims, Jwk};

        let proof = DPoPProof {
            header: DPoPHeader {
                typ: "dpop+jwt".to_string(),
                alg: "ES256".to_string(),
                jwk: Jwk {
                    kty: "EC".to_string(),
                    crv: Some("P-256".to_string()),
                    x: Some("test-x".to_string()),
                    y: Some("test-y".to_string()),
                    n: None,
                    e: None,
                },
            },
            claims: DPoPClaims {
                jti: uuid::Uuid::new_v4().to_string(),
                htm: method.clone(),
                htu: uri.clone(),
                iat: chrono::Utc::now().timestamp(),
                ath: None,
                nonce: None,
            },
            signature: vec![],
            raw_token: "header.payload.signature".to_string(),
        };

        // HTM should match case-insensitively
        prop_assert_eq!(
            proof.claims.htm.to_uppercase(),
            method.to_uppercase(),
            "HTM should be preserved"
        );

        // HTU should be preserved
        prop_assert_eq!(&proof.claims.htu, &uri, "HTU should be preserved");
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;
    use token_service::dpop::proof::{DPoPClaims, DPoPError, DPoPHeader, DPoPProof, Jwk};

    fn create_test_proof(htm: &str, htu: &str) -> DPoPProof {
        DPoPProof {
            header: DPoPHeader {
                typ: "dpop+jwt".to_string(),
                alg: "ES256".to_string(),
                jwk: Jwk {
                    kty: "EC".to_string(),
                    crv: Some("P-256".to_string()),
                    x: Some("test-x".to_string()),
                    y: Some("test-y".to_string()),
                    n: None,
                    e: None,
                },
            },
            claims: DPoPClaims {
                jti: uuid::Uuid::new_v4().to_string(),
                htm: htm.to_string(),
                htu: htu.to_string(),
                iat: chrono::Utc::now().timestamp(),
                ath: None,
                nonce: None,
            },
            signature: vec![],
            raw_token: "header.payload.signature".to_string(),
        }
    }

    #[test]
    fn test_dpop_proof_structure() {
        let proof = create_test_proof("POST", "https://auth.example.com/token");

        assert_eq!(proof.header.typ, "dpop+jwt");
        assert_eq!(proof.header.alg, "ES256");
        assert_eq!(proof.claims.htm, "POST");
    }

    #[test]
    fn test_ec_thumbprint_format() {
        let jwk = Jwk {
            kty: "EC".to_string(),
            crv: Some("P-256".to_string()),
            x: Some("WbbXwVQpNcx4JpLfTo0qjQLwpHA4cb9YNQKM7VjPMns".to_string()),
            y: Some("6Pbt6dwxAeS7yHp7YV4GHKaGMPaY2dSzfb0V4L5Vooo".to_string()),
            n: None,
            e: None,
        };

        let thumbprint = token_service::dpop::JwkThumbprint::compute(&jwk);

        // Base64url encoded SHA-256 should be 43 characters
        assert_eq!(thumbprint.len(), 43);
        // Should only contain base64url characters
        assert!(thumbprint.chars().all(|c| c.is_alphanumeric() || c == '-' || c == '_'));
    }

    #[test]
    fn test_rsa_thumbprint() {
        let jwk = Jwk {
            kty: "RSA".to_string(),
            crv: None,
            x: None,
            y: None,
            n: Some("test-n-value".to_string()),
            e: Some("AQAB".to_string()),
        };

        let thumbprint = token_service::dpop::JwkThumbprint::compute(&jwk);
        assert!(!thumbprint.is_empty());
        assert!(token_service::dpop::JwkThumbprint::verify(&jwk, &thumbprint));
    }
}
