//! Shared proptest generators for all Rust libraries.
//!
//! This module provides reusable generators for domain types used across
//! the auth-platform Rust libraries.

use proptest::prelude::*;
use std::time::Duration;

/// Generate valid CAEP event type strings.
pub fn caep_event_type_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("session-revoked".to_string()),
        Just("credential-change".to_string()),
        Just("assurance-level-change".to_string()),
        Just("token-claims-change".to_string()),
        Just("device-compliance-change".to_string()),
    ]
}

/// Generate valid subject identifier formats.
pub fn subject_identifier_strategy() -> impl Strategy<Value = SubjectIdentifier> {
    prop_oneof![
        ("[a-z]{5,20}", "[a-z0-9]{10,30}").prop_map(|(iss, sub)| {
            SubjectIdentifier::IssSub {
                iss: format!("https://{}.example.com", iss),
                sub,
            }
        }),
        "[a-z0-9._%+-]+@[a-z0-9.-]+\\.[a-z]{2,4}".prop_map(|email| {
            SubjectIdentifier::Email { email }
        }),
        "[a-z0-9]{32}".prop_map(|id| SubjectIdentifier::Opaque { id }),
        "[a-z0-9-]{36}".prop_map(|session_id| SubjectIdentifier::SessionId { session_id }),
    ]
}

/// Subject identifier types for testing.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SubjectIdentifier {
    /// Issuer and subject format
    IssSub {
        /// Issuer URL
        iss: String,
        /// Subject identifier
        sub: String,
    },
    /// Email format
    Email {
        /// Email address
        email: String,
    },
    /// Opaque identifier format
    Opaque {
        /// Opaque ID
        id: String,
    },
    /// Session ID format
    SessionId {
        /// Session ID
        session_id: String,
    },
}

/// Generate valid SPIFFE identities.
pub fn spiffe_identity_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{2,20}".prop_map(|name| {
        format!("spiffe://auth-platform.local/ns/auth-platform/sa/{}", name)
    })
}

/// Generate W3C Trace Context traceparent headers.
pub fn traceparent_strategy() -> impl Strategy<Value = String> {
    (
        Just("00"),
        "[0-9a-f]{32}",
        "[0-9a-f]{16}",
        prop_oneof![Just("00"), Just("01")],
    )
        .prop_map(|(version, trace_id, parent_id, flags)| {
            format!("{}-{}-{}-{}", version, trace_id, parent_id, flags)
        })
}

/// Generate valid secret paths.
pub fn secret_path_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{0,20}(/[a-z][a-z0-9-]{0,20}){0,3}"
}

/// Generate TTL values (1 minute to 24 hours).
pub fn ttl_strategy() -> impl Strategy<Value = Duration> {
    (60u64..86400).prop_map(Duration::from_secs)
}

/// Generate short TTL values for testing (1ms to 100ms).
pub fn short_ttl_strategy() -> impl Strategy<Value = Duration> {
    (1u64..100).prop_map(Duration::from_millis)
}

/// Generate valid correlation IDs.
pub fn correlation_id_strategy() -> impl Strategy<Value = String> {
    "[a-z0-9-]{8,36}"
}

/// Generate valid trace IDs (32 hex characters).
pub fn trace_id_strategy() -> impl Strategy<Value = String> {
    "[a-f0-9]{32}"
}

/// Generate valid span IDs (16 hex characters).
pub fn span_id_strategy() -> impl Strategy<Value = String> {
    "[a-f0-9]{16}"
}

/// Generate valid git commit SHAs (40 hex characters).
pub fn git_commit_sha_strategy() -> impl Strategy<Value = String> {
    "[a-f0-9]{40}"
}

/// Generate valid namespace names.
pub fn namespace_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{2,20}"
}

/// Generate valid cache keys.
pub fn cache_key_strategy() -> impl Strategy<Value = String> {
    "[a-z0-9:_-]{5,50}"
}

/// Generate valid service names.
pub fn service_name_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{2,30}-service"
}

/// Generate valid HTTP status codes.
pub fn http_status_code_strategy() -> impl Strategy<Value = u16> {
    prop_oneof![
        Just(200u16),
        Just(201u16),
        Just(204u16),
        Just(400u16),
        Just(401u16),
        Just(403u16),
        Just(404u16),
        Just(429u16),
        Just(500u16),
        Just(502u16),
        Just(503u16),
    ]
}

/// Generate valid JWT algorithm names.
pub fn jwt_algorithm_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("ES256".to_string()),
        Just("ES384".to_string()),
        Just("RS256".to_string()),
        Just("RS384".to_string()),
        Just("RS512".to_string()),
    ]
}

/// Generate valid log levels.
pub fn log_level_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("DEBUG".to_string()),
        Just("INFO".to_string()),
        Just("WARN".to_string()),
        Just("ERROR".to_string()),
        Just("FATAL".to_string()),
    ]
}

#[cfg(test)]
mod tests {
    use super::*;
    use proptest::strategy::ValueTree;
    use proptest::test_runner::TestRunner;

    #[test]
    fn test_spiffe_identity_format() {
        let mut runner = TestRunner::default();
        for _ in 0..10 {
            let value = spiffe_identity_strategy()
                .new_tree(&mut runner)
                .unwrap()
                .current();
            assert!(value.starts_with("spiffe://"));
            assert!(value.contains("/ns/"));
            assert!(value.contains("/sa/"));
        }
    }

    #[test]
    fn test_traceparent_format() {
        let mut runner = TestRunner::default();
        for _ in 0..10 {
            let value = traceparent_strategy()
                .new_tree(&mut runner)
                .unwrap()
                .current();
            let parts: Vec<&str> = value.split('-').collect();
            assert_eq!(parts.len(), 4);
            assert_eq!(parts[0], "00");
            assert_eq!(parts[1].len(), 32);
            assert_eq!(parts[2].len(), 16);
            assert!(parts[3] == "00" || parts[3] == "01");
        }
    }

    #[test]
    fn test_ttl_range() {
        let mut runner = TestRunner::default();
        for _ in 0..10 {
            let value = ttl_strategy()
                .new_tree(&mut runner)
                .unwrap()
                .current();
            assert!(value.as_secs() >= 60);
            assert!(value.as_secs() < 86400);
        }
    }
}
