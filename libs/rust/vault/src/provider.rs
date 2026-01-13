//! Secret provider traits using native async (Rust 2024).
//!
//! Provides generic traits for secret retrieval abstraction.

use serde::de::DeserializeOwned;
use std::time::Duration;

/// Metadata about a retrieved secret.
#[derive(Debug, Clone)]
pub struct SecretMetadata {
    /// Lease ID for renewable secrets
    pub lease_id: Option<String>,
    /// Time-to-live for the secret
    pub ttl: Duration,
    /// Whether the secret is renewable
    pub renewable: bool,
    /// Version number (for KV v2)
    pub version: Option<u32>,
}

/// Generic trait for secret providers with type-safe retrieval.
///
/// Uses native async traits (Rust 2024) - no async-trait macro needed.
pub trait SecretProvider: Send + Sync {
    /// Error type for this provider
    type Error: std::error::Error + Send + Sync;

    /// Get a secret and deserialize to type T.
    fn get_secret<T>(
        &self,
        path: &str,
    ) -> impl std::future::Future<Output = Result<(T, SecretMetadata), Self::Error>> + Send
    where
        T: DeserializeOwned + Send;

    /// Get a specific version of a secret (KV v2).
    fn get_secret_version<T>(
        &self,
        path: &str,
        version: u32,
    ) -> impl std::future::Future<Output = Result<(T, SecretMetadata), Self::Error>> + Send
    where
        T: DeserializeOwned + Send;

    /// Renew a lease.
    fn renew_lease(
        &self,
        lease_id: &str,
        increment: Duration,
    ) -> impl std::future::Future<Output = Result<Duration, Self::Error>> + Send;

    /// Revoke a lease.
    fn revoke_lease(
        &self,
        lease_id: &str,
    ) -> impl std::future::Future<Output = Result<(), Self::Error>> + Send;
}

/// Trait for database credential providers.
pub trait DatabaseCredentialProvider: Send + Sync {
    /// Error type for this provider
    type Error: std::error::Error + Send + Sync;

    /// Get dynamic database credentials.
    fn get_credentials(
        &self,
        role: &str,
    ) -> impl std::future::Future<Output = Result<DatabaseCredentials, Self::Error>> + Send;
}

/// Database credentials with lease information.
#[derive(Debug, Clone)]
pub struct DatabaseCredentials {
    /// Database username
    pub username: String,
    /// Database password (protected)
    pub password: secrecy::SecretString,
    /// Lease ID for renewal
    pub lease_id: String,
    /// Time-to-live
    pub ttl: Duration,
    /// Whether credentials are renewable
    pub renewable: bool,
}

impl DatabaseCredentials {
    /// Check if credentials should be renewed (at 80% of TTL).
    #[must_use]
    pub fn should_renew(&self, elapsed: Duration) -> bool {
        let threshold = self.ttl.as_secs_f64() * 0.8;
        elapsed.as_secs_f64() >= threshold
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use secrecy::SecretString;

    #[test]
    fn test_should_renew() {
        let creds = DatabaseCredentials {
            username: "test".to_string(),
            password: SecretString::from("pass".to_string()),
            lease_id: "lease-123".to_string(),
            ttl: Duration::from_secs(3600),
            renewable: true,
        };

        // At 70% elapsed, should not renew
        assert!(!creds.should_renew(Duration::from_secs(2520)));

        // At 85% elapsed, should renew
        assert!(creds.should_renew(Duration::from_secs(3060)));
    }
}
