//! Property-Based Tests Module
//!
//! Uses proptest for invariant verification.
//! Each test runs minimum 100 iterations.
//!
//! Test categories:
//! - error_sanitization: Sensitive data never leaks
//! - circuit_breaker: State machine correctness
//! - rate_limiter: Limit enforcement
//! - spiffe: ID parsing round-trip

pub mod generators;
pub mod error_sanitization;
pub mod circuit_breaker;
pub mod rate_limiter;
pub mod spiffe;
