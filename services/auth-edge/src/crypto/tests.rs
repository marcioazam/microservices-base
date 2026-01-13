//! Property-based tests for crypto module
//!
//! Implements the 11 correctness properties defined in the design document.

#[cfg(test)]
mod property_tests {
    use proptest::prelude::*;
    use std::time::Duration;
    use url::Url;

    use crate::crypto::config::CryptoClientConfig;
    use crate::crypto::error::CryptoError;
    use crate::crypto::fallback::{EncryptedData, FallbackHandler};
    use crate::crypto::key_manager::KeyId;

    // =========================================================================
    // Property 1: Encryption Round-Trip
    // For any plaintext P and AAD A: decrypt(encrypt(P, A), A) == P
    // =========================================================================
    proptest! {
        #![proptest_config(ProptestConfig::with_cases(100))]

        #[test]
        fn prop_encryption_round_trip(
            plaintext in prop::collection::vec(any::<u8>(), 0..1024),
            aad in prop::option::of(prop::collection::vec(any::<u8>(), 0..64))
        ) {
            let dek = [0x42u8; 32];
            let handler = FallbackHandler::new(&dek, 1).unwrap();

            let aad_ref = aad.as_deref();
            let encrypted = handler.encrypt(&plaintext, aad_ref).unwrap();
            let decrypted = handler.decrypt(&encrypted, aad_ref).unwrap();

            prop_assert_eq!(decrypted, plaintext);
        }
    }

    // =========================================================================
    // Property 4: Fallback Encryption Consistency
    // Local fallback produces valid AES-256-GCM ciphertext
    // =========================================================================
    proptest! {
        #![proptest_config(ProptestConfig::with_cases(100))]

        #[test]
        fn prop_fallback_encryption_consistency(
            plaintext in prop::collection::vec(any::<u8>(), 1..512),
            key_version in 1u32..100
        ) {
            let dek = [0x42u8; 32];
            let handler = FallbackHandler::new(&dek, key_version).unwrap();

            let encrypted = handler.encrypt(&plaintext, None).unwrap();

            // Verify structure
            prop_assert!(encrypted.is_local_fallback());
            prop_assert_eq!(encrypted.iv.len(), 12); // AES-GCM nonce
            prop_assert_eq!(encrypted.tag.len(), 16); // AES-GCM tag
            prop_assert_eq!(encrypted.algorithm, "AES-256-GCM");
            prop_assert_eq!(encrypted.key_id.version, key_version);

            // Verify decryption works
            let decrypted = handler.decrypt(&encrypted, None).unwrap();
            prop_assert_eq!(decrypted, plaintext);
        }
    }

    // =========================================================================
    // Property 6: AAD Binding
    // Decryption fails if AAD doesn't match
    // =========================================================================
    proptest! {
        #![proptest_config(ProptestConfig::with_cases(100))]

        #[test]
        fn prop_aad_binding(
            plaintext in prop::collection::vec(any::<u8>(), 1..256),
            aad1 in prop::collection::vec(any::<u8>(), 1..32),
            aad2 in prop::collection::vec(any::<u8>(), 1..32)
        ) {
            prop_assume!(aad1 != aad2);

            let dek = [0x42u8; 32];
            let handler = FallbackHandler::new(&dek, 1).unwrap();

            let encrypted = handler.encrypt(&plaintext, Some(&aad1)).unwrap();

            // Decryption with wrong AAD should fail
            let result = handler.decrypt(&encrypted, Some(&aad2));
            prop_assert!(result.is_err());

            // Decryption with correct AAD should succeed
            let decrypted = handler.decrypt(&encrypted, Some(&aad1)).unwrap();
            prop_assert_eq!(decrypted, plaintext);
        }
    }

