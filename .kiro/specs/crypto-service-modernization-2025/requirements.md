# Requirements Document

## Introduction

This document specifies the requirements for modernizing the `crypto-service` to state-of-the-art December 2025 standards. The modernization focuses on:

1. **Integration with Platform Services**: Using centralized `logging-service` and `cache-service` instead of local implementations
2. **Redundancy Elimination**: Removing duplicated code (circuit breaker, logging, caching) that exists in platform services
3. **Architecture Consolidation**: Centralizing cross-cutting concerns and eliminating dead/legacy code
4. **Modern C++23 Standards**: Upgrading to latest C++ features and OpenSSL 3.3+ APIs
5. **Production Readiness**: Ensuring 100% test coverage with all tests passing

## Glossary

- **Crypto_Service**: The cryptographic microservice providing encryption, decryption, signing, and key management operations
- **Logging_Service**: Centralized platform logging microservice (gRPC at port 5001)
- **Cache_Service**: Centralized platform cache microservice (gRPC at port 50051)
- **AES_Engine**: Component responsible for AES-GCM and AES-CBC symmetric encryption operations
- **RSA_Engine**: Component responsible for RSA-OAEP encryption and RSA-PSS signature operations
- **ECDSA_Engine**: Component responsible for elliptic curve digital signature operations
- **Key_Service**: Component managing cryptographic key lifecycle (generation, storage, rotation, deletion)
- **Audit_Logger**: Component recording cryptographic operations for compliance
- **Circuit_Breaker**: Resilience pattern protecting against cascading failures (to be removed - use Service Mesh)
- **Resilience_Policy**: Kubernetes CRD for declarative resilience via Linkerd Service Mesh

## Requirements

### Requirement 1: Platform Logging Integration

**User Story:** As a platform operator, I want the crypto-service to use the centralized logging-service, so that all logs are aggregated in a single location for monitoring and compliance.

#### Acceptance Criteria

1. WHEN the Crypto_Service starts, THE Logging_Client SHALL establish a gRPC connection to the Logging_Service at the configured address
2. WHEN a cryptographic operation occurs, THE Crypto_Service SHALL send structured log entries to the Logging_Service via gRPC
3. WHEN the Logging_Service is unavailable, THE Logging_Client SHALL buffer log entries locally and retry with exponential backoff
4. WHEN log entries are sent, THE Logging_Client SHALL include correlation_id, trace_context, service_id, and operation metadata
5. THE Crypto_Service SHALL remove the local JsonLogger implementation and use the centralized Logging_Client exclusively
6. WHEN the Logging_Client circuit breaker opens, THE Crypto_Service SHALL fall back to local console logging

### Requirement 2: Platform Cache Integration

**User Story:** As a platform operator, I want the crypto-service to use the centralized cache-service for key caching, so that cache management is unified across the platform.

#### Acceptance Criteria

1. WHEN the Crypto_Service starts, THE Cache_Client SHALL establish a gRPC connection to the Cache_Service at the configured address
2. WHEN a key is requested, THE Key_Service SHALL first check the Cache_Service for the cached key material
3. WHEN a key is loaded from storage, THE Key_Service SHALL cache it in the Cache_Service with appropriate TTL
4. WHEN a key is rotated or deleted, THE Key_Service SHALL invalidate the corresponding cache entry
5. THE Crypto_Service SHALL remove the local KeyCache implementation and use the centralized Cache_Client exclusively
6. WHEN the Cache_Service is unavailable, THE Cache_Client SHALL fall back to local in-memory caching with circuit breaker protection

### Requirement 3: Resilience Architecture Modernization

**User Story:** As a platform architect, I want the crypto-service to rely on Service Mesh (Linkerd) for resilience patterns, so that resilience is managed declaratively without code changes.

#### Acceptance Criteria

1. THE Crypto_Service SHALL remove the local CircuitBreaker implementation from the codebase
2. THE Crypto_Service SHALL remove the local Retry implementation from the codebase
3. THE Crypto_Service SHALL be protected by a ResiliencePolicy CRD applied via Kubernetes
4. WHEN external service calls fail, THE Linkerd proxy SHALL handle circuit breaking and retries transparently
5. THE Crypto_Service SHALL expose health endpoints for Linkerd health checking
6. THE Crypto_Service SHALL propagate trace context headers for distributed tracing

### Requirement 4: Code Deduplication and Centralization

**User Story:** As a developer, I want all cross-cutting concerns centralized, so that there is a single source of truth for each behavior.

#### Acceptance Criteria

1. THE Crypto_Service SHALL use a single Result<T> type for all error handling across all components
2. THE Crypto_Service SHALL use a single ErrorCode enumeration for all error types
3. THE Crypto_Service SHALL centralize all OpenSSL RAII wrappers in a single header file
4. THE Crypto_Service SHALL remove duplicate BIODeleter, EVPPKeyCtxDeleter, EVPMDCtxDeleter definitions
5. THE Crypto_Service SHALL centralize hash algorithm selection logic in a single location
6. WHEN adding new error codes, THE Developer SHALL add them only to the centralized ErrorCode enumeration

### Requirement 5: Modern C++23 Upgrade

**User Story:** As a developer, I want the crypto-service to use modern C++23 features, so that the code is more expressive, safer, and maintainable.

#### Acceptance Criteria

