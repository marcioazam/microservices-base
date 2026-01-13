//! CAEP error types using thiserror 2.0.
//!
//! This module provides CAEP-specific error types with integration
//! to the platform's common error handling.

use rust_common::PlatformError;
use thiserror::Error;

/// CAEP-specific errors.
#[derive(Error, Debug)]
pub enum CaepError {
    /// Failed to sign a Security Event Token
    #[error("Failed to sign SET: {0}")]
    SigningError(String),

    /// Failed to verify SET signature
    #[error("Failed to verify SET signature: {0}")]
    VerificationError(String),

    /// Invalid SET structure
    #[error("Invalid SET structure: {0}")]
    InvalidSet(String),

    /// Unknown event type
    #[error("Unknown event type: {0}")]
    UnknownEventType(String),

    /// Stream not found
    #[error("Stream not found: {0}")]
    StreamNotFound(String),

    /// Stream delivery failed
    #[error("Stream delivery failed: {0}")]
    DeliveryFailed(String),

    /// JWKS fetch failed
    #[error("JWKS fetch failed: {0}")]
    JwksFetchError(String),

    /// Event processing failed
    #[error("Event processing failed: {0}")]
    ProcessingError(String),

    /// Configuration error
    #[error("Configuration error: {0}")]
    ConfigError(String),

    /// Network error
    #[error("Network error: {0}")]
    NetworkError(String),

    /// Platform error (from rust-common)
    #[error(transparent)]
    Platform(#[from] PlatformError),

    /// JSON serialization error
    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    /// JWT error
    #[error("JWT error: {0}")]
    Jwt(#[from] jsonwebtoken::errors::Error),

    /// HTTP error
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),
}

impl CaepError {
    /// Check if this error is retryable.
    #[must_use]
    pub const fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::NetworkError(_) | Self::DeliveryFailed(_) | Self::JwksFetchError(_)
        )
    }

    /// Create a signing error.
    #[must_use]
    pub fn signing(msg: impl Into<String>) -> Self {
        Self::SigningError(msg.into())
    }

    /// Create a verification error.
    #[must_use]
    pub fn verification(msg: impl Into<String>) -> Self {
        Self::VerificationError(msg.into())
    }

    /// Create an invalid SET error.
    #[must_use]
    pub fn invalid_set(msg: impl Into<String>) -> Self {
        Self::InvalidSet(msg.into())
    }

    /// Create a stream not found error.
    #[must_use]
    pub fn stream_not_found(stream_id: impl Into<String>) -> Self {
        Self::StreamNotFound(stream_id.into())
    }

    /// Create a delivery failed error.
    #[must_use]
    pub fn delivery_failed(msg: impl Into<String>) -> Self {
        Self::DeliveryFailed(msg.into())
    }

    /// Create a processing error.
    #[must_use]
    pub fn processing(msg: impl Into<String>) -> Self {
        Self::ProcessingError(msg.into())
    }
}

/// Result type for CAEP operations.
pub type CaepResult<T> = Result<T, CaepError>;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_display() {
        let err = CaepError::signing("invalid key");
        assert_eq!(err.to_string(), "Failed to sign SET: invalid key");
    }

    #[test]
    fn test_retryable_errors() {
        assert!(CaepError::NetworkError("timeout".to_string()).is_retryable());
        assert!(CaepError::DeliveryFailed("connection refused".to_string()).is_retryable());
        assert!(!CaepError::InvalidSet("missing field".to_string()).is_retryable());
    }

    #[test]
    fn test_from_platform_error() {
        let platform_err = PlatformError::RateLimited;
        let caep_err: CaepError = platform_err.into();
        assert!(matches!(caep_err, CaepError::Platform(_)));
    }
}
