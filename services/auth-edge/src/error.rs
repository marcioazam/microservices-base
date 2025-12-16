//! Error handling module with type-safe, non-exhaustive error types
//!
//! This module provides a unified error handling approach with:
//! - Non-exhaustive enums for forward compatibility
//! - Structured error variants with contextual information
//! - Automatic conversion from external error types
//! - Sanitization of sensitive information in responses

use chrono::{DateTime, Utc};
use std::time::Duration;
use thiserror::Error;
use tonic::{Code, Status};
use uuid::Uuid;

/// Sensitive patterns that should be sanitized from error messages
const SENSITIVE_PATTERNS: &[&str] = &[
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

/// Non-exhaustive error enum for forward compatibility
/// New variants can be added without breaking existing code
#[non_exhaustive]
#[derive(Error, Debug)]
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

    /// Downstream service is unavailable
    #[error("Service unavailable: {service}")]
    ServiceUnavailable {
        /// Name of the unavailable service
        service: String,
        /// Suggested retry duration
        retry_after: Duration,
    },

    /// Rate limit exceeded
    #[error("Rate limit exceeded")]
    RateLimited {
        /// When the client can retry
        retry_after: Duration,
    },

    /// Operation timed out
    #[error("Operation timed out after {duration:?}")]
    Timeout {
        /// How long the operation ran before timing out
        duration: Duration,
    },

    /// Circuit breaker is open
    #[error("Circuit breaker open for service: {service}")]
    CircuitOpen {
        /// Name of the service with open circuit
        service: String,
        /// When the circuit might close
        retry_after: Duration,
    },

    /// Internal error (details sanitized in responses)
    #[error(transparent)]
    Internal(#[from] anyhow::Error),
}

/// Error codes for gRPC/API responses
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ErrorCode {
    TokenMissing,
    TokenInvalid,
    TokenExpired,
    TokenMalformed,
    ClaimsInvalid,
    SpiffeError,
    CertificateError,
    ServiceUnavailable,
    RateLimited,
    Timeout,
    CircuitOpen,
    Internal,
}

impl ErrorCode {
    /// Get the string representation of the error code
    pub fn as_str(&self) -> &'static str {
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

    /// Get the gRPC status code for this error
    pub fn grpc_code(&self) -> Code {
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

/// Structured error response with correlation ID
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
    /// Create a new error response from an AuthEdgeError
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
                (ErrorCode::ClaimsInvalid, format!("Missing required claims: {:?}", claims), None)
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
            AuthEdgeError::ServiceUnavailable { service, retry_after } => {
                (ErrorCode::ServiceUnavailable, format!("Service {} temporarily unavailable", service), Some(*retry_after))
            }
            AuthEdgeError::RateLimited { retry_after } => {
                (ErrorCode::RateLimited, "Rate limit exceeded".to_string(), Some(*retry_after))
            }
            AuthEdgeError::Timeout { .. } => {
                (ErrorCode::Timeout, "Request timed out".to_string(), None)
            }
            AuthEdgeError::CircuitOpen { service, retry_after } => {
                (ErrorCode::CircuitOpen, format!("Service {} temporarily unavailable", service), Some(*retry_after))
            }
            AuthEdgeError::Internal(_) => {
                // Never expose internal error details
                (ErrorCode::Internal, "Internal error".to_string(), None)
            }
        };

        ErrorResponse {
            code,
            message,
            correlation_id,
            retry_after,
        }
    }

    /// Convert to gRPC Status
    pub fn to_status(&self) -> Status {
        let message = format!("{} [correlation_id: {}]", self.message, self.correlation_id);
        Status::new(self.code.grpc_code(), message)
    }
}

impl AuthEdgeError {
    /// Get the error code for this error
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
            Self::ServiceUnavailable { .. } => ErrorCode::ServiceUnavailable,
            Self::RateLimited { .. } => ErrorCode::RateLimited,
            Self::Timeout { .. } => ErrorCode::Timeout,
            Self::CircuitOpen { .. } => ErrorCode::CircuitOpen,
            Self::Internal(_) => ErrorCode::Internal,
        }
    }

    /// Convert to gRPC Status with correlation ID
    pub fn to_status(&self, correlation_id: Uuid) -> Status {
        ErrorResponse::from_error(self, correlation_id).to_status()
    }

    /// Check if this error is retryable
    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::ServiceUnavailable { .. }
                | Self::RateLimited { .. }
                | Self::Timeout { .. }
                | Self::CircuitOpen { .. }
        )
    }

    /// Get retry-after duration if applicable
    pub fn retry_after(&self) -> Option<Duration> {
        match self {
            Self::ServiceUnavailable { retry_after, .. } => Some(*retry_after),
            Self::RateLimited { retry_after } => Some(*retry_after),
            Self::CircuitOpen { retry_after, .. } => Some(*retry_after),
            _ => None,
        }
    }
}

/// Sanitize a message by removing sensitive information
fn sanitize_message(message: &str) -> String {
    let lower = message.to_lowercase();
    for pattern in SENSITIVE_PATTERNS {
        if lower.contains(pattern) {
            return "Invalid token format".to_string();
        }
    }
    message.to_string()
}

/// Check if a string contains sensitive information
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
                // Try to extract expiration time from error, fallback to now
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
            AuthEdgeError::Timeout {
                duration: Duration::from_secs(30), // Default timeout
            }
        } else if err.is_connect() {
            AuthEdgeError::ServiceUnavailable {
                service: "jwks".to_string(),
                retry_after: Duration::from_secs(5),
            }
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
            Code::Unavailable => AuthEdgeError::ServiceUnavailable {
                service: "downstream".to_string(),
                retry_after: Duration::from_secs(5),
            },
            Code::ResourceExhausted => AuthEdgeError::RateLimited {
                retry_after: Duration::from_secs(60),
            },
            Code::DeadlineExceeded => AuthEdgeError::Timeout {
                duration: Duration::from_secs(30),
            },
            _ => AuthEdgeError::Internal(anyhow::anyhow!("gRPC error: {}", status.message())),
        }
    }
}

impl From<std::io::Error> for AuthEdgeError {
    fn from(err: std::io::Error) -> Self {
        AuthEdgeError::Internal(anyhow::anyhow!("IO error: {}", err))
    }
}

// ============================================================================
// Legacy error code constants for backward compatibility
// ============================================================================

pub const AUTH_TOKEN_MISSING: &str = "AUTH_TOKEN_MISSING";
pub const AUTH_TOKEN_INVALID: &str = "AUTH_TOKEN_INVALID";
pub const AUTH_TOKEN_EXPIRED: &str = "AUTH_TOKEN_EXPIRED";
pub const AUTH_TOKEN_MALFORMED: &str = "AUTH_TOKEN_MALFORMED";
pub const AUTH_CLAIMS_INVALID: &str = "AUTH_CLAIMS_INVALID";