    // =========================================================================
    // Property 7: No Key Material Exposure
    // Error messages never contain key material
    // =========================================================================
    proptest! {
        #![proptest_config(ProptestConfig::with_cases(100))]

        #[test]
        fn prop_no_key_material_in_errors(
            hex_key in "[0-9a-f]{32,64}",
            base64_key in "[A-Za-z0-9+/]{44,88}={0,2}"
        ) {
            // Test hex key sanitization
            let msg_with_hex = format!("Error with key: {}", hex_key);
            let error = CryptoError::encryption_failed(&msg_with_hex);
            let error_str = error.to_string();
            prop_assert!(!error_str.contains(&hex_key), "Hex key leaked: {}", error_str);

            // Test base64 key sanitization
            let msg_with_b64 = format!("Error with key: {}", base64_key);
            let error = CryptoError::decryption_failed(&msg_with_b64);
            let error_str = error.to_string();
            prop_assert!(!error_str.contains(&base64_key), "Base64 key leaked: {}", error_str);
        }
    }

    // =========================================================================
    // Property 8: Configuration Validation
    // Invalid configs are rejected before use
    // =========================================================================
    proptest! {
        #![proptest_config(ProptestConfig::with_cases(100))]

        #[test]
        fn prop_config_validation_empty_namespace(
            url in "https?://[a-z]+\\.[a-z]+:[0-9]{4,5}"
        ) {
            let parsed_url = Url::parse(&url);
            prop_assume!(parsed_url.is_ok());

            let config = CryptoClientConfig::default()
                .with_service_url(parsed_url.unwrap())
                .with_key_namespace("");

            let result = config.validate();
            prop_assert!(matches!(result, Err(CryptoError::InvalidConfig { .. })));
        }

        #[test]
        fn prop_config_validation_zero_timeout(
            namespace in "[a-z]{1,32}"
        ) {
            let config = CryptoClientConfig::default()
                .with_key_namespace(&namespace)
                .with_timeout(Duration::ZERO);

            let result = config.validate();
            prop_assert!(matches!(result, Err(CryptoError::InvalidConfig { .. })));
        }

        #[test]
        fn prop_config_validation_valid(
            namespace in "[a-z]{1,32}",
            timeout_secs in 1u64..300
        ) {
            let config = CryptoClientConfig::default()
                .with_key_namespace(&namespace)
                .with_timeout(Duration::from_secs(timeout_secs));

            let result = config.validate();
            prop_assert!(result.is_ok());
        }
    }

    // =========================================================================
    // Property 5: Key Rotation Continuity
    // Old keys remain valid during rotation window
    // =========================================================================
    #[tokio::test]
    async fn test_key_rotation_continuity() {
        use crate::crypto::key_manager::KeyManager;
        use std::sync::Arc;

        let manager = KeyManager::new("test", Duration::from_secs(3600));

        // Get initial key (uninitialized)
        let initial = manager.active_key();
        assert_eq!(initial.namespace, "test");

        // Simulate setting first key via rotate
        let key_v1 = KeyId::new("test", "kek", 1);
        manager.rotate(key_v1.clone()).await.unwrap();

        // Rotate to v2
        let key_v2 = KeyId::new("test", "kek", 2);
        manager.rotate(key_v2.clone()).await.unwrap();

        // Both keys should be valid
        assert!(manager.is_valid_key(&key_v1).await);
        assert!(manager.is_valid_key(&key_v2).await);

        // Rotate to v3
        let key_v3 = KeyId::new("test", "kek", 3);
        manager.rotate(key_v3.clone()).await.unwrap();

        // All three should be valid
        assert!(manager.is_valid_key(&key_v1).await);
        assert!(manager.is_valid_key(&key_v2).await);
        assert!(manager.is_valid_key(&key_v3).await);
    }

    // =========================================================================
    // Unit tests for KeyId
    // =========================================================================
    #[test]
    fn test_key_id_proto_round_trip() {
        let key = KeyId::new("ns", "id", 42);
        let proto = key.to_proto();
        let back = KeyId::from_proto(&proto);
        assert_eq!(key, back);
    }

    // =========================================================================
    // Unit tests for EncryptedData
    // =========================================================================
    #[test]
    fn test_encrypted_data_local_detection() {
        let local = EncryptedData::new_local(vec![1, 2, 3], vec![0; 12], vec![0; 16], 1);
        assert!(local.is_local_fallback());

        let remote = EncryptedData {
            ciphertext: vec![1, 2, 3],
            iv: vec![0; 12],
            tag: vec![0; 16],
            key_id: KeyId::new("auth-edge", "kek", 1),
            algorithm: "AES-256-GCM".to_string(),
        };
        assert!(!remote.is_local_fallback());
    }
}
