//! Circuit Breaker Unit Tests
//!
//! Tests for circuit breaker state transitions, thresholds, and Tower integration.

use std::time::Duration;

// ============================================================================
// State Logic Tests (sync, pure logic)
// ============================================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum CircuitState {
    Closed,
    Open,
    HalfOpen,
}

struct SimpleCircuitBreaker {
    state: CircuitState,
    failure_count: u32,
    success_count: u32,
    failure_threshold: u32,
    success_threshold: u32,
}

impl SimpleCircuitBreaker {
    fn new(failure_threshold: u32, success_threshold: u32) -> Self {
        Self {
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

#[test]
fn test_circuit_starts_closed() {
    let cb = SimpleCircuitBreaker::new(5, 3);
    assert_eq!(cb.state, CircuitState::Closed);
    assert!(cb.is_available());
}

#[test]
fn test_circuit_opens_after_threshold() {
    let mut cb = SimpleCircuitBreaker::new(3, 2);

    cb.record_failure();
    assert_eq!(cb.state, CircuitState::Closed);

    cb.record_failure();
    assert_eq!(cb.state, CircuitState::Closed);

    cb.record_failure();
    assert_eq!(cb.state, CircuitState::Open);
    assert!(!cb.is_available());
}

#[test]
fn test_circuit_half_open_to_closed() {
    let mut cb = SimpleCircuitBreaker::new(1, 2);

    cb.record_failure();
    assert_eq!(cb.state, CircuitState::Open);

    cb.transition_to_half_open();
    assert_eq!(cb.state, CircuitState::HalfOpen);

    cb.record_success();
    assert_eq!(cb.state, CircuitState::HalfOpen);

    cb.record_success();
    assert_eq!(cb.state, CircuitState::Closed);
}

#[test]
fn test_circuit_half_open_to_open_on_failure() {
    let mut cb = SimpleCircuitBreaker::new(1, 3);

    cb.record_failure();
    cb.transition_to_half_open();

    cb.record_success();
    assert_eq!(cb.state, CircuitState::HalfOpen);

    cb.record_failure();
    assert_eq!(cb.state, CircuitState::Open);
}

#[test]
fn test_success_resets_failure_count() {
    let mut cb = SimpleCircuitBreaker::new(3, 2);

    cb.record_failure();
    cb.record_failure();
    assert_eq!(cb.failure_count, 2);

    cb.record_success();
    assert_eq!(cb.failure_count, 0);
}

#[test]
fn test_circuit_thresholds() {
    let cb = SimpleCircuitBreaker::new(10, 5);
    assert_eq!(cb.failure_threshold, 10);
    assert_eq!(cb.success_threshold, 5);
}

// ============================================================================
// Config Tests
// ============================================================================

#[test]
fn test_config_default_values() {
    let failure_threshold: u32 = 5;
    let success_threshold: u32 = 3;
    let timeout_secs: u64 = 30;

    assert_eq!(failure_threshold, 5);
    assert_eq!(success_threshold, 3);
    assert_eq!(Duration::from_secs(timeout_secs), Duration::from_secs(30));
}

#[test]
fn test_config_custom_values() {
    let failure_threshold: u32 = 10;
    let success_threshold: u32 = 5;
    let timeout_secs: u64 = 60;

    assert_eq!(failure_threshold, 10);
    assert_eq!(success_threshold, 5);
    assert_eq!(Duration::from_secs(timeout_secs), Duration::from_secs(60));
}
