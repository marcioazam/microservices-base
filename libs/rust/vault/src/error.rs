//! Vault error types
//! Requirements: 1.5 - Grace period handling

use thiserror::Error;

#[derive(Error, Debug)]
pub enum VaultError {
    #[error("Vault unavailable: {0}")]
    Unavailable(String),

    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),

    #[error("Secret not found at path: {0}")]
    SecretNotFound(String),

    #[error("Lease renewal failed: {0}")]
    LeaseRenewalFailed(String),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),

    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),

    #[error("Permission denied: {0}")]
    PermissionDenied(String),

    #[error("Rate limited")]
    RateLimited,
}

pub type VaultResult<T> = Result<T, VaultError>;

impl VaultError {
    /// Check if error is retryable
    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            VaultError::Unavailable(_) | VaultError::RateLimited | VaultError::Http(_)
        )
    }
}
