//! JWT Claims structure with validation helpers.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// JWT Claims structure following RFC 7519.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub iss: String,
    pub sub: String,
    pub aud: Vec<String>,
    pub exp: i64,
    pub iat: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nbf: Option<i64>,
    pub jti: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scopes: Option<Vec<String>>,
    #[serde(flatten)]
    pub custom: HashMap<String, serde_json::Value>,
}

impl Claims {
    /// Checks if the token is expired.
    pub fn is_expired(&self) -> bool {
        let now = chrono::Utc::now().timestamp();
        self.exp < now
    }

    /// Checks if the token has a specific scope.
    pub fn has_scope(&self, scope: &str) -> bool {
        self.scopes
            .as_ref()
            .map(|s| s.contains(&scope.to_string()))
            .unwrap_or(false)
    }

    /// Checks if a claim is present (centralized method).
    pub fn has_claim(&self, claim_name: &str) -> bool {
        match claim_name {
            "iss" => !self.iss.is_empty(),
            "sub" => !self.sub.is_empty(),
            "aud" => !self.aud.is_empty(),
            "exp" => true,
            "iat" => true,
            "jti" => !self.jti.is_empty(),
            "session_id" => self.session_id.is_some(),
            "scopes" => self.scopes.is_some(),
            _ => self.custom.contains_key(claim_name),
        }
    }

    /// Converts claims to a string map for gRPC response.
    pub fn to_map(&self) -> HashMap<String, String> {
        let mut map = HashMap::new();
        map.insert("iss".to_string(), self.iss.clone());
        map.insert("sub".to_string(), self.sub.clone());
        map.insert("aud".to_string(), self.aud.join(","));
        map.insert("exp".to_string(), self.exp.to_string());
        map.insert("iat".to_string(), self.iat.to_string());
        map.insert("jti".to_string(), self.jti.clone());
        
        if let Some(ref session_id) = self.session_id {
            map.insert("session_id".to_string(), session_id.clone());
        }
        
        if let Some(ref scopes) = self.scopes {
            map.insert("scopes".to_string(), scopes.join(" "));
        }
        
        map
    }
}
