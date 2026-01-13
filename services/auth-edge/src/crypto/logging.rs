//! Structured logging helpers for crypto operations
//!
//! Provides JSON-formatted logging for crypto operations with sanitization.

use std::time::Duration;
use tracing::{error, info, warn, Span};

use crate::crypto::error::CryptoError;

/// Log a successful crypto operation.
pub fn log_crypto_operation(
    operation: &str,
    correlation_id: &str,
    duration: Duration,
    key_namespace: &str,
) {
    info!(
        target: "crypto",
        operation = %operation,
        correlation_id = %correlation_id,
        duration_ms = duration.as_millis() as u64,
        key_namespace = %key_namespace,
        status = "success",
        "Crypto operation completed"
    );
}

/// Log a crypto operation that used fallback.
pub fn log_crypto_fallback(
    operation: &str,
    correlation_id: &str,
    duration: Duration,
    reason: &str,
) {
    warn!(
        target: "crypto",
        operation = %operation,
        correlation_id = %correlation_id,
        duration_ms = duration.as_millis() as u64,
        fallback = true,
        reason = %reason,
        status = "fallback",
        "Crypto operation used fallback"
    );
}

/// Log a crypto error (sanitized).
pub fn log_crypto_error(
    operation: &str,
    correlation_id: &str,
    error: &CryptoError,
) {
    let error_type = match error {
        CryptoError::ServiceUnavailable { .. } => "service_unavailable",
        CryptoError::EncryptionFailed { .. } => "encryption_failed",
        CryptoError::DecryptionFailed { .. } => "decryption_failed",
        CryptoError::KeyNotFound { .. } => "key_not_found",
        CryptoError::RotationFailed { .. } => "rotation_failed",
        CryptoError::InvalidConfig { .. } => "invalid_config",
        CryptoError::FallbackUnavailable => "fallback_unavailable",
        CryptoError::TransportError { .. } => "transport_error",
        CryptoError::CircuitOpen => "circuit_open",
    };

    error!(
        target: "crypto",
        operation = %operation,
        correlation_id = %correlation_id,
        error_type = %error_type,
        // Note: error message is already sanitized by CryptoError
        error_message = %error,
        status = "error",
        "Crypto operation failed"
    );
}

/// Log key rotation event.
pub fn log_key_rotation(
    correlation_id: &str,
    old_version: u32,
    new_version: u32,
    namespace: &str,
) {
    info!(
        target: "crypto",
        operation = "key_rotation",
        correlation_id = %correlation_id,
        old_version = old_version,
        new_version = new_version,
        key_namespace = %namespace,
        status = "success",
        "Key rotation completed"
    );
}

/// Log circuit breaker state change.
pub fn log_circuit_state_change(
    new_state: &str,
    failure_count: u32,
    threshold: u32,
) {
    warn!(
        target: "crypto",
        operation = "circuit_breaker",
        new_state = %new_state,
        failure_count = failure_count,
        threshold = threshold,
        "Circuit breaker state changed"
    );
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_log_crypto_operation_does_not_panic() {
        // Just verify logging doesn't panic
        log_crypto_operation("encrypt", "test-123", Duration::from_millis(50), "auth-edge");
    }

    #[test]
    fn test_log_crypto_error_sanitizes() {
        let error = CryptoError::encryption_failed("key=secret123");
        // The error message should be sanitized
        let error_str = error.to_string();
        assert!(!error_str.contains("secret123"));
    }

    #[test]
    fn test_error_type_mapping() {
        let errors = vec![
            CryptoError::ServiceUnavailable { reason: "test".into() },
            CryptoError::EncryptionFailed { reason: "test".into() },
            CryptoError::DecryptionFailed { reason: "test".into() },
            CryptoError::KeyNotFound { key_id: "test".into() },
            CryptoError::RotationFailed { reason: "test".into() },
            CryptoError::InvalidConfig { reason: "test".into() },
            CryptoError::FallbackUnavailable,
            CryptoError::TransportError { reason: "test".into() },
            CryptoError::CircuitOpen,
        ];

        for error in errors {
            // Just verify we can log each error type
            log_crypto_error("test", "corr-123", &error);
        }
    }
}
