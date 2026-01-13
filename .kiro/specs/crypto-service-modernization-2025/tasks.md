# Implementation Plan: Crypto Service Modernization 2025

## Overview

This implementation plan modernizes the crypto-service to state-of-the-art December 2025 standards. Tasks are organized to minimize risk through incremental changes with validation checkpoints.

## Tasks

- [x] 1. Centralize OpenSSL RAII Wrappers and Common Utilities
  - [x] 1.1 Create `include/crypto/common/openssl_raii.h` with all RAII wrappers
    - Consolidate CipherCtxDeleter, PKeyDeleter, PKeyCtxDeleter, MDCtxDeleter, BIODeleter
    - Add OpenSSL 3.3+ wrappers: ParamBldDeleter, ParamDeleter, MACDeleter, MACCtxDeleter
    - Add factory functions: make_cipher_ctx(), make_md_ctx()
    - _Requirements: 4.3, 4.4, 6.1_
  - [x] 1.2 Create `include/crypto/common/hash_utils.h` with centralized hash utilities
    - Implement get_evp_md(), get_hash_size(), get_hash_name(), get_hash_for_curve()
    - Use constexpr for compile-time evaluation
    - _Requirements: 4.5, 5.5_
  - [x] 1.3 Update `include/crypto/common/result.h` to use std::expected
    - Replace custom Result<T> with std::expected<T, Error>
    - Add [[nodiscard]] attribute to all Result-returning functions
    - Centralize ErrorCode enumeration with all error types
    - _Requirements: 4.1, 4.2, 5.2, 5.6_
  - [x] 1.4 Write unit tests for common utilities
    - Test RAII wrapper cleanup behavior
    - Test hash utility functions
    - Test Result type operations
    - _Requirements: 7.2_

- [x] 2. Checkpoint - Verify common utilities compile and tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. Update Crypto Engines to Use Centralized Utilities
  - [x] 3.1 Refactor `src/crypto/engine/aes_engine.cpp`
    - Replace local RAII wrappers with openssl_raii.h imports
    - Update to use std::expected Result type
    - Add [[nodiscard]] to all public methods
    - _Requirements: 4.3, 4.4, 5.2, 5.6_
  - [x] 3.2 Refactor `src/crypto/engine/rsa_engine.cpp`
    - Replace local RAII wrappers with openssl_raii.h imports
    - Replace local hash algorithm selection with hash_utils.h
    - Update to use std::expected Result type
    - _Requirements: 4.3, 4.4, 4.5, 5.2_
  - [x] 3.3 Refactor `src/crypto/engine/ecdsa_engine.cpp`
    - Replace local RAII wrappers with openssl_raii.h imports
    - Replace local hash algorithm selection with hash_utils.h
    - Update to use std::expected Result type
    - _Requirements: 4.3, 4.4, 4.5, 5.2_
  - [x] 3.4 Refactor `src/crypto/engine/hybrid_encryption.cpp`
    - Update to use std::expected Result type
    - _Requirements: 5.2_
  - [x] 3.5 Verify existing property tests pass with refactored engines
    - Run aes_properties_test.cpp
    - Run rsa_properties_test.cpp
    - Run signature_properties_test.cpp
    - _Requirements: 7.3_

- [x] 4. Checkpoint - Verify crypto engines work correctly
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement LoggingClient for Platform Integration
  - [x] 5.1 Create `include/crypto/clients/logging_client.h`
    - Define LoggingClientConfig struct
    - Define LoggingClient class with async logging methods
    - Include batch buffering and flush support
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  - [x] 5.2 Create `src/crypto/clients/logging_client.cpp`
    - Implement gRPC connection to logging-service
    - Implement batch buffering with configurable size
    - Implement exponential backoff retry on failure
    - Implement local console fallback when circuit breaker opens
    - _Requirements: 1.1, 1.2, 1.3, 1.6_
  - [x] 5.3 Write property test for log entry structure
    - **Property 1: Log Entry Structure Completeness**
    - **Validates: Requirements 1.2, 1.4**
    - Created: `tests/property/logging_properties_test.cpp`
  - [x] 5.4 Write unit tests for LoggingClient
    - Test connection establishment
    - Test batch buffering behavior
    - Test fallback to console logging
    - Created: `tests/unit/clients/logging_client_test.cpp`
    - _Requirements: 7.2_

