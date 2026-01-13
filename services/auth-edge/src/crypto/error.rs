//! Crypto-specific error types
//!
//! Extends AuthEdgeError with crypto-service specific error variants.

use thiserror::Error;

/// Crypto-specific errors for crypto-service integration
#[non_exhaustive]
#[derive(Error, Debug)]
pub enum CryptoError {
    /// Crypto service is unavailable
    #[error("Crypto service unavailable: {reason}")]
    ServiceUnavailable {
        /// Reason for unavailability
        reason: String,
    },

    /// Encryption operation failed
    #[error("Encryption failed: {reason}")]
    EncryptionFailed {
        /// Reason for failure (sanitized)
        reason: String,
    },

    /// Decryption operation failed
    #[error("Decryption failed: {reason}")]
    DecryptionFailed {
        /// Reason for failure (sanitized)
        reason: String,
    },

    /// Key not found in crypto-service
    #[error("Key not found: {key_id}")]
    KeyNotFound {
        /// Key identifier that was not found
        key_id: String,
    },

    /// Key rotation failed
    #[error("Key rotation failed: {reason}")]
    RotationFailed {
        /// Reason for failure
        reason: String,
    },

    /// Invalid configuration
    #[error("Invalid crypto configuration: {reason}")]
    InvalidConfig {
        /// Reason for invalid configuration
        reason: String,
    },

    /// Fallback encryption not available
    #[error("Fallback encryption not available: no cached DEK")]
    FallbackUnavailable,

    /// gRPC transport error
    #[error("gRPC transport error: {reason}")]
    TransportError {
        /// Reason for transport error
        reason: String,
    },

    /// Circuit breaker is open
    #[error("Circuit breaker open for crypto-service")]
    CircuitOpen,
}

impl CryptoError {
    /// Creates a ServiceUnavailable error
    #[must_use]
    pub fn service_unavailable(reason: impl Into<String>) -> Self {
        Self::ServiceUnavailable {
            reason: sanitize_error_message(&reason.into()),
        }
    }

    /// Creates an EncryptionFailed error
    #[must_use]
    pub fn encryption_failed(reason: impl Into<String>) -> Self {
        Self::EncryptionFailed {
            reason: sanitize_error_message(&reason.into()),
        }
    }

    /// Creates a DecryptionFailed error
    #[must_use]
    pub fn decryption_failed(reason: impl Into<String>) -> Self {
        Self::DecryptionFailed {
            reason: sanitize_error_message(&reason.into()),
        }
    }

    /// Creates a KeyNotFound error
    #[must_use]
    pub fn key_not_found(key_id: impl Into<String>) -> Self {
        Self::KeyNotFound {
            key_id: key_id.into(),
        }
    }

    /// Creates a RotationFailed error
    #[must_use]
    pub fn rotation_failed(reason: impl Into<String>) -> Self {
        Self::RotationFailed {
            reason: sanitize_error_message(&reason.into()),
        }
    }

    /// Creates an InvalidConfig error
    #[must_use]
    pub fn invalid_config(reason: impl Into<String>) -> Self {
        Self::InvalidConfig {
            reason: reason.into(),
        }
    }

    /// Creates a TransportError
    #[must_use]
    pub fn transport_error(reason: impl Into<String>) -> Self {
        Self::TransportError {
            reason: sanitize_error_message(&reason.into()),
        }
    }

    /// Checks if this error is retryable
    #[must_use]
    pub const fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::ServiceUnavailable { .. }
                | Self::TransportError { .. }
                | Self::CircuitOpen
        )
    }

    /// Checks if this error should trigger circuit breaker
    #[must_use]
    pub const fn should_trip_circuit(&self) -> bool {
        matches!(
            self,
            Self::ServiceUnavailable { .. } | Self::TransportError { .. }
        )
    }
}

/// Sensitive patterns that should be sanitized from error messages
const SENSITIVE_PATTERNS: &[&str] = &[
    "key",
    "secret",
    "password",
    "token",
    "credential",
    "private",
    "dek",
    "kek",
    "aes",
    "iv",
    "nonce",
];

/// Sanitizes error messages to remove potential key material
fn sanitize_error_message(message: &str) -> String {
    let lower = message.to_lowercase();
    
    // Check for hex-encoded data (potential key material)
    if looks_like_key_material(&lower) {
        return "Operation failed (details redacted)".to_string();
    }

    // Check for sensitive patterns
    for pattern in SENSITIVE_PATTERNS {
        if lower.contains(pattern) && lower.contains("=") {
            return "Operation failed (details redacted)".to_string();
        }
    }

    message.to_string()
}

/// Checks if a string looks like it might contain key material
fn looks_like_key_material(s: &str) -> bool {
    // Check for long hex strings (32+ chars = 16+ bytes)
    let hex_chars: usize = s.chars().filter(|c| c.is_ascii_hexdigit()).count();
    if hex_chars >= 32 {
        // Check if it's mostly hex
        let total_alnum: usize = s.chars().filter(|c| c.is_alphanumeric()).count();
        if total_alnum > 0 && (hex_chars as f64 / total_alnum as f64) > 0.8 {
            return true;
        }
    }

    // Check for base64-encoded data (44+ chars = 32+ bytes)
    let base64_chars: usize = s
        .chars()
        .filter(|c| c.is_alphanumeric() || *c == '+' || *c == '/' || *c == '=')
        .count();
    if base64_chars >= 44 && s.len() > 0 && (base64_chars as f64 / s.len() as f64) > 0.9 {
        return true;
    }

    false
}

impl From<tonic::Status> for CryptoError {
    fn from(status: tonic::Status) -> Self {
        match status.code() {
            tonic::Code::Unavailable => Self::service_unavailable(status.message()),
            tonic::Code::NotFound => Self::key_not_found(status.message()),
            tonic::Code::InvalidArgument => Self::invalid_config(status.message()),
            tonic::Code::DeadlineExceeded => Self::service_unavailable("Request timed out"),
            _ => Self::transport_error(status.message()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sanitize_removes_hex_key() {
        let msg = "Failed with key: 0123456789abcdef0123456789abcdef";
        let sanitized = sanitize_error_message(msg);
        assert_eq!(sanitized, "Operation failed (details redacted)");
    }

    #[test]
    fn test_sanitize_preserves_normal_message() {
        let msg = "Connection refused";
        let sanitized = sanitize_error_message(msg);
        assert_eq!(sanitized, "Connection refused");
    }

    #[test]
    fn test_sanitize_removes_key_value() {
        let msg = "Error: key=abc123secret";
        let sanitized = sanitize_error_message(msg);
        assert_eq!(sanitized, "Operation failed (details redacted)");
    }

    #[test]
    fn test_error_retryable() {
        assert!(CryptoError::ServiceUnavailable {
            reason: "test".to_string()
        }
        .is_retryable());
        assert!(CryptoError::CircuitOpen.is_retryable());
        assert!(!CryptoError::KeyNotFound {
            key_id: "test".to_string()
        }
        .is_retryable());
    }

    #[test]
    fn test_error_should_trip_circuit() {
        assert!(CryptoError::ServiceUnavailable {
            reason: "test".to_string()
        }
        .should_trip_circuit());
        assert!(!CryptoError::KeyNotFound {
            key_id: "test".to_string()
        }
        .should_trip_circuit());
    }
}
