//! SPIFFE ID Property Tests
//!
//! Validates SPIFFE ID parsing and round-trip.

use proptest::prelude::*;
use super::generators::{arb_trust_domain, arb_spiffe_path};

#[derive(Debug, Clone)]
struct SpiffeId {
    trust_domain: String,
    path: String,
}

impl SpiffeId {
    fn parse(uri: &str) -> Option<Self> {
        if !uri.starts_with("spiffe://") {
            return None;
        }
        let rest = uri.strip_prefix("spiffe://")?;
        let parts: Vec<&str> = rest.splitn(2, '/').collect();
        if parts.is_empty() || parts[0].is_empty() || !parts[0].contains('.') {
            return None;
        }
        Some(SpiffeId {
            trust_domain: parts[0].to_string(),
            path: parts.get(1).unwrap_or(&"").to_string(),
        })
    }

    fn to_uri(&self) -> String {
        if self.path.is_empty() {
            format!("spiffe://{}", self.trust_domain)
        } else {
            format!("spiffe://{}/{}", self.trust_domain, self.path)
        }
    }
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property: SPIFFE ID round-trip preserves trust domain
    #[test]
    fn prop_spiffe_roundtrip(
        trust_domain in arb_trust_domain(),
        path in arb_spiffe_path(),
    ) {
        let uri = if path.is_empty() {
            format!("spiffe://{}", trust_domain)
        } else {
            format!("spiffe://{}/{}", trust_domain, path.trim_matches('/'))
        };
        
        let parsed = SpiffeId::parse(&uri);
        prop_assert!(parsed.is_some(), "Valid SPIFFE URI should parse: {}", uri);
        
        let spiffe_id = parsed.unwrap();
        prop_assert_eq!(spiffe_id.trust_domain, trust_domain);
        
        let reconstructed = spiffe_id.to_uri();
        let reparsed = SpiffeId::parse(&reconstructed);
        prop_assert!(reparsed.is_some());
        prop_assert_eq!(reparsed.unwrap().trust_domain, trust_domain);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_spiffe_id_parse_valid() {
        let id = SpiffeId::parse("spiffe://example.org/ns/default/sa/myservice").unwrap();
        assert_eq!(id.trust_domain, "example.org");
        assert_eq!(id.path, "ns/default/sa/myservice");
    }

    #[test]
    fn test_spiffe_id_parse_invalid() {
        assert!(SpiffeId::parse("http://example.org").is_none());
        assert!(SpiffeId::parse("spiffe://").is_none());
        assert!(SpiffeId::parse("spiffe://nodot").is_none());
    }
}