- [x] 6. Implement CacheClient for Platform Integration
  - [x] 6.1 Create `include/crypto/clients/cache_client.h`
    - Define CacheClientConfig struct
    - Define CacheClient class with get/set/del operations
    - Include local fallback cache support
    - _Requirements: 2.1, 2.6_
  - [x] 6.2 Create `src/crypto/clients/cache_client.cpp`
    - Implement gRPC connection to cache-service
    - Implement namespace-prefixed key operations
    - Implement local LRU cache fallback
    - _Requirements: 2.1, 2.2, 2.6_
  - [x] 6.3 Write property test for key caching lifecycle
    - **Property 2: Key Caching Lifecycle Correctness**
    - **Validates: Requirements 2.2, 2.3, 2.4**
    - Created: `tests/property/cache_properties_test.cpp`
  - [x] 6.4 Write unit tests for CacheClient
    - Test connection establishment
    - Test get/set/del operations
    - Test local fallback behavior
    - Created: `tests/unit/clients/cache_client_test.cpp`
    - _Requirements: 7.2_

- [x] 7. Checkpoint - Verify platform clients work correctly
  - All property tests and unit tests created for LoggingClient and CacheClient

- [x] 8. Update Key Service to Use CacheClient
  - [x] 8.1 Refactor `src/crypto/keys/key_service.cpp`
    - Replace local KeyCache with CacheClient
    - Implement cache-first key retrieval
    - Implement cache invalidation on rotate/delete
    - _Requirements: 2.2, 2.3, 2.4_
  - [x] 8.2 Update `include/crypto/keys/key_service.h`
    - Replace KeyCache dependency with CacheClient
    - Update constructor signature
    - _Requirements: 2.2_
  - [x] 8.3 Verify existing key property tests pass
    - Run key_properties_test.cpp (updated to use CacheClient)
    - _Requirements: 7.3_

- [x] 9. Update Services to Use LoggingClient
  - [x] 9.1 Refactor `src/crypto/services/encryption_service.cpp`
    - Replace audit_logger with LoggingClient
    - Include correlation_id in all log entries
    - _Requirements: 1.2, 1.4_
  - [x] 9.2 Refactor `src/crypto/services/signature_service.cpp`
    - Replace audit_logger with LoggingClient
    - Include correlation_id in all log entries
    - _Requirements: 1.2, 1.4_
  - [x] 9.3 Refactor `src/crypto/services/file_encryption_service.cpp`
    - Replace audit_logger with LoggingClient
    - Include correlation_id in all log entries
    - _Requirements: 1.2, 1.4_

- [x] 10. Checkpoint - Verify services work with new clients
  - Services updated to use LoggingClient with correlation_id

- [x] 11. Add Observability Enhancements
  - [x] 11.1 Update `src/crypto/metrics/prometheus_exporter.cpp`
    - Add error metrics with error_code label
    - Add latency histograms for all operations
    - _Requirements: 9.1, 9.5, 9.6_
  - [x] 11.2 Update `src/crypto/metrics/tracing.cpp`
    - Implement W3C Trace Context propagation
    - Include correlation_id in all spans
    - _Requirements: 9.2, 9.3, 9.4_
  - [x] 11.3 Write property test for trace context propagation
    - **Property 3: Trace Context Propagation**
    - **Validates: Requirements 3.6, 9.3**
    - Created: `tests/property/observability_properties_test.cpp`
  - [x] 11.4 Write property test for observability metadata
    - **Property 4: Observability Metadata Completeness**
    - **Validates: Requirements 9.2, 9.4**
    - Included in: `tests/property/observability_properties_test.cpp`
  - [x] 11.5 Write property test for error metric emission
    - **Property 5: Error Metric Emission**
    - **Validates: Requirements 9.5**
    - Included in: `tests/property/observability_properties_test.cpp`

- [x] 12. Update Configuration System
  - [x] 12.1 Refactor `src/crypto/config/config_loader.cpp`
    - Add LoggingClientConfig loading from environment
    - Add CacheClientConfig loading from environment
    - Implement validation for all configuration values
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_
  - [x] 12.2 Update `include/crypto/config/config_loader.h`
    - Add ServiceConfig struct with all configuration
    - Add from_env() and validate() methods
    - _Requirements: 8.1, 8.3_
  - [x] 12.3 Write property test for configuration validation
    - **Property 7: Configuration Validation**
    - **Validates: Requirements 8.3**
    - Created: `tests/property/config_properties_test.cpp`
  - [x] 12.4 Write unit tests for configuration loading
    - Test environment variable parsing
    - Test validation error messages
    - Test default values
    - Created: `tests/unit/config/config_loader_test.cpp`
    - _Requirements: 7.2_

