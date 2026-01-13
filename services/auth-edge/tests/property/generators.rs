//! Proptest generators for Auth Edge Service types.

use auth_edge::error::AuthEdgeError;
use chrono::Utc;
use proptest::prelude::*;
use rust_common::PlatformError;

/// Generate arbitrary AuthEdgeError domain variants (non-Platform).
pub fn arb_domain_error() -> impl Strategy<Value = AuthEdgeError> {
    prop_oneof![
        Just(AuthEdgeError::TokenMissing),
        Just(AuthEdgeError::TokenInvalid),
        Just(AuthEdgeError::TokenExpired { expired_at: Utc::now() }),
        Just(AuthEdgeError::TokenNotYetValid { valid_from: Utc::now() }),
        "[a-zA-Z0-9 ]{1,50}".prop_map(|reason| AuthEdgeError::TokenMalformed { reason }),
        prop::collection::vec("[a-zA-Z0-9]{1,20}", 1..5)
            .prop_map(|claims| AuthEdgeError::ClaimsInvalid { claims }),
        "[a-zA-Z0-9 ]{1,50}".prop_map(|reason| AuthEdgeError::SpiffeError { reason }),
        "[a-zA-Z0-9 ]{1,50}".prop_map(|reason| AuthEdgeError::CertificateError { reason }),
        "[a-zA-Z0-9 ]{1,50}".prop_map(|reason| AuthEdgeError::JwkCacheError { reason }),
    ]
}

/// Generate arbitrary retryable PlatformError variants.
pub fn arb_retryable_platform_error() -> impl Strategy<Value = PlatformError> {
    prop_oneof![
        Just(PlatformError::RateLimited),
        "[a-zA-Z0-9 ]{1,50}".prop_map(PlatformError::Unavailable),
        "[a-zA-Z0-9 ]{1,50}".prop_map(PlatformError::Timeout),
    ]
}

/// Generate arbitrary non-retryable PlatformError variants.
pub fn arb_non_retryable_platform_error() -> impl Strategy<Value = PlatformError> {
    prop_oneof![
        "[a-zA-Z0-9]{1,30}".prop_map(|s| PlatformError::circuit_open(s)),
        "[a-zA-Z0-9 ]{1,50}".prop_map(PlatformError::NotFound),
        "[a-zA-Z0-9 ]{1,50}".prop_map(PlatformError::AuthFailed),
        "[a-zA-Z0-9 ]{1,50}".prop_map(PlatformError::InvalidInput),
        "[a-zA-Z0-9 ]{1,50}".prop_map(PlatformError::Internal),
        "[a-zA-Z0-9 ]{1,50}".prop_map(PlatformError::Encryption),
    ]
}

/// Generate arbitrary AuthEdgeError wrapping retryable PlatformError.
pub fn arb_retryable_auth_error() -> impl Strategy<Value = AuthEdgeError> {
    arb_retryable_platform_error().prop_map(AuthEdgeError::Platform)
}

/// Generate arbitrary AuthEdgeError wrapping non-retryable PlatformError.
pub fn arb_non_retryable_platform_auth_error() -> impl Strategy<Value = AuthEdgeError> {
    arb_non_retryable_platform_error().prop_map(AuthEdgeError::Platform)
}

/// Generate arbitrary UUID.
pub fn arb_uuid() -> impl Strategy<Value = uuid::Uuid> {
    any::<[u8; 16]>().prop_map(uuid::Uuid::from_bytes)
}

/// Generate strings containing sensitive patterns.
pub fn arb_sensitive_string() -> impl Strategy<Value = String> {
    prop_oneof![
        "[a-zA-Z0-9 ]{0,20}password[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}secret[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}token[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}key[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}credential[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}bearer[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}authorization[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}api_key[a-zA-Z0-9 ]{0,20}",
        "[a-zA-Z0-9 ]{0,20}private[a-zA-Z0-9 ]{0,20}",
    ]
}

/// Generate strings without sensitive patterns.
pub fn arb_safe_string() -> impl Strategy<Value = String> {
    "[a-zA-Z0-9 ]{1,100}".prop_filter("must not contain sensitive patterns", |s| {
        let lower = s.to_lowercase();
        !auth_edge::error::SENSITIVE_PATTERNS.iter().any(|p| lower.contains(p))
    })
}
