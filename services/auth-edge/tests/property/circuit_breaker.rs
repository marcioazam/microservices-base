//! Property tests for circuit breaker error type.
//!
//! **Feature: auth-edge-modernization-2025, Property 3: Circuit Breaker Error Type**
//! **Validates: Requirements 3.5**

use auth_edge::error::{AuthEdgeError, ErrorCode};
use proptest::prelude::*;
use rust_common::{CircuitBreaker, CircuitBreakerConfig, CircuitState, PlatformError};
use std::time::Duration;

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Feature: auth-edge-modernization-2025, Property 3: Circuit Breaker Error Type**
    /// **Validates: Requirements 3.5**
    ///
    /// *For any* circuit breaker that transitions to Open state, subsequent requests
    /// SHALL return `PlatformError::CircuitOpen` with the correct service name.
    #[test]
    fn circuit_open_returns_correct_error_type(
        service_name in "[a-zA-Z][a-zA-Z0-9-]{1,30}"
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config = CircuitBreakerConfig::default()
                .with_failure_threshold(2)
                .with_timeout(Duration::from_secs(60));
            
            let cb = CircuitBreaker::new(config);
            
            // Record failures to open the circuit
            cb.record_failure().await;
            cb.record_failure().await;
            
            // Verify circuit is open
            prop_assert_eq!(cb.state().await, CircuitState::Open);
            
            // Create the error that would be returned
            let error = PlatformError::circuit_open(&service_name);
            let auth_error = AuthEdgeError::Platform(error);
            
            // Verify error code is CircuitOpen
            prop_assert_eq!(auth_error.code(), ErrorCode::CircuitOpen);
            
            // Verify error message contains service name
            let error_string = format!("{}", auth_error);
            prop_assert!(
                error_string.contains(&service_name),
                "Error message '{}' should contain service name '{}'",
                error_string,
                service_name
            );
            
            Ok(())
        })?;
    }

    /// **Feature: auth-edge-modernization-2025, Property 3: Circuit Breaker Error Type**
    /// **Validates: Requirements 3.5**
    ///
    /// *For any* circuit breaker in Open state, allow_request() returns false.
    #[test]
    fn open_circuit_rejects_requests(failure_count in 2u32..10u32) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config = CircuitBreakerConfig::default()
                .with_failure_threshold(2)
                .with_timeout(Duration::from_secs(60));
            
            let cb = CircuitBreaker::new(config);
            
            // Record enough failures to open the circuit
            for _ in 0..failure_count {
                cb.record_failure().await;
            }
            
            // Verify circuit is open
            prop_assert_eq!(cb.state().await, CircuitState::Open);
            
            // Verify requests are rejected
            prop_assert!(!cb.allow_request().await);
            
            Ok(())
        })?;
    }

    /// **Feature: auth-edge-modernization-2025, Property 3: Circuit Breaker Error Type**
    /// **Validates: Requirements 3.5**
    ///
    /// *For any* circuit breaker, success resets failure count in Closed state.
    #[test]
    fn success_resets_failures_in_closed_state(
        initial_failures in 0u32..4u32
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config = CircuitBreakerConfig::default()
                .with_failure_threshold(5);
            
            let cb = CircuitBreaker::new(config);
            
            // Record some failures (but not enough to open)
            for _ in 0..initial_failures {
                cb.record_failure().await;
            }
            
            // Circuit should still be closed
            prop_assert_eq!(cb.state().await, CircuitState::Closed);
            
            // Record a success
            cb.record_success().await;
            
            // Failure count should be reset
            prop_assert_eq!(cb.failure_count(), 0);
            
            Ok(())
        })?;
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[tokio::test]
    async fn test_circuit_breaker_state_transitions() {
        let config = CircuitBreakerConfig {
            failure_threshold: 2,
            success_threshold: 2,
            timeout: Duration::from_millis(10),
            half_open_max_requests: 3,
        };
        
        let cb = CircuitBreaker::new(config);
        
        // Initial state is Closed
        assert_eq!(cb.state().await, CircuitState::Closed);
        assert!(cb.allow_request().await);
        
        // Record failures to open
        cb.record_failure().await;
        cb.record_failure().await;
        assert_eq!(cb.state().await, CircuitState::Open);
        assert!(!cb.allow_request().await);
        
        // Wait for timeout to transition to HalfOpen
        tokio::time::sleep(Duration::from_millis(20)).await;
        assert!(cb.allow_request().await);
        assert_eq!(cb.state().await, CircuitState::HalfOpen);
        
        // Record successes to close
        cb.record_success().await;
        cb.record_success().await;
        assert_eq!(cb.state().await, CircuitState::Closed);
    }

    #[test]
    fn test_circuit_open_error_contains_service_name() {
        let service = "my-service";
        let error = PlatformError::circuit_open(service);
        let auth_error = AuthEdgeError::Platform(error);
        
        assert_eq!(auth_error.code(), ErrorCode::CircuitOpen);
        assert!(format!("{}", auth_error).contains(service));
    }
}
