//! CAEP event types and structures.
//!
//! This module implements OpenID CAEP 1.0 event types with modern Rust patterns.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// CAEP event types conforming to OpenID CAEP 1.0.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
#[serde(rename_all = "kebab-case")]
pub enum CaepEventType {
    /// Session has been revoked
    SessionRevoked,
    /// Credential has changed (added, removed, or modified)
    CredentialChange,
    /// User's assurance level has changed
    AssuranceLevelChange,
    /// Token claims have been updated
    TokenClaimsChange,
    /// Device compliance status has changed
    DeviceComplianceChange,
}

impl CaepEventType {
    /// Get the full URI for this event type.
    ///
    /// Returns the OpenID CAEP event type URI as defined in the specification.
    #[must_use]
    pub const fn uri(&self) -> &'static str {
        match self {
            Self::SessionRevoked => {
                "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
            }
            Self::CredentialChange => {
                "https://schemas.openid.net/secevent/caep/event-type/credential-change"
            }
            Self::AssuranceLevelChange => {
                "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
            }
            Self::TokenClaimsChange => {
                "https://schemas.openid.net/secevent/caep/event-type/token-claims-change"
            }
            Self::DeviceComplianceChange => {
                "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
            }
        }
    }

    /// Get the short name for this event type.
    #[must_use]
    pub const fn name(&self) -> &'static str {
        match self {
            Self::SessionRevoked => "session-revoked",
            Self::CredentialChange => "credential-change",
            Self::AssuranceLevelChange => "assurance-level-change",
            Self::TokenClaimsChange => "token-claims-change",
            Self::DeviceComplianceChange => "device-compliance-change",
        }
    }

    /// Get all event types.
    #[must_use]
    pub const fn all() -> &'static [Self] {
        &[
            Self::SessionRevoked,
            Self::CredentialChange,
            Self::AssuranceLevelChange,
            Self::TokenClaimsChange,
            Self::DeviceComplianceChange,
        ]
    }
}

/// Subject identifier formats per OpenID SSF.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(tag = "format", rename_all = "snake_case")]
pub enum SubjectIdentifier {
    /// Issuer and subject combination
    IssSub {
        /// Issuer URL
        iss: String,
        /// Subject identifier
        sub: String,
    },
    /// Email address
    Email {
        /// Email address
        email: String,
    },
    /// Opaque identifier
    Opaque {
        /// Opaque ID
        id: String,
    },
    /// Session identifier
    SessionId {
        /// Session ID
        session_id: String,
    },
}

impl SubjectIdentifier {
    /// Create an issuer/subject identifier.
    #[must_use]
    pub fn iss_sub(iss: impl Into<String>, sub: impl Into<String>) -> Self {
        Self::IssSub {
            iss: iss.into(),
            sub: sub.into(),
        }
    }

    /// Create an email identifier.
    #[must_use]
    pub fn email(email: impl Into<String>) -> Self {
        Self::Email {
            email: email.into(),
        }
    }

    /// Create an opaque identifier.
    #[must_use]
    pub fn opaque(id: impl Into<String>) -> Self {
        Self::Opaque { id: id.into() }
    }

    /// Create a session identifier.
    #[must_use]
    pub fn session_id(session_id: impl Into<String>) -> Self {
        Self::SessionId {
            session_id: session_id.into(),
        }
    }
}

/// Reason for the event (admin-facing).
#[derive(Debug, Clone, Serialize, Deserialize, Default, PartialEq, Eq)]
pub struct EventReason {
    /// English reason text
    #[serde(skip_serializing_if = "Option::is_none")]
    pub en: Option<String>,
}

impl EventReason {
    /// Create a new event reason.
    #[must_use]
    pub fn new(reason: impl Into<String>) -> Self {
        Self {
            en: Some(reason.into()),
        }
    }
}

/// CAEP Event structure.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CaepEvent {
    /// Event type
    pub event_type: CaepEventType,
    /// Subject of the event
    pub subject: SubjectIdentifier,
    /// Timestamp when the event occurred
    pub event_timestamp: DateTime<Utc>,
    /// Reason for the event (admin-facing)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason_admin: Option<EventReason>,
    /// Additional event-specific data
    #[serde(flatten)]
    pub extra: serde_json::Value,
}

impl CaepEvent {
    /// Create a new CAEP event.
    #[must_use]
    pub fn new(event_type: CaepEventType, subject: SubjectIdentifier) -> Self {
        Self {
            event_type,
            subject,
            event_timestamp: Utc::now(),
            reason_admin: None,
            extra: serde_json::Value::Null,
        }
    }

