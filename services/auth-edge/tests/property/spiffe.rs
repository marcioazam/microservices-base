//! Property-based tests for SPIFFE ID handling.
//!
//! Property 11: SPIFFE ID Round-Trip
//! Validates that SPIFFE IDs can be parsed and serialized without data loss.

use proptest::prelude::*;

/// Generates valid trust domain components
fn arb_trust_domain_label() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{0,10}".prop_map(|s| s.to_string())
}

/// Generates valid trust domains (e.g., "example.org")
fn arb_trust_domain() -> impl Strategy<Value = String> {
    (arb_trust_domain_label(), arb_trust_domain_label())
        .prop_map(|(a, b)| format!("{}.{}", a, b))
}

/// Generates valid path segments
fn arb_path_segment() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{0,20}".prop_map(|s| s.to_string())
}

/// Generates valid SPIFFE URIs
fn arb_spiffe_uri() -> impl Strategy<Value = String> {
    (
        arb_trust_domain(),
        prop::collection::vec(arb_path_segment(), 0..5),
    )
        .prop_map(|(domain, path)| {
            if path.is_empty() {
                format!("spiffe://{}", domain)
            } else {
                format!("spiffe://{}/{}", domain, path.join("/"))
            }
        })
}

/// Generates Kubernetes-style SPIFFE URIs
fn arb_k8s_spiffe_uri() -> impl Strategy<Value = String> {
    (
        arb_trust_domain(),
        arb_path_segment(), // namespace
        arb_path_segment(), // service account
    )
        .prop_map(|(domain, ns, sa)| {
            format!("spiffe://{}/ns/{}/sa/{}", domain, ns, sa)
        })
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 11: SPIFFE ID Round-Trip
    /// 
    /// Validates that:
    /// - Valid SPIFFE URIs can be parsed
    /// - Parsed SPIFFE IDs can be serialized back to the original URI
    /// - Trust domain is preserved
    /// - Path segments are preserved
    #[test]
    fn prop_spiffe_id_roundtrip(uri in arb_spiffe_uri()) {
        use auth_edge::mtls::{SpiffeId, OwnedSpiffeId};

        // Parse the URI
        let parsed = SpiffeId::parse(&uri);
        prop_assert!(parsed.is_ok(), "Failed to parse valid SPIFFE URI: {}", uri);
        
        let spiffe_id = parsed.unwrap();
        
        // Serialize back to URI
        let serialized = spiffe_id.to_uri();
        
        // Should match original (normalized)
        prop_assert_eq!(serialized, uri);
        
        // Owned version should also round-trip
        let owned = OwnedSpiffeId::parse(&uri).unwrap();
        prop_assert_eq!(owned.to_uri(), uri);
    }

    /// Property: Trust domain preservation
    /// 
    /// Validates that the trust domain is correctly extracted and preserved.
    #[test]
    fn prop_trust_domain_preserved(
        domain in arb_trust_domain(),
        path in prop::collection::vec(arb_path_segment(), 0..3),
    ) {
        use auth_edge::mtls::SpiffeId;

        let uri = if path.is_empty() {
            format!("spiffe://{}", domain)
        } else {
            format!("spiffe://{}/{}", domain, path.join("/"))
        };

        let spiffe_id = SpiffeId::parse(&uri).unwrap();
        
        // Trust domain should be preserved exactly
        prop_assert_eq!(spiffe_id.trust_domain.as_ref(), domain.as_str());
    }

    /// Property: Path segments preservation
    /// 
    /// Validates that path segments are correctly extracted and preserved.
    #[test]
    fn prop_path_segments_preserved(
        domain in arb_trust_domain(),
        path in prop::collection::vec(arb_path_segment(), 1..5),
    ) {
        use auth_edge::mtls::SpiffeId;

        let uri = format!("spiffe://{}/{}", domain, path.join("/"));
        let spiffe_id = SpiffeId::parse(&uri).unwrap();
        
        // Path segments should be preserved
        prop_assert_eq!(spiffe_id.path.len(), path.len());
        for (parsed, original) in spiffe_id.path.iter().zip(path.iter()) {
            prop_assert_eq!(parsed.as_ref(), original.as_str());
        }
    }

    /// Property: Service name extraction from K8s-style SPIFFE IDs
    /// 
    /// Validates that service names are correctly extracted from
    /// Kubernetes-style SPIFFE IDs (spiffe://domain/ns/namespace/sa/service).
    #[test]
    fn prop_service_name_extraction(uri in arb_k8s_spiffe_uri()) {
        use auth_edge::mtls::{OwnedSpiffeId, SpiffeValidator};

        let spiffe_id = OwnedSpiffeId::parse(&uri).unwrap();
        let service_name = SpiffeValidator::extract_service_name(&spiffe_id);
        
        // Service name should be extracted (last segment after "sa")
        prop_assert!(service_name.is_some());
        
        // Service name should be the segment after "sa"
        let expected_sa_index = spiffe_id.path.iter().position(|s| s == "sa");
        if let Some(idx) = expected_sa_index {
            if idx + 1 < spiffe_id.path.len() {
                prop_assert_eq!(service_name.as_ref(), Some(&spiffe_id.path[idx + 1]));
            }
        }
    }

    /// Property: Pattern matching
    /// 
    /// Validates that SPIFFE ID pattern matching works correctly.
    #[test]
    fn prop_pattern_matching(
        domain in arb_trust_domain(),
        path in prop::collection::vec(arb_path_segment(), 1..3),
    ) {
        use auth_edge::mtls::SpiffeId;

        let uri = format!("spiffe://{}/{}", domain, path.join("/"));
        let spiffe_id = SpiffeId::parse(&uri).unwrap();
        
        // Exact match should work
        prop_assert!(spiffe_id.matches(&uri));
        
        // Wildcard match should work
        let wildcard = format!("spiffe://{}/*", domain);
        prop_assert!(spiffe_id.matches(&wildcard));
        
        // Different domain should not match
        let other_domain = format!("spiffe://other.domain/*");
        prop_assert!(!spiffe_id.matches(&other_domain));
    }

    /// Property: Validator trust domain enforcement
    /// 
    /// Validates that the validator correctly enforces trust domain allowlist.
    #[test]
    fn prop_validator_trust_domain_enforcement(
        allowed_domain in arb_trust_domain(),
        other_domain in arb_trust_domain(),
    ) {
        use auth_edge::mtls::{SpiffeId, SpiffeValidator};

        // Skip if domains are the same
        prop_assume!(allowed_domain != other_domain);

        let validator = SpiffeValidator::new(vec![allowed_domain.clone()]);
        
        // Allowed domain should pass
        let allowed_uri = format!("spiffe://{}/test", allowed_domain);
        let allowed_id = SpiffeId::parse(&allowed_uri).unwrap();
        prop_assert!(validator.validate(&allowed_id).is_ok());
        
        // Other domain should fail
        let other_uri = format!("spiffe://{}/test", other_domain);
        let other_id = SpiffeId::parse(&other_uri).unwrap();
        prop_assert!(validator.validate(&other_id).is_err());
    }
}

