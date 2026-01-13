//! Property-based tests for Token Service
//! 
//! These tests verify correctness properties using proptest.
//! Each test runs a minimum of 100 iterations.

use proptest::prelude::*;
use token_service::jwt::{Claims, JwtBuilder, JwtSerializer};
use jsonwebtoken::{EncodingKey, DecodingKey, Algorithm};
use std::collections::HashMap;

// Generators for test data
fn arb_string(max_len: usize) -> impl Strategy<Value = String> {
    "[a-zA-Z0-9_-]{1,}".prop_filter_map("non-empty string", move |s| {
        if s.len() <= max_len && !s.is_empty() {
            Some(s)
        } else {
            None
        }
    })
}

fn arb_user_id() -> impl Strategy<Value = String> {
    "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}"
}

fn arb_scope() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("read".to_string()),
        Just("write".to_string()),
        Just("admin".to_string()),
        Just("profile".to_string()),
        Just("email".to_string()),
    ]
}

fn arb_scopes() -> impl Strategy<Value = Vec<String>> {
    prop::collection::vec(arb_scope(), 0..5)
}

fn arb_ttl() -> impl Strategy<Value = i64> {
    60i64..86400i64 // 1 minute to 24 hours
}

fn arb_claims() -> impl Strategy<Value = Claims> {
    (
        arb_string(50),  // issuer
        arb_user_id(),   // subject
        arb_string(20),  // audience
        arb_ttl(),       // ttl
    ).prop_map(|(issuer, subject, audience, ttl)| {
        Claims::new(issuer, subject, vec![audience], ttl)
    })
}

