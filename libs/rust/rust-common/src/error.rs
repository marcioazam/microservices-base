//! Centralized error types for all Rust libraries.
//!
//! This module provides a unified error type that can be used across all
//! auth-platform Rust services, with built-in retryability classification.

use thiserror::Error;

/// Common error type for platform operations.
///
/// All errors are classified as either retryable or non-retryable,
/// which helps callers decide whether to retry failed operations.
#[derive(Error, Debug)]
pub enum PlatformError {
    /// HTTP request failed
    #[error("HTTP request failed: {0}")]
    Http(#[from] reqwest::Error),

    /// gRPC error occurred
    #[error("gRPC error: {0}")]
    Grpc(#[from] tonic::Status),

    /// Serialization/deserialization error
    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    /// Circuit breaker is open for the specified service
    #[error("Circuit breaker open for {service}")]
    CircuitOpen {
        /// The service name that has an open circuit
        service: String,
    },

    /// Service is temporarily unavailable
    #[error("Service unavailable: {0}")]
    Unavailable(String),

    /// Authentication failed
    #[error("Authentication failed: {0}")]
    AuthFailed(String),

    /// Resource not found
    #[error("Not found: {0}")]
    NotFound(String),

    /// Rate limit exceeded
    #[error("Rate limited")]
    RateLimited,

    /// Invalid input provided
    #[error("Invalid input: {0}")]
    InvalidInput(String),

    /// Encryption/decryption error
    #[error("Encryption error: {0}")]
    Encryption(String),

    /// Timeout occurred
    #[error("Operation timed out: {0}")]
    Timeout(String),

    /// Internal error
    #[error("Internal error: {0}")]
    Internal(String),
}

impl PlatformError {
    /// Check if this error is retryable.
    ///
    /// Retryable errors are transient failures that may succeed on retry,
    /// such as network issues, rate limiting, or temporary unavailability.
    ///
    /// # Examples
    ///
    /// ```
    /// use rust_common::PlatformError;
    ///
    /// let err = PlatformError::RateLimited;
    /// assert!(err.is_retryable());
    ///
    /// let err = PlatformError::NotFound("user".to_string());
    /// assert!(!err.is_retryable());
    /// ```
    #[must_use]
    pub const fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::Unavailable(_) | Self::RateLimited | Self::Timeout(_)
        )
    }

    /// Create a circuit open error for the given service.
    #[must_use]
    pub fn circuit_open(service: impl Into<String>) -> Self {
        Self::CircuitOpen {
            service: service.into(),
        }
    }

    /// Create an unavailable error with the given message.
    #[must_use]
    pub fn unavailable(msg: impl Into<String>) -> Self {
        Self::Unavailable(msg.into())
    }

    /// Create an invalid input error with the given message.
    #[must_use]
    pub fn invalid_input(msg: impl Into<String>) -> Self {
        Self::InvalidInput(msg.into())
    }

    /// Create an encryption error with the given message.
    #[must_use]
    pub fn encryption(msg: impl Into<String>) -> Self {
        Self::Encryption(msg.into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_retryable_errors() {
        assert!(PlatformError::RateLimited.is_retryable());
        assert!(PlatformError::Unavailable("test".to_string()).is_retryable());
        assert!(PlatformError::Timeout("test".to_string()).is_retryable());
    }

    #[test]
    fn test_non_retryable_errors() {
        assert!(!PlatformError::NotFound("test".to_string()).is_retryable());
        assert!(!PlatformError::AuthFailed("test".to_string()).is_retryable());
        assert!(!PlatformError::InvalidInput("test".to_string()).is_retryable());
        assert!(!PlatformError::circuit_open("test").is_retryable());
    }

    #[test]
    fn test_error_display() {
        let err = PlatformError::RateLimited;
        assert_eq!(err.to_string(), "Rate limited");

        let err = PlatformError::circuit_open("vault");
        assert_eq!(err.to_string(), "Circuit breaker open for vault");
    }
}
