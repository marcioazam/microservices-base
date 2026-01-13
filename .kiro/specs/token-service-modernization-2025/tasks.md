# Implementation Plan: Token Service Modernization 2025

## Overview

This implementation plan modernizes the Token Service to state-of-the-art December 2025 standards, integrating with platform shared libraries, eliminating redundancy, and ensuring production-ready quality with comprehensive property-based testing.

## Tasks

- [x] 1. Update project configuration and dependencies
  - [x] 1.1 Update Cargo.toml to use workspace dependencies from libs/rust/Cargo.toml
    - Change edition to "2024" and rust-version to "1.85"
    - Replace direct dependencies with workspace references
    - Add rust-common as path dependency
    - Update tonic to 0.12, prost to 0.13, thiserror to 2.0, jsonwebtoken to 9.3, proptest to 1.5
    - Remove async-trait dependency (use native async traits)
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.8_

  - [x] 1.2 Update build.rs for tonic-build 0.12
    - Update proto compilation configuration
    - _Requirements: 1.5_

- [x] 2. Implement centralized error handling
  - [x] 2.1 Refactor src/error.rs to extend rust-common::PlatformError
    - Define TokenError enum with all variants
    - Implement is_retryable() method
    - Implement From<TokenError> for tonic::Status
    - Ensure no internal details exposed in gRPC responses
    - _Requirements: 3.1, 3.2, 3.4, 3.6_

  - [x] 2.2 Write property test for error classification and mapping
    - **Property 12: Error Classification and Mapping**
    - **Validates: Requirements 3.2, 3.4, 3.6**

- [-] 3. Implement centralized configuration
  - [x] 3.1 Refactor src/config.rs with unified Config struct
    - Include CacheClientConfig, LoggingClientConfig, CircuitBreakerConfig
    - Add all JWT, KMS, DPoP, and security settings
    - Implement from_env() with validation
    - _Requirements: 10.5_

- [x] 4. Integrate platform cache service
  - [x] 4.1 Replace Redis direct access with rust-common::CacheClient
    - Remove redis crate direct usage
    - Create CacheClient with namespace "token"
    - Configure encryption key for cached data
    - Implement local fallback when Cache_Service unavailable
    - _Requirements: 2.1, 2.7, 12.5_

  - [x] 4.2 Write property test for cache encryption round-trip
    - **Property 13: Cache Encryption Round-Trip**
    - **Validates: Requirements 12.5**

- [x] 5. Integrate platform logging service
  - [x] 5.1 Replace direct tracing with rust-common::LoggingClient
    - Create LoggingClient with service_id "token-service"
    - Add correlation_id, trace_id, span_id to all log entries
    - Ensure sensitive data is not logged
    - _Requirements: 2.2, 9.1, 9.2, 12.3_

- [x] 6. Checkpoint - Verify platform integration
  - Ensure all tests pass, ask the user if questions arise.
  - Verify CacheClient and LoggingClient are properly integrated

- [x] 7. Refactor JWT module
  - [x] 7.1 Update src/jwt/claims.rs with complete Claims struct
    - Include all standard claims (iss, sub, aud, exp, iat, nbf, jti)
    - Add DPoP binding support (cnf.jkt)
    - Add custom claims support via HashMap
    - _Requirements: 4.2, 4.3_

  - [x] 7.2 Update src/jwt/signer.rs with trait-based abstraction
    - Define JwtSigner trait with async sign method (native async trait)
    - Include algorithm() and key_id() methods
    - _Requirements: 4.4, 4.5, 10.4_

  - [x] 7.3 Update src/jwt/builder.rs for JWT generation
    - Implement JwtBuilder with Claims
    - Include kid in JWT header
    - Sign using KMS provider
    - _Requirements: 4.1, 4.5_

  - [x] 7.4 Write property test for JWT round-trip consistency
    - **Property 1: JWT Claims Round-Trip Consistency**
    - **Validates: Requirements 4.6**

  - [x] 7.5 Write property test for JWT structure completeness
    - **Property 2: JWT Structure Completeness**
    - **Validates: Requirements 4.2, 4.3, 4.5**

- [x] 8. Refactor DPoP module
  - [x] 8.1 Update src/dpop/validator.rs with CacheClient integration
    - Use CacheClient for jti tracking instead of direct Redis
    - Implement all RFC 9449 validation rules
    - Add clock skew validation (60 seconds)
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_

  - [x] 8.2 Update src/dpop/thumbprint.rs with constant-time comparison
    - Use subtle::ConstantTimeEq for thumbprint verification
    - Implement RFC 7638 canonical JSON for EC and RSA keys
    - _Requirements: 5.9, 12.1_

  - [x] 8.3 Write property test for DPoP validation comprehensive
    - **Property 3: DPoP Validation Comprehensive**
    - **Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5, 5.6**

  - [x] 8.4 Write property test for JWK thumbprint determinism
    - **Property 4: JWK Thumbprint Determinism**
    - **Validates: Requirements 5.10**

  - [x] 8.5 Write property test for DPoP replay detection
    - **Property 5: DPoP Replay Detection**
    - **Validates: Requirements 5.7, 5.11**

  - [x] 8.6 Write property test for DPoP token binding
    - **Property 6: DPoP Token Binding**
    - **Validates: Requirements 5.8, 5.9**

