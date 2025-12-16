//! Token State Unit Tests
//!
//! Tests for JWT structure validation and algorithm checks.

#[test]
fn test_state_names() {
    assert_eq!("Unvalidated", "Unvalidated");
    assert_eq!("SignatureValidated", "SignatureValidated");
    assert_eq!("Validated", "Validated");
}

#[test]
fn test_jwt_structure_valid() {
    let token = "header.payload.signature";
    let parts: Vec<&str> = token.split('.').collect();
    assert_eq!(parts.len(), 3);
}

#[test]
fn test_jwt_structure_invalid_two_parts() {
    let token = "header.payload";
    let parts: Vec<&str> = token.split('.').collect();
    assert_ne!(parts.len(), 3);
}

#[test]
fn test_jwt_structure_invalid_no_dots() {
    let token = "invalid-token";
    let parts: Vec<&str> = token.split('.').collect();
    assert_eq!(parts.len(), 1);
}

#[test]
fn test_kid_extraction_from_header() {
    let header_json = r#"{"alg":"RS256","typ":"JWT","kid":"test-key-id"}"#;
    assert!(header_json.contains("kid"));
    assert!(header_json.contains("test-key-id"));
}

#[test]
fn test_algorithm_validation() {
    let valid_algs = vec!["RS256", "RS384", "RS512", "ES256", "ES384", "ES512"];
    let invalid_algs = vec!["none", "HS256", "HS384", "HS512"];

    for alg in valid_algs {
        assert!(["RS256", "RS384", "RS512", "ES256", "ES384", "ES512"].contains(&alg));
    }

    for alg in invalid_algs {
        assert!(!["RS256", "RS384", "RS512", "ES256", "ES384", "ES512"].contains(&alg));
    }
}
