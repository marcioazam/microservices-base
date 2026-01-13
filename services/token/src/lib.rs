//! Token Service library.
//!
//! Provides JWT generation, DPoP validation, refresh token rotation,
//! JWKS publishing, and KMS integration.

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod config;
pub mod crypto;
pub mod dpop;
pub mod error;
pub mod jwks;
pub mod jwt;
pub mod kms;
pub mod metrics;
pub mod refresh;
pub mod storage;

// Re-exports for convenience
pub use config::Config;
pub use error::TokenError;
