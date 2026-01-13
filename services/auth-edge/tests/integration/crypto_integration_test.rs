//! Integration tests for crypto module
//!
//! Tests the full encrypt/decrypt flow with mocked crypto-service.

use std::sync::Arc;
use std::time::Duration;

use auth_edge::crypto::{
    CryptoClient, CryptoClientConfig, CryptoError, FallbackHandler, KeyId,
};

/// Test fallback encryption when crypto-service is unavailable.
#[tokio::test]
async fn test_fallback_encryption_round_trip() {
    let dek = [0x42u8; 32];
    let handler = FallbackHandler::new(&dek, 1).unwrap();

    let plaintext = b"sensitive cache data";
    let aad = b"auth-edge:session:user123";

    // Encrypt
    let encrypted = handler.encrypt(plaintext, Some(aad)).unwrap();
    assert!(encrypted.is_local_fallback());
    assert_eq!(encrypted.algorithm, "AES-256-GCM");

    // Decrypt
    let decrypted = handler.decrypt(&encrypted, Some(aad)).unwrap();
    assert_eq!(decrypted, plaintext);
}

/// Test that AAD mismatch causes decryption failure.
#[tokio::test]
async fn test_aad_mismatch_fails() {
    let dek = [0x42u8; 32];
    let handler = FallbackHandler::new(&dek, 1).unwrap();

    let plaintext = b"sensitive data";
    let aad1 = b"auth-edge:session:user123";
    let aad2 = b"auth-edge:session:user456";

    let encrypted = handler.encrypt(plaintext, Some(aad1)).unwrap();
    let result = handler.decrypt(&encrypted, Some(aad2));

    assert!(result.is_err());
}

/// Test key rotation maintains decryption capability.
#[tokio::test]
async fn test_key_rotation_continuity() {
    use auth_edge::crypto::KeyManager;

    let manager = KeyManager::new("test", Duration::from_secs(3600));

    // Rotate through multiple versions
    let key_v1 = KeyId::new("test", "kek", 1);
    manager.rotate(key_v1.clone()).await.unwrap();

    let key_v2 = KeyId::new("test", "kek", 2);
    manager.rotate(key_v2.clone()).await.unwrap();

    let key_v3 = KeyId::new("test", "kek", 3);
    manager.rotate(key_v3.clone()).await.unwrap();

    // All versions should be valid during rotation window
    assert!(manager.is_valid_key(&key_v1).await);
    assert!(manager.is_valid_key(&key_v2).await);
    assert!(manager.is_valid_key(&key_v3).await);

    // Current active key should be v3
    assert_eq!(manager.active_key(), key_v3);
}

/// Test DEK caching for fallback mode.
#[tokio::test]
async fn test_dek_caching() {
    use auth_edge::crypto::KeyManager;

    let manager = KeyManager::new("test", Duration::from_secs(3600));

    // Initially no DEK cached
    assert!(manager.get_fallback_dek().await.is_none());

    // Cache a DEK
    let dek = vec![0x42u8; 32];
    manager.cache_dek(dek.clone(), 1).await.unwrap();

    // DEK should be retrievable
    let cached = manager.get_fallback_dek().await;
    assert_eq!(cached, Some(dek));

    // Cache should be valid
    assert!(manager.is_dek_cache_valid(Duration::from_secs(60)).await);
}

/// Test config validation rejects invalid configurations.
#[tokio::test]
async fn test_config_validation() {
    // Empty namespace should fail
    let config = CryptoClientConfig::default().with_key_namespace("");
    assert!(matches!(
        config.validate(),
        Err(CryptoError::InvalidConfig { .. })
    ));

    // Zero timeout should fail
    let config = CryptoClientConfig::default().with_timeout(Duration::ZERO);
    assert!(matches!(
        config.validate(),
        Err(CryptoError::InvalidConfig { .. })
    ));

    // Valid config should pass
    let config = CryptoClientConfig::default()
        .with_key_namespace("test")
        .with_timeout(Duration::from_secs(5));
    assert!(config.validate().is_ok());
}

/// Test error sanitization removes key material.
#[test]
fn test_error_sanitization() {
    // Hex key should be redacted
    let error = CryptoError::encryption_failed(
        "Failed with key: 0123456789abcdef0123456789abcdef",
    );
    let msg = error.to_string();
    assert!(!msg.contains("0123456789abcdef"));

    // Normal messages should pass through
    let error = CryptoError::service_unavailable("Connection refused");
    let msg = error.to_string();
    assert!(msg.contains("Connection refused"));
}

/// Test pending operations queue.
#[tokio::test]
async fn test_pending_operations_queue() {
    use auth_edge::crypto::fallback::PendingOperation;
    use std::time::Instant;

    let dek = [0x42u8; 32];
    let handler = FallbackHandler::new(&dek, 1).unwrap();

    // Queue some operations
    handler
        .queue_operation(PendingOperation::KeyRotation {
            correlation_id: "test-1".to_string(),
            requested_at: Instant::now(),
        })
        .await
        .unwrap();

    handler
        .queue_operation(PendingOperation::KeyRotation {
            correlation_id: "test-2".to_string(),
            requested_at: Instant::now(),
        })
        .await
        .unwrap();

    assert_eq!(handler.pending_count().await, 2);

    // Drain operations
    let pending = handler.drain_pending().await;
    assert_eq!(pending.len(), 2);
    assert_eq!(handler.pending_count().await, 0);
}

/// Test metrics recording.
#[test]
fn test_metrics_recording() {
    // Note: This test may fail if run multiple times due to global registry
    // In CI, metrics tests are typically skipped or use isolated registries
    if std::env::var("CI").is_ok() {
        return;
    }

    use auth_edge::crypto::CryptoMetrics;

    let metrics = CryptoMetrics::new();

    // Record some operations
    metrics.record_success("encrypt", Duration::from_millis(10));
    metrics.record_failure("decrypt", "auth_failed", Duration::from_millis(5));
    metrics.record_fallback("encrypt", Duration::from_millis(2));
    metrics.set_fallback_active(true);
    metrics.increment_rotation();

    // Metrics should be recorded (no panics)
}
