//! Property-based tests for JWT module.
//!
//! Property 1: JWT Claims Round-Trip Consistency
//! Property 2: JWT Structure Completeness

use proptest::prelude::*;
use std::collections::HashMap;

/// Generate arbitrary issuer strings.
fn arb_issuer() -> impl Strategy<Value = String> {
    "[a-zA-Z][a-zA-Z0-9-]{0,63}".prop_map(|s| s)
}

/// Generate arbitrary subject strings.
fn arb_subject() -> impl Strategy<Value = String> {
    "[a-zA-Z0-9_-]{1,128}".prop_map(|s| s)
}

/// Generate arbitrary audience lists.
fn arb_audience() -> impl Strategy<Value = Vec<String>> {
    prop::collection::vec("[a-zA-Z][a-zA-Z0-9-]{0,31}".prop_map(|s| s), 1..5)
}

/// Generate arbitrary TTL (1 minute to 24 hours).
fn arb_ttl() -> impl Strategy<Value = i64> {
    60i64..86400i64
}

/// Generate arbitrary scope lists.
fn arb_scopes() -> impl Strategy<Value = Vec<String>> {
    prop::collection::vec("[a-z]{1,16}".prop_map(|s| s), 0..10)
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 1: JWT Claims Round-Trip Consistency
    ///
    /// For any valid claims, serializing and deserializing must
    /// produce identical claims (excluding timing variations).
    #[test]
    fn prop_jwt_round_trip_consistency(
        issuer in arb_issuer(),
        subject in arb_subject(),
        audience in arb_audience(),
        ttl in arb_ttl(),
        scopes in arb_scopes(),
    ) {
        use jsonwebtoken::{Algorithm, DecodingKey, EncodingKey};

        let secret = b"test-secret-key-for-property-testing-32b";
        let encoding_key = EncodingKey::from_secret(secret);
        let decoding_key = DecodingKey::from_secret(secret);

        // Build claims
        let mut builder = token_service::jwt::JwtBuilder::new(issuer.clone())
            .subject(subject.clone())
            .audience(audience.clone())
            .ttl_seconds(ttl);

        if !scopes.is_empty() {
            builder = builder.scopes(scopes.clone());
        }

        let claims = builder.build().unwrap();

        // Serialize
        let serializer = token_service::jwt::JwtSerializer::new(Algorithm::HS256);
        let token = serializer.serialize(&claims, &encoding_key, Some("test-key")).unwrap();

        // Deserialize
        let decoded = serializer.deserialize(&token, &decoding_key).unwrap();

        // Verify round-trip consistency
        prop_assert_eq!(&claims.iss, &decoded.iss, "Issuer must match");
        prop_assert_eq!(&claims.sub, &decoded.sub, "Subject must match");
        prop_assert_eq!(&claims.aud, &decoded.aud, "Audience must match");
        prop_assert_eq!(&claims.jti, &decoded.jti, "JTI must match");
        prop_assert_eq!(&claims.exp, &decoded.exp, "Expiration must match");
        prop_assert_eq!(&claims.iat, &decoded.iat, "Issued-at must match");
        prop_assert_eq!(&claims.scopes, &decoded.scopes, "Scopes must match");
    }

    /// Property 2: JWT Structure Completeness
    ///
    /// All generated JWTs must have required claims and valid structure.
    #[test]
    fn prop_jwt_structure_completeness(
        issuer in arb_issuer(),
        subject in arb_subject(),
        audience in arb_audience(),
        ttl in arb_ttl(),
    ) {
        use jsonwebtoken::EncodingKey;

        let secret = b"test-secret-key-for-property-testing-32b";
        let encoding_key = EncodingKey::from_secret(secret);

        let claims = token_service::jwt::JwtBuilder::new(issuer)
            .subject(subject)
            .audience(audience)
            .ttl_seconds(ttl)
            .build()
            .unwrap();

        let serializer = token_service::jwt::JwtSerializer::new(
            jsonwebtoken::Algorithm::HS256
        );
        let token = serializer.serialize(&claims, &encoding_key, Some("key-1")).unwrap();

        // Verify JWT structure (3 parts separated by dots)
        let parts: Vec<&str> = token.split('.').collect();
        prop_assert_eq!(parts.len(), 3, "JWT must have 3 parts");

        // Verify header contains kid
        let header_json = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            parts[0],
        ).unwrap();
        let header: serde_json::Value = serde_json::from_slice(&header_json).unwrap();

        prop_assert!(header.get("alg").is_some(), "Header must have alg");
        prop_assert!(header.get("kid").is_some(), "Header must have kid");
        prop_assert_eq!(&header["kid"], "key-1", "Kid must match");

        // Verify payload has required claims
        let payload_json = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            parts[1],
        ).unwrap();
        let payload: serde_json::Value = serde_json::from_slice(&payload_json).unwrap();

        prop_assert!(payload.get("iss").is_some(), "Payload must have iss");
        prop_assert!(payload.get("sub").is_some(), "Payload must have sub");
        prop_assert!(payload.get("aud").is_some(), "Payload must have aud");
        prop_assert!(payload.get("exp").is_some(), "Payload must have exp");
        prop_assert!(payload.get("iat").is_some(), "Payload must have iat");
        prop_assert!(payload.get("jti").is_some(), "Payload must have jti");

        // Verify exp > iat
        let exp = payload["exp"].as_i64().unwrap();
        let iat = payload["iat"].as_i64().unwrap();
        prop_assert!(exp > iat, "exp must be greater than iat");
    }

    /// Property: JTI uniqueness across multiple tokens.
    #[test]
    fn prop_jti_uniqueness(
        issuer in arb_issuer(),
        subject in arb_subject(),
    ) {
        let mut jtis = std::collections::HashSet::new();

        for _ in 0..10 {
            let claims = token_service::jwt::JwtBuilder::new(issuer.clone())
                .subject(subject.clone())
                .audience(vec!["api".to_string()])
                .build()
                .unwrap();

            prop_assert!(
                jtis.insert(claims.jti.clone()),
                "JTI must be unique: {}",
                claims.jti
            );
        }
    }

    /// Property: DPoP binding is preserved through serialization.
    #[test]
    fn prop_dpop_binding_preserved(
        issuer in arb_issuer(),
        subject in arb_subject(),
        thumbprint in "[a-zA-Z0-9_-]{43}",
    ) {
        use jsonwebtoken::{Algorithm, DecodingKey, EncodingKey};

        let secret = b"test-secret-key-for-property-testing-32b";
        let encoding_key = EncodingKey::from_secret(secret);
        let decoding_key = DecodingKey::from_secret(secret);

        let claims = token_service::jwt::Claims::new(
            issuer,
            subject,
            vec!["api".to_string()],
            3600,
        ).with_dpop_binding(thumbprint.clone());

        prop_assert!(claims.is_dpop_bound(), "Claims should be DPoP bound");
        prop_assert_eq!(
            claims.dpop_thumbprint(),
            Some(thumbprint.as_str()),
            "Thumbprint should match"
        );

        let serializer = token_service::jwt::JwtSerializer::new(Algorithm::HS256);
        let token = serializer.serialize(&claims, &encoding_key, None).unwrap();
        let decoded = serializer.deserialize(&token, &decoding_key).unwrap();

        prop_assert!(decoded.is_dpop_bound(), "Decoded should be DPoP bound");
        prop_assert_eq!(
            decoded.dpop_thumbprint(),
            Some(thumbprint.as_str()),
            "Decoded thumbprint should match"
        );
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[test]
    fn test_claims_expiration() {
        let claims = token_service::jwt::Claims::new(
            "issuer".to_string(),
            "subject".to_string(),
            vec!["aud".to_string()],
            -1, // Already expired
        );

        assert!(claims.is_expired());
    }

    #[test]
    fn test_claims_not_expired() {
        let claims = token_service::jwt::Claims::new(
            "issuer".to_string(),
            "subject".to_string(),
            vec!["aud".to_string()],
            3600,
        );

        assert!(!claims.is_expired());
    }

    #[test]
    fn test_builder_with_custom_claims() {
        let claims = token_service::jwt::JwtBuilder::new("issuer".to_string())
            .subject("user".to_string())
            .audience(vec!["api".to_string()])
            .custom_claim("role".to_string(), serde_json::json!("admin"))
            .build()
            .unwrap();

        assert_eq!(
            claims.custom.get("role"),
            Some(&serde_json::json!("admin"))
        );
    }
}
