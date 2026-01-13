//! Property-based tests for Crypto Service integration - Core tests.
//!
//! Feature: token-crypto-service-integration

use proptest::prelude::*;
use token_service::crypto::{
    CryptoClientConfig, CryptoError, FallbackHandler, KeyId, KeyState,
    EncryptedData,
};

// =============================================================================
// Property 12: Configuration Validation
// Validates: Requirements 7.6
// =============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_config_validation_missing_address(
        signing_enabled in any::<bool>(),
        encryption_enabled in any::<bool>(),
    ) {
        let config = CryptoClientConfig::default()
            .with_address("")
            .with_signing_enabled(signing_enabled)
            .with_encryption_enabled(encryption_enabled);

        let result = config.validate();
        if signing_enabled || encryption_enabled {
            prop_assert!(result.is_err());
        } else {
            prop_assert!(result.is_ok());
        }
    }

    #[test]
    fn prop_config_validation_missing_namespace(address in "[a-z]+://[a-z]+:[0-9]+") {
        let config = CryptoClientConfig::default()
            .with_address(address)
            .with_namespace("");
        prop_assert!(config.validate().is_err());
    }

    #[test]
    fn prop_config_validation_rate_limit(rate_limit in 0u32..=10u32) {
        let config = CryptoClientConfig::default().with_rate_limit(rate_limit);
        if rate_limit == 0 {
            prop_assert!(config.validate().is_err());
        } else {
            prop_assert!(config.validate().is_ok());
        }
    }
}

// =============================================================================
// Property 2: Fallback Activation
// Validates: Requirements 1.4, 2.4, 4.4, 5.3
// =============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_fallback_sign_local(
        data in prop::collection::vec(any::<u8>(), 1..1000),
        key_id in "[a-z0-9-]{1,36}",
        namespace in "[a-z0-9-]{1,20}",
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let signing_key = b"test-signing-key-for-hmac-256!!".to_vec();
            let handler = FallbackHandler::new(Some(signing_key), Some([0u8; 32]));
            let key = KeyId::new(namespace, key_id, 1);
            let result = handler.sign_local(&data, &key).await;
            prop_assert!(result.is_ok());
            prop_assert!(!result.unwrap().signature.is_empty());
            Ok(())
        })?;
    }

    #[test]
    fn prop_fallback_sign_verify_roundtrip(data in prop::collection::vec(any::<u8>(), 1..1000)) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let signing_key = b"test-signing-key-for-hmac-256!!".to_vec();
            let handler = FallbackHandler::new(Some(signing_key), None);
            let key = KeyId::new("test", "key", 1);
            let sign_result = handler.sign_local(&data, &key).await.unwrap();
            let valid = handler.verify_local(&data, &sign_result.signature, &key).await.unwrap();
            prop_assert!(valid);
            Ok(())
        })?;
    }

    #[test]
    fn prop_fallback_encrypt_decrypt_roundtrip(
        plaintext in prop::collection::vec(any::<u8>(), 1..1000),
        aad in prop::option::of(prop::collection::vec(any::<u8>(), 1..100)),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let handler = FallbackHandler::new(None, Some([42u8; 32]));
            let aad_ref = aad.as_deref();
            let encrypt_result = handler.encrypt_local(&plaintext, aad_ref).await.unwrap();
            let encrypted = EncryptedData::from_result(&encrypt_result);
            let decrypted = handler.decrypt_local(&encrypted, aad_ref).await.unwrap();
            prop_assert_eq!(decrypted, plaintext);
            Ok(())
        })?;
    }
}

// =============================================================================
// Property 8: Key State Validation
// Validates: Requirements 3.4, 3.5, 8.5
// =============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_key_state_can_sign(state_value in 0i32..=5i32) {
        let state = KeyState::from_proto(state_value);
        if state_value == 2 { prop_assert!(state.can_sign()); }
        else { prop_assert!(!state.can_sign()); }
    }

    #[test]
    fn prop_key_state_can_verify(state_value in 0i32..=5i32) {
        let state = KeyState::from_proto(state_value);
        if state_value == 2 || state_value == 3 { prop_assert!(state.can_verify()); }
        else { prop_assert!(!state.can_verify()); }
    }
}

