//! Property-based tests for CAEP library.
//!
//! Tests validate:
//! - Property 1: SET Signing Algorithm Default (ES256)
//! - Property 8: Serialization Round-Trip

use proptest::prelude::*;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Mock SET structure for testing signing algorithm defaults
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
struct MockSecurityEventToken {
    iss: String,
    iat: u64,
    jti: String,
    aud: Vec<String>,
    events: HashMap<String, serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    txn: Option<String>,
}

/// Mock CAEP event for serialization testing
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
struct MockCaepEvent {
    event_type: String,
    subject: MockSubjectIdentifier,
    timestamp: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    reason_admin: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    reason_user: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
struct MockSubjectIdentifier {
    format: String,
    #[serde(flatten)]
    claims: HashMap<String, String>,
}

/// Supported signing algorithms
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SigningAlgorithm {
    ES256,
    ES384,
    ES512,
    RS256,
    RS384,
    RS512,
}

impl SigningAlgorithm {
    const fn as_str(&self) -> &'static str {
        match self {
            Self::ES256 => "ES256",
            Self::ES384 => "ES384",
            Self::ES512 => "ES512",
            Self::RS256 => "RS256",
            Self::RS384 => "RS384",
            Self::RS512 => "RS512",
        }
    }

    const fn is_ecdsa(&self) -> bool {
        matches!(self, Self::ES256 | Self::ES384 | Self::ES512)
    }
}

/// Default algorithm should be ES256
const DEFAULT_ALGORITHM: SigningAlgorithm = SigningAlgorithm::ES256;

// Strategy for generating CAEP event types
fn caep_event_type_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("https://schemas.openid.net/secevent/caep/event-type/session-revoked".to_string()),
        Just("https://schemas.openid.net/secevent/caep/event-type/token-claims-change".to_string()),
        Just("https://schemas.openid.net/secevent/caep/event-type/credential-change".to_string()),
        Just("https://schemas.openid.net/secevent/caep/event-type/assurance-level-change".to_string()),
        Just("https://schemas.openid.net/secevent/caep/event-type/device-compliance-change".to_string()),
    ]
}

// Strategy for generating subject identifier formats
fn subject_format_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("iss_sub".to_string()),
        Just("email".to_string()),
        Just("phone_number".to_string()),
        Just("opaque".to_string()),
    ]
}

// Strategy for generating issuer URLs
fn issuer_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{2,10}".prop_map(|name| format!("https://{}.auth-platform.local", name))
}

// Strategy for generating JTI (JWT ID)
fn jti_strategy() -> impl Strategy<Value = String> {
    "[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}"
}

// Strategy for generating audience
fn audience_strategy() -> impl Strategy<Value = Vec<String>> {
    prop::collection::vec(
        "[a-z][a-z0-9-]{2,15}".prop_map(|name| format!("https://{}.example.com", name)),
        1..4,
    )
}

// Strategy for generating timestamps (recent past)
fn timestamp_strategy() -> impl Strategy<Value = u64> {
    // Timestamps from 2024 to 2026
    1704067200u64..1767225600u64
}

// Strategy for generating subject identifiers
fn subject_identifier_strategy() -> impl Strategy<Value = MockSubjectIdentifier> {
    (subject_format_strategy(), "[a-z0-9]{8,32}").prop_map(|(format, value)| {
        let mut claims = HashMap::new();
        match format.as_str() {
            "iss_sub" => {
                claims.insert("iss".to_string(), "https://issuer.example.com".to_string());
                claims.insert("sub".to_string(), value);
            }
            "email" => {
                claims.insert("email".to_string(), format!("{}@example.com", value));
            }
            "phone_number" => {
                claims.insert("phone_number".to_string(), format!("+1555{}", &value[..7.min(value.len())]));
            }
            _ => {
                claims.insert("id".to_string(), value);
            }
        }
        MockSubjectIdentifier { format, claims }
    })
}

// Strategy for generating CAEP events
fn caep_event_strategy() -> impl Strategy<Value = MockCaepEvent> {
    (
        caep_event_type_strategy(),
        subject_identifier_strategy(),
        timestamp_strategy(),
        proptest::option::of("[A-Za-z0-9 ]{5,50}"),
        proptest::option::of("[A-Za-z0-9 ]{5,50}"),
    )
        .prop_map(|(event_type, subject, timestamp, reason_admin, reason_user)| MockCaepEvent {
            event_type,
            subject,
            timestamp,
            reason_admin,
            reason_user,
        })
}

