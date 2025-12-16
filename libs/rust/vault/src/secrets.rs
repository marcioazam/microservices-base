//! Secret types and structures
//! Requirements: 1.1, 1.2, 1.4

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// JWT signing key secret structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JwtSigningKey {
    pub algorithm: String,
    pub key_id: String,
    pub created_at: DateTime<Utc>,
    #[serde(skip_serializing)]
    pub private_key: String,
    pub public_key: String,
}

/// Service configuration secret
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceConfig {
    #[serde(flatten)]
    pub values: std::collections::HashMap<String, serde_json::Value>,
}

/// Vault KV v2 response wrapper
#[derive(Debug, Deserialize)]
pub struct KvResponse<T> {
    pub data: KvData<T>,
    pub lease_id: String,
    pub lease_duration: u64,
    pub renewable: bool,
}

#[derive(Debug, Deserialize)]
pub struct KvData<T> {
    pub data: T,
    pub metadata: KvMetadata,
}

#[derive(Debug, Deserialize)]
pub struct KvMetadata {
    pub created_time: String,
    pub deletion_time: String,
    pub destroyed: bool,
    pub version: u32,
}

/// Vault database credentials response
#[derive(Debug, Deserialize)]
pub struct DatabaseCredsResponse {
    pub data: DatabaseCredsData,
    pub lease_id: String,
    pub lease_duration: u64,
    pub renewable: bool,
}

#[derive(Debug, Deserialize)]
pub struct DatabaseCredsData {
    pub username: String,
    pub password: String,
}

/// Vault PKI certificate response
#[derive(Debug, Deserialize)]
pub struct PkiCertResponse {
    pub data: PkiCertData,
    pub lease_id: String,
    pub lease_duration: u64,
}

#[derive(Debug, Deserialize)]
pub struct PkiCertData {
    pub certificate: String,
    pub issuing_ca: String,
    pub ca_chain: Vec<String>,
    pub private_key: String,
    pub private_key_type: String,
    pub serial_number: String,
}

/// Vault auth response
#[derive(Debug, Deserialize)]
pub struct AuthResponse {
    pub auth: AuthData,
}

#[derive(Debug, Deserialize)]
pub struct AuthData {
    pub client_token: String,
    pub accessor: String,
    pub policies: Vec<String>,
    pub token_policies: Vec<String>,
    pub lease_duration: u64,
    pub renewable: bool,
}

/// Audit log entry structure (for testing)
/// Requirements: 1.4 - Audit logging
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLogEntry {
    pub time: DateTime<Utc>,
    pub r#type: String,
    pub auth: AuditAuth,
    pub request: AuditRequest,
    pub response: Option<AuditResponse>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditAuth {
    pub accessor: String,
    pub client_token: String,
    pub display_name: String,
    pub policies: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditRequest {
    pub id: String,
    pub operation: String,
    pub path: String,
    pub remote_address: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditResponse {
    pub data: Option<serde_json::Value>,
}
