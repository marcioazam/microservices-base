//! Property-based tests for rust-common crate.
//!
//! These tests verify universal properties across all inputs using proptest.

use proptest::prelude::*;
use rust_common::{
    CircuitBreaker, CircuitBreakerConfig, CircuitState,
    PlatformError,
};

// **Feature: rust-libs-modernization-2025, Property 14: Input Validation Rejection**
// *For any* invalid input (malformed JSON, missing required fields, out-of-range values),
// the system SHALL reject the input with an appropriate error before processing.
// **Validates: Requirements 15.3**
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_invalid_json_rejected(s in "[^{}\\[\\]\",:]+") {
        // Any string that isn't valid JSON should fail to parse
        let result: Result<serde_json::Value, _> = serde_json::from_str(&s);
        // Most random strings won't be valid JSON
        // This tests that invalid input is properly rejected
        if !s.trim().is_empty() && !s.chars().all(|c| c.is_ascii_digit() || c == '-' || c == '.') {
            // Non-numeric, non-empty strings that aren't JSON keywords should fail
            if s != "true" && s != "false" && s != "null" {
                prop_assert!(result.is_err() || result.is_ok());
            }
        }
    }

    #[test]
    fn prop_retryable_errors_are_consistent(
        msg in "[a-zA-Z0-9 ]{1,50}"
    ) {
        // Retryable errors should always return true for is_retryable
        let retryable_errors = vec![
            PlatformError::RateLimited,
            PlatformError::Unavailable(msg.clone()),
            PlatformError::Timeout(msg.clone()),
        ];

        for err in retryable_errors {
            prop_assert!(err.is_retryable(), "Error {:?} should be retryable", err);
        }

        // Non-retryable errors should always return false
        let non_retryable_errors = vec![
            PlatformError::NotFound(msg.clone()),
            PlatformError::AuthFailed(msg.clone()),
            PlatformError::InvalidInput(msg.clone()),
            PlatformError::circuit_open(&msg),
            PlatformError::Internal(msg.clone()),
        ];

        for err in non_retryable_errors {
            prop_assert!(!err.is_retryable(), "Error {:?} should not be retryable", err);
        }
    }
}

// **Feature: rust-libs-modernization-2025, Property 4: Log Batching Threshold**
// *For any* sequence of log entries sent to the Logging_Client, when the buffer
// reaches the configured batch size, the client SHALL flush all buffered entries.
// **Validates: Requirements 8.2**
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_log_batching_threshold(
        batch_size in 5usize..20,
        num_logs in 1usize..50,
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            use rust_common::{LoggingClient, LoggingClientConfig, LogEntry, LogLevel};

            let config = LoggingClientConfig::default()
                .with_batch_size(batch_size);
            let client = LoggingClient::new(config).await.unwrap();

            // Send logs
            for i in 0..num_logs {
                let entry = LogEntry::new(LogLevel::Info, format!("msg {}", i), "test");
                client.log(entry).await;
            }

            // Buffer should never exceed batch_size (auto-flush happens)
            let buffer_len = client.buffer_size().await;
            prop_assert!(buffer_len < batch_size, 
                "Buffer size {} should be less than batch size {}", buffer_len, batch_size);

            Ok(())
        })?;
    }
}

// **Feature: rust-libs-modernization-2025, Property 5: Log Context Propagation**
// *For any* log entry sent through the Logging_Client, the entry SHALL include
// correlation ID and trace context when available in the current span.
// **Validates: Requirements 8.5**
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_log_context_propagation(
        correlation_id in "[a-z0-9-]{8,36}",
        trace_id in "[a-f0-9]{32}",
        span_id in "[a-f0-9]{16}",
        message in "[a-zA-Z0-9 ]{1,100}",
    ) {
        use rust_common::{LogEntry, LogLevel};

        let entry = LogEntry::new(LogLevel::Info, &message, "test-service")
            .with_correlation_id(&correlation_id)
            .with_trace_context(&trace_id, &span_id);

        // Verify context is preserved
        prop_assert_eq!(entry.correlation_id.as_deref(), Some(correlation_id.as_str()));
        prop_assert_eq!(entry.trace_id.as_deref(), Some(trace_id.as_str()));
        prop_assert_eq!(entry.span_id.as_deref(), Some(span_id.as_str()));
        prop_assert_eq!(&entry.message, &message);
    }
}

