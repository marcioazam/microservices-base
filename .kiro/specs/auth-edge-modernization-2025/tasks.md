# Implementation Plan: Auth Edge Service Modernization 2025

## Overview

This implementation plan modernizes the Auth Edge Service to December 2025 state-of-the-art standards. Tasks are organized to eliminate redundancies first, then integrate with platform services, and finally ensure production readiness with comprehensive testing.

## Tasks

- [x] 1. Update Dependencies to December 2025 Versions
  - Update Cargo.toml with latest stable versions
  - Update edition to 2024, rust-version to 1.85
  - Remove deprecated failsafe and unused borrow crates
  - Update workspace dependency references
  - _Requirements: 1.1-1.11_

- [x] 2. Centralize Error Handling via rust-common
  - [x] 2.1 Refactor error.rs to extend PlatformError
  - [x] 2.2 Write property test for error retryability classification
  - [x] 2.3 Implement error sanitization using rust-common
  - [x] 2.4 Write property test for sensitive data protection

- [x] 3. Checkpoint - Verify error handling compiles and tests pass

- [x] 4. Remove Local Circuit Breaker, Use rust-common
  - [x] 4.1 Update imports to use rust-common CircuitBreaker
  - [x] 4.2 Delete local circuit_breaker module
  - [x] 4.3 Write property test for circuit breaker error type

- [x] 5. Integrate JWK Cache with Cache_Service
  - [x] 5.1 Add CacheClient to JwkCache
  - [x] 5.2 Implement distributed cache with local fallback
  - [x] 5.3 Write property test for cache fallback behavior
  - [x] 5.4 Write property test for single-flight refresh

- [x] 6. Checkpoint - Verify cache integration works

- [x] 7. Integrate with Logging_Service
  - [x] 7.1 Create AuthEdgeLogger wrapper
  - [x] 7.2 Update gRPC service to use AuthEdgeLogger
  - [x] 7.3 Implement logging fallback to local tracing
  - [x] 7.4 Write property test for log level classification
  - [x] 7.5 Write property test for correlation ID propagation

- [x] 8. Eliminate Redundant Code
  - [x] 8.1 Remove duplicate has_claim method (centralized in Claims)
  - [x] 8.2 Consolidate SPIFFE validation (extract_from_certificate added)
  - [x] 8.3 Consolidate middleware stack (removed local circuit breaker)

- [x] 9. Checkpoint - Verify redundancy elimination

- [x] 10. Modernize Observability
  - [x] 10.1 Update OpenTelemetry to 0.27+
  - [x] 10.2 Implement W3C Trace Context propagation
  - [x] 10.3 Add span attributes for gRPC methods

- [x] 11. Preserve and Verify JWT Type-State Pattern
  - [x] 11.1 Verify type-state implementation is preserved
  - [x] 11.2 Write property test for JWT type-state transitions

- [x] 12. Modernize Configuration
  - [x] 12.1 Update Config with URL validation
  - [x] 12.2 Ensure environment variable overrides work
  - [x] 12.3 Write property test for configuration validation

- [x] 13. Checkpoint - Verify configuration and observability

- [x] 14. Implement Graceful Shutdown
  - [x] 14.1 Update shutdown coordinator
  - [x] 14.2 Add cleanup for LoggingClient and CacheClient
  - [x] 14.3 Write property test for graceful shutdown behavior

- [x] 15. Security Hardening
  - [x] 15.1 Verify constant-time signature comparison (jsonwebtoken uses it)
  - [x] 15.2 Implement algorithm confusion rejection
  - [x] 15.3 Enforce minimum key sizes
  - [x] 15.4 Configure rustls with secure cipher suites
  - [x] 15.5 Write property test for algorithm confusion rejection
  - [x] 15.6 Write property test for minimum key size enforcement

- [x] 16. SPIFFE Property Tests
  - [x] 16.1 Write property test for SPIFFE ID round-trip

- [x] 17. Final Checkpoint - All Tests Pass
  - Run full test suite
  - Verify 90%+ coverage
  - Ensure all property tests pass with 100 iterations

- [x] 18. Cleanup and Documentation
  - [x] 18.1 Remove any remaining dead code
  - [x] 18.2 Update README.md with new dependencies
  - [x] 18.3 Verify test organization

- [x] 19. Final Verification
  - Run cargo clippy -- -D warnings
  - Run cargo fmt --check
  - Run cargo test
  - Run cargo tarpaulin to verify 90%+ coverage

## Completed Property Tests

1. `error_retryability.rs` - Error retryability classification
2. `error_sanitization.rs` - Sensitive data protection
3. `circuit_breaker.rs` - Circuit breaker error types
4. `cache_fallback.rs` - Cache fallback and single-flight
5. `logging.rs` - Log level classification and correlation ID
6. `jwt_typestate.rs` - JWT type-state transitions
7. `config.rs` - Configuration validation
8. `shutdown.rs` - Graceful shutdown behavior
9. `security.rs` - Algorithm confusion and key size
10. `spiffe.rs` - SPIFFE ID round-trip

## Files Modified

### Source Files
- `Cargo.toml` - Updated dependencies to Dec 2025
- `src/main.rs` - Modernized with graceful shutdown
- `src/lib.rs` - Module exports
- `src/config.rs` - URL validation, OTLP endpoint
- `src/error.rs` - PlatformError integration
- `src/grpc/mod.rs` - rust-common CircuitBreaker, AuthEdgeLogger
- `src/jwt/claims.rs` - Centralized has_claim
- `src/jwt/token.rs` - Uses Claims::has_claim
- `src/jwt/validator.rs` - Removed duplicate has_claim
- `src/jwt/jwk_cache.rs` - CacheClient integration
- `src/mtls/spiffe.rs` - extract_from_certificate added
- `src/middleware/stack.rs` - Removed local circuit breaker
- `src/observability/mod.rs` - Export AuthEdgeLogger
- `src/observability/logging.rs` - AuthEdgeLogger wrapper
- `src/shutdown.rs` - Logger cleanup support

### Deleted Files
- `src/circuit_breaker/mod.rs`
- `src/circuit_breaker/state.rs`

### Test Files
- `tests/property/mod.rs`
- `tests/property/generators.rs`
- `tests/property/error_retryability.rs`
- `tests/property/error_sanitization.rs`
- `tests/property/circuit_breaker.rs`
- `tests/property/cache_fallback.rs`
- `tests/property/logging.rs`
- `tests/property/jwt_typestate.rs`
- `tests/property/config.rs`
- `tests/property/shutdown.rs`
- `tests/property/security.rs`
- `tests/property/spiffe.rs`
