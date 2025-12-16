//! SPIFFE ID parsing and validation
//!
//! Implements SPIFFE (Secure Production Identity Framework for Everyone)
//! for workload identity in Zero Trust architecture.
//!
//! Uses Cow<str> for zero-copy parsing where possible.

use std::borrow::Cow;
use std::collections::HashSet;

/// SPIFFE ID structure with zero-copy support
#[derive(Debug, Clone, PartialEq)]
pub struct SpiffeId<'a> {
    /// Trust domain (e.g., "example.org")
    pub trust_domain: Cow<'a, str>,
    /// Path segments (e.g., ["ns", "default", "sa", "myservice"])
    pub path: Vec<Cow<'a, str>>,
}

/// Owned version of SpiffeId for when lifetime management is needed
#[derive(Debug, Clone, PartialEq)]
pub struct OwnedSpiffeId {
    /// Trust domain (e.g., "example.org")
    pub trust_domain: String,
    /// Path segments (e.g., ["ns", "default", "sa", "myservice"])
    pub path: Vec<String>,
}

/// SPIFFE ID validation error
#[derive(Debug, thiserror::Error)]
pub enum SpiffeError {
    #[error("Invalid SPIFFE URI scheme: expected 'spiffe://'")]
    InvalidScheme,
    
    #[error("Empty trust domain")]
    EmptyTrustDomain,
    
    #[error("Invalid trust domain: {0}")]
    InvalidTrustDomain(String),
    
    #[error("Trust domain not in allowlist: {0}")]
    UntrustedDomain(String),
    
    #[error("Invalid path segment")]
    InvalidPath,
}

impl<'a> SpiffeId<'a> {
    /// Parses a SPIFFE ID from a URI string with zero-copy where possible
    /// Format: spiffe://trust-domain/path/segments
    pub fn parse(uri: &'a str) -> Result<Self, SpiffeError> {
        // Check scheme
        if !uri.starts_with("spiffe://") {
            return Err(SpiffeError::InvalidScheme);
        }

        let rest = &uri[9..]; // Skip "spiffe://"
        
        // Split trust domain and path
        let (trust_domain, path_str) = match rest.find('/') {
            Some(idx) => (&rest[..idx], &rest[idx + 1..]),
            None => (rest, ""),
        };

        // Validate trust domain
        if trust_domain.is_empty() {
            return Err(SpiffeError::EmptyTrustDomain);
        }

        if !Self::is_valid_trust_domain(trust_domain) {
            return Err(SpiffeError::InvalidTrustDomain(trust_domain.to_string()));
        }

        // Parse path segments with zero-copy
        let path: Vec<Cow<'a, str>> = if path_str.is_empty() {
            vec![]
        } else {
            path_str
                .split('/')
                .filter(|s| !s.is_empty())
                .map(Cow::Borrowed)
                .collect()
        };

        Ok(SpiffeId {
            trust_domain: Cow::Borrowed(trust_domain),
            path,
        })
    }

    /// Validates trust domain format
    fn is_valid_trust_domain(domain: &str) -> bool {
        // Trust domain must be a valid DNS name
        if domain.is_empty() || domain.len() > 255 {
            return false;
        }

        // Must contain at least one dot (e.g., "example.org")
        if !domain.contains('.') {
            return false;
        }

        // Check each label
        for label in domain.split('.') {
            if label.is_empty() || label.len() > 63 {
                return false;
            }
            
            // Must start with alphanumeric
            if !label.chars().next().map(|c| c.is_alphanumeric()).unwrap_or(false) {
                return false;
            }

            // Must contain only alphanumeric and hyphens
            if !label.chars().all(|c| c.is_alphanumeric() || c == '-') {
                return false;
            }
        }

        true
    }

    /// Converts to URI string
    pub fn to_uri(&self) -> String {
        if self.path.is_empty() {
            format!("spiffe://{}", self.trust_domain)
        } else {
            let path_str: Vec<&str> = self.path.iter().map(|s| s.as_ref()).collect();
            format!("spiffe://{}/{}", self.trust_domain, path_str.join("/"))
        }
    }

    /// Checks if this SPIFFE ID matches a pattern
    /// Supports wildcards: spiffe://example.org/* matches any path
    pub fn matches(&self, pattern: &str) -> bool {
        if pattern.ends_with("/*") {
            let prefix = &pattern[..pattern.len() - 2];
            self.to_uri().starts_with(prefix)
        } else {
            self.to_uri() == pattern
        }
    }

    /// Converts to owned version
    pub fn to_owned(&self) -> OwnedSpiffeId {
        OwnedSpiffeId {
            trust_domain: self.trust_domain.to_string(),
            path: self.path.iter().map(|s| s.to_string()).collect(),
        }
    }
}

