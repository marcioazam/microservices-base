//! JWT Claims Unit Tests
//!
//! Tests for claims validation, expiration, and scope handling.
//! Extracted from src/jwt/claims.rs inline tests.

use std::collections::HashMap;

// ============================================================================
// Test Helpers (mock implementation for testing logic)
// ============================================================================

#[derive(Debug, Clone)]
struct Claims {
    iss: String,
    sub: String,
    aud: Vec<String>,
    exp: i64,
    iat: i64,
    nbf: Option<i64>,
    jti: String,
    session_id: Option<String>,
    scopes: Option<Vec<String>>,
    custom: HashMap<String, serde_json::Value>,
}

impl Claims {
    fn is_expired(&self) -> bool {
        let now = chrono::Utc::now().timestamp();
        self.exp < now
    }

    fn has_scope(&self, scope: &str) -> bool {
        self.scopes
            .as_ref()
            .map(|s| s.contains(&scope.to_string()))
            .unwrap_or(false)
    }

    fn to_map(&self) -> HashMap<String, String> {
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

fn create_test_claims(exp_offset: i64) -> Claims {
    Claims {
        iss: "test-issuer".to_string(),
        sub: "user-123".to_string(),
        aud: vec!["api".to_string()],
        exp: chrono::Utc::now().timestamp() + exp_offset,
        iat: chrono::Utc::now().timestamp(),
        nbf: None,
        jti: "jti-123".to_string(),
        session_id: Some("session-xyz".to_string()),
        scopes: Some(vec!["read".to_string(), "write".to_string()]),
        custom: HashMap::new(),
    }
}

// ============================================================================
// Expiration Tests
// ============================================================================

#[test]
fn test_is_expired_false() {
    let claims = create_test_claims(3600);
    assert!(!claims.is_expired());
}

#[test]
fn test_is_expired_true() {
    let claims = create_test_claims(-3600);
    assert!(claims.is_expired());
}

// ============================================================================
// Scope Tests
// ============================================================================

#[test]
fn test_has_scope_existing() {
    let claims = create_test_claims(3600);
    assert!(claims.has_scope("read"));
    assert!(claims.has_scope("write"));
}

#[test]
fn test_has_scope_missing() {
    let claims = create_test_claims(3600);
    assert!(!claims.has_scope("admin"));
}

#[test]
fn test_has_scope_none() {
    let mut claims = create_test_claims(3600);
    claims.scopes = None;
    assert!(!claims.has_scope("read"));
}

// ============================================================================
// Map Conversion Tests
// ============================================================================

#[test]
fn test_to_map_contains_required_fields() {
    let claims = create_test_claims(3600);
    let map = claims.to_map();

    assert_eq!(map.get("iss").unwrap(), "test-issuer");
    assert_eq!(map.get("sub").unwrap(), "user-123");
    assert_eq!(map.get("jti").unwrap(), "jti-123");
}

#[test]
fn test_to_map_optional_fields() {
    let claims = create_test_claims(3600);
    let map = claims.to_map();

    assert!(map.contains_key("session_id"));
    assert!(map.contains_key("scopes"));
}

#[test]
fn test_to_map_without_optional_fields() {
    let mut claims = create_test_claims(3600);
    claims.session_id = None;
    claims.scopes = None;
    let map = claims.to_map();

    assert!(!map.contains_key("session_id"));
    assert!(!map.contains_key("scopes"));
}