1. THE Crypto_Service SHALL compile with C++23 standard (-std=c++23)
2. THE Crypto_Service SHALL use std::expected<T, E> instead of custom Result<T> where appropriate
3. THE Crypto_Service SHALL use std::format for string formatting instead of std::ostringstream
4. THE Crypto_Service SHALL use std::ranges algorithms where applicable
5. THE Crypto_Service SHALL use constexpr where possible for compile-time evaluation
6. THE Crypto_Service SHALL use [[nodiscard]] attribute on all functions returning Result types

### Requirement 6: OpenSSL 3.3+ Modernization

**User Story:** As a security engineer, I want the crypto-service to use the latest OpenSSL 3.3+ APIs, so that deprecated functions are eliminated and security is maximized.

#### Acceptance Criteria

1. THE Crypto_Service SHALL require OpenSSL 3.3.0 or later
2. THE Crypto_Service SHALL use EVP_PKEY_fromdata for key import instead of deprecated functions
3. THE Crypto_Service SHALL use OSSL_PARAM for algorithm configuration instead of legacy APIs
4. THE Crypto_Service SHALL use EVP_MAC for HMAC operations instead of deprecated HMAC functions
5. THE Crypto_Service SHALL enable FIPS mode when FIPS_MODE environment variable is set
6. WHEN using deprecated OpenSSL functions, THE Build SHALL fail with a compilation error

### Requirement 7: Test Architecture Modernization

**User Story:** As a developer, I want tests separated from source code with clear organization, so that the codebase is maintainable and test coverage is comprehensive.

#### Acceptance Criteria

1. THE Test directory SHALL be organized as tests/unit/, tests/property/, tests/integration/
2. THE Unit tests SHALL mirror the source directory structure (tests/unit/crypto/engine/aes_engine_test.cpp)
3. THE Property tests SHALL use RapidCheck with minimum 100 iterations per property
4. THE Integration tests SHALL use Testcontainers for external dependencies
5. WHEN a source file is modified, THE corresponding test file SHALL be updated
6. THE Test coverage SHALL be at least 80% for all modules

### Requirement 8: Configuration Modernization

**User Story:** As a platform operator, I want configuration to be environment-based and validated at startup, so that misconfigurations are caught early.

#### Acceptance Criteria

1. THE Crypto_Service SHALL load configuration from environment variables
2. WHEN a required configuration is missing, THE Crypto_Service SHALL fail fast with a descriptive error
3. THE Crypto_Service SHALL validate all configuration values at startup
4. THE Crypto_Service SHALL support configuration for Logging_Service address, Cache_Service address, and TLS settings
5. THE Crypto_Service SHALL use sensible defaults for optional configuration
6. WHEN configuration changes, THE Crypto_Service SHALL NOT require code changes

### Requirement 9: Observability Enhancement

**User Story:** As a platform operator, I want comprehensive observability, so that I can monitor service health and performance.

#### Acceptance Criteria

1. THE Crypto_Service SHALL expose Prometheus metrics at /metrics endpoint
2. THE Crypto_Service SHALL emit OpenTelemetry traces for all operations
3. THE Crypto_Service SHALL propagate W3C Trace Context headers
4. THE Crypto_Service SHALL include correlation_id in all log entries and traces
5. WHEN an operation fails, THE Crypto_Service SHALL emit an error metric with error_code label
6. THE Crypto_Service SHALL expose latency histograms for all cryptographic operations

### Requirement 10: Security Hardening

**User Story:** As a security engineer, I want the crypto-service to follow security best practices, so that cryptographic operations are protected.

#### Acceptance Criteria

1. THE Crypto_Service SHALL use secure memory allocation (mlock) for all key material
2. THE Crypto_Service SHALL zero memory containing key material before deallocation
3. THE Crypto_Service SHALL use constant-time comparison for all authentication tags and signatures
4. THE Crypto_Service SHALL require TLS 1.3 for all gRPC and REST communications
5. THE Crypto_Service SHALL validate all input sizes before processing
6. WHEN a security-sensitive operation fails, THE Crypto_Service SHALL NOT leak sensitive information in error messages

### Requirement 11: API Compatibility

**User Story:** As a service consumer, I want the crypto-service API to remain backward compatible, so that existing integrations continue to work.

#### Acceptance Criteria

1. THE Crypto_Service SHALL maintain the existing gRPC API contract (crypto_service.proto)
2. THE Crypto_Service SHALL maintain the existing REST API endpoints and response formats
3. WHEN adding new functionality, THE Crypto_Service SHALL use additive changes only
4. THE Crypto_Service SHALL version the API using semantic versioning
5. WHEN deprecating functionality, THE Crypto_Service SHALL provide a migration path
6. THE Crypto_Service SHALL pass all existing integration tests without modification

### Requirement 12: Build System Modernization

**User Story:** As a developer, I want a modern build system, so that builds are fast, reproducible, and easy to maintain.

#### Acceptance Criteria

1. THE CMakeLists.txt SHALL require CMake 3.28 or later
2. THE Build SHALL use CMake presets for common configurations
3. THE Build SHALL support sanitizers (ASan, UBSan, TSan) via build options
4. THE Build SHALL generate compile_commands.json for IDE integration
5. THE Build SHALL use FetchContent for external dependencies (RapidCheck, gRPC)
6. WHEN dependencies are updated, THE Build SHALL use version pinning with checksums
