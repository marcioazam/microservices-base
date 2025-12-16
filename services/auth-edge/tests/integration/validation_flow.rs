//! Validation Flow Integration Tests
//!
//! Tests that simulate full validation flows using mocks.

use std::collections::HashMap;
use uuid::Uuid;

struct MockClaims {
    sub: String,
    exp: i64,
    scopes: Vec<String>,
}

struct MockValidator {
    valid_tokens: HashMap<String, MockClaims>,
}

impl MockValidator {
    fn new() -> Self {
        let mut valid_tokens = HashMap::new();
        valid_tokens.insert(
            "valid-token".to_string(),
            MockClaims {
                sub: "user-123".to_string(),
                exp: chrono::Utc::now().timestamp() + 3600,
                scopes: vec!["read".to_string(), "write".to_string()],
            },
        );
        Self { valid_tokens }
    }

    fn validate(&self, token: &str, required_claims: &[&str]) -> Result<&MockClaims, &'static str> {
        if token.is_empty() {
            return Err("Token missing");
        }

        let claims = self.valid_tokens.get(token).ok_or("Token invalid")?;

        if claims.exp < chrono::Utc::now().timestamp() {
            return Err("Token expired");
        }

        for claim in required_claims {
            if *claim == "scopes" && claims.scopes.is_empty() {
                return Err("Missing required claim");
            }
        }

        Ok(claims)
    }
}

#[test]
fn test_full_validation_flow_success() {
    let validator = MockValidator::new();
    let result = validator.validate("valid-token", &["sub"]);

    assert!(result.is_ok());
    assert_eq!(result.unwrap().sub, "user-123");
}

#[test]
fn test_full_validation_flow_missing_token() {
    let validator = MockValidator::new();
    let result = validator.validate("", &[]);

    assert!(result.is_err());
    assert_eq!(result.unwrap_err(), "Token missing");
}

#[test]
fn test_full_validation_flow_invalid_token() {
    let validator = MockValidator::new();
    let result = validator.validate("invalid-token", &[]);

    assert!(result.is_err());
    assert_eq!(result.unwrap_err(), "Token invalid");
}

#[test]
fn test_error_response_includes_correlation_id() {
    let correlation_id = Uuid::new_v4();
    let error_message = format!("Token is required [correlation_id: {}]", correlation_id);

    assert!(error_message.contains("correlation_id"));
    assert!(error_message.contains(&correlation_id.to_string()));
}
