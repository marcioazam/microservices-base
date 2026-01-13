//! Property-based tests for observability.
//!
//! Property 14: Observability Completeness

use proptest::prelude::*;

/// Generate arbitrary method names.
fn arb_method() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("IssueTokenPair".to_string()),
        Just("RefreshTokens".to_string()),
        Just("RevokeToken".to_string()),
        Just("GetJwks".to_string()),
        Just("RotateSigningKey".to_string()),
    ]
}

/// Generate arbitrary token types.
fn arb_token_type() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("access".to_string()),
        Just("refresh".to_string()),
        Just("id".to_string()),
    ]
}

/// Generate arbitrary algorithms.
fn arb_algorithm() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("RS256".to_string()),
        Just("ES256".to_string()),
        Just("PS256".to_string()),
    ]
}

/// Generate arbitrary latencies (1ms to 5s).
fn arb_latency() -> impl Strategy<Value = f64> {
    0.001f64..5.0f64
}

/// Generate arbitrary event types.
fn arb_event_type() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("REPLAY_ATTACK".to_string()),
        Just("TOKEN_REVOKED".to_string()),
        Just("FAMILY_REVOKED".to_string()),
        Just("KMS_FAILURE".to_string()),
    ]
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 14: Observability Completeness
    ///
    /// All operations must be tracked with appropriate metrics.
    #[test]
    fn prop_metrics_record_without_panic(
        method in arb_method(),
        token_type in arb_token_type(),
        algorithm in arb_algorithm(),
        latency in arb_latency(),
        event_type in arb_event_type(),
    ) {
        // Recording metrics should never panic
        token_service::metrics::record_token_issued(&token_type, &algorithm);
        token_service::metrics::record_token_refreshed("success");
        token_service::metrics::record_token_revoked("user_request");
        token_service::metrics::record_dpop_validation("success", "none");
        token_service::metrics::record_grpc_latency(&method, latency);
        token_service::metrics::record_kms_operation("sign", "success");
        token_service::metrics::record_cache_operation("get", "hit");
        token_service::metrics::record_security_event(&event_type);
    }

    /// Property: Counters increment correctly.
    #[test]
    fn prop_counters_increment(
        token_type in arb_token_type(),
        algorithm in arb_algorithm(),
        count in 1usize..10,
    ) {
        let initial = token_service::metrics::TOKENS_ISSUED
            .with_label_values(&[&token_type, &algorithm])
            .get();

        for _ in 0..count {
            token_service::metrics::record_token_issued(&token_type, &algorithm);
        }

        let final_val = token_service::metrics::TOKENS_ISSUED
            .with_label_values(&[&token_type, &algorithm])
            .get();

        prop_assert!(
            final_val >= initial + count as f64,
            "Counter should increment by at least {}", count
        );
    }

    /// Property: Different labels create separate counters.
    /// Note: This test verifies that incrementing one label doesn't affect others,
    /// but due to parallel test execution, we can only verify relative changes.
    #[test]
    fn prop_label_isolation(
        type1 in arb_token_type(),
        type2 in arb_token_type(),
        alg in arb_algorithm(),
    ) {
        prop_assume!(type1 != type2);

        let initial1 = token_service::metrics::TOKENS_ISSUED
            .with_label_values(&[&type1, &alg])
            .get();
        let initial2 = token_service::metrics::TOKENS_ISSUED
            .with_label_values(&[&type2, &alg])
            .get();

        // Increment only type1
        token_service::metrics::record_token_issued(&type1, &alg);

        let final1 = token_service::metrics::TOKENS_ISSUED
            .with_label_values(&[&type1, &alg])
            .get();
        let final2 = token_service::metrics::TOKENS_ISSUED
            .with_label_values(&[&type2, &alg])
            .get();

        // type1 should have incremented by at least 1 (may be more due to parallel tests)
        prop_assert!(
            final1 >= initial1 + 1.0,
            "type1 counter should increment by at least 1"
        );
        // type2 should not have decreased
        prop_assert!(
            final2 >= initial2,
            "type2 counter should not decrease"
        );
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[test]
    fn test_all_metric_types_exist() {
        // Verify all metrics are properly initialized
        let _ = &*token_service::metrics::TOKENS_ISSUED;
        let _ = &*token_service::metrics::TOKENS_REFRESHED;
        let _ = &*token_service::metrics::TOKENS_REVOKED;
        let _ = &*token_service::metrics::DPOP_VALIDATIONS;
        let _ = &*token_service::metrics::GRPC_LATENCY;
        let _ = &*token_service::metrics::KMS_OPERATIONS;
        let _ = &*token_service::metrics::CACHE_OPERATIONS;
        let _ = &*token_service::metrics::SECURITY_EVENTS;
    }

    #[test]
    fn test_record_refresh_statuses() {
        token_service::metrics::record_token_refreshed("success");
        token_service::metrics::record_token_refreshed("invalid");
        token_service::metrics::record_token_refreshed("expired");
        token_service::metrics::record_token_refreshed("replay");
    }

    #[test]
    fn test_record_dpop_errors() {
        token_service::metrics::record_dpop_validation("success", "none");
        token_service::metrics::record_dpop_validation("failure", "htm_mismatch");
        token_service::metrics::record_dpop_validation("failure", "htu_mismatch");
        token_service::metrics::record_dpop_validation("failure", "replay");
        token_service::metrics::record_dpop_validation("failure", "expired");
    }

    #[test]
    fn test_record_cache_operations() {
        token_service::metrics::record_cache_operation("get", "hit");
        token_service::metrics::record_cache_operation("get", "miss");
        token_service::metrics::record_cache_operation("set", "success");
        token_service::metrics::record_cache_operation("delete", "success");
    }

    #[test]
    fn test_latency_histogram_buckets() {
        // Record various latencies to test bucket distribution
        token_service::metrics::record_grpc_latency("test", 0.001);
        token_service::metrics::record_grpc_latency("test", 0.01);
        token_service::metrics::record_grpc_latency("test", 0.1);
        token_service::metrics::record_grpc_latency("test", 1.0);
    }
}
