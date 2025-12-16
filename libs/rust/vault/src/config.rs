//! Vault client configuration
//! Requirements: 1.1, 1.5

use std::time::Duration;

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
    /// Grace period for cached credentials when Vault unavailable (Requirements 1.5)
    pub grace_period: Duration,
    /// Renewal threshold (percentage of TTL remaining to trigger renewal)
    pub renewal_threshold: f64,
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
            grace_period: Duration::from_secs(300), // 5 minutes - Requirements 1.5
            renewal_threshold: 0.2, // Renew at 20% remaining TTL - Requirements 1.3
        }
    }
}

impl VaultConfig {
    pub fn new(addr: impl Into<String>, role: impl Into<String>) -> Self {
        Self {
            addr: addr.into(),
            role: role.into(),
            ..Default::default()
        }
    }

    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    pub fn with_grace_period(mut self, grace_period: Duration) -> Self {
        self.grace_period = grace_period;
        self
    }

    pub fn with_renewal_threshold(mut self, threshold: f64) -> Self {
        self.renewal_threshold = threshold.clamp(0.1, 0.5);
        self
    }
}
