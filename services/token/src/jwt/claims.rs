use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// DPoP Confirmation claim (cnf) for token binding per RFC 9449
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Confirmation {
    /// JWK Thumbprint per RFC 7638
    pub jkt: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Claims {
    // Standard JWT claims
    pub iss: String,
    pub sub: String,
    pub aud: Vec<String>,
    pub exp: i64,
    pub iat: i64,
    pub nbf: Option<i64>,
    pub jti: String,

    // OIDC claims
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nonce: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub auth_time: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub acr: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub amr: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub azp: Option<String>,

    // DPoP token binding (RFC 9449)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cnf: Option<Confirmation>,

    // Custom claims
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scopes: Option<Vec<String>>,
    #[serde(flatten)]
    pub custom: HashMap<String, serde_json::Value>,
}

impl Claims {
    pub fn new(issuer: String, subject: String, audience: Vec<String>, ttl_seconds: i64) -> Self {
        let now = chrono::Utc::now().timestamp();
        Claims {
            iss: issuer,
            sub: subject,
            aud: audience,
            exp: now + ttl_seconds,
            iat: now,
            nbf: Some(now),
            jti: uuid::Uuid::new_v4().to_string(),
            nonce: None,
            auth_time: None,
            acr: None,
            amr: None,
            azp: None,
            cnf: None,
            session_id: None,
            scopes: None,
            custom: HashMap::new(),
        }
    }

    pub fn with_nonce(mut self, nonce: String) -> Self {
        self.nonce = Some(nonce);
        self
    }

    pub fn with_session_id(mut self, session_id: String) -> Self {
        self.session_id = Some(session_id);
        self
    }

    pub fn with_scopes(mut self, scopes: Vec<String>) -> Self {
        self.scopes = Some(scopes);
        self
    }

    /// Binds the token to a DPoP proof using JWK thumbprint
    pub fn with_dpop_binding(mut self, jkt: String) -> Self {
        self.cnf = Some(Confirmation { jkt });
        self
    }

    /// Adds authentication method references (amr)
    pub fn with_amr(mut self, methods: Vec<String>) -> Self {
        self.amr = Some(methods);
        self
    }

    pub fn with_custom_claim(mut self, key: String, value: serde_json::Value) -> Self {
        self.custom.insert(key, value);
        self
    }

    /// Checks if this token is DPoP-bound
    pub fn is_dpop_bound(&self) -> bool {
        self.cnf.is_some()
    }

    /// Gets the DPoP thumbprint if bound
    pub fn dpop_thumbprint(&self) -> Option<&str> {
        self.cnf.as_ref().map(|c| c.jkt.as_str())
    }

    pub fn is_expired(&self) -> bool {
        let now = chrono::Utc::now().timestamp();
        self.exp < now
    }

    pub fn is_valid_at(&self, timestamp: i64) -> bool {
        if let Some(nbf) = self.nbf {
            if timestamp < nbf {
                return false;
            }
        }
        timestamp < self.exp
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_claims_creation() {
        let claims = Claims::new(
            "test-issuer".to_string(),
            "user-123".to_string(),
            vec!["api".to_string()],
            900,
        );

        assert_eq!(claims.iss, "test-issuer");
        assert_eq!(claims.sub, "user-123");
        assert!(!claims.is_expired());
    }

    #[test]
    fn test_claims_with_custom() {
        let claims = Claims::new(
            "test-issuer".to_string(),
            "user-123".to_string(),
            vec!["api".to_string()],
            900,
        )
        .with_session_id("session-456".to_string())
        .with_scopes(vec!["read".to_string(), "write".to_string()]);

        assert_eq!(claims.session_id, Some("session-456".to_string()));
        assert_eq!(claims.scopes, Some(vec!["read".to_string(), "write".to_string()]));
    }
}
