//! Circuit breaker implementation for resilience.
//!
//! This module provides a circuit breaker pattern implementation to protect
//! services from cascading failures when downstream dependencies are unavailable.

use std::sync::atomic::{AtomicU32, Ordering};
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

/// Circuit breaker state.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CircuitState {
    /// Circuit is closed, requests are allowed
    Closed,
    /// Circuit is open, requests are rejected
    Open,
    /// Circuit is half-open, limited requests are allowed to test recovery
    HalfOpen,
}

/// Circuit breaker configuration.
#[derive(Debug, Clone)]
pub struct CircuitBreakerConfig {
    /// Number of consecutive failures before opening the circuit
    pub failure_threshold: u32,
    /// Number of consecutive successes in half-open state to close the circuit
    pub success_threshold: u32,
    /// Time to wait before transitioning from open to half-open
    pub timeout: Duration,
    /// Maximum requests allowed in half-open state
    pub half_open_max_requests: u32,
}

impl Default for CircuitBreakerConfig {
    fn default() -> Self {
        Self {
            failure_threshold: 5,
            success_threshold: 2,
            timeout: Duration::from_secs(30),
            half_open_max_requests: 3,
        }
    }
}

impl CircuitBreakerConfig {
    /// Create a new config with custom failure threshold.
    #[must_use]
    pub const fn with_failure_threshold(mut self, threshold: u32) -> Self {
        self.failure_threshold = threshold;
        self
    }

    /// Create a new config with custom success threshold.
    #[must_use]
    pub const fn with_success_threshold(mut self, threshold: u32) -> Self {
        self.success_threshold = threshold;
        self
    }

    /// Create a new config with custom timeout.
    #[must_use]
    pub const fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }
}

/// Circuit breaker for protecting external services.
///
/// Implements the circuit breaker pattern with three states:
/// - Closed: Normal operation, requests are allowed
/// - Open: Failure threshold exceeded, requests are rejected
/// - Half-Open: Testing recovery, limited requests allowed
pub struct CircuitBreaker {
    config: CircuitBreakerConfig,
    state: RwLock<CircuitState>,
    failures: AtomicU32,
    successes: AtomicU32,
    last_failure: RwLock<Option<Instant>>,
    half_open_requests: AtomicU32,
}

impl CircuitBreaker {
    /// Create a new circuit breaker with the given configuration.
    #[must_use]
    pub fn new(config: CircuitBreakerConfig) -> Self {
        Self {
            config,
            state: RwLock::new(CircuitState::Closed),
            failures: AtomicU32::new(0),
            successes: AtomicU32::new(0),
            last_failure: RwLock::new(None),
            half_open_requests: AtomicU32::new(0),
        }
    }

    /// Create a circuit breaker with default configuration.
    #[must_use]
    pub fn with_defaults() -> Self {
        Self::new(CircuitBreakerConfig::default())
    }

    /// Check if a request is allowed.
    ///
    /// Returns `true` if the request should proceed, `false` if it should be rejected.
    pub async fn allow_request(&self) -> bool {
        let state = *self.state.read().await;
        match state {
            CircuitState::Closed => true,
            CircuitState::Open => {
                // Check if timeout has elapsed
                if let Some(last) = *self.last_failure.read().await {
                    if last.elapsed() >= self.config.timeout {
                        // Transition to half-open
                        *self.state.write().await = CircuitState::HalfOpen;
                        self.half_open_requests.store(0, Ordering::SeqCst);
                        self.successes.store(0, Ordering::SeqCst);
                        true
                    } else {
                        false
                    }
                } else {
                    false
                }
            }
            CircuitState::HalfOpen => {
                // Allow limited requests in half-open state
                let current = self.half_open_requests.fetch_add(1, Ordering::SeqCst);
                current < self.config.half_open_max_requests
            }
        }
    }

    /// Record a successful request.
    ///
    /// In half-open state, consecutive successes will close the circuit.
    pub async fn record_success(&self) {
        let state = *self.state.read().await;
        match state {
            CircuitState::HalfOpen => {
                let successes = self.successes.fetch_add(1, Ordering::SeqCst) + 1;
                if successes >= self.config.success_threshold {
                    // Close the circuit
                    *self.state.write().await = CircuitState::Closed;
                    self.failures.store(0, Ordering::SeqCst);
                    self.successes.store(0, Ordering::SeqCst);
                }
            }
            CircuitState::Closed => {
                // Reset failure count on success
                self.failures.store(0, Ordering::SeqCst);
            }
            CircuitState::Open => {
                // Shouldn't happen, but ignore
            }
        }
    }

