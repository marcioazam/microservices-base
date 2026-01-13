//! Error handling module with type-safe, non-exhaustive error types.
//!
//! This module provides a unified error handling approach extending PlatformError
//! from rust-common with domain-specific AuthEdgeError variants.

use chrono::{DateTime, Utc};
use rust_common::PlatformError;
use std::time::Duration;
use thiserror::Error;
use tonic::{Code, Status};
use uuid::Uuid;

/// Sensitive patterns that should be sanitized from error messages.
pub const SENSITIVE_PATTERNS: &[&str] = &[
    "password",
    "secret",
    "token",
    "key",
    "credential",
    "bearer",
    "authorization",
    "api_key",
    "apikey",
    "private",
];

/// Non-exhaustive error enum for forward compatibility.
/// Extends PlatformError with domain-specific variants.
#[non_exhaustive]
#[derive(Error, Debug, Clone)]
pub enum AuthEdgeError {
    /// Token was not provided in the request
    #[error("Token missing from request")]
    TokenMissing,

    /// Token signature verification failed
    #[error("Token signature invalid")]
    TokenInvalid,

    /// Token has expired
    #[error("Token expired at {expired_at}")]
    TokenExpired {
        /// When the token expired
        expired_at: DateTime<Utc>,
    },

    /// Token is not yet valid (nbf claim)
    #[error("Token not yet valid until {valid_from}")]
    TokenNotYetValid {
        /// When the token becomes valid
        valid_from: DateTime<Utc>,
    },

    /// Token structure is malformed
    #[error("Token malformed: {reason}")]
    TokenMalformed {
        /// Description of the malformation
        reason: String,
    },

    /// Required claims are missing or invalid
    #[error("Required claims invalid: {claims:?}")]
    ClaimsInvalid {
        /// List of invalid or missing claims
        claims: Vec<String>,
    },

    /// SPIFFE ID extraction or validation failed
    #[error("SPIFFE ID error: {reason}")]
    SpiffeError {
        /// Description of the SPIFFE error
        reason: String,
    },

    /// Certificate validation failed
    #[error("Certificate validation failed: {reason}")]
    CertificateError {
        /// Description of the certificate error
        reason: String,
    },

    /// JWK cache operation failed
    #[error("JWK cache error: {reason}")]
    JwkCacheError {
        /// Description of the cache error
        reason: String,
    },

    /// Request was rate limited
    #[error("Rate limit exceeded, retry after {retry_after:?}")]
    RateLimited {
        /// Duration to wait before retrying
        retry_after: u64,
    },

    /// Request exceeded timeout
    #[error("Request timeout after {duration:?}")]
    Timeout {
        /// Duration that was exceeded
        duration: Duration,
    },

    /// Wraps PlatformError for infrastructure errors
    #[error(transparent)]
    Platform(#[from] PlatformError),
}

/// Error codes for gRPC/API responses.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ErrorCode {
    /// Token missing
    TokenMissing,
    /// Token invalid
    TokenInvalid,
    /// Token expired
    TokenExpired,
    /// Token malformed
    TokenMalformed,
    /// Claims invalid
    ClaimsInvalid,
    /// SPIFFE error
    SpiffeError,
    /// Certificate error
    CertificateError,
    /// Service unavailable
    ServiceUnavailable,
    /// Rate limited
    RateLimited,
    /// Timeout
    Timeout,
    /// Circuit open
    CircuitOpen,
    /// Internal error
    Internal,
}

impl ErrorCode {
    /// Get the string representation of the error code.
    #[must_use]
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::TokenMissing => "AUTH_TOKEN_MISSING",
            Self::TokenInvalid => "AUTH_TOKEN_INVALID",
            Self::TokenExpired => "AUTH_TOKEN_EXPIRED",
            Self::TokenMalformed => "AUTH_TOKEN_MALFORMED",
            Self::ClaimsInvalid => "AUTH_CLAIMS_INVALID",
            Self::SpiffeError => "AUTH_SPIFFE_ERROR",
            Self::CertificateError => "AUTH_CERTIFICATE_ERROR",
            Self::ServiceUnavailable => "SERVICE_UNAVAILABLE",
            Self::RateLimited => "RATE_LIMITED",
            Self::Timeout => "TIMEOUT",
            Self::CircuitOpen => "CIRCUIT_OPEN",
            Self::Internal => "INTERNAL_ERROR",
        }
    }

    /// Get the gRPC status code for this error.
    #[must_use]
    pub const fn grpc_code(&self) -> Code {
        match self {
            Self::TokenMissing | Self::TokenInvalid | Self::TokenExpired => Code::Unauthenticated,
            Self::TokenMalformed => Code::InvalidArgument,
            Self::ClaimsInvalid => Code::PermissionDenied,
            Self::SpiffeError | Self::CertificateError => Code::Unauthenticated,
            Self::ServiceUnavailable | Self::CircuitOpen => Code::Unavailable,
            Self::RateLimited => Code::ResourceExhausted,
            Self::Timeout => Code::DeadlineExceeded,
            Self::Internal => Code::Internal,
        }
    }
}

