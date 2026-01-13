//! Property tests for error message sanitization.
//!
//! **Feature: auth-edge-modernization-2025, Property 14: Sensitive Data Protection**
//! **Validates: Requirements 13.2, 13.3, 13.4**

use auth_edge::error::{sanitize_message, ErrorResponse, AuthEdgeError, SENSITIVE_PATTERNS};
use proptest::prelude::*;
use uuid::Uuid;

mod generators {
    include!("generators.rs");
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Feature: auth-edge-modernization-2025, Property 14: Sensitive Data Protection**
    /// **Validates: Requirements 13.2, 13.3, 13.4**
    ///
    /// *For any* string containing sensitive patterns, sanitize_message SHALL return
    /// a generic message that does not contain the sensitive pattern.
    #[test]
    fn sensitive_strings_are_sanitized(input in generators::arb_sensitive_string()) {
        let sanitized = sanitize_message(&input);
        let lower_sanitized = sanitized.to_lowercase();
        
        for pattern in SENSITIVE_PATTERNS {
            prop_assert!(
                !lower_sanitized.contains(pattern),
                "Sanitized message '{}' should not contain sensitive pattern '{}'",
                sanitized,
                pattern
            );
        }
    }

    /// **Feature: auth-edge-modernization-2025, Property 14: Sensitive Data Protection**
    /// **Validates: Requirements 13.2, 13.3, 13.4**
    ///
    /// *For any* string without sensitive patterns, sanitize_message SHALL preserve
    /// the original message.
    #[test]
    fn safe_strings_are_preserved(input in generators::arb_safe_string()) {
        let sanitized = sanitize_message(&input);
        prop_assert_eq!(
            sanitized,
            input,
            "Safe message should be preserved unchanged"
        );
    }

    /// **Feature: auth-edge-modernization-2025, Property 14: Sensitive Data Protection**
    /// **Validates: Requirements 13.2, 13.3, 13.4**
    ///
    /// *For any* error converted to ErrorResponse, the message SHALL NOT contain
    /// any sensitive patterns.
    #[test]
    fn error_responses_never_contain_sensitive_data(
        error in generators::arb_domain_error(),
        correlation_id in generators::arb_uuid()
    ) {
        let response = ErrorResponse::from_error(&error, correlation_id);
        let lower_message = response.message.to_lowercase();
        
        for pattern in SENSITIVE_PATTERNS {
            prop_assert!(
                !lower_message.contains(pattern),
                "Error response message '{}' should not contain sensitive pattern '{}'",
                response.message,
                pattern
            );
        }
    }

    /// **Feature: auth-edge-modernization-2025, Property 14: Sensitive Data Protection**
    /// **Validates: Requirements 13.2, 13.3, 13.4**
    ///
    /// *For any* TokenMalformed error with sensitive reason, the ErrorResponse
    /// message SHALL be sanitized.
    #[test]
    fn token_malformed_with_sensitive_reason_is_sanitized(
        sensitive_reason in generators::arb_sensitive_string(),
        correlation_id in generators::arb_uuid()
    ) {
        let error = AuthEdgeError::TokenMalformed { reason: sensitive_reason };
        let response = ErrorResponse::from_error(&error, correlation_id);
        let lower_message = response.message.to_lowercase();
        
        for pattern in SENSITIVE_PATTERNS {
            prop_assert!(
                !lower_message.contains(pattern),
                "TokenMalformed error response '{}' should not contain sensitive pattern '{}'",
                response.message,
                pattern
            );
        }
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[test]
    fn test_specific_sensitive_patterns() {
        // Test each sensitive pattern explicitly
        let test_cases = [
            ("my password is 123", "Invalid request"),
            ("secret key here", "Invalid request"),
            ("bearer token abc", "Invalid request"),
            ("api_key=xyz", "Invalid request"),
            ("private data", "Invalid request"),
            ("authorization header", "Invalid request"),
            ("credential leak", "Invalid request"),
        ];

        for (input, expected) in test_cases {
            let result = sanitize_message(input);
            assert_eq!(result, expected, "Failed for input: {}", input);
        }
    }

    #[test]
    fn test_case_insensitive_sanitization() {
        assert_eq!(sanitize_message("PASSWORD"), "Invalid request");
        assert_eq!(sanitize_message("PaSsWoRd"), "Invalid request");
        assert_eq!(sanitize_message("SECRET"), "Invalid request");
        assert_eq!(sanitize_message("TOKEN"), "Invalid request");
    }

    #[test]
    fn test_error_response_correlation_id_present() {
        let correlation_id = Uuid::new_v4();
        let error = AuthEdgeError::TokenMissing;
        let response = ErrorResponse::from_error(&error, correlation_id);
        
        assert_eq!(response.correlation_id, correlation_id);
        let status = response.to_status();
        assert!(status.message().contains(&correlation_id.to_string()));
    }
}