    /// Record a failed request.
    ///
    /// Consecutive failures will open the circuit.
    pub async fn record_failure(&self) {
        let failures = self.failures.fetch_add(1, Ordering::SeqCst) + 1;
        *self.last_failure.write().await = Some(Instant::now());

        let state = *self.state.read().await;
        match state {
            CircuitState::Closed | CircuitState::HalfOpen => {
                if failures >= self.config.failure_threshold {
                    *self.state.write().await = CircuitState::Open;
                    self.successes.store(0, Ordering::SeqCst);
                }
            }
            CircuitState::Open => {
                // Already open, nothing to do
            }
        }
    }

    /// Get the current circuit state.
    pub async fn state(&self) -> CircuitState {
        *self.state.read().await
    }

    /// Get the current failure count.
    #[must_use]
    pub fn failure_count(&self) -> u32 {
        self.failures.load(Ordering::SeqCst)
    }

    /// Reset the circuit breaker to closed state.
    pub async fn reset(&self) {
        *self.state.write().await = CircuitState::Closed;
        self.failures.store(0, Ordering::SeqCst);
        self.successes.store(0, Ordering::SeqCst);
        self.half_open_requests.store(0, Ordering::SeqCst);
        *self.last_failure.write().await = None;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_initial_state_closed() {
        let cb = CircuitBreaker::with_defaults();
        assert_eq!(cb.state().await, CircuitState::Closed);
        assert!(cb.allow_request().await);
    }

    #[tokio::test]
    async fn test_opens_after_failures() {
        let config = CircuitBreakerConfig::default()
            .with_failure_threshold(3);
        let cb = CircuitBreaker::new(config);

        for _ in 0..3 {
            cb.record_failure().await;
        }

        assert_eq!(cb.state().await, CircuitState::Open);
        assert!(!cb.allow_request().await);
    }

    #[tokio::test]
    async fn test_success_resets_failures() {
        let config = CircuitBreakerConfig::default()
            .with_failure_threshold(3);
        let cb = CircuitBreaker::new(config);

        cb.record_failure().await;
        cb.record_failure().await;
        cb.record_success().await;

        assert_eq!(cb.failure_count(), 0);
        assert_eq!(cb.state().await, CircuitState::Closed);
    }

    #[tokio::test]
    async fn test_half_open_transition() {
        let config = CircuitBreakerConfig {
            failure_threshold: 2,
            success_threshold: 1,
            timeout: Duration::from_millis(1),
            half_open_max_requests: 3,
        };
        let cb = CircuitBreaker::new(config);

        // Open the circuit
        cb.record_failure().await;
        cb.record_failure().await;
        assert_eq!(cb.state().await, CircuitState::Open);

        // Wait for timeout
        tokio::time::sleep(Duration::from_millis(5)).await;

        // Should transition to half-open
        assert!(cb.allow_request().await);
        assert_eq!(cb.state().await, CircuitState::HalfOpen);
    }

    #[tokio::test]
    async fn test_closes_after_successes_in_half_open() {
        let config = CircuitBreakerConfig {
            failure_threshold: 2,
            success_threshold: 2,
            timeout: Duration::from_millis(1),
            half_open_max_requests: 5,
        };
        let cb = CircuitBreaker::new(config);

        // Open the circuit
        cb.record_failure().await;
        cb.record_failure().await;

        // Wait for timeout and transition to half-open
        tokio::time::sleep(Duration::from_millis(5)).await;
        cb.allow_request().await;

        // Record successes
        cb.record_success().await;
        cb.record_success().await;

        assert_eq!(cb.state().await, CircuitState::Closed);
    }

    #[tokio::test]
    async fn test_reset() {
        let cb = CircuitBreaker::with_defaults();

        cb.record_failure().await;
        cb.record_failure().await;

        cb.reset().await;

        assert_eq!(cb.state().await, CircuitState::Closed);
        assert_eq!(cb.failure_count(), 0);
    }
}
