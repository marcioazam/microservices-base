//! Shared test utilities for auth-platform Rust libraries.
//!
//! This crate provides:
//! - Proptest generators for all domain types
//! - Mock implementations for service clients
//! - Test fixtures with sample data

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod generators;
pub mod mocks;
pub mod fixtures;

pub use generators::*;