/// Structured error response with correlation ID.
#[derive(Debug, Clone)]
pub struct ErrorResponse {
    /// Error code for programmatic handling
    pub code: ErrorCode,
    /// Human-readable message (sanitized)
    pub message: String,
    /// Correlation ID for tracing
    pub correlation_id: Uuid,
    /// Optional retry-after duration
    pub retry_after: Option<Duration>,
}

impl ErrorResponse {
    /// Create a new error response from an AuthEdgeError.
    #[must_use]
    pub fn from_error(error: &AuthEdgeError, correlation_id: Uuid) -> Self {
        let (code, message, retry_after) = match error {
            AuthEdgeError::TokenMissing => {
                (ErrorCode::TokenMissing, "Token is required".to_string(), None)
            }
            AuthEdgeError::TokenInvalid => {
                (ErrorCode::TokenInvalid, "Token signature is invalid".to_string(), None)
            }
            AuthEdgeError::TokenExpired { .. } => {
                (ErrorCode::TokenExpired, "Token has expired".to_string(), None)
            }
            AuthEdgeError::TokenNotYetValid { .. } => {
                (ErrorCode::TokenMalformed, "Token is not yet valid".to_string(), None)
            }
            AuthEdgeError::TokenMalformed { reason } => {
                (ErrorCode::TokenMalformed, sanitize_message(reason), None)
            }
            AuthEdgeError::ClaimsInvalid { claims } => {
                (ErrorCode::ClaimsInvalid, format!("Missing required claims: {claims:?}"), None)
            }
            AuthEdgeError::SpiffeError { .. } => {
                (ErrorCode::SpiffeError, "SPIFFE ID validation failed".to_string(), None)
            }
            AuthEdgeError::CertificateError { .. } => {
                (ErrorCode::CertificateError, "Certificate validation failed".to_string(), None)
            }
            AuthEdgeError::JwkCacheError { .. } => {
                (ErrorCode::Internal, "Key validation temporarily unavailable".to_string(), None)
            }
            AuthEdgeError::RateLimited { retry_after } => {
                (ErrorCode::RateLimited, "Rate limit exceeded".to_string(), Some(Duration::from_secs(*retry_after)))
            }
            AuthEdgeError::Timeout { .. } => {
                (ErrorCode::Timeout, "Request timed out".to_string(), None)
            }
            AuthEdgeError::Platform(platform_err) => {
                map_platform_error(platform_err)
            }
        };

        ErrorResponse {
            code,
            message,
            correlation_id,
            retry_after,
        }
    }

    /// Convert to gRPC Status.
    #[must_use]
    pub fn to_status(&self) -> Status {
        let message = format!("{} [correlation_id: {}]", self.message, self.correlation_id);
        Status::new(self.code.grpc_code(), message)
    }
}

/// Map PlatformError to ErrorCode, message, and retry_after.
fn map_platform_error(err: &PlatformError) -> (ErrorCode, String, Option<Duration>) {
    match err {
        PlatformError::CircuitOpen { service } => {
            (ErrorCode::CircuitOpen, format!("Service {service} temporarily unavailable"), Some(Duration::from_secs(30)))
        }
        PlatformError::Unavailable(_) => {
            (ErrorCode::ServiceUnavailable, "Service temporarily unavailable".to_string(), Some(Duration::from_secs(5)))
        }
        PlatformError::RateLimited => {
            (ErrorCode::RateLimited, "Rate limit exceeded".to_string(), Some(Duration::from_secs(60)))
        }
        PlatformError::Timeout(_) => {
            (ErrorCode::Timeout, "Request timed out".to_string(), None)
        }
        _ => {
            (ErrorCode::Internal, "Internal error".to_string(), None)
        }
    }
}

