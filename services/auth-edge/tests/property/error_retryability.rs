//! Property tests for error retryability classification.
//!
//! **Feature: auth-edge-modernization-2025, Property 1: Error Retryability Classification**
//! **Validates: Requirements 2.3**

use proptest::prelude::*;

mod generators {
    include!("generators.rs");
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Feature: auth-edge-modernization-2025, Property 1: Error Retryability Classification**
    /// **Validates: Requirements 2.3**
    ///
    /// *For any* AuthEdgeError, calling `is_retryable()` SHALL return `true` only for
    /// transient infrastructure errors (Unavailable, RateLimited, Timeout) and `false`
    /// for all domain errors (TokenMissing, TokenInvalid, TokenExpired, ClaimsInvalid, SpiffeError).
    #[test]
    fn domain_errors_are_never_retryable(error in generators::arb_domain_error()) {
        prop_assert!(
            !error.is_retryable(),
            "Domain error {:?} should not be retryable",
            error
        );
    }

    /// **Feature: auth-edge-modernization-2025, Property 1: Error Retryability Classification**
    /// **Validates: Requirements 2.3**
    ///
    /// *For any* AuthEdgeError wrapping a retryable PlatformError, is_retryable() returns true.
    #[test]
    fn retryable_platform_errors_are_retryable(error in generators::arb_retryable_auth_error()) {
        prop_assert!(
            error.is_retryable(),
            "Retryable platform error {:?} should be retryable",
            error
        );
    }

    /// **Feature: auth-edge-modernization-2025, Property 1: Error Retryability Classification**
    /// **Validates: Requirements 2.3**
    ///
    /// *For any* AuthEdgeError wrapping a non-retryable PlatformError, is_retryable() returns false.
    #[test]
    fn non_retryable_platform_errors_are_not_retryable(
        error in generators::arb_non_retryable_platform_auth_error()
    ) {
        prop_assert!(
            !error.is_retryable(),
            "Non-retryable platform error {:?} should not be retryable",
            error
        );
    }
}
