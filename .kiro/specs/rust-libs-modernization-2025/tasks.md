# Implementation Plan: Rust Libraries Modernization 2025

## Overview

This implementation plan transforms the `libs/rust` directory into a state-of-the-art December 2025 Rust workspace with centralized dependencies, zero redundancy, platform service integration, and comprehensive property-based testing.

## Tasks

- [x] 1. Create workspace root and common crate structure
  - [x] 1.1 Create root Cargo.toml with workspace configuration
    - Define workspace members, Rust 2024 edition, dependency inheritance
    - Configure workspace-level lints and clippy settings
    - _Requirements: 1.1, 1.2, 1.4, 1.5_
  - [x] 1.2 Create rust-common crate skeleton
    - Create Cargo.toml inheriting workspace dependencies
    - Create src/lib.rs with module declarations
    - _Requirements: 7.1_
  - [x] 1.3 Create test-utils crate skeleton
    - Create Cargo.toml with proptest 1.7 dependency
    - Create src/lib.rs with module declarations
    - _Requirements: 10.7_

- [x] 2. Implement rust-common core modules
  - [x] 2.1 Implement centralized error types
    - Create src/error.rs with PlatformError enum
    - Implement is_retryable() method
    - Add From implementations for common error types
    - _Requirements: 7.1, 4.1_
  - [x] 2.2 Write property test for error retryability
    - **Property 14: Input Validation Rejection**
    - **Validates: Requirements 15.3**
  - [x] 2.3 Implement HTTP client builder
    - Create src/http.rs with HttpConfig and build_http_client
    - Configure rustls-tls, connection pooling, timeouts
    - _Requirements: 7.2, 4.2_
  - [x] 2.4 Implement retry policy
    - Create src/retry.rs with exponential backoff
    - Support configurable max retries and delays
    - _Requirements: 7.3_
  - [x] 2.5 Implement circuit breaker
    - Create src/circuit_breaker.rs with CircuitBreaker struct
    - Implement state transitions (Closed, Open, HalfOpen)
    - _Requirements: 7.4_
  - [x] 2.6 Write property test for circuit breaker
    - **Property 2: Circuit Breaker State Transitions**
    - **Validates: Requirements 6.5, 8.3**

- [x] 3. Checkpoint - Verify rust-common compiles
  - All tests pass.

- [x] 4. Implement platform service clients
  - [x] 4.1 Implement Logging_Service gRPC client
    - Create src/logging_client.rs with LoggingClient struct
    - Implement batching with configurable batch size
    - Implement circuit breaker for service unavailability
    - Implement fallback to local tracing
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.6_
  - [x] 4.2 Write property test for log batching
    - **Property 4: Log Batching Threshold**
    - **Validates: Requirements 8.2**
  - [x] 4.3 Write property test for log context propagation
    - **Property 5: Log Context Propagation**
    - **Validates: Requirements 8.5**
  - [x] 4.4 Implement Cache_Service gRPC client
    - Create src/cache_client.rs with CacheClient struct
    - Implement namespace-based key isolation
    - Implement local fallback cache
    - Implement AES-GCM encryption for secrets
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_
  - [x] 4.5 Write property test for cache namespace isolation
    - **Property 6: Cache Namespace Isolation**
    - **Validates: Requirements 9.2**
  - [x] 4.6 Write property test for cache TTL enforcement
    - **Property 7: Cache TTL Enforcement**
    - **Validates: Requirements 9.4**
  - [x] 4.7 Write property test for credential encryption round-trip
    - **Property 3: Credential Encryption Round-Trip**
    - **Validates: Requirements 6.6, 9.5**


- [x] 5. Checkpoint - Verify platform clients work
  - All tests pass.

- [x] 6. Implement shared test utilities
  - [x] 6.1 Create shared proptest generators
    - Create src/generators.rs with all domain type generators
    - Include caep_event_type_strategy, subject_identifier_strategy
    - Include spiffe_identity_strategy, traceparent_strategy
    - Include secret_path_strategy, ttl_strategy
    - _Requirements: 10.3, 10.7_
  - [x] 6.2 Create mock implementations
    - Create src/mocks.rs with mock service clients
    - Include MockLoggingClient, MockCacheClient
    - _Requirements: 10.7_
  - [x] 6.3 Create test fixtures
    - Create src/fixtures.rs with sample data
    - Include sample events, credentials, contracts
    - _Requirements: 10.7_

- [x] 7. Modernize CAEP library
  - [x] 7.1 Update CAEP Cargo.toml
    - Inherit workspace dependencies
    - Add rust-common dependency
    - Remove async-trait where possible
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.9, 5.1_
  - [x] 7.2 Modernize CAEP error types
    - Update src/error.rs to use thiserror 2.0
    - Add From<PlatformError> implementation
    - _Requirements: 5.2_
  - [x] 7.3 Modernize CAEP event types
    - Update src/event.rs with const fn for URIs
    - Ensure serde derives are complete
    - _Requirements: 5.1_
  - [x] 7.4 Integrate Logging_Service client
    - Add structured logging throughout CAEP
    - Include correlation ID in all log entries
    - _Requirements: 5.4_
  - [x] 7.5 Integrate Cache_Service for JWKS
    - Cache JWKS keys via Cache_Service client
    - Implement cache invalidation on key rotation
    - _Requirements: 5.5_
  - [x] 7.6 Update SET signing to default ES256
    - Ensure SecurityEventToken::sign defaults to ES256
    - _Requirements: 5.6_
  - [x] 7.7 Write property test for SET signing algorithm
    - **Property 1: SET Signing Algorithm Default**
    - **Validates: Requirements 5.6**
  - [x] 7.8 Write property test for serialization round-trip
    - **Property 8: Serialization Round-Trip**
    - **Validates: Requirements 10.4**

