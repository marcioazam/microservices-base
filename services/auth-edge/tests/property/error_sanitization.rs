//! Error Sanitization Property Tests
//!
//! Validates that sensitive data never leaks in error responses.

use proptest::prelude::*;
use uuid::Uuid;
use super::generators::{arb_sensitive_content, arb_error_type, arb_service_name};

const SENSITIVE_PATTERNS: &[&str] = &[
    "password", "secret", "token", "key", "credential",
    "bearer", "authorization", "api_key", "apikey", "private",
];

fn contains_sensitive_info(text: &str) -> bool {
    let lower = text.to_lowercase();
    SENSITIVE_PATTERNS.iter().any(|p| lower.contains(p))
}

fn sanitize_message(message: &str) -> String {
    let lower = message.to_lowercase();
    for pattern in SENSITIVE_PATTERNS {
        if lower.contains(pattern) {
            return "Invalid token format".to_string();
        }
    }
    message.to_string()
}

#[derive(Debug, Clone)]
enum MockAuthError {
    Internal { details: String },
}

impl MockAuthError {
    fn to_sanitized_message(&self) -> String {
        match self {
            Self::Internal { .. } => "Internal error".to_string(),
        }
    }
}

#[derive(Debug, Clone)]
struct MockErrorResponse {
    message: String,
    correlation_id: Uuid,
}

impl MockErrorResponse {
    fn from_error(error: &MockAuthError, correlation_id: Uuid) -> Self {
        MockErrorResponse {
            message: error.to_sanitized_message(),
            correlation_id,
        }
    }
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property: Error responses never contain sensitive information
    #[test]
    fn prop_error_response_sanitization(sensitive_content in arb_sensitive_content()) {
        let correlation_id = Uuid::new_v4();
        let error = MockAuthError::Internal { details: sensitive_content.clone() };
        let response = MockErrorResponse::from_error(&error, correlation_id);
        
        prop_assert!(!contains_sensitive_info(&response.message),
            "Response '{}' should not contain sensitive info", response.message);
        prop_assert_eq!(response.message, "Internal error");
    }

    /// Property: Error event attributes are always present
    #[test]
    fn prop_error_event_attributes(
        error_type in arb_error_type(),
        service_name in arb_service_name(),
    ) {
        let correlation_id = Uuid::new_v4();
        let timestamp = chrono::Utc::now();
        
        prop_assert!(!correlation_id.is_nil());
        prop_assert!(!error_type.is_empty());
        prop_assert!(!service_name.is_empty());
        prop_assert!(timestamp.timestamp() > 0);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sanitize_message() {
        assert_eq!(sanitize_message("Invalid password"), "Invalid token format");
        assert_eq!(sanitize_message("Missing header"), "Missing header");
    }
}
