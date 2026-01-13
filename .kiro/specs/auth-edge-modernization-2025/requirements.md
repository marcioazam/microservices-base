# Requirements Document

## Introduction

This document specifies the requirements for modernizing the Auth Edge Service to state-of-the-art December 2025 standards. The modernization focuses on eliminating redundancies, centralizing shared logic via `libs/rust/rust-common`, integrating with platform services (`cache-service`, `logging-service`), upgrading dependencies to latest stable versions, and ensuring production-ready quality with comprehensive testing.

## Glossary

- **Auth_Edge_Service**: Ultra-low latency JWT validation and edge routing gRPC service
- **JWT_Validator**: Component responsible for validating JSON Web Tokens using type-state pattern
- **JWK_Cache**: Cache for JSON Web Keys with single-flight refresh pattern
- **Circuit_Breaker**: Resilience pattern that prevents cascading failures
- **Rate_Limiter**: Component that enforces request rate limits per client
- **SPIFFE_Validator**: Component for validating SPIFFE workload identities
- **Cache_Client**: gRPC client for centralized Cache_Service from rust-common
- **Logging_Client**: gRPC client for centralized Logging_Service from rust-common
- **Tower_Middleware**: Composable service middleware using Tower crate
- **Platform_Error**: Centralized error type from rust-common with retryability classification
- **Rust_Common**: Shared library at `libs/rust/rust-common` for cross-cutting concerns

## Requirements

### Requirement 1: Dependency Modernization

**User Story:** As a platform engineer, I want all dependencies upgraded to December 2025 stable versions, so that the service benefits from latest security patches, performance improvements, and language features.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL use Rust 2024 edition with rust-version 1.85+
2. THE Auth_Edge_Service SHALL use tokio 1.42+ for async runtime
3. THE Auth_Edge_Service SHALL use tonic 0.12+ for gRPC
4. THE Auth_Edge_Service SHALL use jsonwebtoken 9.3+ for JWT handling
5. THE Auth_Edge_Service SHALL use thiserror 2.0+ for error handling
6. THE Auth_Edge_Service SHALL use opentelemetry 0.27+ for observability
7. THE Auth_Edge_Service SHALL use rustls 0.23+ for TLS (replacing 0.21)
8. THE Auth_Edge_Service SHALL use reqwest 0.12+ with rustls-tls feature
9. THE Auth_Edge_Service SHALL use proptest 1.5+ for property-based testing
10. THE Auth_Edge_Service SHALL remove deprecated failsafe crate dependency
11. THE Auth_Edge_Service SHALL remove unused borrow crate dependency

### Requirement 2: Centralized Error Handling via rust-common

**User Story:** As a developer, I want error handling centralized in rust-common, so that error types are consistent across all Rust services and redundancy is eliminated.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL use PlatformError from rust-common as base error type
2. THE Auth_Edge_Service SHALL extend PlatformError with domain-specific AuthEdgeError variants
3. WHEN an error occurs, THE Auth_Edge_Service SHALL classify it using is_retryable() from PlatformError
4. THE Auth_Edge_Service SHALL remove duplicate error code constants (AUTH_TOKEN_MISSING, etc.)
5. THE Auth_Edge_Service SHALL use centralized error sanitization from rust-common
6. WHEN converting errors to gRPC Status, THE Auth_Edge_Service SHALL include correlation_id

### Requirement 3: Centralized Circuit Breaker via rust-common

**User Story:** As a developer, I want circuit breaker logic centralized in rust-common, so that resilience patterns are consistent and not duplicated.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL use CircuitBreaker from rust-common instead of local implementation
2. THE Auth_Edge_Service SHALL use CircuitBreakerConfig from rust-common for configuration
3. THE Auth_Edge_Service SHALL remove local circuit_breaker module (mod.rs, state.rs)
4. THE Auth_Edge_Service SHALL remove StandaloneCircuitBreaker duplicate implementation
5. WHEN circuit breaker opens, THE Auth_Edge_Service SHALL return PlatformError::CircuitOpen

### Requirement 4: Integration with Cache_Service

**User Story:** As a platform engineer, I want JWK caching to use the centralized Cache_Service, so that cache management is consistent and benefits from platform-wide features.

#### Acceptance Criteria

1. THE JWK_Cache SHALL use CacheClient from rust-common for distributed caching
2. THE JWK_Cache SHALL use namespace "auth-edge:jwk" for key isolation
3. THE JWK_Cache SHALL encrypt cached JWK data using CacheClient encryption
4. THE JWK_Cache SHALL maintain local fallback when Cache_Service is unavailable
5. WHEN Cache_Service circuit opens, THE JWK_Cache SHALL fall back to local cache
6. THE JWK_Cache SHALL preserve single-flight refresh pattern for thundering herd prevention

### Requirement 5: Integration with Logging_Service

**User Story:** As a platform engineer, I want structured logging to use the centralized Logging_Service, so that logs are aggregated and searchable across all services.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL use LoggingClient from rust-common for structured logging
2. THE Auth_Edge_Service SHALL configure service_id as "auth-edge-service"
3. WHEN logging, THE Auth_Edge_Service SHALL include correlation_id in all log entries
4. WHEN logging, THE Auth_Edge_Service SHALL include trace_id and span_id from OpenTelemetry context
5. THE Auth_Edge_Service SHALL use LogLevel::Error for authentication failures
6. THE Auth_Edge_Service SHALL use LogLevel::Info for successful validations
7. WHEN Logging_Service is unavailable, THE Auth_Edge_Service SHALL fall back to local tracing

### Requirement 6: Eliminate Redundant Code

