//! Security Event Token (SET) implementation per RFC 8417.

use crate::{CaepError, CaepEvent};
use chrono::{DateTime, Utc};
use jsonwebtoken::{encode, Algorithm, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

/// Security Event Token (SET) structure per RFC 8417
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityEventToken {
    /// Issuer
    pub iss: String,
    /// Issued at timestamp
    pub iat: i64,
    /// JWT ID (unique identifier)
    pub jti: String,
    /// Audience
    pub aud: String,
    /// Events map (event URI -> event data)
    pub events: HashMap<String, serde_json::Value>,
}

impl SecurityEventToken {
    /// Create a new SET from a CAEP event
    pub fn from_event(event: &CaepEvent, issuer: &str, audience: &str) -> Self {
        let mut events = HashMap::new();
        
        let event_data = serde_json::json!({
            "subject": event.subject,
            "event_timestamp": event.event_timestamp.timestamp(),
            "reason_admin": event.reason_admin
        });
        
        events.insert(event.event_type.uri().to_string(), event_data);

        Self {
            iss: issuer.to_string(),
            iat: Utc::now().timestamp(),
            jti: Uuid::new_v4().to_string(),
            aud: audience.to_string(),
            events,
        }
    }

    /// Sign the SET and return the JWT string
    pub fn sign(&self, key: &EncodingKey, algorithm: Algorithm) -> Result<String, CaepError> {
        let header = Header::new(algorithm);
        encode(&header, self, key).map_err(|e| CaepError::SigningError(e.to_string()))
    }

    /// Get the event type URIs in this SET
    pub fn event_types(&self) -> Vec<&str> {
        self.events.keys().map(|s| s.as_str()).collect()
    }
}

/// SET Builder for fluent construction
pub struct SetBuilder {
    issuer: String,
    audience: String,
    events: Vec<CaepEvent>,
}

impl SetBuilder {
    pub fn new(issuer: impl Into<String>, audience: impl Into<String>) -> Self {
        Self {
            issuer: issuer.into(),
            audience: audience.into(),
            events: Vec::new(),
        }
    }

    pub fn add_event(mut self, event: CaepEvent) -> Self {
        self.events.push(event);
        self
    }

    pub fn build(self) -> Vec<SecurityEventToken> {
        self.events
            .iter()
            .map(|e| SecurityEventToken::from_event(e, &self.issuer, &self.audience))
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::event::SubjectIdentifier;

    #[test]
    fn test_set_from_event() {
        let subject = SubjectIdentifier::IssSub {
            iss: "https://auth.example.com".to_string(),
            sub: "user-123".to_string(),
        };
        let event = CaepEvent::session_revoked(subject, None);
        let set = SecurityEventToken::from_event(&event, "https://auth.example.com", "https://receiver.example.com");

        assert_eq!(set.iss, "https://auth.example.com");
        assert_eq!(set.aud, "https://receiver.example.com");
        assert!(!set.jti.is_empty());
        assert!(set.events.contains_key(
            "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
        ));
    }
}
