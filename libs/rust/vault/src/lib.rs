//! HashiCorp Vault Client for Auth Platform
//!
//! Provides type-safe secret retrieval with automatic renewal support,
//! circuit breaker resilience, and platform service integration.
//!
//! # December 2025 Modernization
//! - Native async traits (Rust 2024)
//! - secrecy 0.10 with zeroize
//! - Circuit breaker integration
//! - Logging and cache service integration

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod client;
pub mod config;
pub mod error;
pub mod provider;
pub mod secrets;

pub use client::VaultClient;
pub use config::VaultConfig;
pub use error::{VaultError, VaultResult};
pub use provider::{DatabaseCredentialProvider, DatabaseCredentials, SecretMetadata, SecretProvider};
