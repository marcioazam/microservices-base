//! SPIFFE ID Unit Tests
//!
//! Tests for SPIFFE ID parsing, validation, and trust domain management.

use std::collections::HashSet;

// ============================================================================
// SPIFFE ID Types
// ============================================================================

#[derive(Debug, Clone, PartialEq)]
struct SpiffeId {
    trust_domain: String,
    path: Vec<String>,
}

#[derive(Debug)]
enum SpiffeError {
    InvalidScheme,
    EmptyTrustDomain,
    InvalidTrustDomain(String),
    UntrustedDomain(String),
}

impl SpiffeId {
    fn parse(uri: &str) -> Result<Self, SpiffeError> {
        if !uri.starts_with("spiffe://") {
            return Err(SpiffeError::InvalidScheme);
        }

        let rest = &uri[9..];
        let (trust_domain, path_str) = match rest.find('/') {
            Some(idx) => (&rest[..idx], &rest[idx + 1..]),
            None => (rest, ""),
        };

        if trust_domain.is_empty() {
            return Err(SpiffeError::EmptyTrustDomain);
        }

        if !trust_domain.contains('.') {
            return Err(SpiffeError::InvalidTrustDomain(trust_domain.to_string()));
        }

        let path: Vec<String> = if path_str.is_empty() {
            vec![]
        } else {
            path_str.split('/').filter(|s| !s.is_empty()).map(String::from).collect()
        };

        Ok(SpiffeId { trust_domain: trust_domain.to_string(), path })
    }

    fn to_uri(&self) -> String {
        if self.path.is_empty() {
            format!("spiffe://{}", self.trust_domain)
        } else {
            format!("spiffe://{}/{}", self.trust_domain, self.path.join("/"))
        }
    }

    fn matches(&self, pattern: &str) -> bool {
        if pattern.ends_with("/*") {
            let prefix = &pattern[..pattern.len() - 2];
            self.to_uri().starts_with(prefix)
        } else {
            self.to_uri() == pattern
        }
    }
}

// ============================================================================
// SPIFFE Validator
// ============================================================================

struct SpiffeValidator {
    allowed_domains: HashSet<String>,
}

impl SpiffeValidator {
    fn new(domains: Vec<&str>) -> Self {
        Self {
            allowed_domains: domains.into_iter().map(String::from).collect(),
        }
    }

    fn is_trusted(&self, domain: &str) -> bool {
        self.allowed_domains.contains(domain)
    }

    fn add_domain(&mut self, domain: &str) {
        self.allowed_domains.insert(domain.to_string());
    }

    fn remove_domain(&mut self, domain: &str) {
        self.allowed_domains.remove(domain);
    }
}

// ============================================================================
// Service Name Extraction
// ============================================================================

fn extract_service_name(spiffe_id: &str) -> Option<String> {
    let parts: Vec<&str> = spiffe_id.split('/').collect();

    for (i, part) in parts.iter().enumerate() {
        if *part == "sa" && i + 1 < parts.len() {
            return Some(parts[i + 1].to_string());
        }
    }

    parts.last().map(|s| s.to_string())
}

// ============================================================================
// SPIFFE ID Parsing Tests
// ============================================================================

#[test]
fn test_parse_full_spiffe_id() {
    let id = SpiffeId::parse("spiffe://example.org/ns/default/sa/myservice").unwrap();
    assert_eq!(id.trust_domain, "example.org");
    assert_eq!(id.path, vec!["ns", "default", "sa", "myservice"]);
}

#[test]
fn test_parse_spiffe_id_no_path() {
    let id = SpiffeId::parse("spiffe://example.org").unwrap();
    assert_eq!(id.trust_domain, "example.org");
    assert!(id.path.is_empty());
}

#[test]
fn test_parse_spiffe_id_single_path() {
    let id = SpiffeId::parse("spiffe://example.org/service").unwrap();
    assert_eq!(id.path, vec!["service"]);
}

#[test]
fn test_parse_invalid_scheme_http() {
    let result = SpiffeId::parse("http://example.org/path");
    assert!(matches!(result, Err(SpiffeError::InvalidScheme)));
}

#[test]
fn test_parse_invalid_scheme_https() {
    let result = SpiffeId::parse("https://example.org/path");
    assert!(matches!(result, Err(SpiffeError::InvalidScheme)));
}