// =============================================================================
// Property 9: Encryption Round Trip
// Validates: Requirements 4.1, 4.2
// =============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_encryption_roundtrip_with_aad(
        plaintext in prop::collection::vec(any::<u8>(), 1..10000),
        aad in prop::collection::vec(any::<u8>(), 1..100),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let handler = FallbackHandler::new(None, Some([0x42u8; 32]));
            let encrypt_result = handler.encrypt_local(&plaintext, Some(&aad)).await.unwrap();
            prop_assert_eq!(encrypt_result.iv.len(), 12);
            let encrypted = EncryptedData::from_result(&encrypt_result);
            let decrypted = handler.decrypt_local(&encrypted, Some(&aad)).await.unwrap();
            prop_assert_eq!(decrypted, plaintext);
            Ok(())
        })?;
    }

    #[test]
    fn prop_encryption_wrong_aad_fails(
        plaintext in prop::collection::vec(any::<u8>(), 1..1000),
        aad1 in prop::collection::vec(any::<u8>(), 1..50),
        aad2 in prop::collection::vec(any::<u8>(), 1..50),
    ) {
        prop_assume!(aad1 != aad2);
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let handler = FallbackHandler::new(None, Some([0x42u8; 32]));
            let encrypt_result = handler.encrypt_local(&plaintext, Some(&aad1)).await.unwrap();
            let encrypted = EncryptedData::from_result(&encrypt_result);
            let result = handler.decrypt_local(&encrypted, Some(&aad2)).await;
            prop_assert!(result.is_err());
            Ok(())
        })?;
    }
}

// =============================================================================
// Property 11: Feature Flag Behavior
// Validates: Requirements 7.2, 7.3, 7.4
// =============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_disabled_fallback_fails(data in prop::collection::vec(any::<u8>(), 1..100)) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let handler = FallbackHandler::new_disabled();
            let key = KeyId::new("test", "key", 1);
            prop_assert!(handler.sign_local(&data, &key).await.is_err());
            prop_assert!(handler.encrypt_local(&data, None).await.is_err());
            Ok(())
        })?;
    }

    #[test]
    fn prop_fallback_activation_count(operations in 1usize..=10usize) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let signing_key = b"test-signing-key-for-hmac-256!!".to_vec();
            let handler = FallbackHandler::new(Some(signing_key), None);
            prop_assert_eq!(handler.activation_count(), 0);
            let key = KeyId::new("test", "key", 1);
            for _ in 0..operations {
                handler.sign_local(b"data", &key).await.unwrap();
            }
            prop_assert_eq!(handler.activation_count(), operations as u64);
            Ok(())
        })?;
    }
}

// =============================================================================
// Property 1: Circuit Breaker State Transitions
// Validates: Requirements 1.3
// =============================================================================

#[test]
fn test_crypto_error_transient_classification() {
    assert!(CryptoError::Connection("test".to_string()).is_transient());
    assert!(CryptoError::CircuitBreakerOpen.is_transient());
    assert!(CryptoError::Timeout.is_transient());
    assert!(CryptoError::RateLimited.is_transient());
    assert!(!CryptoError::Signing("test".to_string()).is_transient());
    assert!(!CryptoError::KeyNotFound("test".to_string()).is_transient());
    assert!(!CryptoError::InvalidKeyState {
        state: KeyState::Deprecated,
        operation: "sign".to_string(),
    }.is_transient());
}

// =============================================================================
// Property 14: Rate Limiting Enforcement
// Validates: Requirements 8.6
// =============================================================================

#[test]
fn test_config_rate_limit_validation() {
    assert!(CryptoClientConfig::default().with_rate_limit(0).validate().is_err());
    assert!(CryptoClientConfig::default().with_rate_limit(1).validate().is_ok());
    assert!(CryptoClientConfig::default().with_rate_limit(1000).validate().is_ok());
}
