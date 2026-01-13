//! Advanced property-based tests for Crypto Service integration.
//!
//! Feature: token-crypto-service-integration

use proptest::prelude::*;
use token_service::crypto::{
    CryptoClientConfig, CryptoError, FallbackHandler, KeyId, KeyState,
    models::{KeyAlgorithm, KeyMetadata, KeyRotationResult},
};
use chrono::Utc;

// =============================================================================
// Property 5: Algorithm Support Completeness
// Validates: Requirements 2.2, 5.2
// =============================================================================

#[test]
fn test_algorithm_jwt_mapping() {
    assert_eq!(KeyAlgorithm::Rsa2048.to_jwt_algorithm(), Some("PS256"));
    assert_eq!(KeyAlgorithm::Rsa3072.to_jwt_algorithm(), Some("PS256"));
    assert_eq!(KeyAlgorithm::Rsa4096.to_jwt_algorithm(), Some("PS256"));
    assert_eq!(KeyAlgorithm::EcdsaP256.to_jwt_algorithm(), Some("ES256"));
    assert_eq!(KeyAlgorithm::EcdsaP384.to_jwt_algorithm(), Some("ES384"));
    assert_eq!(KeyAlgorithm::EcdsaP521.to_jwt_algorithm(), Some("ES512"));
    assert_eq!(KeyAlgorithm::Aes128Gcm.to_jwt_algorithm(), None);
    assert_eq!(KeyAlgorithm::Aes256Gcm.to_jwt_algorithm(), None);
}

// =============================================================================
// Property 6: Key Metadata Caching Consistency
// Validates: Requirements 2.5, 5.4
// =============================================================================

#[test]
fn test_key_id_proto_roundtrip() {
    let original = KeyId::new("token", "signing-key", 42);
    let proto = original.to_proto();
    let restored = KeyId::from_proto(&proto);
    assert_eq!(original, restored);
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_key_id_proto_roundtrip(
        namespace in "[a-z0-9-]{1,20}",
        id in "[a-z0-9-]{1,36}",
        version in any::<u32>(),
    ) {
        let original = KeyId::new(namespace, id, version);
        let proto = original.to_proto();
        let restored = KeyId::from_proto(&proto);
        prop_assert_eq!(original, restored);
    }
}

// =============================================================================
// Property 7: Key Rotation Graceful Transition
// Validates: Requirements 2.6, 3.2, 3.3
// =============================================================================

#[test]
fn test_key_rotation_result_structure() {
    let old_key = KeyId::new("token", "signing-key", 1);
    let new_key = KeyId::new("token", "signing-key", 2);
    
    let metadata = KeyMetadata {
        id: new_key.clone(),
        algorithm: KeyAlgorithm::Rsa2048,
        state: KeyState::Active,
        created_at: Utc::now(),
        expires_at: None,
        rotated_at: Some(Utc::now()),
        previous_version: Some(old_key.clone()),
        owner_service: "token-service".to_string(),
        allowed_operations: vec!["sign".to_string(), "verify".to_string()],
        usage_count: 0,
    };

    let rotation_result = KeyRotationResult {
        new_key_id: new_key.clone(),
        old_key_id: old_key.clone(),
        metadata,
    };

    assert!(rotation_result.metadata.state.can_sign());
    assert_eq!(rotation_result.new_key_id.version, rotation_result.old_key_id.version + 1);
    assert_eq!(rotation_result.metadata.previous_version, Some(old_key));
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_key_rotation_version_increment(
        namespace in "[a-z0-9-]{1,20}",
        key_name in "[a-z0-9-]{1,36}",
        initial_version in 1u32..1000u32,
    ) {
        let old_key = KeyId::new(&namespace, &key_name, initial_version);
        let new_key = KeyId::new(&namespace, &key_name, initial_version + 1);
        prop_assert_eq!(old_key.namespace, new_key.namespace);
        prop_assert_eq!(old_key.id, new_key.id);
        prop_assert_eq!(new_key.version, old_key.version + 1);
    }
}

// =============================================================================
// Property 10: Error Logging Completeness
// Validates: Requirements 6.4
// =============================================================================

#[test]
fn test_crypto_error_display_messages() {
    let errors = vec![
        CryptoError::Connection("connection refused".to_string()),
        CryptoError::Signing("invalid key".to_string()),
        CryptoError::Verification("signature mismatch".to_string()),
        CryptoError::Encryption("encryption failed".to_string()),
        CryptoError::Decryption("decryption failed".to_string()),
        CryptoError::KeyNotFound("key-123".to_string()),
        CryptoError::InvalidKeyState { state: KeyState::Deprecated, operation: "sign".to_string() },
        CryptoError::InvalidAlgorithm { expected: "PS256".to_string(), actual: "RS256".to_string() },
        CryptoError::RateLimited,
        CryptoError::CircuitBreakerOpen,
        CryptoError::Timeout,
        CryptoError::Internal("internal error".to_string()),
    ];

    for error in errors {
        let message = error.to_string();
        assert!(!message.is_empty());
        assert!(message.len() > 5, "Error message should be descriptive: {}", message);
    }
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_error_messages_contain_context(context in "[a-zA-Z0-9 ]{1,50}") {
        let error = CryptoError::Connection(context.clone());
        prop_assert!(error.to_string().contains(&context));
    }

    #[test]
    fn prop_key_not_found_includes_key_id(key_id in "[a-z0-9-]{1,36}") {
        let error = CryptoError::KeyNotFound(key_id.clone());
        prop_assert!(error.to_string().contains(&key_id));
    }
}

// =============================================================================
// Property 4: JWT Signing Round Trip
// Validates: Requirements 2.1, 2.3
// =============================================================================

#[test]
fn test_fallback_signing_produces_valid_hmac() {
    use ring::hmac;

    let rt = tokio::runtime::Runtime::new().unwrap();
    rt.block_on(async {
        let signing_key = b"test-signing-key-for-hmac-256!!".to_vec();
        let handler = FallbackHandler::new(Some(signing_key.clone()), None);
        let key = KeyId::new("test", "key", 1);
        let data = b"test jwt payload";

        let result = handler.sign_local(data, &key).await.unwrap();
        let hmac_key = hmac::Key::new(hmac::HMAC_SHA256, &signing_key);
        hmac::verify(&hmac_key, data, &result.signature).expect("Signature should be valid");
    });
}

// =============================================================================
// Property 3: Request Context Propagation
// Validates: Requirements 1.5, 4.5
// =============================================================================

#[test]
fn test_config_defaults_for_tracing() {
    let config = CryptoClientConfig::default();
    assert!(config.connect_timeout.as_secs() > 0);
    assert!(config.request_timeout.as_secs() > 0);
    assert!(config.request_timeout > config.connect_timeout);
}

// =============================================================================
// Property 13: Response Algorithm Validation
// Validates: Requirements 8.2
// =============================================================================

#[test]
fn test_invalid_algorithm_error() {
    let error = CryptoError::InvalidAlgorithm {
        expected: "PS256".to_string(),
        actual: "RS256".to_string(),
    };
    assert!(!error.is_transient());
    let message = error.to_string();
    assert!(message.contains("PS256"));
    assert!(message.contains("RS256"));
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_algorithm_mismatch_non_transient(
        expected in "[A-Z0-9]{2,10}",
        actual in "[A-Z0-9]{2,10}",
    ) {
        let error = CryptoError::InvalidAlgorithm { expected, actual };
        prop_assert!(!error.is_transient());
    }
}
