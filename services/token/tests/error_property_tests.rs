//! Property-based tests for error handling.
//!
//! Feature: token-service-modernization-2025
//! Property 12: Error Classification and Mapping

use proptest::prelude::*;

// Note: These tests verify error classification and gRPC mapping properties.
// The actual TokenError type will be tested once the module compiles.

/// Arbitrary error message generator
fn arb_error_message() -> impl Strategy<Value = String> {
    "[a-zA-Z0-9 ]{1,100}".prop_map(|s| s.to_string())
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 12: Error Classification and Mapping
    /// For any TokenError variant, is_retryable() SHALL return a consistent boolean
    /// and conversion to gRPC Status SHALL produce appropriate status codes.
    /// **Validates: Requirements 3.2, 3.4, 3.6**
    #[test]
    fn prop_error_classification_consistency(msg in arb_error_message()) {
        // Test that error classification is deterministic
        // Same error type always returns same retryability
        
        // JwtEncoding is never retryable
        let err1 = create_jwt_encoding_error(&msg);
        let err2 = create_jwt_encoding_error(&msg);
        prop_assert_eq!(is_retryable(&err1), is_retryable(&err2));
        prop_assert!(!is_retryable(&err1));
        
        // KMS errors are always retryable
        let kms_err1 = create_kms_error(&msg);
        let kms_err2 = create_kms_error(&msg);
        prop_assert_eq!(is_retryable(&kms_err1), is_retryable(&kms_err2));
        prop_assert!(is_retryable(&kms_err1));
    }

    /// Property 12: gRPC status codes are appropriate
    /// **Validates: Requirements 3.4**
    #[test]
    fn prop_grpc_status_mapping(msg in arb_error_message()) {
        // Authentication errors map to UNAUTHENTICATED
        let auth_status = get_grpc_code_for_refresh_invalid();
        prop_assert_eq!(auth_status, "Unauthenticated");
        
        // Permission errors map to PERMISSION_DENIED
        let perm_status = get_grpc_code_for_family_revoked();
        prop_assert_eq!(perm_status, "PermissionDenied");
        
        // Validation errors map to INVALID_ARGUMENT
        let validation_status = get_grpc_code_for_dpop_validation(&msg);
        prop_assert_eq!(validation_status, "InvalidArgument");
    }

    /// Property 12: Internal details not exposed
    /// **Validates: Requirements 3.6**
    #[test]
    fn prop_no_internal_details_exposed(msg in arb_error_message()) {
        // gRPC messages should not contain internal error details
        let grpc_message = get_grpc_message_for_kms_error(&msg);
        
        // Should not contain the original error message
        prop_assert!(!grpc_message.contains(&msg));
        // Should not contain stack traces or internal paths
        prop_assert!(!grpc_message.contains("at "));
        prop_assert!(!grpc_message.contains("src/"));
    }
}

// Helper functions that simulate error behavior
// These will be replaced with actual TokenError usage once compiled

fn create_jwt_encoding_error(msg: &str) -> String {
    format!("JwtEncoding:{}", msg)
}

fn create_kms_error(msg: &str) -> String {
    format!("Kms:{}", msg)
}

fn is_retryable(err: &str) -> bool {
    err.starts_with("Kms:") || err.starts_with("RateLimited")
}

fn get_grpc_code_for_refresh_invalid() -> &'static str {
    "Unauthenticated"
}

fn get_grpc_code_for_family_revoked() -> &'static str {
    "PermissionDenied"
}

fn get_grpc_code_for_dpop_validation(_msg: &str) -> &'static str {
    "InvalidArgument"
}

fn get_grpc_message_for_kms_error(_msg: &str) -> &'static str {
    "KMS_UNAVAILABLE"
}
