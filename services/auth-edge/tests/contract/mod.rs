//! Contract Tests Module (Pact Consumer)
//!
//! Consumer-driven contract tests for service dependencies.
//! Uses Pact framework to verify API contracts.
//!
//! Providers tested:
//! - token-service: Token issuance and refresh
//! - session-identity-core: Session management
//! - iam-policy-service: Authorization checks

pub mod token_service;
pub mod session_service;
pub mod iam_service;
