//! Property-based tests for JWKS module.
//!
//! Property 11: JWKS Key Rotation

use proptest::prelude::*;

/// Generate arbitrary key IDs.
fn arb_key_id() -> impl Strategy<Value = String> {
    "[a-zA-Z0-9_-]{8,32}".prop_map(|s| s)
}

/// Generate arbitrary algorithms.
fn arb_algorithm() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("RS256".to_string()),
        Just("RS384".to_string()),
        Just("RS512".to_string()),
        Just("ES256".to_string()),
        Just("ES384".to_string()),
        Just("PS256".to_string()),
    ]
}

fn create_test_key(kid: &str, alg: &str) -> token_service::jwks::Jwk {
    token_service::jwks::Jwk {
        kty: "RSA".to_string(),
        kid: kid.to_string(),
        key_use: "sig".to_string(),
        alg: alg.to_string(),
        n: Some("test-n".to_string()),
        e: Some("AQAB".to_string()),
        x: None,
        y: None,
        crv: None,
    }
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 11: JWKS Key Rotation
    ///
    /// After rotation, both current and previous keys must be available.
    /// Previous keys are retained for the configured period.
    #[test]
    fn prop_jwks_key_rotation(
        key1_id in arb_key_id(),
        key2_id in arb_key_id(),
        alg in arb_algorithm(),
    ) {
        // Skip if key IDs are the same
        prop_assume!(key1_id != key2_id);

        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let publisher = token_service::jwks::JwksPublisher::new();

            // Add initial key
            publisher.add_key(create_test_key(&key1_id, &alg)).await;

            let jwks1 = publisher.get_jwks().await;
            prop_assert_eq!(jwks1.keys.len(), 1, "Should have 1 key initially");
            prop_assert_eq!(&jwks1.keys[0].kid, &key1_id);

            // Rotate to new key
            publisher.rotate_keys(create_test_key(&key2_id, &alg)).await;

            let jwks2 = publisher.get_jwks().await;
            prop_assert_eq!(jwks2.keys.len(), 2, "Should have 2 keys after rotation");

            // Both keys should be present
            let key_ids: Vec<&str> = jwks2.keys.iter().map(|k| k.kid.as_str()).collect();
            prop_assert!(key_ids.contains(&key1_id.as_str()), "Previous key must be retained");
            prop_assert!(key_ids.contains(&key2_id.as_str()), "New key must be present");

            // Current key ID should be the new one
            let current = publisher.get_current_key_id().await;
            prop_assert_eq!(current, Some(key2_id.clone()), "Current key should be new key");

            Ok(())
        })?;
    }

    /// Property: Multiple rotations preserve all keys within retention.
    #[test]
    fn prop_multiple_rotations(
        key_ids in prop::collection::vec(arb_key_id(), 3..6),
    ) {
        // Ensure unique key IDs
        let unique_ids: std::collections::HashSet<_> = key_ids.iter().collect();
        prop_assume!(unique_ids.len() == key_ids.len());

        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let publisher = token_service::jwks::JwksPublisher::new();

            // Add first key
            publisher.add_key(create_test_key(&key_ids[0], "RS256")).await;

            // Rotate through remaining keys
            for kid in key_ids.iter().skip(1) {
                publisher.rotate_keys(create_test_key(kid, "RS256")).await;
            }

            let jwks = publisher.get_jwks().await;

            // All keys should be present (within retention period)
            prop_assert_eq!(
                jwks.keys.len(),
                key_ids.len(),
                "All keys should be retained"
            );

            // Current key should be the last one
            let current = publisher.get_current_key_id().await;
            prop_assert_eq!(
                current,
                Some(key_ids.last().unwrap().clone()),
                "Current key should be last rotated"
            );

            Ok(())
        })?;
    }

    /// Property: JWKS JSON serialization is valid.
    #[test]
    fn prop_jwks_json_valid(
        key_id in arb_key_id(),
        alg in arb_algorithm(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let publisher = token_service::jwks::JwksPublisher::new();
            publisher.add_key(create_test_key(&key_id, &alg)).await;

            let jwks = publisher.get_jwks().await;
            let json = jwks.to_json();

            // Should be valid JSON
            let parsed: Result<serde_json::Value, _> = serde_json::from_str(&json);
            prop_assert!(parsed.is_ok(), "JWKS JSON must be valid");

            let value = parsed.unwrap();
            prop_assert!(value.get("keys").is_some(), "Must have 'keys' array");
            prop_assert!(value["keys"].is_array(), "'keys' must be array");

            Ok(())
        })?;
    }

    /// Property: Key lookup by ID is consistent.
    #[test]
    fn prop_key_lookup_consistent(
        key_ids in prop::collection::vec(arb_key_id(), 1..5),
    ) {
        let unique_ids: std::collections::HashSet<_> = key_ids.iter().collect();
        prop_assume!(unique_ids.len() == key_ids.len());

        let mut jwks = token_service::jwks::Jwks::new();
        for kid in &key_ids {
            jwks.add_key(create_test_key(kid, "RS256"));
        }

        // All added keys should be findable
        for kid in &key_ids {
            let found = jwks.find_key(kid);
            prop_assert!(found.is_some(), "Key {} should be found", kid);
            prop_assert_eq!(&found.unwrap().kid, kid);
        }

        // Non-existent key should not be found
        let not_found = jwks.find_key("non-existent-key-id");
        prop_assert!(not_found.is_none(), "Non-existent key should not be found");
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[tokio::test]
    async fn test_empty_publisher() {
        let publisher = token_service::jwks::JwksPublisher::new();

        let jwks = publisher.get_jwks().await;
        assert!(jwks.keys.is_empty());

        let current = publisher.get_current_key_id().await;
        assert!(current.is_none());
    }

    #[tokio::test]
    async fn test_ec_key() {
        let key = token_service::jwks::Jwk {
            kty: "EC".to_string(),
            kid: "ec-key-1".to_string(),
            key_use: "sig".to_string(),
            alg: "ES256".to_string(),
            n: None,
            e: None,
            x: Some("x-coord".to_string()),
            y: Some("y-coord".to_string()),
            crv: Some("P-256".to_string()),
        };

        let publisher = token_service::jwks::JwksPublisher::new();
        publisher.add_key(key).await;

        let jwks = publisher.get_jwks().await;
        assert_eq!(jwks.keys[0].kty, "EC");
        assert_eq!(jwks.keys[0].crv, Some("P-256".to_string()));
    }

    #[test]
    fn test_jwks_default() {
        let jwks = token_service::jwks::Jwks::default();
        assert!(jwks.keys.is_empty());
    }
}
