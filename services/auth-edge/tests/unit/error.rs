//! Error Handling Unit Tests
//!
//! Tests for error sanitization, sensitive data detection, and error codes.

use uuid::Uuid;

const SENSITIVE_PATTERNS: &[&str] = &[
    "password", "secret", "token", "key", "credential",
    "bearer", "authorization", "api_key", "apikey", "private",
];

fn sanitize_message(message: &str) -> String {
    let lower = message.to_lowercase();
    for pattern in SENSITIVE_PATTERNS {
        if lower.contains(pattern) {
            return "Invalid token format".to_string();
        }
    }
    message.to_string()
}

fn contains_sensitive_info(text: &str) -> bool {
    let lower = text.to_lowercase();
    SENSITIVE_PATTERNS.iter().any(|p| lower.contains(p))
}

// ============================================================================
// Sanitization Tests
// ============================================================================

#[test]
fn test_sanitize_message_with_password() {
    assert_eq!(sanitize_message("Invalid password"), "Invalid token format");
    assert_eq!(sanitize_message("PASSWORD_INVALID"), "Invalid token format");
}

#[test]
fn test_sanitize_message_with_secret() {
    assert_eq!(sanitize_message("Missing secret key"), "Invalid token format");
    assert_eq!(sanitize_message("SECRET_NOT_FOUND"), "Invalid token format");
}

#[test]
fn test_sanitize_message_with_token() {
    assert_eq!(sanitize_message("Token expired"), "Invalid token format");
    assert_eq!(sanitize_message("Invalid TOKEN"), "Invalid token format");
}

#[test]
fn test_sanitize_message_with_bearer() {
    assert_eq!(sanitize_message("Bearer auth failed"), "Invalid token format");
}

#[test]
fn test_sanitize_message_with_api_key() {
    assert_eq!(sanitize_message("api_key missing"), "Invalid token format");
    assert_eq!(sanitize_message("apikey invalid"), "Invalid token format");
}

#[test]
fn test_sanitize_message_safe_content() {
    assert_eq!(sanitize_message("Invalid header"), "Invalid header");
    assert_eq!(sanitize_message("Missing claim"), "Missing claim");
    assert_eq!(sanitize_message("Expired"), "Expired");
}

#[test]
fn test_contains_sensitive_info_true() {
    assert!(contains_sensitive_info("password=123"));
    assert!(contains_sensitive_info("Bearer eyJ..."));
    assert!(contains_sensitive_info("api_key: sk-123"));
    assert!(contains_sensitive_info("private_key data"));
    assert!(contains_sensitive_info("credential: admin"));
}

#[test]
fn test_contains_sensitive_info_false() {
    assert!(!contains_sensitive_info("Invalid header format"));
    assert!(!contains_sensitive_info("Missing required claim"));
    assert!(!contains_sensitive_info("Service unavailable"));
}

// ============================================================================
// Correlation ID Tests
// ============================================================================

#[test]
fn test_correlation_id_uniqueness() {
    let id1 = Uuid::new_v4();
    let id2 = Uuid::new_v4();
    assert_ne!(id1, id2);
}

#[test]
fn test_correlation_id_format() {
    let id = Uuid::new_v4();
    let str_id = id.to_string();
    assert_eq!(str_id.len(), 36);
    assert!(str_id.contains('-'));
}

// ============================================================================
// Error Code Tests
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum ErrorCode {
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
    fn as_str(&self) -> &'static str {
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

    fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::ServiceUnavailable | Self::RateLimited | Self::Timeout | Self::CircuitOpen
        )
    }
}

#[test]
fn test_error_code_strings() {
    assert_eq!(ErrorCode::TokenMissing.as_str(), "AUTH_TOKEN_MISSING");
    assert_eq!(ErrorCode::TokenInvalid.as_str(), "AUTH_TOKEN_INVALID");
    assert_eq!(ErrorCode::TokenExpired.as_str(), "AUTH_TOKEN_EXPIRED");
    assert_eq!(ErrorCode::Internal.as_str(), "INTERNAL_ERROR");
}

#[test]
fn test_retryable_errors() {
    assert!(ErrorCode::ServiceUnavailable.is_retryable());
    assert!(ErrorCode::RateLimited.is_retryable());
    assert!(ErrorCode::Timeout.is_retryable());
    assert!(ErrorCode::CircuitOpen.is_retryable());
}

#[test]
fn test_non_retryable_errors() {
    assert!(!ErrorCode::TokenMissing.is_retryable());
    assert!(!ErrorCode::TokenInvalid.is_retryable());
    assert!(!ErrorCode::TokenExpired.is_retryable());
    assert!(!ErrorCode::ClaimsInvalid.is_retryable());
    assert!(!ErrorCode::Internal.is_retryable());
}
