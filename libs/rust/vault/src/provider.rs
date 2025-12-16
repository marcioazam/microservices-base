//! Generic Secret Provider trait
//! Requirements: 13.1 - Generic traits for secret retrieval abstraction

use async_trait::async_trait;
use serde::de::DeserializeOwned;
use std::time::Duration;

/// Metadata about a retrieved secret
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

/// Generic trait for secret providers with type-safe retrieval
/// Requirements: 13.1 - Generic traits for secret retrieval abstraction
#[async_trait]
pub trait SecretProvider: Send + Sync {
    type Error: std::error::Error + Send + Sync;

    /// Get a secret and deserialize to type T
    async fn get_secret<T>(&self, path: &str) -> Result<(T, SecretMetadata), Self::Error>
    where
        T: DeserializeOwned + Send;

    /// Get a specific version of a secret (KV v2)
    async fn get_secret_version<T>(
        &self,
        path: &str,
        version: u32,
    ) -> Result<(T, SecretMetadata), Self::Error>
    where
        T: DeserializeOwned + Send;

    /// Renew a lease
    async fn renew_lease(&self, lease_id: &str, increment: Duration) -> Result<Duration, Self::Error>;

    /// Revoke a lease
    async fn revoke_lease(&self, lease_id: &str) -> Result<(), Self::Error>;
}

/// Trait for database credential providers
/// Requirements: 1.2 - Dynamic database credentials
#[async_trait]
pub trait DatabaseCredentialProvider: Send + Sync {
    type Error: std::error::Error + Send + Sync;

    /// Get dynamic database credentials
    async fn get_credentials(&self, role: &str) -> Result<DatabaseCredentials, Self::Error>;
}

/// Database credentials with lease information
#[derive(Debug, Clone)]
pub struct DatabaseCredentials {
    pub username: String,
    pub password: secrecy::SecretString,
    pub lease_id: String,
    pub ttl: Duration,
    pub renewable: bool,
}

impl DatabaseCredentials {
    /// Check if credentials should be renewed (at 80% of TTL)
    /// Requirements: 1.3 - Automatic renewal at 80% TTL
    pub fn should_renew(&self, elapsed: Duration) -> bool {
        let threshold = self.ttl.as_secs_f64() * 0.8;
        elapsed.as_secs_f64() >= threshold
    }
}
