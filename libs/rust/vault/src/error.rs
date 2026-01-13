//! Vault error types using thiserror 2.0.
//!
//! Provides Vault-specific errors with retryability classification
//! and integration with platform common errors.

use rust_common::PlatformError;
use thiserror::Error;

/// Vault-specific errors.
#[derive(Error, Debug)]
pub enum VaultError {
    /// Vault server unavailable
    #[error("Vault unavailable: {0}")]
    Unavailable(String),

    /// Authentication failed
    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),

    /// Secret not found
    #[error("Secret not found at path: {0}")]
    SecretNotFound(String),

    /// Lease renewal failed
    #[error("Lease renewal failed: {0}")]
    LeaseRenewalFailed(String),

    /// Serialization error
    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    /// HTTP error
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),

    /// Invalid configuration
    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),

    /// Permission denied
    #[error("Permission denied: {0}")]
    PermissionDenied(String),

    /// Rate limited
    #[error("Rate limited")]
    RateLimited,

    /// Circuit breaker open
    #[error("Circuit breaker open")]
    CircuitBreakerOpen,

    /// Platform error
    #[error(transparent)]
    Platform(#[from] PlatformError),
}

/// Result type for Vault operations.
pub type VaultResult<T> = Result<T, VaultError>;

impl VaultError {
    /// Check if error is retryable.
    #[must_use]
    pub const fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::Unavailable(_) | Self::RateLimited | Self::Http(_)
        )
    }

    /// Create an unavailable error.
    #[must_use]
    pub fn unavailable(msg: impl Into<String>) -> Self {
        Self::Unavailable(msg.into())
    }

    /// Create an authentication failed error.
    #[must_use]
    pub fn auth_failed(msg: impl Into<String>) -> Self {
        Self::AuthenticationFailed(msg.into())
    }

    /// Create a secret not found error.
    #[must_use]
    pub fn not_found(path: impl Into<String>) -> Self {
        Self::SecretNotFound(path.into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_display() {
        let err = VaultError::unavailable("connection refused");
        assert_eq!(err.to_string(), "Vault unavailable: connection refused");
    }

    #[test]
    fn test_retryable_errors() {
        assert!(VaultError::Unavailable("timeout".to_string()).is_retryable());
        assert!(VaultError::RateLimited.is_retryable());
        assert!(!VaultError::SecretNotFound("path".to_string()).is_retryable());
        assert!(!VaultError::CircuitBreakerOpen.is_retryable());
    }

    #[test]
    fn test_from_platform_error() {
        let platform_err = PlatformError::RateLimited;
        let vault_err: VaultError = platform_err.into();
        assert!(matches!(vault_err, VaultError::Platform(_)));
    }
}