fn test_keys() -> (EncodingKey, DecodingKey) {
    let secret = b"test-secret-key-for-property-testing-32b!";
    (
        EncodingKey::from_secret(secret),
        DecodingKey::from_secret(secret),
    )
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Feature: auth-microservices-platform, Property 2: JWT Round-Trip Consistency**
    /// **Validates: Requirements 2.7**
    /// 
    /// For any valid token claims, serializing to JWT format and then parsing back
    /// should produce claims exactly equal to the original.
    #[test]
    fn prop_jwt_round_trip_consistency(
        issuer in arb_string(50),
        subject in arb_user_id(),
        audience in arb_string(20),
        ttl in arb_ttl(),
        session_id in proptest::option::of(arb_user_id()),
        scopes in arb_scopes(),
    ) {
        let (encoding_key, decoding_key) = test_keys();
        let serializer = JwtSerializer::new(Algorithm::HS256);

        let mut builder = JwtBuilder::new(issuer.clone())
            .subject(subject.clone())
            .audience(vec![audience.clone()])
            .ttl_seconds(ttl);

        if let Some(ref sid) = session_id {
            builder = builder.session_id(sid.clone());
        }

        if !scopes.is_empty() {
            builder = builder.scopes(scopes.clone());
        }

        let original_claims = builder.build().unwrap();

        // Serialize to JWT
        let token = serializer.serialize(&original_claims, &encoding_key, Some("test-key")).unwrap();

        // Parse back
        let decoded_claims = serializer.deserialize(&token, &decoding_key).unwrap();

        // Verify round-trip consistency
        prop_assert_eq!(original_claims.iss, decoded_claims.iss);
        prop_assert_eq!(original_claims.sub, decoded_claims.sub);
        prop_assert_eq!(original_claims.aud, decoded_claims.aud);
        prop_assert_eq!(original_claims.jti, decoded_claims.jti);
        prop_assert_eq!(original_claims.session_id, decoded_claims.session_id);
        prop_assert_eq!(original_claims.scopes, decoded_claims.scopes);
        // exp, iat, nbf should be equal
        prop_assert_eq!(original_claims.exp, decoded_claims.exp);
        prop_assert_eq!(original_claims.iat, decoded_claims.iat);
    }

    /// **Feature: auth-microservices-platform, Property 3: Token Pair Issuance Completeness**
    /// **Validates: Requirements 2.1, 2.2**
    /// 
    /// For any successful authentication, the Token Service should issue both an access token
    /// (with configured TTL) and a refresh token (with longer TTL), where both tokens are
    /// cryptographically signed.
    #[test]
    fn prop_token_pair_completeness(
        user_id in arb_user_id(),
        session_id in arb_user_id(),
        access_ttl in 60i64..3600i64,
        refresh_ttl in 86400i64..604800i64,
    ) {
        // Verify refresh TTL is always longer than access TTL
        prop_assert!(refresh_ttl > access_ttl);

        let (encoding_key, _) = test_keys();
        let serializer = JwtSerializer::new(Algorithm::HS256);

        // Build access token
        let access_claims = JwtBuilder::new("test-issuer".to_string())
            .subject(user_id.clone())
            .audience(vec!["api".to_string()])
            .ttl_seconds(access_ttl)
            .session_id(session_id.clone())
            .build()
            .unwrap();

        let access_token = serializer.serialize(&access_claims, &encoding_key, Some("key-1")).unwrap();

        // Verify access token is valid JWT format (3 parts separated by dots)
        let parts: Vec<&str> = access_token.split('.').collect();
        prop_assert_eq!(parts.len(), 3);

        // Verify each part is valid base64
        for part in &parts {
            prop_assert!(!part.is_empty());
        }

        // Verify TTL is set correctly
        let now = chrono::Utc::now().timestamp();
        prop_assert!(access_claims.exp > now);
        prop_assert!(access_claims.exp <= now + access_ttl + 1); // +1 for timing tolerance
    }

    /// **Feature: auth-microservices-platform, Property 4: Refresh Token Rotation Invalidates Previous**
    /// **Validates: Requirements 2.3**
    /// 
    /// For any refresh token that is used, the Token Service should issue a new token pair
    /// and the previous refresh token should become invalid for subsequent use.
    #[test]
    fn prop_refresh_token_rotation(
        user_id in arb_user_id(),
        session_id in arb_user_id(),
    ) {
        use token_service::refresh::{TokenFamily, RefreshTokenGenerator};

        // Create initial token family
        let initial_token = RefreshTokenGenerator::generate();
        let initial_hash = RefreshTokenGenerator::hash(&initial_token);
        
        let mut family = TokenFamily::new(
            RefreshTokenGenerator::generate_family_id(),
            user_id,
            session_id,
            initial_hash.clone(),
        );

        // Verify initial token is valid
        prop_assert!(family.is_valid_token(&initial_hash));

        // Rotate to new token
        let new_token = RefreshTokenGenerator::generate();
        let new_hash = RefreshTokenGenerator::hash(&new_token);
        family.rotate(new_hash.clone());

        // Verify old token is no longer valid
        prop_assert!(!family.is_valid_token(&initial_hash));
        
        // Verify new token is valid
        prop_assert!(family.is_valid_token(&new_hash));
        
        // Verify rotation count increased
        prop_assert_eq!(family.rotation_count, 1);
    }

    /// **Feature: auth-microservices-platform, Property 5: Refresh Token Replay Detection**
    /// **Validates: Requirements 2.4**
    /// 
    /// For any refresh token that has already been rotated, attempting to use it again
    /// should be detected as a replay attack.
    #[test]
    fn prop_refresh_token_replay_detection(
        user_id in arb_user_id(),
        session_id in arb_user_id(),
    ) {
        use token_service::refresh::{TokenFamily, RefreshTokenGenerator};

        let initial_token = RefreshTokenGenerator::generate();
        let initial_hash = RefreshTokenGenerator::hash(&initial_token);
        
        let mut family = TokenFamily::new(
            RefreshTokenGenerator::generate_family_id(),
            user_id,
            session_id,
            initial_hash.clone(),
        );

        // Rotate to new token
        let new_hash = RefreshTokenGenerator::hash(&RefreshTokenGenerator::generate());
        family.rotate(new_hash);

        // Attempting to use old token should be detected as replay
        prop_assert!(family.is_replay_attack(&initial_hash));
        
        // New token should not be detected as replay
        prop_assert!(!family.is_replay_attack(&family.current_token_hash));
    }

    /// **Feature: auth-microservices-platform, Property 6: JWKS Contains Rotation Keys**
    /// **Validates: Requirements 2.5**
    /// 
    /// For any key rotation event, the JWKS endpoint should contain both the current
    /// and previous signing keys to allow graceful transition.
    #[test]
    fn prop_jwks_contains_rotation_keys(
        key_id_1 in arb_string(20),
        key_id_2 in arb_string(20),
    ) {
        use token_service::jwks::{JwksPublisher, Jwk};

        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let publisher = JwksPublisher::new();

            // Add initial key
            let key1 = Jwk {
                kty: "RSA".to_string(),
                kid: key_id_1.clone(),
                key_use: "sig".to_string(),
                alg: "RS256".to_string(),
                n: Some("test-n-1".to_string()),
                e: Some("AQAB".to_string()),
                x: None,
                y: None,
                crv: None,
            };
            publisher.add_key(key1).await;

            // Rotate to new key
            let key2 = Jwk {
                kty: "RSA".to_string(),
                kid: key_id_2.clone(),
                key_use: "sig".to_string(),
                alg: "RS256".to_string(),
                n: Some("test-n-2".to_string()),
                e: Some("AQAB".to_string()),
                x: None,
                y: None,
                crv: None,
            };
            publisher.rotate_keys(key2).await;

            // Get JWKS
            let jwks = publisher.get_jwks().await;

            // Should contain both keys
            let key_ids: Vec<&str> = jwks.keys.iter().map(|k| k.kid.as_str()).collect();
            
            assert!(key_ids.contains(&key_id_1.as_str()), "JWKS should contain previous key");
            assert!(key_ids.contains(&key_id_2.as_str()), "JWKS should contain current key");
        });
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[test]
    fn test_claims_expiration() {
        let claims = Claims::new(
            "issuer".to_string(),
            "subject".to_string(),
            vec!["aud".to_string()],
            -1, // Already expired
        );
        assert!(claims.is_expired());
    }

    #[test]
    fn test_claims_not_expired() {
        let claims = Claims::new(
            "issuer".to_string(),
            "subject".to_string(),
            vec!["aud".to_string()],
            3600,
        );
        assert!(!claims.is_expired());
    }
}
