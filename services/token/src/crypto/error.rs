//! Error types for CryptoClient operations.

use std::fmt;

/// Errors from CryptoClient operations.
#[derive(Debug)]
pub enum CryptoError {
    /// Connection to Crypto Service failed
    Connection(String),
    /// Signing operation failed
    Signing(String),
    /// Verification operation failed
    Verification(String),
    /// Encryption operation failed
    Encryption(String),
    /// Decryption operation failed
    Decryption(String),
    /// Key not found
    KeyNotFound(String),
    /// Invalid key state for operation
    InvalidKeyState {
        state: super::models::KeyState,
        operation: String,
    },
    /// Invalid algorithm returned
    InvalidAlgorithm { expected: String, actual: String },
    /// Rate limited
    RateLimited,
    /// Circuit breaker is open
    CircuitBreakerOpen,
    /// Request timeout
    Timeout,
    /// Internal error
    Internal(String),
}

impl CryptoError {
    /// Check if error is transient (suitable for fallback).
    #[must_use]
    pub fn is_transient(&self) -> bool {
        matches!(
            self,
            CryptoError::Connection(_)
                | CryptoError::CircuitBreakerOpen
                | CryptoError::Timeout
                | CryptoError::RateLimited
        )
    }

    /// Create a connection error.
    #[must_use]
    pub fn connection(msg: impl Into<String>) -> Self {
        CryptoError::Connection(msg.into())
    }

    /// Create a signing error.
    #[must_use]
    pub fn signing(msg: impl Into<String>) -> Self {
        CryptoError::Signing(msg.into())
    }

    /// Create a verification error.
    #[must_use]
    pub fn verification(msg: impl Into<String>) -> Self {
        CryptoError::Verification(msg.into())
    }

    /// Create an encryption error.
    #[must_use]
    pub fn encryption(msg: impl Into<String>) -> Self {
        CryptoError::Encryption(msg.into())
    }

    /// Create a decryption error.
    #[must_use]
    pub fn decryption(msg: impl Into<String>) -> Self {
        CryptoError::Decryption(msg.into())
    }

    /// Create a key not found error.
    #[must_use]
    pub fn key_not_found(msg: impl Into<String>) -> Self {
        CryptoError::KeyNotFound(msg.into())
    }

    /// Create an internal error.
    #[must_use]
    pub fn internal(msg: impl Into<String>) -> Self {
        CryptoError::Internal(msg.into())
    }
}

impl fmt::Display for CryptoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CryptoError::Connection(msg) => write!(f, "Connection failed: {}", msg),
            CryptoError::Signing(msg) => write!(f, "Signing failed: {}", msg),
            CryptoError::Verification(msg) => write!(f, "Verification failed: {}", msg),
            CryptoError::Encryption(msg) => write!(f, "Encryption failed: {}", msg),
            CryptoError::Decryption(msg) => write!(f, "Decryption failed: {}", msg),
            CryptoError::KeyNotFound(msg) => write!(f, "Key not found: {}", msg),
            CryptoError::InvalidKeyState { state, operation } => {
                write!(f, "Invalid key state {:?} for operation {}", state, operation)
            }
            CryptoError::InvalidAlgorithm { expected, actual } => {
                write!(f, "Invalid algorithm: expected {}, got {}", expected, actual)
            }
            CryptoError::RateLimited => write!(f, "Rate limited"),
            CryptoError::CircuitBreakerOpen => write!(f, "Circuit breaker open"),
            CryptoError::Timeout => write!(f, "Request timeout"),
            CryptoError::Internal(msg) => write!(f, "Internal error: {}", msg),
        }
    }
}

impl std::error::Error for CryptoError {}

impl From<tonic::Status> for CryptoError {
    fn from(status: tonic::Status) -> Self {
        match status.code() {
            tonic::Code::NotFound => CryptoError::KeyNotFound(status.message().to_string()),
            tonic::Code::DeadlineExceeded => CryptoError::Timeout,
            tonic::Code::ResourceExhausted => CryptoError::RateLimited,
            tonic::Code::Unavailable => CryptoError::Connection(status.message().to_string()),
            _ => CryptoError::Internal(status.message().to_string()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_transient_errors() {
        assert!(CryptoError::Connection("test".to_string()).is_transient());
        assert!(CryptoError::CircuitBreakerOpen.is_transient());
        assert!(CryptoError::Timeout.is_transient());
        assert!(CryptoError::RateLimited.is_transient());
        assert!(!CryptoError::Signing("test".to_string()).is_transient());
        assert!(!CryptoError::KeyNotFound("test".to_string()).is_transient());
    }

    #[test]
    fn test_error_display() {
        let err = CryptoError::connection("failed to connect");
        assert_eq!(err.to_string(), "Connection failed: failed to connect");
    }
}