#[test]
fn test_parse_empty_trust_domain() {
    let result = SpiffeId::parse("spiffe:///path");
    assert!(matches!(result, Err(SpiffeError::EmptyTrustDomain)));
}

#[test]
fn test_parse_invalid_trust_domain_no_dot() {
    let result = SpiffeId::parse("spiffe://localhost/path");
    assert!(matches!(result, Err(SpiffeError::InvalidTrustDomain(_))));
}

#[test]
fn test_to_uri_roundtrip() {
    let original = "spiffe://example.org/ns/default/sa/myservice";
    let id = SpiffeId::parse(original).unwrap();
    assert_eq!(id.to_uri(), original);
}

#[test]
fn test_to_uri_no_path() {
    let id = SpiffeId::parse("spiffe://example.org").unwrap();
    assert_eq!(id.to_uri(), "spiffe://example.org");
}

#[test]
fn test_matches_exact() {
    let id = SpiffeId::parse("spiffe://example.org/ns/default").unwrap();
    assert!(id.matches("spiffe://example.org/ns/default"));
    assert!(!id.matches("spiffe://example.org/ns/other"));
}

#[test]
fn test_matches_wildcard() {
    let id = SpiffeId::parse("spiffe://example.org/ns/default/sa/myservice").unwrap();
    assert!(id.matches("spiffe://example.org/*"));
    assert!(id.matches("spiffe://example.org/ns/*"));
    assert!(id.matches("spiffe://example.org/ns/default/*"));
}

#[test]
fn test_matches_wildcard_different_domain() {
    let id = SpiffeId::parse("spiffe://example.org/path").unwrap();
    assert!(!id.matches("spiffe://other.org/*"));
}

// ============================================================================
// SPIFFE Validator Tests
// ============================================================================

#[test]
fn test_validator_empty_allowlist() {
    let validator = SpiffeValidator::new(vec![]);
    assert!(!validator.is_trusted("example.org"));
}

#[test]
fn test_validator_single_domain() {
    let validator = SpiffeValidator::new(vec!["example.org"]);
    assert!(validator.is_trusted("example.org"));
    assert!(!validator.is_trusted("other.org"));
}

#[test]
fn test_validator_multiple_domains() {
    let validator = SpiffeValidator::new(vec!["prod.example.org", "staging.example.org"]);
    assert!(validator.is_trusted("prod.example.org"));
    assert!(validator.is_trusted("staging.example.org"));
    assert!(!validator.is_trusted("dev.example.org"));
}

#[test]
fn test_validator_add_domain() {
    let mut validator = SpiffeValidator::new(vec!["example.org"]);
    assert!(!validator.is_trusted("new.org"));

    validator.add_domain("new.org");
    assert!(validator.is_trusted("new.org"));
}

#[test]
fn test_validator_remove_domain() {
    let mut validator = SpiffeValidator::new(vec!["example.org", "other.org"]);
    assert!(validator.is_trusted("example.org"));

    validator.remove_domain("example.org");
    assert!(!validator.is_trusted("example.org"));
    assert!(validator.is_trusted("other.org"));
}

// ============================================================================
// Service Name Extraction Tests
// ============================================================================

#[test]
fn test_extract_service_name_standard_format() {
    let spiffe_id = "spiffe://example.org/ns/default/sa/myservice";
    let name = extract_service_name(spiffe_id);
    assert_eq!(name, Some("myservice".to_string()));
}

#[test]
fn test_extract_service_name_different_namespace() {
    let spiffe_id = "spiffe://example.org/ns/production/sa/auth-service";
    let name = extract_service_name(spiffe_id);
    assert_eq!(name, Some("auth-service".to_string()));
}

#[test]
fn test_extract_service_name_no_sa_segment() {
    let spiffe_id = "spiffe://example.org/workload/myworkload";
    let name = extract_service_name(spiffe_id);
    assert_eq!(name, Some("myworkload".to_string()));
}

#[test]
fn test_extract_service_name_minimal() {
    let spiffe_id = "spiffe://example.org";
    let name = extract_service_name(spiffe_id);
    assert_eq!(name, Some("example.org".to_string()));
}
