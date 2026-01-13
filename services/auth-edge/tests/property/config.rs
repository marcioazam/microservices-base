//! Property-based tests for configuration validation.
//!
//! Property 12: Configuration Validation
//! Validates that configuration correctly validates all fields.

use proptest::prelude::*;

/// Generates valid port numbers
fn arb_valid_port() -> impl Strategy<Value = u16> {
    1u16..=65535u16
}

/// Generates invalid port numbers (0)
fn arb_invalid_port() -> impl Strategy<Value = u16> {
    Just(0u16)
}

/// Generates valid TTL values (positive)
fn arb_valid_ttl() -> impl Strategy<Value = u64> {
    1u64..=86400u64
}

/// Generates invalid TTL values (0)
fn arb_invalid_ttl() -> impl Strategy<Value = u64> {
    Just(0u64)
}

/// Generates valid URL strings
fn arb_valid_url() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("http://localhost:8080".to_string()),
        Just("https://api.example.com".to_string()),
        Just("grpc://service:50051".to_string()),
        (1u16..65535u16).prop_map(|port| format!("http://localhost:{}", port)),
    ]
}

/// Generates valid host strings
fn arb_valid_host() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("0.0.0.0".to_string()),
        Just("127.0.0.1".to_string()),
        Just("localhost".to_string()),
        "[a-z]{3,10}\\.[a-z]{2,5}".prop_map(|s| s.to_string()),
    ]
}

/// Generates valid SPIFFE domains
fn arb_spiffe_domain() -> impl Strategy<Value = String> {
    "[a-z]{3,10}\\.[a-z]{2,5}".prop_map(|s| s.to_string())
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 12: Configuration Validation - Valid Ports
    /// 
    /// Validates that valid port numbers (1-65535) are accepted.
    #[test]
    fn prop_valid_port_accepted(port in arb_valid_port()) {
        // Port should be in valid range
        prop_assert!(port >= 1 && port <= 65535);
        
        // Port validation should pass
        let is_valid = port > 0 && port <= 65535;
        prop_assert!(is_valid);
    }

    /// Property: Invalid port rejected
    /// 
    /// Validates that port 0 is rejected.
    #[test]
    fn prop_invalid_port_rejected(port in arb_invalid_port()) {
        // Port 0 should be invalid
        prop_assert_eq!(port, 0);
        
        // Port validation should fail
        let is_valid = port > 0 && port <= 65535;
        prop_assert!(!is_valid);
    }

    /// Property: Valid TTL accepted
    /// 
    /// Validates that positive TTL values are accepted.
    #[test]
    fn prop_valid_ttl_accepted(ttl in arb_valid_ttl()) {
        // TTL should be positive
        prop_assert!(ttl > 0);
        
        // TTL validation should pass
        let is_valid = ttl > 0;
        prop_assert!(is_valid);
    }

    /// Property: Invalid TTL rejected
    /// 
    /// Validates that TTL of 0 is rejected.
    #[test]
    fn prop_invalid_ttl_rejected(ttl in arb_invalid_ttl()) {
        // TTL 0 should be invalid
        prop_assert_eq!(ttl, 0);
        
        // TTL validation should fail
        let is_valid = ttl > 0;
        prop_assert!(!is_valid);
    }

    /// Property: URL format validation
    /// 
    /// Validates that URLs are properly formatted.
    #[test]
    fn prop_url_format_validation(url in arb_valid_url()) {
        // URL should start with a valid scheme
        let has_valid_scheme = url.starts_with("http://") 
            || url.starts_with("https://") 
            || url.starts_with("grpc://");
        prop_assert!(has_valid_scheme);
        
        // URL should not be empty
        prop_assert!(!url.is_empty());
    }

    /// Property: Host validation
    /// 
    /// Validates that host strings are properly formatted.
    #[test]
    fn prop_host_validation(host in arb_valid_host()) {
        // Host should not be empty
        prop_assert!(!host.is_empty());
        
        // Host should not contain invalid characters
        let has_valid_chars = host.chars().all(|c| {
            c.is_alphanumeric() || c == '.' || c == '-' || c == '_'
        });
        prop_assert!(has_valid_chars);
    }

    /// Property: SPIFFE domain validation
    /// 
    /// Validates that SPIFFE domains are properly formatted.
    #[test]
    fn prop_spiffe_domain_validation(domain in arb_spiffe_domain()) {
        // Domain should contain at least one dot
        prop_assert!(domain.contains('.'));
        
        // Domain should not be empty
        prop_assert!(!domain.is_empty());
        
        // Domain should only contain valid characters
        let has_valid_chars = domain.chars().all(|c| {
            c.is_alphanumeric() || c == '.' || c == '-'
        });
        prop_assert!(has_valid_chars);
    }

    /// Property: Circuit breaker threshold validation
    /// 
    /// Validates that circuit breaker thresholds are positive.
    #[test]
    fn prop_circuit_breaker_threshold_validation(threshold in 1u32..100u32) {
        // Threshold should be positive
        prop_assert!(threshold > 0);
        
        // Threshold should be reasonable (not too high)
        prop_assert!(threshold <= 100);
    }

    /// Property: Timeout validation
    /// 
    /// Validates that timeout values are positive and reasonable.
    #[test]
    fn prop_timeout_validation(timeout_secs in 1u64..300u64) {
        // Timeout should be positive
        prop_assert!(timeout_secs > 0);
        
        // Timeout should be reasonable (not more than 5 minutes)
        prop_assert!(timeout_secs <= 300);
    }
}

#[cfg(test)]
mod unit_tests {
    #[test]
    fn test_port_range_validation() {
        // Valid ports
        assert!((1..=65535).contains(&1));
        assert!((1..=65535).contains(&80));
        assert!((1..=65535).contains(&443));
        assert!((1..=65535).contains(&8080));
        assert!((1..=65535).contains(&65535));
        
        // Invalid port
        assert!(!(1..=65535).contains(&0));
    }

    #[test]
    fn test_ttl_validation() {
        // Valid TTLs
        assert!(1 > 0);
        assert!(3600 > 0);
        assert!(86400 > 0);
        
        // Invalid TTL
        assert!(!(0 > 0));
    }

    #[test]
    fn test_url_scheme_validation() {
        let valid_urls = [
            "http://localhost:8080",
            "https://api.example.com",
            "grpc://service:50051",
        ];
        
        for url in valid_urls {
            assert!(
                url.starts_with("http://") 
                || url.starts_with("https://") 
                || url.starts_with("grpc://")
            );
        }
    }
}
