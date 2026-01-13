//! Vault client configuration.

use std::time::Duration;

/// Vault client configuration.
#[derive(Debug, Clone)]
pub struct VaultConfig {
    /// Vault server address
    pub addr: String,
    /// Kubernetes auth role name
    pub role: String,
    /// Service account token path
    pub token_path: String,
    /// Request timeout
    pub timeout: Duration,
    /// Maximum retry attempts
    pub max_retries: u32,
    /// Base retry delay
    pub retry_delay: Duration,
    /// Grace period for cached credentials when Vault unavailable
    pub grace_period: Duration,
    /// Renewal threshold (percentage of TTL remaining to trigger renewal)
    pub renewal_threshold: f64,
    /// Circuit breaker failure threshold
    pub circuit_breaker_threshold: u32,
    /// Circuit breaker reset timeout
    pub circuit_breaker_timeout: Duration,
}

impl Default for VaultConfig {
    fn default() -> Self {
        Self {
            addr: std::env::var("VAULT_ADDR")
                .unwrap_or_else(|_| "https://vault.vault.svc:8200".to_string()),
            role: std::env::var("VAULT_ROLE").unwrap_or_default(),
            token_path: "/var/run/secrets/kubernetes.io/serviceaccount/token".to_string(),
            timeout: Duration::from_secs(30),
            max_retries: 3,
            retry_delay: Duration::from_millis(100),
            grace_period: Duration::from_secs(300),
            renewal_threshold: 0.2,
            circuit_breaker_threshold: 5,
            circuit_breaker_timeout: Duration::from_secs(30),
        }
    }
}

impl VaultConfig {
    /// Create a new configuration.
    #[must_use]
    pub fn new(addr: impl Into<String>, role: impl Into<String>) -> Self {
        Self {
            addr: addr.into(),
            role: role.into(),
            ..Default::default()
        }
    }

    /// Set request timeout.
    #[must_use]
    pub const fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Set grace period.
    #[must_use]
    pub const fn with_grace_period(mut self, grace_period: Duration) -> Self {
        self.grace_period = grace_period;
        self
    }

    /// Set renewal threshold (clamped to 0.1-0.5).
    #[must_use]
    pub fn with_renewal_threshold(mut self, threshold: f64) -> Self {
        self.renewal_threshold = threshold.clamp(0.1, 0.5);
        self
    }

    /// Set circuit breaker threshold.
    #[must_use]
    pub const fn with_circuit_breaker_threshold(mut self, threshold: u32) -> Self {
        self.circuit_breaker_threshold = threshold;
        self
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = VaultConfig::default();
        assert_eq!(config.grace_period, Duration::from_secs(300));
        assert!((config.renewal_threshold - 0.2).abs() < f64::EPSILON);
        assert_eq!(config.circuit_breaker_threshold, 5);
    }

    #[test]
    fn test_renewal_threshold_clamping() {
        let config = VaultConfig::default().with_renewal_threshold(0.05);
        assert!((config.renewal_threshold - 0.1).abs() < f64::EPSILON);

        let config = VaultConfig::default().with_renewal_threshold(0.8);
        assert!((config.renewal_threshold - 0.5).abs() < f64::EPSILON);
    }
}
