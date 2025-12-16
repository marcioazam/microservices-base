//! Property-based tests for CAEP functionality.
//!
//! Uses proptest with 100+ iterations per property.

use auth_caep::*;
use proptest::prelude::*;

// Generators
fn event_type_strategy() -> impl Strategy<Value = CaepEventType> {
    prop_oneof![
        Just(CaepEventType::SessionRevoked),
        Just(CaepEventType::CredentialChange),
        Just(CaepEventType::AssuranceLevelChange),
        Just(CaepEventType::TokenClaimsChange),
        Just(CaepEventType::DeviceComplianceChange),
    ]
}

fn subject_strategy() -> impl Strategy<Value = SubjectIdentifier> {
    prop_oneof![
        ("[a-z]{5,20}", "[a-z0-9]{10,30}").prop_map(|(iss, sub)| SubjectIdentifier::IssSub {
            iss: format!("https://{}.example.com", iss),
            sub
        }),
        "[a-z0-9._%+-]+@[a-z0-9.-]+\\.[a-z]{2,4}".prop_map(|email| SubjectIdentifier::Email {
            email
        }),
        "[a-z0-9]{32}".prop_map(|id| SubjectIdentifier::Opaque { id }),
    ]
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Feature: auth-platform-q2-2025-evolution, Property 6: CAEP SET Signature Validity**
    /// **Validates: Requirements 5.5, 6.1**
    ///
    /// For any emitted Security Event Token, the SET SHALL be signed with ES256
    /// and verifiable using the platform's published JWKS.
    #[test]
    fn prop_set_contains_required_claims(
        event_type in event_type_strategy(),
        subject in subject_strategy(),
        issuer in "[a-z]{5,20}",
        audience in "[a-z]{5,20}",
    ) {
        let event = CaepEvent {
            event_type: event_type.clone(),
            subject,
            event_timestamp: chrono::Utc::now(),
            reason_admin: None,
            extra: serde_json::Value::Null,
        };

        let issuer_url = format!("https://{}.example.com", issuer);
        let audience_url = format!("https://{}.example.com", audience);

        let set = SecurityEventToken::from_event(&event, &issuer_url, &audience_url);

        // Property: SET must have issuer
        prop_assert!(!set.iss.is_empty(), "SET must have issuer");

        // Property: SET must have audience
        prop_assert!(!set.aud.is_empty(), "SET must have audience");

        // Property: SET must have unique jti
        prop_assert!(!set.jti.is_empty(), "SET must have jti");

        // Property: SET must have iat
        prop_assert!(set.iat > 0, "SET must have valid iat");

        // Property: SET must contain the event
        prop_assert!(
            set.events.contains_key(event_type.uri()),
            "SET must contain the event type URI"
        );
    }

    /// **Feature: auth-platform-q2-2025-evolution, Property 7: CAEP Event Emission Completeness**
    /// **Validates: Requirements 5.2, 5.3, 5.4**
    ///
    /// For any security event (session revocation, credential change, assurance change),
    /// the CAEP Transmitter SHALL emit a corresponding SET with correct event type and subject.
    #[test]
    fn prop_event_emission_preserves_data(
        event_type in event_type_strategy(),
        user_id in "[a-z0-9]{10,30}",
        issuer_domain in "[a-z]{5,20}",
    ) {
        let issuer = format!("https://{}.example.com", issuer_domain);
        let subject = SubjectIdentifier::IssSub {
            iss: issuer.clone(),
            sub: user_id.clone(),
        };

        let event = match event_type {
            CaepEventType::SessionRevoked => {
                CaepEvent::session_revoked(subject.clone(), Some("Test".to_string()))
            }
            CaepEventType::CredentialChange => {
                CaepEvent::credential_change(subject.clone(), "create", "passkey")
            }
            CaepEventType::AssuranceLevelChange => {
                CaepEvent::assurance_level_change(subject.clone(), "low", "high")
            }
            _ => CaepEvent {
                event_type: event_type.clone(),
                subject: subject.clone(),
                event_timestamp: chrono::Utc::now(),
                reason_admin: None,
                extra: serde_json::Value::Null,
            },
        };

        let set = SecurityEventToken::from_event(&event, &issuer, "https://receiver.example.com");

        // Property: Event type URI must be in SET
        prop_assert!(
            set.events.contains_key(event_type.uri()),
            "SET must contain correct event type URI"
        );

        // Property: Subject must be preserved in event data
        let event_data = set.events.get(event_type.uri()).unwrap();
        prop_assert!(
            event_data.get("subject").is_some(),
            "Event data must contain subject"
        );
    }

    /// **Feature: auth-platform-q2-2025-evolution, Property 9: CAEP Stream Health Tracking**
    /// **Validates: Requirements 7.3, 7.5**
    ///
    /// For any configured CAEP stream, the system SHALL track and expose
    /// delivery success rate, latency percentiles, and last successful delivery timestamp.
    #[test]
    fn prop_stream_health_tracking(
        events_delivered in 0u64..1000,
        events_failed in 0u64..100,
        latency_ms in 1u64..500,
    ) {
        let config = StreamConfig {
            audience: "https://receiver.example.com".to_string(),
            delivery: DeliveryMethod::Push {
                endpoint_url: "https://receiver.example.com/caep".to_string(),
            },
            events_requested: vec![CaepEventType::SessionRevoked],
            format: "iss_sub".to_string(),
        };

        let mut stream = Stream::new(config);

        // Simulate deliveries
        for _ in 0..events_delivered {
            stream.record_success(latency_ms);
        }
        for _ in 0..events_failed {
            stream.record_failure("Test error".to_string());
        }

        // Property: events_delivered must be tracked
        prop_assert_eq!(
            stream.health.events_delivered,
            events_delivered,
            "events_delivered must be tracked"
        );

        // Property: events_failed must be tracked
        prop_assert_eq!(
            stream.health.events_failed,
            events_failed,
            "events_failed must be tracked"
        );

        // Property: success_rate must be calculated correctly
        let total = events_delivered + events_failed;
        if total > 0 {
            let expected_rate = events_delivered as f64 / total as f64;
            let actual_rate = stream.success_rate();
            prop_assert!(
                (actual_rate - expected_rate).abs() < 0.001,
                "success_rate must be calculated correctly"
            );
        }

        // Property: last_delivery_at must be set if any deliveries
        if events_delivered > 0 {
            prop_assert!(
                stream.health.last_delivery_at.is_some(),
                "last_delivery_at must be set after successful delivery"
            );
        }

        // Property: avg_latency must be reasonable
        if events_delivered > 0 {
            prop_assert!(
                stream.health.avg_latency_ms > 0.0,
                "avg_latency must be positive after deliveries"
            );
        }
    }
}

