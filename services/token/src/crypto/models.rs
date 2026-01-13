//! Data models for CryptoClient operations.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Key identifier matching Crypto Service proto.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct KeyId {
    /// Namespace for key isolation
    pub namespace: String,
    /// Unique key identifier
    pub id: String,
    /// Key version
    pub version: u32,
}

impl KeyId {
    /// Create a new KeyId.
    #[must_use]
    pub fn new(namespace: impl Into<String>, id: impl Into<String>, version: u32) -> Self {
        Self {
            namespace: namespace.into(),
            id: id.into(),
            version,
        }
    }

    /// Create KeyId from proto message.
    #[must_use]
    pub fn from_proto(proto: &super::proto::KeyId) -> Self {
        Self {
            namespace: proto.namespace.clone(),
            id: proto.id.clone(),
            version: proto.version,
        }
    }

    /// Convert to proto message.
    #[must_use]
    pub fn to_proto(&self) -> super::proto::KeyId {
        super::proto::KeyId {
            namespace: self.namespace.clone(),
            id: self.id.clone(),
            version: self.version,
        }
    }
}

/// Key state from Crypto Service.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum KeyState {
    /// Key state is unknown
    Unspecified,
    /// Key is pending activation
    PendingActivation,
    /// Key is active and can be used
    Active,
    /// Key is deprecated (can verify but not sign)
    Deprecated,
    /// Key is pending destruction
    PendingDestruction,
    /// Key is destroyed
    Destroyed,
}

impl KeyState {
    /// Check if key can be used for signing.
    #[must_use]
    pub fn can_sign(&self) -> bool {
        matches!(self, KeyState::Active)
    }

    /// Check if key can be used for verification.
    #[must_use]
    pub fn can_verify(&self) -> bool {
        matches!(self, KeyState::Active | KeyState::Deprecated)
    }

    /// Create from proto enum value.
    #[must_use]
    pub fn from_proto(value: i32) -> Self {
        match value {
            1 => KeyState::PendingActivation,
            2 => KeyState::Active,
            3 => KeyState::Deprecated,
            4 => KeyState::PendingDestruction,
            5 => KeyState::Destroyed,
            _ => KeyState::Unspecified,
        }
    }
}

/// Key algorithm.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum KeyAlgorithm {
    Unspecified,
    Aes128Gcm,
    Aes256Gcm,
    Aes128Cbc,
    Aes256Cbc,
    Rsa2048,
    Rsa3072,
    Rsa4096,
    EcdsaP256,
    EcdsaP384,
    EcdsaP521,
}

impl KeyAlgorithm {
    /// Create from proto enum value.
    #[must_use]
    pub fn from_proto(value: i32) -> Self {
        match value {
            1 => KeyAlgorithm::Aes128Gcm,
            2 => KeyAlgorithm::Aes256Gcm,
            3 => KeyAlgorithm::Aes128Cbc,
            4 => KeyAlgorithm::Aes256Cbc,
            5 => KeyAlgorithm::Rsa2048,
            6 => KeyAlgorithm::Rsa3072,
            7 => KeyAlgorithm::Rsa4096,
            8 => KeyAlgorithm::EcdsaP256,
            9 => KeyAlgorithm::EcdsaP384,
            10 => KeyAlgorithm::EcdsaP521,
            _ => KeyAlgorithm::Unspecified,
        }
    }

    /// Convert to proto enum value.
    #[must_use]
    pub fn to_proto(&self) -> i32 {
        match self {
            KeyAlgorithm::Unspecified => 0,
            KeyAlgorithm::Aes128Gcm => 1,
            KeyAlgorithm::Aes256Gcm => 2,
            KeyAlgorithm::Aes128Cbc => 3,
            KeyAlgorithm::Aes256Cbc => 4,
            KeyAlgorithm::Rsa2048 => 5,
            KeyAlgorithm::Rsa3072 => 6,
            KeyAlgorithm::Rsa4096 => 7,
            KeyAlgorithm::EcdsaP256 => 8,
            KeyAlgorithm::EcdsaP384 => 9,
            KeyAlgorithm::EcdsaP521 => 10,
        }
    }

    /// Get JWT algorithm string.
    #[must_use]
    pub fn to_jwt_algorithm(&self) -> Option<&'static str> {
        match self {
            KeyAlgorithm::Rsa2048 | KeyAlgorithm::Rsa3072 | KeyAlgorithm::Rsa4096 => Some("PS256"),
            KeyAlgorithm::EcdsaP256 => Some("ES256"),
            KeyAlgorithm::EcdsaP384 => Some("ES384"),
            KeyAlgorithm::EcdsaP521 => Some("ES512"),
            _ => None,
        }
    }
}