- [x] 9. Checkpoint - Verify JWT and DPoP modules
  - Ensure all tests pass, ask the user if questions arise.
  - Verify property tests run with 100+ iterations

- [x] 10. Refactor refresh token module
  - [x] 10.1 Update src/refresh/family.rs with TokenFamily struct
    - Include family_id, user_id, session_id, current_token_hash
    - Add rotation_count, revoked, revoked_at fields
    - Implement rotate(), revoke(), is_valid_token(), is_replay() methods
    - _Requirements: 6.1, 6.4_

  - [x] 10.2 Update src/refresh/rotator.rs with CacheClient and LoggingClient
    - Use CacheClient for token family storage
    - Use LoggingClient for security events
    - Implement create_family(), rotate(), revoke_family() methods
    - Detect replay attacks and revoke entire family
    - _Requirements: 6.2, 6.3, 6.5, 6.6, 6.7_

  - [x] 10.3 Write property test for refresh token rotation invalidation
    - **Property 7: Refresh Token Rotation Invalidation**
    - **Validates: Requirements 6.2, 6.6**

  - [x] 10.4 Write property test for refresh token replay detection
    - **Property 8: Refresh Token Replay Detection**
    - **Validates: Requirements 6.3**

  - [x] 10.5 Write property test for token family uniqueness
    - **Property 9: Token Family Uniqueness**
    - **Validates: Requirements 6.1**

  - [x] 10.6 Write property test for token family revocation
    - **Property 10: Token Family Revocation**
    - **Validates: Requirements 6.7**

- [x] 11. Refactor KMS module
  - [x] 11.1 Update src/kms/mod.rs with trait and factory
    - Define KmsSigner trait with async sign method
    - Implement KmsFactory for provider selection
    - Add CircuitBreaker integration
    - _Requirements: 8.1, 8.2, 8.3, 10.4_

  - [x] 11.2 Update src/kms/aws.rs with circuit breaker
    - Wrap AWS KMS calls with CircuitBreaker
    - Implement fallback signing when enabled
    - Log all KMS failures as security events
    - Map KMS algorithms to JWT algorithms
    - _Requirements: 8.3, 8.4, 8.5, 8.6_

  - [x] 11.3 Update src/kms/mock.rs for testing
    - Implement MockKmsSigner for development
    - Support ES256 and RS256 algorithms
    - _Requirements: 8.2_

- [x] 12. Refactor JWKS module
  - [x] 12.1 Update src/jwks/publisher.rs with key rotation support
    - Include both current and previous keys during rotation
    - Format keys per RFC 7517
    - Retain previous key for configurable period
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [x] 12.2 Write property test for JWKS key rotation
    - **Property 11: JWKS Key Rotation**
    - **Validates: Requirements 7.2, 7.4, 7.5**

- [x] 13. Checkpoint - Verify all modules
  - Ensure all tests pass, ask the user if questions arise.
  - Verify all property tests pass with 100+ iterations

- [x] 14. Update gRPC service implementation
  - [x] 14.1 Refactor src/grpc/mod.rs with all integrations
    - Wire JWT, DPoP, Refresh, JWKS, KMS modules
    - Add correlation ID to all requests
    - Implement rate limiting for token endpoints
    - Add metrics tracking for all operations
    - _Requirements: 9.4, 9.5, 12.6_

  - [x] 14.2 Update src/main.rs with graceful shutdown
    - Initialize CacheClient, LoggingClient, CircuitBreaker
    - Configure OpenTelemetry tracing
    - Implement graceful shutdown handling
    - _Requirements: 9.6_

- [x] 15. Implement observability
  - [x] 15.1 Add Prometheus metrics
    - tokens_issued_total, tokens_refreshed_total, tokens_revoked_total
    - dpop_validations_total
    - Latency histograms for all gRPC methods
    - _Requirements: 9.3, 9.4, 9.5_

  - [x] 15.2 Write property test for observability completeness
    - **Property 14: Observability Completeness**
    - **Validates: Requirements 9.2, 9.4, 9.5**

- [x] 16. Final validation and cleanup
  - [x] 16.1 Verify file size limits
    - Ensure no file exceeds 400 lines
    - Split large files if necessary
    - _Requirements: 10.2_

  - [x] 16.2 Verify test organization
    - Ensure src/ and tests/ are properly separated
    - Verify all property tests reference design document property numbers
    - _Requirements: 10.1, 11.7_

  - [x] 16.3 Remove legacy code
    - Remove direct Redis access code
    - Remove async-trait usage
    - Remove deprecated patterns
    - _Requirements: 1.3, 2.1_

- [x] 17. Final checkpoint - Production readiness
  - Ensure all tests pass, ask the user if questions arise.
  - Run `cargo test --workspace` to verify all tests pass
  - Run `cargo clippy` to verify no warnings
  - Verify all 14 property tests pass with 100+ iterations

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- All property tests must run minimum 100 iterations
- All property tests must reference design document property numbers
