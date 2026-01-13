//! Retry policy implementation with exponential backoff.
//!
//! This module provides a configurable retry mechanism for handling
//! transient failures in distributed systems.

use std::time::Duration;
use crate::PlatformError;

/// Retry policy configuration.
#[derive(Debug, Clone)]
pub struct RetryConfig {
    /// Maximum number of retry attempts
    pub max_retries: u32,
    /// Initial delay between retries
    pub initial_delay: Duration,
    /// Maximum delay between retries
    pub max_delay: Duration,
    /// Multiplier for exponential backoff
    pub multiplier: f64,
    /// Whether to add jitter to delays
    pub jitter: bool,
}

impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            max_retries: 3,
            initial_delay: Duration::from_millis(100),
            max_delay: Duration::from_secs(10),
            multiplier: 2.0,
            jitter: true,
        }
    }
}

impl RetryConfig {
    /// Create a new retry config with custom max retries.
    #[must_use]
    pub const fn with_max_retries(mut self, max_retries: u32) -> Self {
        self.max_retries = max_retries;
        self
    }

    /// Create a new retry config with custom initial delay.
    #[must_use]
    pub const fn with_initial_delay(mut self, delay: Duration) -> Self {
        self.initial_delay = delay;
        self
    }

    /// Create a new retry config with custom max delay.
    #[must_use]
    pub const fn with_max_delay(mut self, delay: Duration) -> Self {
        self.max_delay = delay;
        self
    }

    /// Create a new retry config without jitter.
    #[must_use]
    pub const fn without_jitter(mut self) -> Self {
        self.jitter = false;
        self
    }
}

/// Retry policy for executing operations with automatic retries.
#[derive(Debug, Clone)]
pub struct RetryPolicy {
    config: RetryConfig,
}

impl RetryPolicy {
    /// Create a new retry policy with the given configuration.
    #[must_use]
    pub const fn new(config: RetryConfig) -> Self {
        Self { config }
    }

    /// Create a retry policy with default configuration.
    #[must_use]
    pub fn with_defaults() -> Self {
        Self::new(RetryConfig::default())
    }

    /// Calculate the delay for a given attempt number.
    ///
    /// Uses exponential backoff with optional jitter.
    #[must_use]
    pub fn delay_for_attempt(&self, attempt: u32) -> Duration {
        let base_delay = self.config.initial_delay.as_millis() as f64
            * self.config.multiplier.powi(attempt as i32);

        let delay_ms = base_delay.min(self.config.max_delay.as_millis() as f64);

        let final_delay = if self.config.jitter {
            // Add up to 25% jitter
            let jitter_factor = 1.0 + (rand::random::<f64>() * 0.25);
            delay_ms * jitter_factor
        } else {
            delay_ms
        };

        Duration::from_millis(final_delay as u64)
    }

    /// Check if an error should be retried.
    #[must_use]
    pub fn should_retry(&self, error: &PlatformError, attempt: u32) -> bool {
        attempt < self.config.max_retries && error.is_retryable()
    }

    /// Execute an async operation with retries.
    ///
    /// # Errors
    ///
    /// Returns the last error if all retries are exhausted.
    pub async fn execute<F, Fut, T>(&self, mut operation: F) -> Result<T, PlatformError>
    where
        F: FnMut() -> Fut,
        Fut: std::future::Future<Output = Result<T, PlatformError>>,
    {
        let mut attempt = 0;
        loop {
            match operation().await {
                Ok(result) => return Ok(result),
                Err(error) => {
                    if !self.should_retry(&error, attempt) {
                        return Err(error);
                    }
                    let delay = self.delay_for_attempt(attempt);
                    tokio::time::sleep(delay).await;
                    attempt += 1;
                }
            }
        }
    }

    /// Get the maximum number of retries.
    #[must_use]
    pub const fn max_retries(&self) -> u32 {
        self.config.max_retries
    }
}

impl Default for RetryPolicy {
    fn default() -> Self {
        Self::with_defaults()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = RetryConfig::default();
        assert_eq!(config.max_retries, 3);
        assert_eq!(config.initial_delay, Duration::from_millis(100));
    }

    #[test]
    fn test_delay_calculation_no_jitter() {
        let config = RetryConfig::default().without_jitter();
        let policy = RetryPolicy::new(config);

        let delay0 = policy.delay_for_attempt(0);
        let delay1 = policy.delay_for_attempt(1);
        let delay2 = policy.delay_for_attempt(2);

        assert_eq!(delay0, Duration::from_millis(100));
        assert_eq!(delay1, Duration::from_millis(200));
        assert_eq!(delay2, Duration::from_millis(400));
    }

    #[test]
    fn test_max_delay_cap() {
        let config = RetryConfig::default()
            .without_jitter()
            .with_max_delay(Duration::from_millis(150));
        let policy = RetryPolicy::new(config);

        let delay2 = policy.delay_for_attempt(2);
        assert_eq!(delay2, Duration::from_millis(150));
    }

    #[test]
    fn test_should_retry() {
        let policy = RetryPolicy::with_defaults();

        // Retryable error within limit
        assert!(policy.should_retry(&PlatformError::RateLimited, 0));
        assert!(policy.should_retry(&PlatformError::RateLimited, 2));

        // Retryable error at limit
        assert!(!policy.should_retry(&PlatformError::RateLimited, 3));

        // Non-retryable error
        assert!(!policy.should_retry(&PlatformError::NotFound("test".to_string()), 0));
    }

    #[tokio::test]
    async fn test_execute_success() {
        let policy = RetryPolicy::with_defaults();
        let result: Result<i32, PlatformError> = policy.execute(|| async { Ok(42) }).await;
        assert_eq!(result.unwrap(), 42);
    }

    #[tokio::test]
    async fn test_execute_non_retryable_error() {
        let policy = RetryPolicy::with_defaults();
        let result: Result<i32, PlatformError> = policy
            .execute(|| async { Err(PlatformError::NotFound("test".to_string())) })
            .await;
        assert!(result.is_err());
    }
}
