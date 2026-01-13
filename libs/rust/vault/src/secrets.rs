//! Secret types and Vault response structures.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// JWT signing key secret structure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JwtSigningKey {
    /// Signing algorithm
    pub algorithm: String,
    /// Key identifier
    pub key_id: String,
    /// Creation timestamp
    pub created_at: DateTime<Utc>,
    /// Private key (never serialized)
    #[serde(skip_serializing)]
    pub private_key: String,
    /// Public key
    pub public_key: String,
}

/// Service configuration secret.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceConfig {
    /// Configuration values
    #[serde(flatten)]
    pub values: std::collections::HashMap<String, serde_json::Value>,
}

/// Vault KV v2 response wrapper.
#[derive(Debug, Deserialize)]
pub struct KvResponse<T> {
    /// Response data
    pub data: KvData<T>,
    /// Lease ID
    pub lease_id: String,
    /// Lease duration in seconds
    pub lease_duration: u64,
    /// Whether lease is renewable
    pub renewable: bool,
}

/// KV data wrapper.
#[derive(Debug, Deserialize)]
pub struct KvData<T> {
    /// Actual secret data
    pub data: T,
    /// KV metadata
    pub metadata: KvMetadata,
}

/// KV metadata.
#[derive(Debug, Deserialize)]
pub struct KvMetadata {
    /// Creation time
    pub created_time: String,
    /// Deletion time (empty if not deleted)
    pub deletion_time: String,
    /// Whether secret is destroyed
    pub destroyed: bool,
    /// Version number
    pub version: u32,
}

/// Database credentials response.
#[derive(Debug, Deserialize)]
pub struct DatabaseCredsResponse {
    /// Credentials data
    pub data: DatabaseCredsData,
    /// Lease ID
    pub lease_id: String,
    /// Lease duration in seconds
    pub lease_duration: u64,
    /// Whether lease is renewable
    pub renewable: bool,
}

/// Database credentials data.
#[derive(Debug, Deserialize)]
pub struct DatabaseCredsData {
    /// Database username
    pub username: String,
    /// Database password
    pub password: String,
}

/// Vault auth response.
#[derive(Debug, Deserialize)]
pub struct AuthResponse {
    /// Auth data
    pub auth: AuthData,
}

/// Auth data.
#[derive(Debug, Deserialize)]
pub struct AuthData {
    /// Client token
    pub client_token: String,
    /// Token accessor
    pub accessor: String,
    /// Policies
    pub policies: Vec<String>,
    /// Token policies
    pub token_policies: Vec<String>,
    /// Lease duration in seconds
    pub lease_duration: u64,
    /// Whether token is renewable
    pub renewable: bool,
}
