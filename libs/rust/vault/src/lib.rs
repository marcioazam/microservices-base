//! HashiCorp Vault Client for Auth Platform
//! 
//! Provides type-safe secret retrieval with automatic renewal support.
//! Requirements: 1.1, 1.2, 1.3, 1.4

pub mod client;
pub mod config;
pub mod error;
pub mod secrets;
pub mod provider;

pub use client::VaultClient;
pub use config::VaultConfig;
pub use error::{VaultError, VaultResult};
pub use provider::SecretProvider;
