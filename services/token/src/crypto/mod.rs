//! Crypto Service integration module.
//!
//! Provides client for centralized cryptographic operations via Crypto Service,
//! with circuit breaker resilience and local fallback capabilities.

pub mod client;
pub mod config;
pub mod encryptor;
pub mod error;
pub mod factory;
pub mod fallback;
pub mod metrics;
pub mod models;
pub mod signer;

// Re-exports
pub use client::{CryptoClient, CryptoClientCore};
pub use config::CryptoClientConfig;
pub use encryptor::CryptoEncryptor;
pub use error::CryptoError;
pub use factory::CryptoClientFactory;
pub use fallback::FallbackHandler;
pub use models::{EncryptResult, EncryptedData, KeyId, KeyMetadata, KeyState, SignResult};
pub use signer::CryptoSigner;

/// Generated protobuf types for Crypto Service
pub mod proto {
    tonic::include_proto!("crypto.v1");
}
