//! Centralized error handling for Token Service.
//!
//! All errors extend rust-common::PlatformError and are classified
//! as retryable or non-retryable for proper handling.

use rust_common::PlatformError;
use thiserror::Error;
use tonic::Status;

/// Token Service error types.
///
/// All errors are classified as retryable or non-retryable to help
/// callers decide whether to retry failed operations.
#[derive(Error, Debug)]
pub enum TokenError {
    /// Platform infrastructure error
    #[error("Platform error: {0}")]
    Platform(#[from] PlatformError),

    /// JWT encoding failed
    #[error("JWT encoding failed: {0}")]
    JwtEncoding(String),

    /// JWT decoding failed
    #[error("JWT decoding failed: {0}")]
    JwtDecoding(String),

    /// DPoP proof validation failed
    #[error("DPoP validation failed: {0}")]
    DpopValidation(String),

    /// DPoP replay attack detected
    #[error("DPoP replay detected: jti={0}")]
    DpopReplay(String),

    /// Refresh token not found or invalid
    #[error("Refresh token invalid")]
    RefreshInvalid,

    /// Refresh token has expired
    #[error("Refresh token expired")]
    RefreshExpired,

    /// Refresh token replay attack detected
    #[error("Refresh token replay detected - family revoked")]
    RefreshReplay,

    /// Token family has been revoked
    #[error("Token family revoked")]
    FamilyRevoked,

    /// KMS operation failed
    #[error("KMS operation failed: {0}")]
    Kms(String),

    /// Configuration error
    #[error("Configuration error: {0}")]
    Config(String),

    /// Serialization error
    #[error("Serialization error: {0}")]
    Serialization(String),

    /// Rate limit exceeded
    #[error("Rate limit exceeded")]
    RateLimited,

    /// Cache operation failed
    #[error("Cache error: {0}")]
    Cache(String),

    /// Internal error
    #[error("Internal error: {0}")]
    Internal(String),

    /// Legacy Redis error (deprecated)
    #[deprecated(since = "2.0.0", note = "Use Cache variant")]
    #[error("Redis error: {0}")]
    RedisError(String),
}

impl TokenError {
    /// Check if this error is retryable.
    ///
    /// Retryable errors are transient failures that may succeed on retry.
    #[must_use]
    #[allow(deprecated)]
    pub fn is_retryable(&self) -> bool {
        match self {
            Self::Platform(e) => e.is_retryable(),
            Self::Kms(_) => true,
            Self::RateLimited => true,
            Self::Cache(_) => true,
            Self::RedisError(_) => true,
            _ => false,
        }
    }

    /// Create a cache error.
    #[must_use]
    pub fn cache(msg: impl Into<String>) -> Self {
        Self::Cache(msg.into())
    }

    /// Create an internal error.
    #[must_use]
    pub fn internal(msg: impl Into<String>) -> Self {
        Self::Internal(msg.into())
    }

    /// Create a JWT encoding error.
    #[must_use]
    pub fn jwt_encoding(msg: impl Into<String>) -> Self {
        Self::JwtEncoding(msg.into())
    }

    /// Create a JWT decoding error.
    #[must_use]
    pub fn jwt_decoding(msg: impl Into<String>) -> Self {
        Self::JwtDecoding(msg.into())
    }

    /// Create a DPoP validation error.
    #[must_use]
    pub fn dpop_validation(msg: impl Into<String>) -> Self {
        Self::DpopValidation(msg.into())
    }

    /// Create a DPoP replay error.
    #[must_use]
    pub fn dpop_replay(jti: impl Into<String>) -> Self {
        Self::DpopReplay(jti.into())
    }

    /// Create a KMS error.
    #[must_use]
    pub fn kms(msg: impl Into<String>) -> Self {
        Self::Kms(msg.into())
    }

    /// Create a configuration error.
    #[must_use]
    pub fn config(msg: impl Into<String>) -> Self {
        Self::Config(msg.into())
    }

