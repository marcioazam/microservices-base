//! Circuit Breaker Property Tests
//!
//! Validates state machine correctness.

use proptest::prelude::*;

#[derive(Debug, Clone, PartialEq)]
enum CircuitState { Closed, Open, HalfOpen }

struct CircuitBreaker {
    state: CircuitState,
    failure_count: u32,
    success_count: u32,
    failure_threshold: u32,
    success_threshold: u32,
}

impl CircuitBreaker {
    fn new(failure_threshold: u32, success_threshold: u32) -> Self {
        CircuitBreaker {
            state: CircuitState::Closed,
            failure_count: 0,
            success_count: 0,
            failure_threshold,
            success_threshold,
        }
    }

    fn record_failure(&mut self) {
        match self.state {
            CircuitState::Closed => {
                self.failure_count += 1;
                if self.failure_count >= self.failure_threshold {
                    self.state = CircuitState::Open;
                }
            }
            CircuitState::HalfOpen => {
                self.state = CircuitState::Open;
                self.success_count = 0;
            }
            _ => {}
        }
    }

    fn record_success(&mut self) {
        match self.state {
            CircuitState::HalfOpen => {
                self.success_count += 1;
                if self.success_count >= self.success_threshold {
                    self.state = CircuitState::Closed;
                    self.failure_count = 0;
                    self.success_count = 0;
                }
            }
            CircuitState::Closed => {
                self.failure_count = 0;
            }
            _ => {}
        }
    }

    fn transition_to_half_open(&mut self) {
        if self.state == CircuitState::Open {
            self.state = CircuitState::HalfOpen;
            self.success_count = 0;
        }
    }

    fn is_available(&self) -> bool {
        self.state != CircuitState::Open
    }
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property: Circuit opens after failure threshold
    #[test]
    fn prop_circuit_open_fail_fast(failure_threshold in 1u32..10u32) {
        let mut cb = CircuitBreaker::new(failure_threshold, 3);
        
        for _ in 0..failure_threshold {
            cb.record_failure();
        }
        
        prop_assert_eq!(cb.state, CircuitState::Open);
        prop_assert!(!cb.is_available(), "Open circuit should not be available");
    }

    /// Property: Circuit breaker state machine transitions correctly
    #[test]
    fn prop_circuit_breaker_state_machine(
        failure_threshold in 1u32..10u32,
        success_threshold in 1u32..5u32,
        num_failures in 0u32..20u32,
    ) {
        let mut cb = CircuitBreaker::new(failure_threshold, success_threshold);
        
        prop_assert_eq!(cb.state, CircuitState::Closed);
        
        for i in 0..num_failures {
            cb.record_failure();
            if i + 1 >= failure_threshold {
                prop_assert_eq!(cb.state, CircuitState::Open);
            }
        }
        
        if cb.state == CircuitState::Open {
            cb.transition_to_half_open();
            prop_assert_eq!(cb.state, CircuitState::HalfOpen);
            
            for _ in 0..success_threshold {
                cb.record_success();
            }
            prop_assert_eq!(cb.state, CircuitState::Closed);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_circuit_breaker_opens() {
        let mut cb = CircuitBreaker::new(3, 2);
        
        cb.record_failure();
        cb.record_failure();
        assert!(cb.is_available());
        
        cb.record_failure();
        assert!(!cb.is_available());
        assert_eq!(cb.state, CircuitState::Open);
    }

    #[test]
    fn test_circuit_breaker_closes() {
        let mut cb = CircuitBreaker::new(1, 2);
        
        cb.record_failure();
        assert_eq!(cb.state, CircuitState::Open);
        
        cb.transition_to_half_open();
        assert_eq!(cb.state, CircuitState::HalfOpen);
        
        cb.record_success();
        cb.record_success();
        assert_eq!(cb.state, CircuitState::Closed);
    }
}
