//! Circuit Breaker State Management
//!
//! Shared state logic used by both Tower and Legacy implementations.

use std::time::{Duration, Instant};
use tracing::{info, warn};

/// Circuit breaker states
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CircuitState {
    /// Circuit is closed, requests flow through normally
    Closed,
    /// Circuit is open, requests fail fast
    Open,
    /// Circuit is testing if service recovered
    HalfOpen,
}

/// Internal state for the circuit breaker
#[derive(Debug)]
pub struct CircuitBreakerState {
    pub state: CircuitState,
    pub failure_count: u32,
    pub success_count: u32,
    pub last_failure_time: Option<Instant>,
}

impl CircuitBreakerState {
    pub fn new() -> Self {
        Self {
            state: CircuitState::Closed,
            failure_count: 0,
            success_count: 0,
            last_failure_time: None,
        }
    }

    /// Check if circuit should transition from Open to HalfOpen
    pub fn check_timeout_transition(&mut self, timeout: Duration, name: &str) -> bool {
        if self.state != CircuitState::Open {
            return self.state != CircuitState::Open;
        }

        if let Some(last_failure) = self.last_failure_time {
            if last_failure.elapsed() >= timeout {
                self.state = CircuitState::HalfOpen;
                self.success_count = 0;
                info!(circuit = %name, "Circuit transitioning to half-open");
                return true;
            }
        }
        false
    }

    /// Record a successful request
    pub fn record_success(&mut self, success_threshold: u32, name: &str) {
        match self.state {
            CircuitState::HalfOpen => {
                self.success_count += 1;
                if self.success_count >= success_threshold {
                    self.state = CircuitState::Closed;
                    self.failure_count = 0;
                    self.success_count = 0;
                    info!(circuit = %name, "Circuit closed after recovery");
                }
            }
            CircuitState::Closed => {
                self.failure_count = 0;
            }
            _ => {}
        }
    }

    /// Record a failed request
    pub fn record_failure(&mut self, failure_threshold: u32, name: &str) {
        match self.state {
            CircuitState::Closed => {
                self.failure_count += 1;
                if self.failure_count >= failure_threshold {
                    self.state = CircuitState::Open;
                    self.last_failure_time = Some(Instant::now());
                    warn!(
                        circuit = %name,
                        failures = self.failure_count,
                        "Circuit opened due to failures"
                    );
                }
            }
            CircuitState::HalfOpen => {
                self.state = CircuitState::Open;
                self.last_failure_time = Some(Instant::now());
                self.success_count = 0;
                warn!(circuit = %name, "Circuit re-opened from half-open");
            }
            _ => {}
        }
    }

    /// Check if circuit allows requests
    pub fn is_available(&mut self, timeout: Duration, name: &str) -> bool {
        match self.state {
            CircuitState::Closed | CircuitState::HalfOpen => true,
            CircuitState::Open => self.check_timeout_transition(timeout, name),
        }
    }
}

impl Default for CircuitBreakerState {
    fn default() -> Self {
        Self::new()
    }
}
