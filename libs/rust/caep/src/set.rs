//! Security Event Token (SET) implementation per RFC 8417.
//!
//! This module provides SET generation and signing with ES256 as the default algorithm.

use crate::{CaepError, CaepEvent, CaepResult};
use chrono::Utc;
use jsonwebtoken::{encode, Algorithm, EncodingKey, Header};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

/// Default signing algorithm (ES256 per CAEP spec).
pub const DEFAULT_ALGORITHM: Algorithm = Algorithm::ES256;

/// Security Event Token (SET) structure per RFC 8417.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
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
    /// Create a new SET from a CAEP event.
    #[must_use]
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

    /// Sign the SET using the default ES256 algorithm.
    ///
    /// # Errors
    ///
    /// Returns an error if signing fails.
    pub fn sign(&self, key: &EncodingKey) -> CaepResult<String> {
        self.sign_with_algorithm(key, DEFAULT_ALGORITHM)
    }

    /// Sign the SET with a specific algorithm.
    ///
    /// # Errors
    ///
    /// Returns an error if signing fails.
    pub fn sign_with_algorithm(
        &self,
        key: &EncodingKey,
        algorithm: Algorithm,
    ) -> CaepResult<String> {
        let header = Header::new(algorithm);
        encode(&header, self, key).map_err(|e| CaepError::signing(e.to_string()))
    }

    /// Get the event type URIs in this SET.
    #[must_use]
    pub fn event_types(&self) -> Vec<&str> {
        self.events.keys().map(String::as_str).collect()
    }

    /// Check if this SET contains a specific event type.
    #[must_use]
    pub fn contains_event_type(&self, uri: &str) -> bool {
        self.events.contains_key(uri)
    }

    /// Get the number of events in this SET.
    #[must_use]
    pub fn event_count(&self) -> usize {
        self.events.len()
    }

    /// Get the algorithm used for signing (default).
    #[must_use]
    pub const fn default_algorithm() -> Algorithm {
        DEFAULT_ALGORITHM
    }
}

/// SET Builder for fluent construction.
pub struct SetBuilder {
    issuer: String,
    audience: String,
    events: Vec<CaepEvent>,
}

impl SetBuilder {
    /// Create a new SET builder.
    #[must_use]
    pub fn new(issuer: impl Into<String>, audience: impl Into<String>) -> Self {
        Self {
            issuer: issuer.into(),
            audience: audience.into(),
            events: Vec::new(),
        }
    }

    /// Add an event to the builder.
    #[must_use]
    pub fn add_event(mut self, event: CaepEvent) -> Self {
        self.events.push(event);
        self
    }

    /// Build SETs from the added events.
    #[must_use]
    pub fn build(self) -> Vec<SecurityEventToken> {
        self.events
            .iter()
            .map(|e| SecurityEventToken::from_event(e, &self.issuer, &self.audience))
            .collect()
    }

    /// Build a single SET containing all events.
    #[must_use]
    pub fn build_combined(self) -> SecurityEventToken {
        let mut events = HashMap::new();

        for event in &self.events {
            let event_data = serde_json::json!({
                "subject": event.subject,
                "event_timestamp": event.event_timestamp.timestamp(),
                "reason_admin": event.reason_admin
            });
            events.insert(event.event_type.uri().to_string(), event_data);
        }

        SecurityEventToken {
            iss: self.issuer,
            iat: Utc::now().timestamp(),
            jti: Uuid::new_v4().to_string(),
            aud: self.audience,
            events,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::event::SubjectIdentifier;

    #[test]
    fn test_default_algorithm_is_es256() {
        assert_eq!(SecurityEventToken::default_algorithm(), Algorithm::ES256);
        assert_eq!(DEFAULT_ALGORITHM, Algorithm::ES256);
    }

    #[test]
    fn test_set_from_event() {
        let subject = SubjectIdentifier::iss_sub("https://auth.example.com", "user-123");
        let event = CaepEvent::session_revoked(subject, None);
        let set = SecurityEventToken::from_event(
            &event,
            "https://auth.example.com",
            "https://receiver.example.com",
        );

        assert_eq!(set.iss, "https://auth.example.com");
        assert_eq!(set.aud, "https://receiver.example.com");
        assert!(!set.jti.is_empty());
        assert!(set.contains_event_type(
            "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
        ));
    }

    #[test]
    fn test_set_builder() {
        let subject1 = SubjectIdentifier::email("user1@example.com");
        let subject2 = SubjectIdentifier::email("user2@example.com");

        let event1 = CaepEvent::session_revoked(subject1, None);
        let event2 = CaepEvent::session_revoked(subject2, None);

        let sets = SetBuilder::new("https://issuer.com", "https://audience.com")
            .add_event(event1)
            .add_event(event2)
            .build();

        assert_eq!(sets.len(), 2);
    }

    #[test]
    fn test_set_builder_combined() {
        let subject1 = SubjectIdentifier::email("user1@example.com");
        let subject2 = SubjectIdentifier::email("user2@example.com");

        let event1 = CaepEvent::session_revoked(subject1, None);
        let event2 = CaepEvent::credential_change(subject2, "update", "password");

        let set = SetBuilder::new("https://issuer.com", "https://audience.com")
            .add_event(event1)
            .add_event(event2)
            .build_combined();

        assert_eq!(set.event_count(), 2);
    }

    #[test]
    fn test_set_serialization() {
        let subject = SubjectIdentifier::opaque("user-abc");
        let event = CaepEvent::session_revoked(subject, None);
        let set = SecurityEventToken::from_event(&event, "issuer", "audience");

        let json = serde_json::to_string(&set).unwrap();
        let parsed: SecurityEventToken = serde_json::from_str(&json).unwrap();

        assert_eq!(set.iss, parsed.iss);
        assert_eq!(set.aud, parsed.aud);
        assert_eq!(set.events.len(), parsed.events.len());
    }
}