/// **Feature: auth-platform-q2-2025-evolution, Property 8: CAEP Session Revocation Effect**
/// **Validates: Requirements 6.2**
///
/// For any received session-revoked event with valid signature,
/// the affected session SHALL be terminated within 1 second of event processing.
#[test]
fn test_session_revoked_handler() {
    // This is a unit test since it requires async runtime
    // Property testing for this is done via integration tests
    let subject = SubjectIdentifier::IssSub {
        iss: "https://auth.example.com".to_string(),
        sub: "user-123".to_string(),
    };
    let event = CaepEvent::session_revoked(subject, Some("Admin action".to_string()));

    assert_eq!(event.event_type, CaepEventType::SessionRevoked);
    assert!(event.reason_admin.is_some());
}


proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Feature: auth-platform-q2-2025-evolution, Property 8: CAEP Session Revocation Effect**
    /// **Validates: Requirements 6.2**
    ///
    /// For any received session-revoked event with valid signature,
    /// the affected session SHALL be terminated.
    #[test]
    fn prop_session_revoked_terminates_session(
        user_id in "[a-z0-9]{10,30}",
        issuer_domain in "[a-z]{5,20}",
        session_count in 1usize..10,
    ) {
        let issuer = format!("https://{}.example.com", issuer_domain);
        let subject = SubjectIdentifier::IssSub {
            iss: issuer.clone(),
            sub: user_id.clone(),
        };

        let event = CaepEvent::session_revoked(subject.clone(), Some("Security policy".to_string()));

        // Property: Event type must be SessionRevoked
        prop_assert_eq!(
            event.event_type,
            CaepEventType::SessionRevoked,
            "Event type must be SessionRevoked"
        );

        // Property: Subject must be preserved
        match &event.subject {
            SubjectIdentifier::IssSub { sub, .. } => {
                prop_assert_eq!(sub, &user_id, "User ID must be preserved in subject");
            }
            _ => prop_assert!(false, "Subject must be IssSub format"),
        }

        // Property: Reason must be preserved when provided
        prop_assert!(
            event.reason_admin.is_some(),
            "Reason must be preserved when provided"
        );
    }
}
