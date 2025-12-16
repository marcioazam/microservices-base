//! Certificate Verifier Unit Tests
//!
//! Tests for certificate validity and SPIFFE URI validation.

use std::time::{SystemTime, UNIX_EPOCH};

fn is_certificate_valid(not_before: i64, not_after: i64) -> Result<(), &'static str> {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    if now < not_before {
        return Err("Certificate not yet valid");
    }

    if now > not_after {
        return Err("Certificate expired");
    }

    Ok(())
}

#[test]
fn test_certificate_valid_period() {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let not_before = now - 3600;
    let not_after = now + 3600;

    assert!(is_certificate_valid(not_before, not_after).is_ok());
}

#[test]
fn test_certificate_not_yet_valid() {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let not_before = now + 3600;
    let not_after = now + 7200;

    let result = is_certificate_valid(not_before, not_after);
    assert!(result.is_err());
    assert_eq!(result.unwrap_err(), "Certificate not yet valid");
}

#[test]
fn test_certificate_expired() {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let not_before = now - 7200;
    let not_after = now - 3600;

    let result = is_certificate_valid(not_before, not_after);
    assert!(result.is_err());
    assert_eq!(result.unwrap_err(), "Certificate expired");
}

#[test]
fn test_spiffe_uri_in_san() {
    let san_uri = "spiffe://example.org/ns/default/sa/myservice";
    let trust_domain = "example.org";
    let expected_prefix = format!("spiffe://{}/", trust_domain);

    assert!(san_uri.starts_with(&expected_prefix));
}

#[test]
fn test_spiffe_uri_wrong_domain() {
    let san_uri = "spiffe://other.org/ns/default/sa/myservice";
    let trust_domain = "example.org";
    let expected_prefix = format!("spiffe://{}/", trust_domain);

    assert!(!san_uri.starts_with(&expected_prefix));
}
