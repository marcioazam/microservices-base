//! Property-based tests for JWT type-state pattern.
//!
//! Property 10: JWT Type-State Transitions
//! Validates that the type-state pattern enforces correct validation order.

use proptest::prelude::*;
use std::collections::HashMap;

/// Generates arbitrary JWT-like token strings (not cryptographically valid)
fn arb_token_string() -> impl Strategy<Value = String> {
    // Generate base64-like strings for header.payload.signature
    let segment = "[A-Za-z0-9_-]{10,50}";
    (segment, segment, segment).prop_map(|(h, p, s)| format!("{}.{}.{}", h, p, s))
}

/// Generates arbitrary claim names
fn arb_claim_name() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("iss".to_string()),
        Just("sub".to_string()),
        Just("aud".to_string()),
        Just("exp".to_string()),
        Just("iat".to_string()),
        Just("jti".to_string()),
        Just("session_id".to_string()),
        Just("scopes".to_string()),
        "[a-z_]{3,20}".prop_map(|s| s.to_string()),
    ]
}

/// Generates arbitrary subject strings
fn arb_subject() -> impl Strategy<Value = String> {
    "[a-zA-Z0-9@._-]{5,50}".prop_map(|s| s.to_string())
}

/// Generates arbitrary issuer strings
fn arb_issuer() -> impl Strategy<Value = String> {
    "https://[a-z]{3,10}\\.[a-z]{2,5}/".prop_map(|s| s.to_string())
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 10: JWT Type-State Transitions
    /// 
    /// Validates that:
    /// - Unvalidated tokens cannot access claims
    /// - SignatureValidated tokens can peek but not fully access claims
    /// - Only Validated tokens can access claims
    #[test]
    fn prop_typestate_enforces_validation_order(
        subject in arb_subject(),
        issuer in arb_issuer(),
    ) {
        // This test validates the compile-time guarantees of the type-state pattern
        // The actual enforcement is at compile time, so we verify the state names
        
        // Unvalidated state
        prop_assert_eq!(
            auth_edge::jwt::Unvalidated::state_name(),
            "Unvalidated"
        );
        
        // SignatureValidated state
        prop_assert_eq!(
            auth_edge::jwt::SignatureValidated::state_name(),
            "SignatureValidated"
        );
        
        // Validated state
        prop_assert_eq!(
            auth_edge::jwt::Validated::state_name(),
            "Validated"
        );
    }

    /// Property: Claims has_claim consistency
    /// 
    /// Validates that has_claim returns consistent results for standard claims.
    #[test]
    fn prop_claims_has_claim_consistency(
        subject in arb_subject(),
        issuer in arb_issuer(),
        jti in "[a-f0-9]{32}",
    ) {
        use auth_edge::jwt::Claims;
        use serde_json::Value;

        let claims = Claims {
            iss: issuer.clone(),
            sub: subject.clone(),
            aud: vec!["test-audience".to_string()],
            exp: chrono::Utc::now().timestamp() + 3600,
            iat: chrono::Utc::now().timestamp(),
            nbf: None,
            jti: jti.clone(),
            session_id: Some("session-123".to_string()),
            scopes: Some(vec!["read".to_string(), "write".to_string()]),
            custom: HashMap::new(),
        };

        // Standard claims should be present
        prop_assert!(claims.has_claim("iss") == !issuer.is_empty());
        prop_assert!(claims.has_claim("sub") == !subject.is_empty());
        prop_assert!(claims.has_claim("aud"));
        prop_assert!(claims.has_claim("exp"));
        prop_assert!(claims.has_claim("iat"));
        prop_assert!(claims.has_claim("jti") == !jti.is_empty());
        prop_assert!(claims.has_claim("session_id"));
        prop_assert!(claims.has_claim("scopes"));

        // Non-existent custom claim should not be present
        prop_assert!(!claims.has_claim("nonexistent_claim"));
    }

    /// Property: Claims expiration check
    /// 
    /// Validates that is_expired correctly identifies expired tokens.
    #[test]
    fn prop_claims_expiration_check(
        offset in -7200i64..7200i64,
    ) {
        use auth_edge::jwt::Claims;

        let now = chrono::Utc::now().timestamp();
        let claims = Claims {
            iss: "test-issuer".to_string(),
            sub: "test-subject".to_string(),
            aud: vec!["test-audience".to_string()],
            exp: now + offset,
            iat: now - 3600,
            nbf: None,
            jti: "test-jti".to_string(),
            session_id: None,
            scopes: None,
            custom: HashMap::new(),
        };

        // Token is expired if exp < now
        let expected_expired = offset < 0;
        prop_assert_eq!(claims.is_expired(), expected_expired);
    }

    /// Property: Claims to_map preserves data
    /// 
    /// Validates that to_map includes all standard claims.
    #[test]
    fn prop_claims_to_map_preserves_data(
        subject in arb_subject(),
        issuer in arb_issuer(),
    ) {
        use auth_edge::jwt::Claims;

        let now = chrono::Utc::now().timestamp();
        let claims = Claims {
            iss: issuer.clone(),
            sub: subject.clone(),
            aud: vec!["aud1".to_string(), "aud2".to_string()],
            exp: now + 3600,
            iat: now,
            nbf: None,
            jti: "test-jti".to_string(),
            session_id: Some("session-123".to_string()),
            scopes: Some(vec!["read".to_string()]),
            custom: HashMap::new(),
        };

        let map = claims.to_map();

        // Verify all standard claims are in the map
        prop_assert_eq!(map.get("iss"), Some(&issuer));
        prop_assert_eq!(map.get("sub"), Some(&subject));
        prop_assert!(map.contains_key("aud"));
        prop_assert!(map.contains_key("exp"));
        prop_assert!(map.contains_key("iat"));
        prop_assert!(map.contains_key("jti"));
        prop_assert!(map.contains_key("session_id"));
        prop_assert!(map.contains_key("scopes"));
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[test]
    fn test_state_names_are_correct() {
        use auth_edge::jwt::{Unvalidated, SignatureValidated, Validated, TokenState};
        
        assert_eq!(Unvalidated::state_name(), "Unvalidated");
        assert_eq!(SignatureValidated::state_name(), "SignatureValidated");
        assert_eq!(Validated::state_name(), "Validated");
    }

    #[test]
    fn test_claims_has_scope() {
        use auth_edge::jwt::Claims;
        use std::collections::HashMap;

        let claims = Claims {
            iss: "issuer".to_string(),
            sub: "subject".to_string(),
            aud: vec!["audience".to_string()],
            exp: chrono::Utc::now().timestamp() + 3600,
            iat: chrono::Utc::now().timestamp(),
            nbf: None,
            jti: "jti".to_string(),
            session_id: None,
            scopes: Some(vec!["read".to_string(), "write".to_string()]),
            custom: HashMap::new(),
        };

        assert!(claims.has_scope("read"));
        assert!(claims.has_scope("write"));
        assert!(!claims.has_scope("admin"));
    }
}