/// Key metadata from Crypto Service.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyMetadata {
    pub id: KeyId,
    pub algorithm: KeyAlgorithm,
    pub state: KeyState,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
    pub rotated_at: Option<DateTime<Utc>>,
    pub previous_version: Option<KeyId>,
    pub owner_service: String,
    pub allowed_operations: Vec<String>,
    pub usage_count: u64,
}

impl KeyMetadata {
    /// Create from proto message.
    #[must_use]
    pub fn from_proto(proto: &super::proto::KeyMetadata) -> Self {
        Self {
            id: proto.id.as_ref().map(KeyId::from_proto).unwrap_or_else(|| {
                KeyId::new("", "", 0)
            }),
            algorithm: KeyAlgorithm::from_proto(proto.algorithm),
            state: KeyState::from_proto(proto.state),
            created_at: DateTime::from_timestamp(proto.created_at, 0)
                .unwrap_or_else(Utc::now),
            expires_at: if proto.expires_at > 0 {
                DateTime::from_timestamp(proto.expires_at, 0)
            } else {
                None
            },
            rotated_at: if proto.rotated_at > 0 {
                DateTime::from_timestamp(proto.rotated_at, 0)
            } else {
                None
            },
            previous_version: proto.previous_version.as_ref().map(KeyId::from_proto),
            owner_service: proto.owner_service.clone(),
            allowed_operations: proto.allowed_operations.clone(),
            usage_count: proto.usage_count,
        }
    }
}

/// Result of a signing operation.
#[derive(Debug, Clone)]
pub struct SignResult {
    /// Signature bytes
    pub signature: Vec<u8>,
    /// Key ID used for signing
    pub key_id: KeyId,
    /// Algorithm used
    pub algorithm: String,
}

/// Result of an encryption operation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptResult {
    /// Ciphertext
    pub ciphertext: Vec<u8>,
    /// Initialization vector
    pub iv: Vec<u8>,
    /// Authentication tag
    pub tag: Vec<u8>,
    /// Key ID used
    pub key_id: KeyId,
    /// Algorithm used
    pub algorithm: String,
}

impl EncryptResult {
    /// Serialize to bytes for storage.
    pub fn to_bytes(&self) -> Result<Vec<u8>, super::CryptoError> {
        serde_json::to_vec(self)
            .map_err(|e| super::CryptoError::internal(format!("Serialization failed: {}", e)))
    }
}

/// Encrypted data for storage/transmission.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptedData {
    pub ciphertext: Vec<u8>,
    pub iv: Vec<u8>,
    pub tag: Vec<u8>,
}

impl EncryptedData {
    /// Create from EncryptResult.
    #[must_use]
    pub fn from_result(result: &EncryptResult) -> Self {
        Self {
            ciphertext: result.ciphertext.clone(),
            iv: result.iv.clone(),
            tag: result.tag.clone(),
        }
    }

    /// Deserialize from bytes.
    pub fn from_bytes(data: &[u8]) -> Result<Self, super::CryptoError> {
        serde_json::from_slice(data)
            .map_err(|e| super::CryptoError::internal(format!("Deserialization failed: {}", e)))
    }
}

/// Key rotation result.
#[derive(Debug, Clone)]
pub struct KeyRotationResult {
    pub new_key_id: KeyId,
    pub old_key_id: KeyId,
    pub metadata: KeyMetadata,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_key_id_creation() {
        let key_id = KeyId::new("token", "signing-key", 1);
        assert_eq!(key_id.namespace, "token");
        assert_eq!(key_id.id, "signing-key");
        assert_eq!(key_id.version, 1);
    }

    #[test]
    fn test_key_state_can_sign() {
        assert!(KeyState::Active.can_sign());
        assert!(!KeyState::Deprecated.can_sign());
        assert!(!KeyState::Destroyed.can_sign());
    }

    #[test]
    fn test_key_state_can_verify() {
        assert!(KeyState::Active.can_verify());
        assert!(KeyState::Deprecated.can_verify());
        assert!(!KeyState::Destroyed.can_verify());
    }

    #[test]
    fn test_algorithm_to_jwt() {
        assert_eq!(KeyAlgorithm::EcdsaP256.to_jwt_algorithm(), Some("ES256"));
        assert_eq!(KeyAlgorithm::Rsa2048.to_jwt_algorithm(), Some("PS256"));
        assert_eq!(KeyAlgorithm::Aes256Gcm.to_jwt_algorithm(), None);
    }
}
