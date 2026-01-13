//! Integration tests for Crypto Service integration.
//!
//! Feature: token-crypto-service-integration

use token_service::crypto::{
    CryptoClient, CryptoClientConfig, CryptoClientFactory, CryptoEncryptor,
    CryptoSigner, FallbackHandler, KeyId,
};
use token_service::kms::KmsSigner;
use std::sync::Arc;

/// Test CryptoClientFactory creates valid client.
#[tokio::test]
async fn test_factory_creates_client() {
    let config = CryptoClientConfig::default();
    let signing_key = Some(b"test-signing-key-for-hmac-256!!".to_vec());
    let encryption_key = Some([0u8; 32]);

    let result = CryptoClientFactory::create(config, signing_key, encryption_key).await;
    assert!(result.is_ok());
}

/// Test CryptoClientFactory creates client without fallback.
#[tokio::test]
async fn test_factory_creates_client_without_fallback() {
    let config = CryptoClientConfig::default();
    let result = CryptoClientFactory::create_without_fallback(config).await;
    assert!(result.is_ok());
}

/// Test CryptoSigner creation via factory.
#[tokio::test]
async fn test_factory_creates_signer() {
    let config = CryptoClientConfig::default();
    let signing_key = Some(b"test-signing-key-for-hmac-256!!".to_vec());
    let client = CryptoClientFactory::create(config, signing_key, None).await.unwrap();

    let key_id = KeyId::new("token", "signing-key", 1);
    let signer = CryptoClientFactory::create_signer(client, key_id, "PS256");

    assert_eq!(signer.key_id(), "signing-key");
    assert_eq!(signer.algorithm(), "PS256");
}

/// Test CryptoEncryptor creation via factory.
#[tokio::test]
async fn test_factory_creates_encryptor() {
    let config = CryptoClientConfig::default();
    let encryption_key = Some([0u8; 32]);
    let client = CryptoClientFactory::create(config, None, encryption_key).await.unwrap();

    let key_id = KeyId::new("token-cache", "enc-key", 1);
    let encryptor = CryptoClientFactory::create_encryptor(client, key_id.clone(), "token-cache");

    assert_eq!(encryptor.namespace(), "token-cache");
    assert_eq!(encryptor.key_id(), &key_id);
}

/// Test fallback signing when Crypto Service is unavailable.
#[tokio::test]
async fn test_fallback_signing_integration() {
    let signing_key = b"test-signing-key-for-hmac-256!!".to_vec();
    let handler = FallbackHandler::new(Some(signing_key), None);

    let key_id = KeyId::new("token", "signing-key", 1);
    let data = b"test data for signing";

    let result = handler.sign_local(data, &key_id).await;
    assert!(result.is_ok());

    let sign_result = result.unwrap();
    assert!(!sign_result.signature.is_empty());
    assert_eq!(sign_result.algorithm, "HS256");

    // Verify signature
    let valid = handler.verify_local(data, &sign_result.signature, &key_id).await;
    assert!(valid.is_ok());
    assert!(valid.unwrap());
}

/// Test fallback encryption when Crypto Service is unavailable.
#[tokio::test]
async fn test_fallback_encryption_integration() {
    let encryption_key = [42u8; 32];
    let handler = FallbackHandler::new(None, Some(encryption_key));

    let plaintext = b"sensitive token data";
    let aad = b"family-id-123";

    let encrypt_result = handler.encrypt_local(plaintext, Some(aad)).await;
    assert!(encrypt_result.is_ok());

    let encrypted = encrypt_result.unwrap();
    assert!(!encrypted.ciphertext.is_empty());
    assert_eq!(encrypted.iv.len(), 12);
    assert_eq!(encrypted.tag.len(), 16);

    // Decrypt
    let encrypted_data = token_service::crypto::EncryptedData::from_result(&encrypted);
    let decrypted = handler.decrypt_local(&encrypted_data, Some(aad)).await;
    assert!(decrypted.is_ok());
    assert_eq!(decrypted.unwrap(), plaintext);
}

/// Test fallback activation counter.
#[tokio::test]
async fn test_fallback_activation_counter() {
    let signing_key = b"test-signing-key-for-hmac-256!!".to_vec();
    let handler = FallbackHandler::new(Some(signing_key), None);

    assert_eq!(handler.activation_count(), 0);

    let key_id = KeyId::new("token", "key", 1);
    for _ in 0..5 {
        handler.sign_local(b"data", &key_id).await.unwrap();
    }

    assert_eq!(handler.activation_count(), 5);
}

/// Test disabled fallback returns error.
#[tokio::test]
async fn test_disabled_fallback_returns_error() {
    let handler = FallbackHandler::new_disabled();
    let key_id = KeyId::new("token", "key", 1);

    let sign_result = handler.sign_local(b"data", &key_id).await;
    assert!(sign_result.is_err());

    let encrypt_result = handler.encrypt_local(b"data", None).await;
    assert!(encrypt_result.is_err());
}

/// Test config validation.
#[tokio::test]
async fn test_config_validation() {
    // Valid config
    let config = CryptoClientConfig::default();
    assert!(config.validate().is_ok());

    // Missing address with signing enabled
    let config = CryptoClientConfig::default()
        .with_address("")
        .with_signing_enabled(true);
    assert!(config.validate().is_err());

    // Missing namespace
    let config = CryptoClientConfig::default().with_namespace("");
    assert!(config.validate().is_err());

    // Zero rate limit
    let config = CryptoClientConfig::default().with_rate_limit(0);
    assert!(config.validate().is_err());
}

/// Test KeyId serialization.
#[test]
fn test_key_id_serialization() {
    let key_id = KeyId::new("token", "signing-key", 1);
    let json = serde_json::to_string(&key_id).unwrap();
    let deserialized: KeyId = serde_json::from_str(&json).unwrap();
    assert_eq!(key_id, deserialized);
}

/// Test KeyId proto conversion.
#[test]
fn test_key_id_proto_conversion() {
    let original = KeyId::new("token", "signing-key", 42);
    let proto = original.to_proto();
    let restored = KeyId::from_proto(&proto);
    assert_eq!(original, restored);
}