impl OwnedSpiffeId {
    /// Parses a SPIFFE ID from a URI string (owned version)
    pub fn parse(uri: &str) -> Result<Self, SpiffeError> {
        let borrowed = SpiffeId::parse(uri)?;
        Ok(borrowed.to_owned())
    }

    /// Converts to URI string
    pub fn to_uri(&self) -> String {
        if self.path.is_empty() {
            format!("spiffe://{}", self.trust_domain)
        } else {
            format!("spiffe://{}/{}", self.trust_domain, self.path.join("/"))
        }
    }

    /// Checks if this SPIFFE ID matches a pattern
    pub fn matches(&self, pattern: &str) -> bool {
        if pattern.ends_with("/*") {
            let prefix = &pattern[..pattern.len() - 2];
            self.to_uri().starts_with(prefix)
        } else {
            self.to_uri() == pattern
        }
    }
}

/// SPIFFE ID validator with trust domain allowlist
pub struct SpiffeValidator {
    allowed_domains: HashSet<String>,
}

impl SpiffeValidator {
    pub fn new(allowed_domains: Vec<String>) -> Self {
        SpiffeValidator {
            allowed_domains: allowed_domains.into_iter().collect(),
        }
    }

    /// Validates a SPIFFE ID against the allowlist
    pub fn validate<'a>(&self, spiffe_id: &SpiffeId<'a>) -> Result<(), SpiffeError> {
        if !self.allowed_domains.contains(spiffe_id.trust_domain.as_ref()) {
            return Err(SpiffeError::UntrustedDomain(spiffe_id.trust_domain.to_string()));
        }
        Ok(())
    }

    /// Validates an owned SPIFFE ID against the allowlist
    pub fn validate_owned(&self, spiffe_id: &OwnedSpiffeId) -> Result<(), SpiffeError> {
        if !self.allowed_domains.contains(&spiffe_id.trust_domain) {
            return Err(SpiffeError::UntrustedDomain(spiffe_id.trust_domain.clone()));
        }
        Ok(())
    }

    /// Parses and validates a SPIFFE URI
    pub fn parse_and_validate<'a>(&self, uri: &'a str) -> Result<SpiffeId<'a>, SpiffeError> {
        let spiffe_id = SpiffeId::parse(uri)?;
        self.validate(&spiffe_id)?;
        Ok(spiffe_id)
    }

    /// Parses and validates a SPIFFE URI, returning owned version
    pub fn parse_and_validate_owned(&self, uri: &str) -> Result<OwnedSpiffeId, SpiffeError> {
        let spiffe_id = OwnedSpiffeId::parse(uri)?;
        self.validate_owned(&spiffe_id)?;
        Ok(spiffe_id)
    }

    /// Adds a trust domain to the allowlist
    pub fn add_trust_domain(&mut self, domain: String) {
        self.allowed_domains.insert(domain);
    }

    /// Removes a trust domain from the allowlist
    pub fn remove_trust_domain(&mut self, domain: &str) {
        self.allowed_domains.remove(domain);
    }
}

impl Default for SpiffeValidator {
    fn default() -> Self {
        SpiffeValidator::new(vec![])
    }
}