    /// Create an encryption error.
    #[must_use]
    pub fn encryption(msg: impl Into<String>) -> Self {
        Self::Internal(format!("Encryption failed: {}", msg.into()))
    }

    /// Create a decryption error.
    #[must_use]
    pub fn decryption(msg: impl Into<String>) -> Self {
        Self::Internal(format!("Decryption failed: {}", msg.into()))
    }

    /// Create a signing error.
    #[must_use]
    pub fn signing(msg: impl Into<String>) -> Self {
        Self::Kms(format!("Signing failed: {}", msg.into()))
    }
}

impl From<TokenError> for Status {
    #[allow(deprecated)]
    fn from(err: TokenError) -> Self {
        match err {
            TokenError::RefreshInvalid | TokenError::RefreshExpired => {
                Status::unauthenticated("UNAUTHENTICATED")
            }
            TokenError::RefreshReplay | TokenError::FamilyRevoked => {
                Status::permission_denied("TOKEN_REVOKED")
            }
            TokenError::DpopValidation(_) => {
                Status::invalid_argument("INVALID_DPOP_PROOF")
            }
            TokenError::DpopReplay(_) => {
                Status::invalid_argument("DPOP_REPLAY_DETECTED")
            }
            TokenError::RateLimited => {
                Status::resource_exhausted("RATE_LIMITED")
            }
            TokenError::Cache(_) | TokenError::RedisError(_) if err.is_retryable() => {
                Status::unavailable("CACHE_UNAVAILABLE")
            }
            TokenError::Kms(_) if err.is_retryable() => {
                Status::unavailable("KMS_UNAVAILABLE")
            }
            TokenError::Platform(ref e) if e.is_retryable() => {
                Status::unavailable("SERVICE_UNAVAILABLE")
            }
            _ => Status::internal("INTERNAL_ERROR"),
        }
    }
}

impl From<serde_json::Error> for TokenError {
    fn from(err: serde_json::Error) -> Self {
        Self::Serialization(err.to_string())
    }
}

impl From<std::string::FromUtf8Error> for TokenError {
    fn from(err: std::string::FromUtf8Error) -> Self {
        Self::Serialization(err.to_string())
    }
}

impl From<jsonwebtoken::errors::Error> for TokenError {
    fn from(err: jsonwebtoken::errors::Error) -> Self {
        Self::JwtEncoding(err.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_retryable_errors() {
        assert!(TokenError::Kms("test".to_string()).is_retryable());
        assert!(TokenError::RateLimited.is_retryable());
    }

    #[test]
    fn test_non_retryable_errors() {
        assert!(!TokenError::RefreshInvalid.is_retryable());
        assert!(!TokenError::RefreshExpired.is_retryable());
        assert!(!TokenError::RefreshReplay.is_retryable());
        assert!(!TokenError::FamilyRevoked.is_retryable());
        assert!(!TokenError::DpopValidation("test".to_string()).is_retryable());
        assert!(!TokenError::DpopReplay("test".to_string()).is_retryable());
    }

    #[test]
    fn test_grpc_status_mapping() {
        let status: Status = TokenError::RefreshInvalid.into();
        assert_eq!(status.code(), tonic::Code::Unauthenticated);

        let status: Status = TokenError::RefreshReplay.into();
        assert_eq!(status.code(), tonic::Code::PermissionDenied);

        let status: Status = TokenError::DpopValidation("test".to_string()).into();
        assert_eq!(status.code(), tonic::Code::InvalidArgument);

        let status: Status = TokenError::RateLimited.into();
        assert_eq!(status.code(), tonic::Code::ResourceExhausted);
    }

    #[test]
    fn test_error_messages_do_not_expose_internals() {
        let status: Status = TokenError::Kms("secret key error".to_string()).into();
        assert!(!status.message().contains("secret"));
        assert!(!status.message().contains("key"));
    }
}
