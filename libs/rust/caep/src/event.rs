//! CAEP event types and structures.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// CAEP event types conforming to OpenID CAEP 1.0
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
    /// Get the full URI for this event type
    pub fn uri(&self) -> &'static str {
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
}

/// Subject identifier formats per OpenID SSF
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "format", rename_all = "snake_case")]
pub enum SubjectIdentifier {
    /// Issuer and subject combination
    IssSub { iss: String, sub: String },
    /// Email address
    Email { email: String },
    /// Opaque identifier
    Opaque { id: String },
    /// Session identifier
    SessionId { session_id: String },
}

/// Reason for the event (admin-facing)
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct EventReason {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub en: Option<String>,
}

/// CAEP Event structure
#[derive(Debug, Clone, Serialize, Deserialize)]
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
    /// Create a new session-revoked event
    pub fn session_revoked(subject: SubjectIdentifier, reason: Option<String>) -> Self {
        Self {
            event_type: CaepEventType::SessionRevoked,
            subject,
            event_timestamp: Utc::now(),
            reason_admin: reason.map(|r| EventReason { en: Some(r) }),
            extra: serde_json::Value::Null,
        }
    }

    /// Create a new credential-change event
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

    /// Create a new assurance-level-change event
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
    }

    #[test]
    fn test_session_revoked_event() {
        let subject = SubjectIdentifier::IssSub {
            iss: "https://auth.example.com".to_string(),
            sub: "user-123".to_string(),
        };
        let event = CaepEvent::session_revoked(subject, Some("Admin action".to_string()));

        assert_eq!(event.event_type, CaepEventType::SessionRevoked);
        assert!(event.reason_admin.is_some());
    }
}
