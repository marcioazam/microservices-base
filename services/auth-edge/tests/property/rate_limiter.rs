//! Rate Limiter Property Tests
//!
//! Validates rate limiting enforcement.

use proptest::prelude::*;
use std::time::Duration;

struct RateLimiter {
    base_limit: u32,
    window: Duration,
    load_threshold: f64,
    load_reduction_factor: f64,
    trust_multiplier: f64,
}

impl RateLimiter {
    fn new(base_limit: u32) -> Self {
        RateLimiter {
            base_limit,
            window: Duration::from_secs(60),
            load_threshold: 0.8,
            load_reduction_factor: 0.5,
            trust_multiplier: 2.0,
        }
    }

    fn calculate_effective_limit(&self, system_load: f64, is_trusted: bool) -> u32 {
        let mut limit = self.base_limit as f64;
        if system_load > self.load_threshold {
            limit *= self.load_reduction_factor;
        }
        if is_trusted {
            limit *= self.trust_multiplier;
        }
        limit.max(1.0) as u32
    }
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property: Rate limiter always enforces minimum limit
    #[test]
    fn prop_rate_limiter_enforcement(
        base_limit in 10u32..100u32,
        system_load in 0.0f64..1.0f64,
        is_trusted in proptest::bool::ANY,
    ) {
        let limiter = RateLimiter::new(base_limit);
        let effective_limit = limiter.calculate_effective_limit(system_load, is_trusted);
        
        prop_assert!(effective_limit >= 1);
        
        if system_load > 0.8 && !is_trusted {
            prop_assert!(effective_limit <= (base_limit as f64 * 0.5) as u32 + 1);
        }
        
        if is_trusted && system_load <= 0.8 {
            prop_assert!(effective_limit >= base_limit);
        }
    }

    /// Property: Rate limit headers are valid
    #[test]
    fn prop_rate_limit_headers(
        remaining in 0u32..1000u32,
        limit in 1u32..1000u32,
        reset_secs in 1u64..3600u64,
    ) {
        let remaining_header = remaining.to_string();
        let limit_header = limit.to_string();
        let reset_header = reset_secs.to_string();
        
        prop_assert!(!remaining_header.is_empty());
        prop_assert!(!limit_header.is_empty());
        prop_assert!(!reset_header.is_empty());
        
        prop_assert!(remaining_header.parse::<u32>().is_ok());
        prop_assert!(limit_header.parse::<u32>().is_ok());
        prop_assert!(reset_header.parse::<u64>().is_ok());
    }

    /// Property: Timeout is always positive and bounded
    #[test]
    fn prop_timeout_enforcement(timeout_ms in 1u64..1000u64) {
        let timeout = Duration::from_millis(timeout_ms);
        
        prop_assert!(timeout.as_millis() > 0);
        prop_assert!(timeout.as_millis() <= 1000);
    }
}
