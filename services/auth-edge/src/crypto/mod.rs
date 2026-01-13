//! Crypto-Service Integration Module
//!
//! This module provides integration with the centralized crypto-service
//! for encryption, decryption, and key management operations.

pub mod cache_integration;
pub mod client;
pub mod config;
pub mod error;
pub mod fallback;
pub mod key_manager;
pub mod logging;
pub mod metrics;

#[cfg(test)]
mod tests;

// Re-exports for convenience
pub use cache_integration::EncryptedCacheClient;
pub use client::CryptoClient;
pub use config::CryptoClientConfig;
pub use error::CryptoError;
pub use fallback::FallbackHandler;
pub use key_manager::{KeyId, KeyManager, KeyMetadata};
pub use logging::{log_crypto_error, log_crypto_fallback, log_crypto_operation, log_key_rotation};
pub use metrics::CryptoMetrics;

/// Generated gRPC client code from crypto_service.proto
pub mod proto {
    tonic::include_proto!("crypto.v1");
}