impl AuthEdgeError {
    /// Get the error code for this error.
    #[must_use]
    pub fn code(&self) -> ErrorCode {
        match self {
            Self::TokenMissing => ErrorCode::TokenMissing,
            Self::TokenInvalid => ErrorCode::TokenInvalid,
            Self::TokenExpired { .. } => ErrorCode::TokenExpired,
            Self::TokenNotYetValid { .. } => ErrorCode::TokenMalformed,
            Self::TokenMalformed { .. } => ErrorCode::TokenMalformed,
            Self::ClaimsInvalid { .. } => ErrorCode::ClaimsInvalid,
            Self::SpiffeError { .. } => ErrorCode::SpiffeError,
            Self::CertificateError { .. } => ErrorCode::CertificateError,
            Self::JwkCacheError { .. } => ErrorCode::Internal,
            Self::RateLimited { .. } => ErrorCode::RateLimited,
            Self::Timeout { .. } => ErrorCode::Timeout,
            Self::Platform(e) => match e {
                PlatformError::CircuitOpen { .. } => ErrorCode::CircuitOpen,
                PlatformError::Unavailable(_) => ErrorCode::ServiceUnavailable,
                PlatformError::RateLimited => ErrorCode::RateLimited,
                PlatformError::Timeout(_) => ErrorCode::Timeout,
                _ => ErrorCode::Internal,
            },
        }
    }

    /// Convert to gRPC Status with correlation ID.
    #[must_use]
    pub fn to_status(&self, correlation_id: Uuid) -> Status {
        ErrorResponse::from_error(self, correlation_id).to_status()
    }

    /// Check if this error is retryable.
    /// Delegates to PlatformError for infrastructure errors.
    #[must_use]
    pub fn is_retryable(&self) -> bool {
        match self {
            Self::Platform(e) => e.is_retryable(),
            _ => false,
        }
    }

    /// Get retry-after duration if applicable.
    #[must_use]
    pub fn retry_after(&self) -> Option<Duration> {
        match self {
            Self::RateLimited { retry_after } => Some(Duration::from_secs(*retry_after)),
            Self::Platform(PlatformError::CircuitOpen { .. }) => Some(Duration::from_secs(30)),
            Self::Platform(PlatformError::Unavailable(_)) => Some(Duration::from_secs(5)),
            Self::Platform(PlatformError::RateLimited) => Some(Duration::from_secs(60)),
            _ => None,
        }
    }
}

/// Sanitize a message by removing sensitive information.
#[must_use]
pub fn sanitize_message(message: &str) -> String {
    let lower = message.to_lowercase();
    for pattern in SENSITIVE_PATTERNS {
        if lower.contains(pattern) {
            return "Invalid request".to_string();
        }
    }
    message.to_string()
}

/// Check if a string contains sensitive information.
#[must_use]
pub fn contains_sensitive_info(text: &str) -> bool {
    let lower = text.to_lowercase();
    SENSITIVE_PATTERNS.iter().any(|p| lower.contains(p))
}

// ============================================================================
// From trait implementations for automatic error conversion
// ============================================================================

impl From<jsonwebtoken::errors::Error> for AuthEdgeError {
    fn from(err: jsonwebtoken::errors::Error) -> Self {
        use jsonwebtoken::errors::ErrorKind;
        
        match err.kind() {
            ErrorKind::ExpiredSignature => {
                AuthEdgeError::TokenExpired {
                    expired_at: Utc::now(),
                }
            }
            ErrorKind::ImmatureSignature => {
                AuthEdgeError::TokenNotYetValid {
                    valid_from: Utc::now(),
                }
            }
            ErrorKind::InvalidSignature => AuthEdgeError::TokenInvalid,
            ErrorKind::InvalidToken
            | ErrorKind::InvalidAlgorithm
            | ErrorKind::InvalidAlgorithmName
            | ErrorKind::MissingAlgorithm => {
                AuthEdgeError::TokenMalformed {
                    reason: sanitize_message(&err.to_string()),
                }
            }
            ErrorKind::MissingRequiredClaim(claim) => {
                AuthEdgeError::ClaimsInvalid {
                    claims: vec![claim.to_string()],
                }
            }
            _ => AuthEdgeError::TokenMalformed {
                reason: "Token validation failed".to_string(),
            },
        }
    }
}

