//! Test fixtures with sample data.
//!
//! This module provides pre-built test data for use in tests.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Sample CAEP event for testing.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SampleCaepEvent {
    /// Event type
    pub event_type: String,
    /// Subject
    pub subject: SampleSubject,
    /// Event timestamp
    pub event_timestamp: DateTime<Utc>,
    /// Reason (admin-facing)
    pub reason_admin: Option<String>,
}

/// Sample subject for testing.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SampleSubject {
    /// Format type
    pub format: String,
    /// Issuer (for iss_sub format)
    pub iss: Option<String>,
    /// Subject (for iss_sub format)
    pub sub: Option<String>,
    /// Email (for email format)
    pub email: Option<String>,
}

impl SampleCaepEvent {
    /// Create a sample session revoked event.
    #[must_use]
    pub fn session_revoked() -> Self {
        Self {
            event_type: "session-revoked".to_string(),
            subject: SampleSubject::iss_sub(
                "https://issuer.example.com",
                "user-123",
            ),
            event_timestamp: Utc::now(),
            reason_admin: Some("User logged out".to_string()),
        }
    }

    /// Create a sample credential change event.
    #[must_use]
    pub fn credential_change() -> Self {
        Self {
            event_type: "credential-change".to_string(),
            subject: SampleSubject::email("user@example.com"),
            event_timestamp: Utc::now(),
            reason_admin: Some("Password changed".to_string()),
        }
    }
}

impl SampleSubject {
    /// Create an iss_sub format subject.
    #[must_use]
    pub fn iss_sub(iss: &str, sub: &str) -> Self {
        Self {
            format: "iss_sub".to_string(),
            iss: Some(iss.to_string()),
            sub: Some(sub.to_string()),
            email: None,
        }
    }

    /// Create an email format subject.
    #[must_use]
    pub fn email(email: &str) -> Self {
        Self {
            format: "email".to_string(),
            iss: None,
            sub: None,
            email: Some(email.to_string()),
        }
    }
}

/// Sample database credentials for testing.
#[derive(Debug, Clone)]
pub struct SampleDatabaseCredentials {
    /// Username
    pub username: String,
    /// Password (should be treated as secret)
    pub password: String,
    /// Lease ID
    pub lease_id: String,
    /// TTL in seconds
    pub ttl_seconds: u64,
    /// Whether the lease is renewable
    pub renewable: bool,
}

impl SampleDatabaseCredentials {
    /// Create sample PostgreSQL credentials.
    #[must_use]
    pub fn postgres() -> Self {
        Self {
            username: "v-token-postgres-readonly-abc123".to_string(),
            password: "A1b2C3d4E5f6G7h8I9j0".to_string(),
            lease_id: "database/creds/readonly/abc123".to_string(),
            ttl_seconds: 3600,
            renewable: true,
        }
    }

    /// Create sample MongoDB credentials.
    #[must_use]
    pub fn mongodb() -> Self {
        Self {
            username: "v-token-mongodb-readwrite-xyz789".to_string(),
            password: "X9y8Z7w6V5u4T3s2R1q0".to_string(),
            lease_id: "database/creds/readwrite/xyz789".to_string(),
            ttl_seconds: 7200,
            renewable: true,
        }
    }
}

/// Sample Pact contract for testing.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SamplePactContract {
    /// Consumer name
    pub consumer: String,
    /// Provider name
    pub provider: String,
    /// Contract version
    pub version: String,
    /// Interactions
    pub interactions: Vec<SampleInteraction>,
}

/// Sample Pact interaction for testing.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SampleInteraction {
    /// Description
    pub description: String,
    /// Request method
    pub request_method: String,
    /// Request path
    pub request_path: String,
    /// Response status
    pub response_status: u16,
}

impl SamplePactContract {
    /// Create a sample user service contract.
    #[must_use]
    pub fn user_service() -> Self {
        Self {
            consumer: "web-frontend".to_string(),
            provider: "user-service".to_string(),
            version: "1.0.0".to_string(),
            interactions: vec![
                SampleInteraction {
                    description: "Get user by ID".to_string(),
                    request_method: "GET".to_string(),
                    request_path: "/users/123".to_string(),
                    response_status: 200,
                },
                SampleInteraction {
                    description: "Create user".to_string(),
                    request_method: "POST".to_string(),
                    request_path: "/users".to_string(),
                    response_status: 201,
                },
            ],
        }
    }
}

/// Sample Linkerd mTLS connection info for testing.
#[derive(Debug, Clone)]
pub struct SampleMtlsConnection {
    /// Source identity (SPIFFE URI)
    pub source_identity: String,
    /// Destination identity (SPIFFE URI)
    pub destination_identity: String,
    /// TLS version
    pub tls_version: String,
    /// Whether the connection is encrypted
    pub encrypted: bool,
}

impl SampleMtlsConnection {
    /// Create a sample mTLS connection.
    #[must_use]
    pub fn default_connection() -> Self {
        Self {
            source_identity: "spiffe://auth-platform.local/ns/auth-platform/sa/web-frontend".to_string(),
            destination_identity: "spiffe://auth-platform.local/ns/auth-platform/sa/user-service".to_string(),
            tls_version: "TLSv1.3".to_string(),
            encrypted: true,
        }
    }
}

/// Sample trace context for testing.
#[derive(Debug, Clone)]
pub struct SampleTraceContext {
    /// Trace ID (32 hex chars)
    pub trace_id: String,
    /// Parent span ID (16 hex chars)
    pub parent_id: String,
    /// Trace flags
    pub flags: String,
}

impl SampleTraceContext {
    /// Create a sample trace context.
    #[must_use]
    pub fn default_context() -> Self {
        Self {
            trace_id: "0af7651916cd43dd8448eb211c80319c".to_string(),
            parent_id: "b7ad6b7169203331".to_string(),
            flags: "01".to_string(),
        }
    }

    /// Format as W3C traceparent header.
    #[must_use]
    pub fn to_traceparent(&self) -> String {
        format!("00-{}-{}-{}", self.trace_id, self.parent_id, self.flags)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sample_caep_event() {
        let event = SampleCaepEvent::session_revoked();
        assert_eq!(event.event_type, "session-revoked");
        assert!(event.reason_admin.is_some());
    }

    #[test]
    fn test_sample_credentials() {
        let creds = SampleDatabaseCredentials::postgres();
        assert!(creds.username.starts_with("v-token-postgres"));
        assert!(creds.renewable);
    }

    #[test]
    fn test_sample_pact_contract() {
        let contract = SamplePactContract::user_service();
        assert_eq!(contract.consumer, "web-frontend");
        assert_eq!(contract.interactions.len(), 2);
    }

    #[test]
    fn test_sample_trace_context() {
        let ctx = SampleTraceContext::default_context();
        let traceparent = ctx.to_traceparent();
        assert!(traceparent.starts_with("00-"));
        assert_eq!(traceparent.split('-').count(), 4);
    }

    #[test]
    fn test_caep_event_serialization() {
        let event = SampleCaepEvent::session_revoked();
        let json = serde_json::to_string(&event).unwrap();
        let parsed: SampleCaepEvent = serde_json::from_str(&json).unwrap();
        assert_eq!(event.event_type, parsed.event_type);
    }
}
