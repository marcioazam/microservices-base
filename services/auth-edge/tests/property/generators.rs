//! Proptest Generators
//!
//! Shared generators for property-based tests.

use proptest::prelude::*;

/// Generates sensitive content patterns
pub fn arb_sensitive_content() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("password=secret123".to_string()),
        Just("Bearer eyJhbGciOiJIUzI1NiJ9".to_string()),
        Just("api_key: sk-1234567890".to_string()),
        Just("credential: admin:password".to_string()),
        Just("private_key: -----BEGIN RSA-----".to_string()),
        "[a-zA-Z0-9_]{5,20}".prop_map(|s| format!("password={}", s)),
    ]
}

/// Generates valid SPIFFE trust domains
pub fn arb_trust_domain() -> impl Strategy<Value = String> {
    "[a-z0-9][a-z0-9-]{2,20}\\.[a-z]{2,6}"
}

/// Generates SPIFFE paths
pub fn arb_spiffe_path() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("".to_string()),
        "[a-z0-9/]{1,30}".prop_map(|s| s.replace("//", "/")),
    ]
}

/// Generates error type names
pub fn arb_error_type() -> impl Strategy<Value = String> {
    "[A-Z][a-zA-Z]{5,20}"
}

/// Generates service names
pub fn arb_service_name() -> impl Strategy<Value = String> {
    "[a-z][a-z-]{5,20}"
}