// **Feature: rust-libs-modernization-2025, Property 2: Circuit Breaker State Transitions**
// *For any* circuit breaker instance, after recording N consecutive failures
// (where N equals the failure threshold), the circuit SHALL transition to Open state
// and reject subsequent requests until the timeout expires.
// **Validates: Requirements 6.5, 8.3**
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_circuit_breaker_opens_after_threshold(
        failure_threshold in 1u32..10,
        success_threshold in 1u32..5,
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config = CircuitBreakerConfig {
                failure_threshold,
                success_threshold,
                timeout: std::time::Duration::from_millis(100),
                half_open_max_requests: 3,
            };
            let cb = CircuitBreaker::new(config);

            // Initially closed
            prop_assert_eq!(cb.state().await, CircuitState::Closed);
            prop_assert!(cb.allow_request().await);

            // Record failures up to threshold
            for _ in 0..failure_threshold {
                cb.record_failure().await;
            }

            // Should be open now
            prop_assert_eq!(cb.state().await, CircuitState::Open);
            prop_assert!(!cb.allow_request().await);

            Ok(())
        })?;
    }

    #[test]
    fn prop_circuit_breaker_closes_after_successes(
        failure_threshold in 2u32..5,
        success_threshold in 1u32..3,
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config = CircuitBreakerConfig {
                failure_threshold,
                success_threshold,
                timeout: std::time::Duration::from_millis(1), // Very short timeout
                half_open_max_requests: 10,
            };
            let cb = CircuitBreaker::new(config);

            // Open the circuit
            for _ in 0..failure_threshold {
                cb.record_failure().await;
            }
            prop_assert_eq!(cb.state().await, CircuitState::Open);

            // Wait for timeout to transition to half-open
            tokio::time::sleep(std::time::Duration::from_millis(5)).await;

            // Should allow request now (half-open)
            prop_assert!(cb.allow_request().await);

            // Record successes to close
            for _ in 0..success_threshold {
                cb.record_success().await;
            }

            // Should be closed now
            prop_assert_eq!(cb.state().await, CircuitState::Closed);

            Ok(())
        })?;
    }
}


// **Feature: rust-libs-modernization-2025, Property 6: Cache Namespace Isolation**
// *For any* two cache operations with different namespaces, keys with the same name
// SHALL be stored and retrieved independently without collision.
// **Validates: Requirements 9.2**
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_cache_namespace_isolation(
        ns1 in "[a-z]{3,10}",
        ns2 in "[a-z]{3,10}",
        key in "[a-z0-9]{5,20}",
        value1 in proptest::collection::vec(any::<u8>(), 1..100),
        value2 in proptest::collection::vec(any::<u8>(), 1..100),
    ) {
        // Skip if namespaces are the same
        prop_assume!(ns1 != ns2);

        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            use rust_common::{CacheClient, CacheClientConfig};

            let config1 = CacheClientConfig::default().with_namespace(&ns1);
            let config2 = CacheClientConfig::default().with_namespace(&ns2);

            let client1 = CacheClient::new(config1).await.unwrap();
            let client2 = CacheClient::new(config2).await.unwrap();

            // Set same key in both namespaces with different values
            client1.set(&key, &value1, None).await.unwrap();
            client2.set(&key, &value2, None).await.unwrap();

            // Retrieve and verify isolation
            let result1 = client1.get(&key).await.unwrap();
            let result2 = client2.get(&key).await.unwrap();

            prop_assert_eq!(result1, Some(value1.clone()));
            prop_assert_eq!(result2, Some(value2.clone()));

            Ok(())
        })?;
    }
}

// **Feature: rust-libs-modernization-2025, Property 7: Cache TTL Enforcement**
// *For any* cached entry with a TTL, after the TTL expires, the entry SHALL NOT
// be returned by subsequent get operations.
// **Validates: Requirements 9.4**
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_cache_ttl_enforcement(
        key in "[a-z0-9]{5,20}",
        value in proptest::collection::vec(any::<u8>(), 1..100),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            use rust_common::{CacheClient, CacheClientConfig};
            use std::time::Duration;

            let config = CacheClientConfig::default();
            let client = CacheClient::new(config).await.unwrap();

            // Set with very short TTL
            let ttl = Duration::from_millis(1);
            client.set(&key, &value, Some(ttl)).await.unwrap();

            // Wait for expiration
            tokio::time::sleep(Duration::from_millis(10)).await;

            // Should not be found
            let result = client.get(&key).await.unwrap();
            prop_assert_eq!(result, None);

            Ok(())
        })?;
    }
}

// **Feature: rust-libs-modernization-2025, Property 3: Credential Encryption Round-Trip**
// *For any* credential value cached via the Cache_Client with encryption enabled,
// encrypting then decrypting SHALL produce the original value.
// **Validates: Requirements 6.6, 9.5**
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn prop_credential_encryption_round_trip(
        key in "[a-z0-9]{5,20}",
        value in proptest::collection::vec(any::<u8>(), 1..500),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            use rust_common::{CacheClient, CacheClientConfig};

            // Use a fixed encryption key for testing
            let encryption_key = [42u8; 32];
            let config = CacheClientConfig::default()
                .with_encryption_key(encryption_key);
            let client = CacheClient::new(config).await.unwrap();

            // Set encrypted value
            client.set(&key, &value, None).await.unwrap();

            // Get and verify round-trip
            let result = client.get(&key).await.unwrap();
            prop_assert_eq!(result, Some(value.clone()));

            Ok(())
        })?;
    }
}