impl From<reqwest::Error> for AuthEdgeError {
    fn from(err: reqwest::Error) -> Self {
        if err.is_timeout() {
            AuthEdgeError::Platform(PlatformError::Timeout("JWKS fetch timed out".to_string()))
        } else if err.is_connect() {
            AuthEdgeError::Platform(PlatformError::Unavailable("JWKS endpoint unavailable".to_string()))
        } else {
            AuthEdgeError::JwkCacheError {
                reason: sanitize_message(&err.to_string()),
            }
        }
    }
}

impl From<tonic::Status> for AuthEdgeError {
    fn from(status: tonic::Status) -> Self {
        match status.code() {
            Code::Unauthenticated => AuthEdgeError::TokenInvalid,
            Code::PermissionDenied => AuthEdgeError::ClaimsInvalid {
                claims: vec!["permission".to_string()],
            },
            Code::Unavailable => AuthEdgeError::Platform(
                PlatformError::Unavailable("downstream service".to_string())
            ),
            Code::ResourceExhausted => AuthEdgeError::Platform(PlatformError::RateLimited),
            Code::DeadlineExceeded => AuthEdgeError::Platform(
                PlatformError::Timeout("request".to_string())
            ),
            _ => AuthEdgeError::Platform(
                PlatformError::Internal(sanitize_message(status.message()))
            ),
        }
    }
}

impl From<std::io::Error> for AuthEdgeError {
    fn from(err: std::io::Error) -> Self {
        AuthEdgeError::Platform(PlatformError::Internal(format!("IO error: {err}")))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_domain_errors_not_retryable() {
        assert!(!AuthEdgeError::TokenMissing.is_retryable());
        assert!(!AuthEdgeError::TokenInvalid.is_retryable());
        assert!(!AuthEdgeError::TokenExpired { expired_at: Utc::now() }.is_retryable());
        assert!(!AuthEdgeError::ClaimsInvalid { claims: vec![] }.is_retryable());
        assert!(!AuthEdgeError::SpiffeError { reason: "test".to_string() }.is_retryable());
    }

    #[test]
    fn test_platform_errors_retryable() {
        assert!(AuthEdgeError::Platform(PlatformError::RateLimited).is_retryable());
        assert!(AuthEdgeError::Platform(PlatformError::Unavailable("test".to_string())).is_retryable());
        assert!(AuthEdgeError::Platform(PlatformError::Timeout("test".to_string())).is_retryable());
    }

    #[test]
    fn test_platform_errors_not_retryable() {
        assert!(!AuthEdgeError::Platform(PlatformError::circuit_open("test")).is_retryable());
        assert!(!AuthEdgeError::Platform(PlatformError::NotFound("test".to_string())).is_retryable());
        assert!(!AuthEdgeError::Platform(PlatformError::AuthFailed("test".to_string())).is_retryable());
    }

    #[test]
    fn test_sanitize_message() {
        assert_eq!(sanitize_message("normal message"), "normal message");
        assert_eq!(sanitize_message("contains password"), "Invalid request");
        assert_eq!(sanitize_message("has SECRET data"), "Invalid request");
        assert_eq!(sanitize_message("bearer TOKEN here"), "Invalid request");
        assert_eq!(sanitize_message("api_key exposed"), "Invalid request");
    }

    #[test]
    fn test_contains_sensitive_info() {
        assert!(!contains_sensitive_info("normal message"));
        assert!(contains_sensitive_info("contains password"));
        assert!(contains_sensitive_info("has SECRET data"));
        assert!(contains_sensitive_info("bearer TOKEN here"));
    }

    #[test]
    fn test_error_response_includes_correlation_id() {
        let correlation_id = Uuid::new_v4();
        let error = AuthEdgeError::TokenMissing;
        let response = ErrorResponse::from_error(&error, correlation_id);
        
        assert_eq!(response.correlation_id, correlation_id);
        let status = response.to_status();
        assert!(status.message().contains(&correlation_id.to_string()));
    }
}