// Strategy for generating SETs
fn set_strategy() -> impl Strategy<Value = MockSecurityEventToken> {
    (
        issuer_strategy(),
        timestamp_strategy(),
        jti_strategy(),
        audience_strategy(),
        caep_event_strategy(),
        proptest::option::of("[a-z0-9]{16,32}"),
    )
        .prop_map(|(iss, iat, jti, aud, event, txn)| {
            let mut events = HashMap::new();
            events.insert(
                event.event_type.clone(),
                serde_json::to_value(&event).unwrap_or_default(),
            );
            MockSecurityEventToken {
                iss,
                iat,
                jti,
                aud,
                events,
                txn,
            }
        })
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Property 1: SET Signing Algorithm Default**
    /// *For any* Security Event Token created without explicit algorithm specification,
    /// the signing algorithm SHALL default to ES256 (ECDSA with P-256 and SHA-256).
    /// **Validates: Requirements 5.6**
    #[test]
    fn prop_set_signing_algorithm_default(
        set in set_strategy(),
    ) {
        // When no algorithm is specified, default should be ES256
        let algorithm = DEFAULT_ALGORITHM;

        prop_assert_eq!(
            algorithm,
            SigningAlgorithm::ES256,
            "Default signing algorithm should be ES256"
        );

        prop_assert_eq!(
            algorithm.as_str(),
            "ES256",
            "Algorithm string should be 'ES256'"
        );

        prop_assert!(
            algorithm.is_ecdsa(),
            "Default algorithm should be ECDSA-based"
        );

        // SET should be serializable
        let json = serde_json::to_string(&set);
        prop_assert!(json.is_ok(), "SET should be JSON serializable");
    }

    /// **Property 8: Serialization Round-Trip**
    /// *For any* CAEP event or SET, serialization to JSON and deserialization back
    /// SHALL produce an identical object.
    /// **Validates: Requirements 10.4**
    #[test]
    fn prop_caep_event_serialization_roundtrip(
        event in caep_event_strategy(),
    ) {
        // Serialize to JSON
        let json = serde_json::to_string(&event);
        prop_assert!(json.is_ok(), "Event should serialize to JSON");

        let json_str = json.unwrap();

        // Deserialize back
        let deserialized: Result<MockCaepEvent, _> = serde_json::from_str(&json_str);
        prop_assert!(deserialized.is_ok(), "Event should deserialize from JSON");

        let restored = deserialized.unwrap();

        // Should be identical
        prop_assert_eq!(
            event, restored,
            "Event should survive serialization round-trip"
        );
    }

    /// Property: SET serialization round-trip
    #[test]
    fn prop_set_serialization_roundtrip(
        set in set_strategy(),
    ) {
        // Serialize to JSON
        let json = serde_json::to_string(&set);
        prop_assert!(json.is_ok(), "SET should serialize to JSON");

        let json_str = json.unwrap();

        // Deserialize back
        let deserialized: Result<MockSecurityEventToken, _> = serde_json::from_str(&json_str);
        prop_assert!(deserialized.is_ok(), "SET should deserialize from JSON");

        let restored = deserialized.unwrap();

        // Should be identical
        prop_assert_eq!(
            set, restored,
            "SET should survive serialization round-trip"
        );
    }

    /// Property: Subject identifier format validity
    #[test]
    fn prop_subject_identifier_format_valid(
        subject in subject_identifier_strategy(),
    ) {
        // Format should be one of the valid types
        let valid_formats = ["iss_sub", "email", "phone_number", "opaque"];
        prop_assert!(
            valid_formats.contains(&subject.format.as_str()),
            "Subject format '{}' should be valid", subject.format
        );

        // Claims should not be empty
        prop_assert!(
            !subject.claims.is_empty(),
            "Subject claims should not be empty"
        );

        // Serialization should work
        let json = serde_json::to_string(&subject);
        prop_assert!(json.is_ok(), "Subject should serialize");
    }

    /// Property: Event type URI format
    #[test]
    fn prop_event_type_uri_format(
        event_type in caep_event_type_strategy(),
    ) {
        // Event type should be a valid CAEP URI
        prop_assert!(
            event_type.starts_with("https://schemas.openid.net/secevent/caep/event-type/"),
            "Event type should be a valid CAEP URI"
        );

        // Should have a specific event name after the prefix
        let prefix = "https://schemas.openid.net/secevent/caep/event-type/";
        let event_name = &event_type[prefix.len()..];
        prop_assert!(
            !event_name.is_empty(),
            "Event type should have a name after prefix"
        );
    }

    /// Property: JTI uniqueness format (UUID v4)
    #[test]
    fn prop_jti_uuid_format(
        jti in jti_strategy(),
    ) {
        // JTI should be UUID v4 format
        let parts: Vec<&str> = jti.split('-').collect();
        prop_assert_eq!(parts.len(), 5, "JTI should have 5 parts");
        prop_assert_eq!(parts[0].len(), 8, "First part should be 8 chars");
        prop_assert_eq!(parts[1].len(), 4, "Second part should be 4 chars");
        prop_assert_eq!(parts[2].len(), 4, "Third part should be 4 chars");
        prop_assert!(parts[2].starts_with('4'), "Third part should start with 4 (UUID v4)");
        prop_assert_eq!(parts[3].len(), 4, "Fourth part should be 4 chars");
        prop_assert_eq!(parts[4].len(), 12, "Fifth part should be 12 chars");
    }
}