    /// Create a new session-revoked event.
    #[must_use]
    pub fn session_revoked(subject: SubjectIdentifier, reason: Option<String>) -> Self {
        Self {
            event_type: CaepEventType::SessionRevoked,
            subject,
            event_timestamp: Utc::now(),
            reason_admin: reason.map(EventReason::new),
            extra: serde_json::Value::Null,
        }
    }

    /// Create a new credential-change event.
    #[must_use]
    pub fn credential_change(
        subject: SubjectIdentifier,
        change_type: &str,
        credential_type: &str,
    ) -> Self {
        Self {
            event_type: CaepEventType::CredentialChange,
            subject,
            event_timestamp: Utc::now(),
            reason_admin: None,
            extra: serde_json::json!({
                "change_type": change_type,
                "credential_type": credential_type
            }),
        }
    }

    /// Create a new assurance-level-change event.
    #[must_use]
    pub fn assurance_level_change(
        subject: SubjectIdentifier,
        previous_level: &str,
        current_level: &str,
    ) -> Self {
        Self {
            event_type: CaepEventType::AssuranceLevelChange,
            subject,
            event_timestamp: Utc::now(),
            reason_admin: None,
            extra: serde_json::json!({
                "previous_level": previous_level,
                "current_level": current_level
            }),
        }
    }

    /// Create a new token-claims-change event.
    #[must_use]
    pub fn token_claims_change(subject: SubjectIdentifier, claims: serde_json::Value) -> Self {
        Self {
            event_type: CaepEventType::TokenClaimsChange,
            subject,
            event_timestamp: Utc::now(),
            reason_admin: None,
            extra: claims,
        }
    }

    /// Create a new device-compliance-change event.
    #[must_use]
    pub fn device_compliance_change(
        subject: SubjectIdentifier,
        previous_status: &str,
        current_status: &str,
    ) -> Self {
        Self {
            event_type: CaepEventType::DeviceComplianceChange,
            subject,
            event_timestamp: Utc::now(),
            reason_admin: None,
            extra: serde_json::json!({
                "previous_status": previous_status,
                "current_status": current_status
            }),
        }
    }

    /// Add a reason to the event.
    #[must_use]
    pub fn with_reason(mut self, reason: impl Into<String>) -> Self {
        self.reason_admin = Some(EventReason::new(reason));
        self
    }

    /// Add extra data to the event.
    #[must_use]
    pub fn with_extra(mut self, extra: serde_json::Value) -> Self {
        self.extra = extra;
        self
    }

    /// Get the event type URI.
    #[must_use]
    pub const fn event_uri(&self) -> &'static str {
        self.event_type.uri()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_event_type_uri() {
        assert_eq!(
            CaepEventType::SessionRevoked.uri(),
            "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
        );
        assert_eq!(CaepEventType::SessionRevoked.name(), "session-revoked");
    }

    #[test]
    fn test_all_event_types() {
        let all = CaepEventType::all();
        assert_eq!(all.len(), 5);
    }

    #[test]
    fn test_subject_identifier_constructors() {
        let iss_sub = SubjectIdentifier::iss_sub("https://issuer.com", "user-123");
        assert!(matches!(iss_sub, SubjectIdentifier::IssSub { .. }));

        let email = SubjectIdentifier::email("user@example.com");
        assert!(matches!(email, SubjectIdentifier::Email { .. }));
    }

    #[test]
    fn test_session_revoked_event() {
        let subject = SubjectIdentifier::iss_sub("https://auth.example.com", "user-123");
        let event = CaepEvent::session_revoked(subject, Some("Admin action".to_string()));

        assert_eq!(event.event_type, CaepEventType::SessionRevoked);
        assert!(event.reason_admin.is_some());
    }

    #[test]
    fn test_event_serialization() {
        let subject = SubjectIdentifier::email("user@example.com");
        let event = CaepEvent::session_revoked(subject, None);

        let json = serde_json::to_string(&event).unwrap();
        let parsed: CaepEvent = serde_json::from_str(&json).unwrap();

        assert_eq!(event.event_type, parsed.event_type);
    }

    #[test]
    fn test_event_builder() {
        let subject = SubjectIdentifier::opaque("user-abc");
        let event = CaepEvent::new(CaepEventType::CredentialChange, subject)
            .with_reason("Password updated")
            .with_extra(serde_json::json!({"change_type": "update"}));

        assert!(event.reason_admin.is_some());
        assert!(!event.extra.is_null());
    }
}