#[cfg(test)]
mod unit_tests {
    use auth_edge::mtls::{SpiffeId, OwnedSpiffeId, SpiffeValidator, SpiffeError};

    #[test]
    fn test_invalid_scheme_rejected() {
        let result = SpiffeId::parse("https://example.org/path");
        assert!(matches!(result, Err(SpiffeError::InvalidScheme)));
    }

    #[test]
    fn test_empty_trust_domain_rejected() {
        let result = SpiffeId::parse("spiffe:///path");
        assert!(matches!(result, Err(SpiffeError::EmptyTrustDomain)));
    }

    #[test]
    fn test_invalid_trust_domain_rejected() {
        // Trust domain without dot
        let result = SpiffeId::parse("spiffe://localhost/path");
        assert!(matches!(result, Err(SpiffeError::InvalidTrustDomain(_))));
    }

    #[test]
    fn test_valid_spiffe_id_parsed() {
        let result = SpiffeId::parse("spiffe://example.org/ns/default/sa/myservice");
        assert!(result.is_ok());
        
        let spiffe_id = result.unwrap();
        assert_eq!(spiffe_id.trust_domain.as_ref(), "example.org");
        assert_eq!(spiffe_id.path.len(), 4);
        assert_eq!(spiffe_id.path[0].as_ref(), "ns");
        assert_eq!(spiffe_id.path[1].as_ref(), "default");
        assert_eq!(spiffe_id.path[2].as_ref(), "sa");
        assert_eq!(spiffe_id.path[3].as_ref(), "myservice");
    }

    #[test]
    fn test_service_name_extraction() {
        let spiffe_id = OwnedSpiffeId::parse("spiffe://example.org/ns/default/sa/myservice").unwrap();
        let service_name = SpiffeValidator::extract_service_name(&spiffe_id);
        assert_eq!(service_name, Some("myservice".to_string()));
    }

    #[test]
    fn test_validator_allows_trusted_domain() {
        let validator = SpiffeValidator::new(vec!["example.org".to_string()]);
        let spiffe_id = SpiffeId::parse("spiffe://example.org/test").unwrap();
        assert!(validator.validate(&spiffe_id).is_ok());
    }

    #[test]
    fn test_validator_rejects_untrusted_domain() {
        let validator = SpiffeValidator::new(vec!["example.org".to_string()]);
        let spiffe_id = SpiffeId::parse("spiffe://other.org/test").unwrap();
        assert!(matches!(validator.validate(&spiffe_id), Err(SpiffeError::UntrustedDomain(_))));
    }
}
