//! Pact contract testing types.
//!
//! Provides types for consumer-driven contract testing with Pact.
//!
//! # December 2025 Modernization
//! - Rust 2024 edition
//! - Complete type exports for contract testing
//! - Property-based testing support

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod contract;
pub mod matrix;
pub mod verification;

pub use contract::{Contract, ContractMetadata, Interaction, PactSpecification, Participant, Request, Response};
pub use matrix::{CanIDeployResult, MatrixEntry};
pub use verification::{ContractVersion, VerificationResult};