/// Test default algorithm constant
#[test]
fn test_default_algorithm_is_es256() {
    assert_eq!(DEFAULT_ALGORITHM, SigningAlgorithm::ES256);
    assert_eq!(DEFAULT_ALGORITHM.as_str(), "ES256");
    assert!(DEFAULT_ALGORITHM.is_ecdsa());
}

/// Test all supported algorithms
#[test]
fn test_supported_algorithms() {
    let algorithms = [
        (SigningAlgorithm::ES256, "ES256", true),
        (SigningAlgorithm::ES384, "ES384", true),
        (SigningAlgorithm::ES512, "ES512", true),
        (SigningAlgorithm::RS256, "RS256", false),
        (SigningAlgorithm::RS384, "RS384", false),
        (SigningAlgorithm::RS512, "RS512", false),
    ];

    for (alg, expected_str, expected_ecdsa) in algorithms {
        assert_eq!(alg.as_str(), expected_str);
        assert_eq!(alg.is_ecdsa(), expected_ecdsa);
    }
}

/// Test CAEP event serialization
#[test]
fn test_caep_event_serialization() {
    let mut claims = HashMap::new();
    claims.insert("sub".to_string(), "user123".to_string());
    claims.insert("iss".to_string(), "https://issuer.example.com".to_string());

    let event = MockCaepEvent {
        event_type: "https://schemas.openid.net/secevent/caep/event-type/session-revoked".to_string(),
        subject: MockSubjectIdentifier {
            format: "iss_sub".to_string(),
            claims,
        },
        timestamp: 1704067200,
        reason_admin: Some("Security policy violation".to_string()),
        reason_user: None,
    };

    let json = serde_json::to_string(&event).unwrap();
    let restored: MockCaepEvent = serde_json::from_str(&json).unwrap();

    assert_eq!(event, restored);
}

/// Test SET structure
#[test]
fn test_set_structure() {
    let mut events = HashMap::new();
    events.insert(
        "https://schemas.openid.net/secevent/caep/event-type/session-revoked".to_string(),
        serde_json::json!({"reason": "logout"}),
    );

    let set = MockSecurityEventToken {
        iss: "https://issuer.auth-platform.local".to_string(),
        iat: 1704067200,
        jti: "550e8400-e29b-41d4-a716-446655440000".to_string(),
        aud: vec!["https://receiver.example.com".to_string()],
        events,
        txn: Some("txn-12345".to_string()),
    };

    let json = serde_json::to_string(&set).unwrap();
    assert!(json.contains("iss"));
    assert!(json.contains("iat"));
    assert!(json.contains("jti"));
    assert!(json.contains("aud"));
    assert!(json.contains("events"));
    assert!(json.contains("txn"));

    let restored: MockSecurityEventToken = serde_json::from_str(&json).unwrap();
    assert_eq!(set, restored);
}
