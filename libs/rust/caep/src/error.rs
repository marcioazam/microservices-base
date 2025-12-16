//! CAEP error types.

use thiserror::Error;

/// CAEP-specific errors
#[derive(Error, Debug)]
pub enum CaepError {
    #[error("Failed to sign SET: {0}")]
    SigningError(String),

    #[error("Failed to verify SET signature: {0}")]
    VerificationError(String),

    #[error("Invalid SET structure: {0}")]
    InvalidSet(String),

    #[error("Unknown event type: {0}")]
    UnknownEventType(String),

    #[error("Stream not found: {0}")]
    StreamNotFound(String),

    #[error("Stream delivery failed: {0}")]
    DeliveryFailed(String),

    #[error("JWKS fetch failed: {0}")]
    JwksFetchError(String),

    #[error("Event processing failed: {0}")]
    ProcessingError(String),

    #[error("Configuration error: {0}")]
    ConfigError(String),

    #[error("Network error: {0}")]
    NetworkError(String),
}