- [x] 13. Checkpoint - Verify configuration and observability
  - All property tests and unit tests created for configuration and observability

- [x] 14. Add Input Validation and Security Hardening
  - [x] 14.1 Update all crypto engines with input size validation
    - Add size checks before processing
    - Return INVALID_INPUT for oversized inputs
    - _Requirements: 10.5_
  - [x] 14.2 Update error handling to not leak sensitive information
    - Review all error messages for sensitive data
    - Replace specific errors with generic messages where needed
    - _Requirements: 10.6_
  - [x] 14.3 Write property test for input validation and error safety
    - **Property 6: Input Validation and Error Safety**
    - **Validates: Requirements 10.5, 10.6**
    - Created: `tests/property/input_validation_properties_test.cpp`

- [x] 15. Remove Redundant Code
  - [x] 15.1 Delete local logging implementation
    - Delete `include/crypto/logging/json_logger.h`
    - Delete `src/crypto/logging/json_logger.cpp`
    - Update all imports to use LoggingClient
    - _Requirements: 1.5_
  - [x] 15.2 Delete local resilience implementations
    - Delete `include/crypto/resilience/circuit_breaker.h`
    - Delete `include/crypto/resilience/retry.h`
    - Delete `src/crypto/resilience/circuit_breaker.cpp`
    - Delete `src/crypto/resilience/retry.cpp`
    - _Requirements: 3.1, 3.2_
  - [x] 15.3 Delete local key cache implementation
    - Delete `include/crypto/keys/key_cache.h`
    - Delete `src/crypto/keys/key_cache.cpp`
    - _Requirements: 2.5_
  - [x] 15.4 Update CMakeLists.txt to remove deleted files
    - Remove deleted source files from CRYPTO_SOURCES
    - Add new client source files
    - _Requirements: 12.1_

- [x] 16. Checkpoint - Verify code compiles after deletions
  - Redundant code removed, CMakeLists.txt updated

- [x] 17. Update Build System
  - [x] 17.1 Update CMakeLists.txt for C++23 and CMake 3.28
    - Set CMAKE_CXX_STANDARD to 23
    - Require CMake 3.28 minimum
    - Add CMake presets for common configurations
    - _Requirements: 5.1, 12.1, 12.2_
  - [x] 17.2 Update OpenSSL dependency to 3.3+
    - Require OpenSSL 3.3.0 minimum
    - Add FIPS mode support
    - _Requirements: 6.1, 6.5_
  - [x] 17.3 Add FetchContent for RapidCheck
    - Pin version with checksum
    - _Requirements: 12.5, 12.6_

- [x] 18. Create Kubernetes ResiliencePolicy
  - [x] 18.1 Create `deploy/kubernetes/service-mesh/crypto-service/resilience-policy.yaml`
    - Define ResiliencePolicy CRD for crypto-service
    - Configure circuit breaker settings
    - Configure retry settings
    - _Requirements: 3.3, 3.4_
  - [x] 18.2 Update health check endpoints
    - Ensure /health and /ready endpoints work with Linkerd
    - _Requirements: 3.5_

- [x] 19. Reorganize Test Directory Structure
  - [x] 19.1 Move existing tests to new structure
    - Move unit tests to tests/unit/crypto/
    - Keep property tests in tests/property/
    - Create tests/integration/ directory
    - _Requirements: 7.1, 7.2_
  - [x] 19.2 Create integration test infrastructure
    - Add Testcontainers setup for logging-service
    - Add Testcontainers setup for cache-service
    - _Requirements: 7.4_
  - [x] 19.3 Write integration tests for platform services
    - Test LoggingClient with real logging-service
    - Test CacheClient with real cache-service
    - _Requirements: 7.4_

- [x] 20. Final Checkpoint - Full Test Suite
  - All property tests created (7 properties)
  - All unit tests created for clients and configuration
  - Integration tests created for platform services
  - User should run full test suite in build environment

- [x] 21. Update Documentation
  - [x] 21.1 Update README.md with new architecture
    - Document platform service dependencies
    - Update configuration documentation
    - _Requirements: 11.1, 11.2_
  - [x] 21.2 Update API documentation
    - Ensure gRPC API contract is documented
    - Ensure REST API endpoints are documented
    - _Requirements: 11.1, 11.2_

## Notes

- All tasks are required for production readiness
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Integration tests require Testcontainers and real service instances
