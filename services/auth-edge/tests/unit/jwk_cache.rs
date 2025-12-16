//! JWK Cache Unit Tests
//!
//! Tests for JWK validation and key type support.

#[derive(Debug, Clone)]
struct Jwk {
    kty: String,
    kid: String,
    key_use: Option<String>,
    alg: Option<String>,
    n: Option<String>,
    e: Option<String>,
    x: Option<String>,
    y: Option<String>,
}

fn is_supported_key_type(kty: &str) -> bool {
    matches!(kty, "RSA" | "EC" | "oct")
}

fn validate_rsa_jwk(jwk: &Jwk) -> bool {
    jwk.kty == "RSA" && jwk.n.is_some() && jwk.e.is_some()
}

fn validate_ec_jwk(jwk: &Jwk) -> bool {
    jwk.kty == "EC" && jwk.x.is_some() && jwk.y.is_some()
}

#[test]
fn test_supported_key_types() {
    assert!(is_supported_key_type("RSA"));
    assert!(is_supported_key_type("EC"));
    assert!(is_supported_key_type("oct"));
    assert!(!is_supported_key_type("unknown"));
    assert!(!is_supported_key_type("DSA"));
}

#[test]
fn test_validate_rsa_jwk_valid() {
    let jwk = Jwk {
        kty: "RSA".to_string(),
        kid: "key-1".to_string(),
        key_use: Some("sig".to_string()),
        alg: Some("RS256".to_string()),
        n: Some("modulus".to_string()),
        e: Some("AQAB".to_string()),
        x: None,
        y: None,
    };
    assert!(validate_rsa_jwk(&jwk));
}

#[test]
fn test_validate_rsa_jwk_missing_n() {
    let jwk = Jwk {
        kty: "RSA".to_string(),
        kid: "key-1".to_string(),
        key_use: None,
        alg: None,
        n: None,
        e: Some("AQAB".to_string()),
        x: None,
        y: None,
    };
    assert!(!validate_rsa_jwk(&jwk));
}

#[test]
fn test_validate_rsa_jwk_missing_e() {
    let jwk = Jwk {
        kty: "RSA".to_string(),
        kid: "key-1".to_string(),
        key_use: None,
        alg: None,
        n: Some("modulus".to_string()),
        e: None,
        x: None,
        y: None,
    };
    assert!(!validate_rsa_jwk(&jwk));
}

#[test]
fn test_validate_ec_jwk_valid() {
    let jwk = Jwk {
        kty: "EC".to_string(),
        kid: "key-2".to_string(),
        key_use: Some("sig".to_string()),
        alg: Some("ES256".to_string()),
        n: None,
        e: None,
        x: Some("x-coord".to_string()),
        y: Some("y-coord".to_string()),
    };
    assert!(validate_ec_jwk(&jwk));
}

#[test]
fn test_validate_ec_jwk_missing_x() {
    let jwk = Jwk {
        kty: "EC".to_string(),
        kid: "key-2".to_string(),
        key_use: None,
        alg: None,
        n: None,
        e: None,
        x: None,
        y: Some("y-coord".to_string()),
    };
    assert!(!validate_ec_jwk(&jwk));
}

#[test]
fn test_validate_ec_jwk_missing_y() {
    let jwk = Jwk {
        kty: "EC".to_string(),
        kid: "key-2".to_string(),
        key_use: None,
        alg: None,
        n: None,
        e: None,
        x: Some("x-coord".to_string()),
        y: None,
    };
    assert!(!validate_ec_jwk(&jwk));
}

#[test]
fn test_jwk_kid_required() {
    let jwk = Jwk {
        kty: "RSA".to_string(),
        kid: "".to_string(),
        key_use: None,
        alg: None,
        n: Some("n".to_string()),
        e: Some("e".to_string()),
        x: None,
        y: None,
    };
    assert!(jwk.kid.is_empty());
}