**User Story:** As a developer, I want all redundant code eliminated, so that the codebase is minimal, maintainable, and follows DRY principles.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL remove duplicate has_claim() method (exists in both validator.rs and token.rs)
2. THE Auth_Edge_Service SHALL remove duplicate sanitize_message() function (centralize in rust-common)
3. THE Auth_Edge_Service SHALL remove duplicate CircuitBreakerState (use rust-common)
4. THE Auth_Edge_Service SHALL remove duplicate error code string constants
5. THE Auth_Edge_Service SHALL consolidate middleware layers into single composable stack
6. THE Auth_Edge_Service SHALL remove unused SpiffeExtractor (consolidate with SpiffeValidator)

### Requirement 7: Architecture Reorganization

**User Story:** As a developer, I want clear separation between source code and tests, so that the codebase follows standard Rust project conventions.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL maintain src/ directory for all source code
2. THE Auth_Edge_Service SHALL maintain tests/ directory for all test code
3. THE Auth_Edge_Service SHALL organize tests into unit/, integration/, property/, and contract/ subdirectories
4. THE Auth_Edge_Service SHALL use #[cfg(test)] for unit tests co-located with source
5. THE Auth_Edge_Service SHALL export public API through lib.rs for testability

### Requirement 8: Modernized Observability

**User Story:** As an SRE, I want comprehensive observability with OpenTelemetry 0.27+, so that I can monitor, trace, and debug the service effectively.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL use opentelemetry 0.27+ with OTLP exporter
2. THE Auth_Edge_Service SHALL propagate W3C Trace Context headers
3. THE Auth_Edge_Service SHALL record span attributes for all gRPC methods
4. THE Auth_Edge_Service SHALL export metrics in Prometheus format
5. WHEN errors occur, THE Auth_Edge_Service SHALL record error events with correlation_id
6. THE Auth_Edge_Service SHALL use tracing-opentelemetry 0.28+ for integration

### Requirement 9: Type-State JWT Validation Preservation

**User Story:** As a security engineer, I want the type-state JWT validation pattern preserved, so that compile-time guarantees prevent accessing unvalidated claims.

#### Acceptance Criteria

1. THE JWT_Validator SHALL maintain Token<Unvalidated>, Token<SignatureValidated>, Token<Validated> states
2. THE JWT_Validator SHALL only expose claims() method on Token<Validated>
3. THE JWT_Validator SHALL use sealed trait pattern to prevent external state implementations
4. WHEN parsing a token, THE JWT_Validator SHALL return Token<Unvalidated>
5. WHEN validating signature, THE JWT_Validator SHALL transition to Token<SignatureValidated>
6. WHEN validating claims, THE JWT_Validator SHALL transition to Token<Validated>

### Requirement 10: Production-Ready Testing

**User Story:** As a QA engineer, I want comprehensive test coverage with property-based tests, so that the service is production-ready with verified correctness.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL maintain 90%+ test coverage
2. THE Auth_Edge_Service SHALL use proptest 1.5+ for property-based testing
3. WHEN running property tests, THE Auth_Edge_Service SHALL execute minimum 100 iterations per property
4. THE Auth_Edge_Service SHALL include property tests for error sanitization
5. THE Auth_Edge_Service SHALL include property tests for circuit breaker state machine
6. THE Auth_Edge_Service SHALL include property tests for rate limiter enforcement
7. THE Auth_Edge_Service SHALL include property tests for SPIFFE ID parsing round-trip
8. THE Auth_Edge_Service SHALL include contract tests for downstream service interactions

### Requirement 11: Configuration Modernization

**User Story:** As a DevOps engineer, I want type-safe configuration with validation, so that misconfigurations are caught at startup.

#### Acceptance Criteria

1. THE Config SHALL use type-state builder pattern for required fields
2. THE Config SHALL validate all URLs are well-formed at startup
3. THE Config SHALL validate numeric ranges (port 1-65535, TTL > 0)
4. IF required configuration is missing, THEN THE Auth_Edge_Service SHALL fail fast with descriptive error
5. THE Config SHALL support environment variable overrides for all settings
6. THE Config SHALL use serde for deserialization with validation

### Requirement 12: Graceful Shutdown

**User Story:** As an SRE, I want graceful shutdown with configurable timeout, so that in-flight requests complete before termination.

#### Acceptance Criteria

1. WHEN SIGTERM is received, THE Auth_Edge_Service SHALL initiate graceful shutdown
2. WHEN SIGINT is received, THE Auth_Edge_Service SHALL initiate graceful shutdown
3. THE Auth_Edge_Service SHALL wait for in-flight requests up to configurable timeout
4. THE Auth_Edge_Service SHALL flush LoggingClient buffer before shutdown
5. THE Auth_Edge_Service SHALL close CacheClient connections gracefully
6. IF shutdown timeout exceeded, THEN THE Auth_Edge_Service SHALL abort remaining tasks

### Requirement 13: Security Hardening

**User Story:** As a security engineer, I want all security best practices applied, so that the service is hardened against common vulnerabilities.

#### Acceptance Criteria

1. THE Auth_Edge_Service SHALL use constant-time comparison for token signatures
2. THE Auth_Edge_Service SHALL sanitize all error messages before external exposure
3. THE Auth_Edge_Service SHALL never log sensitive data (tokens, keys, credentials)
4. THE Auth_Edge_Service SHALL validate all input before processing
5. THE Auth_Edge_Service SHALL use rustls with secure cipher suites only
6. THE Auth_Edge_Service SHALL reject tokens with algorithm confusion attacks
7. THE Auth_Edge_Service SHALL enforce minimum key sizes for cryptographic operations
