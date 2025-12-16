//! Unit Tests Module
//!
//! Organized by domain following the test pyramid.
//! Each submodule focuses on a specific domain area.
//!
//! Structure:
//! - claims: JWT claims validation and scope handling
//! - circuit_breaker: Circuit breaker state transitions
//! - config: Configuration builder and validation
//! - error: Error sanitization and error codes
//! - grpc: gRPC service response building
//! - jwk_cache: JWK validation and key types
//! - metrics: Telemetry and metrics recording
//! - middleware: Rate limit headers and timeouts
//! - rate_limiter: Rate limiting and trust levels
//! - spiffe: SPIFFE ID parsing and validation
//! - token: JWT structure validation
//! - certificate: Certificate validity checks
//! - shutdown: Graceful shutdown configuration

pub mod certificate;
pub mod circuit_breaker;
pub mod claims;
pub mod config;
pub mod error;
pub mod grpc;
pub mod jwk_cache;
pub mod metrics;
pub mod middleware;
pub mod rate_limiter;
pub mod shutdown;
pub mod spiffe;
pub mod token;