- [x] 8. Checkpoint - Verify CAEP library works
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Modernize Vault client
  - [x] 9.1 Update Vault Cargo.toml
    - Inherit workspace dependencies
    - Add rust-common dependency
    - Update secrecy to 0.10 with zeroize
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.8, 2.9, 6.4_
  - [x] 9.2 Modernize Vault error types
    - Update src/error.rs to use thiserror 2.0
    - Add From<PlatformError> implementation
    - _Requirements: 6.1_
  - [x] 9.3 Update SecretProvider to native async trait
    - Remove async-trait macro usage
    - Use Rust 2024 native async trait syntax
    - _Requirements: 6.1_
  - [x] 9.4 Integrate Logging_Service client
    - Add structured logging for all Vault operations
    - Never log secret values
    - _Requirements: 6.2_
  - [x] 9.5 Integrate Cache_Service for credentials
    - Cache credentials via Cache_Service client
    - Encrypt cached credentials with AES-GCM
    - _Requirements: 6.3, 6.6_
  - [x] 9.6 Implement circuit breaker for Vault
    - Use rust-common CircuitBreaker
    - Configure appropriate thresholds
    - _Requirements: 6.5_
  - [x] 9.7 Write property test for secret non-exposure
    - **Property 15: Secret Non-Exposure in Debug Output**
    - **Validates: Requirements 15.6**

- [x] 10. Checkpoint - Verify Vault client works
  - Ensure all tests pass, ask the user if questions arise.


- [x] 11. Modernize Linkerd library
  - [x] 11.1 Create Linkerd crate structure
    - Create Cargo.toml inheriting workspace dependencies
    - Create src/lib.rs with module declarations
    - Move types from tests to src/
    - _Requirements: 3.1, 3.5_
  - [x] 11.2 Implement Linkerd types
    - Create src/mtls.rs with MtlsConnection type
    - Create src/trace.rs with TraceContext type
    - Create src/metrics.rs with LinkerdMetrics type
    - _Requirements: 11.1, 11.2_
  - [x] 11.3 Write property test for mTLS validity
    - **Property 9: mTLS Connection Validity**
    - **Validates: Requirements 11.1**
  - [x] 11.4 Write property test for trace context propagation
    - **Property 10: Trace Context Propagation**
    - **Validates: Requirements 11.2**
  - [x] 11.5 Write property test for latency overhead
    - **Property 11: Linkerd Latency Overhead**
    - **Validates: Requirements 11.3**

- [x] 12. Modernize Pact library
  - [x] 12.1 Create Pact crate structure
    - Create Cargo.toml inheriting workspace dependencies
    - Create src/lib.rs with module declarations
    - Move types from tests to src/
    - _Requirements: 3.1, 3.5_
  - [x] 12.2 Implement Pact types
    - Create src/contract.rs with Contract, Interaction types
    - Create src/verification.rs with VerificationResult type
    - Create src/matrix.rs with CanIDeployResult type
    - _Requirements: 12.1, 12.2_
  - [x] 12.3 Write property test for contract serialization
    - **Property 12: Contract Serialization Round-Trip**
    - **Validates: Requirements 12.1**
  - [x] 12.4 Write property test for version git commit match
    - **Property 13: Contract Version Git Commit Match**
    - **Validates: Requirements 12.2**

- [x] 13. Checkpoint - Verify Linkerd and Pact libraries work
  - Ensure all tests pass, ask the user if questions arise.

- [x] 14. Modernize integration tests
  - [x] 14.1 Update integration crate structure
    - Create Cargo.toml inheriting workspace dependencies
    - Add dependencies on all library crates
    - _Requirements: 13.1_
  - [x] 14.2 Implement Vault through mesh tests
    - Test secret retrieval through Linkerd mesh
    - Verify mTLS is active
    - _Requirements: 13.1, 13.2_
  - [x] 14.3 Implement secret rotation tests
    - Test rotation continuity without errors
    - Verify zero error rate increase
    - _Requirements: 13.3_
  - [x] 14.4 Implement Logging_Service integration tests
    - Test log delivery through mesh
    - Verify correlation ID propagation
    - _Requirements: 13.4_
  - [x] 14.5 Implement Cache_Service integration tests
    - Test cache operations through mesh
    - Verify namespace isolation
    - _Requirements: 13.5_

- [x] 15. Cleanup and documentation
  - [x] 15.1 Remove redundant code
    - Delete duplicate error handling patterns
    - Delete duplicate HTTP client configurations
    - Delete duplicate test utilities
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_
  - [x] 15.2 Update README files
    - Create libs/rust/README.md with workspace overview
    - Update each crate's README.md
    - _Requirements: 14.1_
  - [x] 15.3 Add rustdoc comments
    - Document all public APIs
    - Include usage examples in doc comments
    - _Requirements: 14.2, 14.3_
  - [x] 15.4 Create CHANGELOG.md
    - Document all changes from modernization
    - Include migration notes
    - _Requirements: 14.4_

- [x] 16. Final checkpoint - Full test suite
  - Run `cargo test --workspace`
  - Run `cargo clippy --workspace`
  - Run `cargo doc --workspace`
  - Note: Initial build requires dependency download (~3-5 min first time)

## Notes

- All tasks are required including property tests for comprehensive testing
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (15 properties total)
- Unit tests validate specific examples and edge cases
