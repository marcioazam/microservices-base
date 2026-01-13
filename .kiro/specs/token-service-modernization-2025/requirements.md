# Requirements Document

## Introduction

This document specifies the requirements for modernizing the Token Service (`services/token`) to state-of-the-art December 2025 standards. The modernization focuses on eliminating redundancy, integrating with platform shared libraries (`libs/rust/rust-common`), upgrading to Rust 2024 edition with latest stable dependencies, and ensuring production-ready quality with comprehensive property-based testing.

## Glossary

- **Token_Service**: The gRPC microservice responsible for JWT generation, DPoP validation, refresh token rotation, and JWKS publishing
- **JWT**: JSON Web Token per RFC 7519, used for access and ID tokens
- **DPoP**: Demonstrating Proof of Possession per RFC 9449, sender-constrained tokens
- **JWKS**: JSON Web Key Set per RFC 7517, public keys for token verification
- **Refresh_Token**: Opaque token for obtaining new access tokens without re-authentication
- **Token_Family**: Group of related refresh tokens for rotation tracking and replay detection
- **KMS**: Key Management Service for HSM-backed cryptographic operations
- **Cache_Service**: Platform centralized caching service (`platform/cache-service`)
- **Logging_Service**: Platform centralized logging service (`platform/logging-service`)
- **rust-common**: Shared Rust library with cross-cutting concerns (`libs/rust/rust-common`)
- **Circuit_Breaker**: Resilience pattern for fail-fast behavior on external service failures
- **PBT**: Property-Based Testing using proptest for correctness verification

## Requirements

### Requirement 1: Rust 2024 Edition Migration

**User Story:** As a platform maintainer, I want the Token Service to use Rust 2024 edition with latest stable dependencies, so that we benefit from modern language features and security patches.

#### Acceptance Criteria

1. THE Token_Service SHALL use Rust edition 2024 with rust-version 1.85 or higher
2. THE Token_Service SHALL use workspace dependencies from `libs/rust/Cargo.toml` for shared crates
3. THE Token_Service SHALL use native async traits (removing async-trait crate dependency)
4. THE Token_Service SHALL use thiserror 2.0 for error handling
5. THE Token_Service SHALL use tonic 0.12 for gRPC with prost 0.13
6. THE Token_Service SHALL use jsonwebtoken 9.3 for JWT operations
7. THE Token_Service SHALL use redis 0.27 with cluster-async and tokio-comp features
8. THE Token_Service SHALL use proptest 1.5 for property-based testing

### Requirement 2: Platform Library Integration

**User Story:** As a platform architect, I want the Token Service to use centralized platform libraries, so that cross-cutting concerns are consistent and maintainable.

#### Acceptance Criteria

1. THE Token_Service SHALL use rust-common::CacheClient for all caching operations instead of direct Redis access
2. THE Token_Service SHALL use rust-common::LoggingClient for structured logging to Logging_Service
3. THE Token_Service SHALL use rust-common::PlatformError as the base error type
4. THE Token_Service SHALL use rust-common::CircuitBreaker for external service resilience
5. THE Token_Service SHALL use rust-common::RetryPolicy for retryable operations
6. THE Token_Service SHALL use rust-common metrics module for Prometheus-compatible metrics
7. WHEN Cache_Service is unavailable, THE Token_Service SHALL fallback to local cache with encryption

### Requirement 3: Error Handling Centralization

**User Story:** As a developer, I want unified error handling across the Token Service, so that error responses are consistent and debuggable.

#### Acceptance Criteria

1. THE Token_Service SHALL define TokenError as an extension of rust-common::PlatformError
2. THE Token_Service SHALL classify all errors as retryable or non-retryable
3. THE Token_Service SHALL include correlation IDs in all error responses
4. THE Token_Service SHALL map TokenError variants to appropriate gRPC status codes
5. WHEN an error occurs, THE Token_Service SHALL log the error with full context to Logging_Service
6. THE Token_Service SHALL NOT expose internal error details in gRPC responses

### Requirement 4: JWT Token Generation

**User Story:** As an authentication consumer, I want to receive properly signed JWT tokens, so that I can authenticate to protected resources.

#### Acceptance Criteria

1. WHEN a valid token request is received, THE Token_Service SHALL generate a signed JWT access token
2. THE Token_Service SHALL include standard claims: iss, sub, aud, exp, iat, nbf, jti
3. THE Token_Service SHALL support custom claims via the request payload
4. THE Token_Service SHALL sign tokens using the configured KMS provider
5. THE Token_Service SHALL include the key ID (kid) in the JWT header
6. FOR ALL valid Claims objects, serializing to JWT and parsing back SHALL produce equivalent claims (round-trip property)

### Requirement 5: DPoP Token Binding (RFC 9449)

**User Story:** As a security engineer, I want DPoP support for sender-constrained tokens, so that stolen tokens cannot be used by attackers.

#### Acceptance Criteria

1. WHEN a DPoP proof is provided, THE Token_Service SHALL validate the proof per RFC 9449
2. THE Token_Service SHALL validate DPoP proof typ header equals "dpop+jwt"
3. THE Token_Service SHALL validate DPoP proof alg is ES256 or RS256
4. THE Token_Service SHALL validate htm claim matches the HTTP method
5. THE Token_Service SHALL validate htu claim matches the request URI
6. THE Token_Service SHALL validate iat claim is within acceptable clock skew (60 seconds)
7. THE Token_Service SHALL detect and reject replayed DPoP proofs via jti tracking
8. WHEN DPoP is valid, THE Token_Service SHALL bind the access token with cnf.jkt claim
9. THE Token_Service SHALL compute JWK thumbprints per RFC 7638 using SHA-256
10. FOR ALL valid JWKs, computing thumbprint twice SHALL produce identical results (determinism property)
11. FOR ALL DPoP proofs with previously seen jti, THE Token_Service SHALL reject as replay attack

### Requirement 6: Refresh Token Rotation

**User Story:** As a security engineer, I want refresh token rotation with replay detection, so that compromised tokens are quickly invalidated.

#### Acceptance Criteria

1. WHEN issuing tokens, THE Token_Service SHALL create a new token family with unique family_id
2. WHEN a refresh token is used, THE Token_Service SHALL issue a new refresh token and invalidate the old one
3. WHEN a rotated refresh token is reused, THE Token_Service SHALL detect replay attack and revoke the entire family
4. THE Token_Service SHALL track rotation count per family
5. THE Token_Service SHALL store token families in Cache_Service with configurable TTL
6. FOR ALL refresh token rotations, the old token SHALL become invalid immediately
7. FOR ALL token families, revocation SHALL prevent all tokens in the family from being used

### Requirement 7: JWKS Publishing

**User Story:** As a resource server, I want to retrieve public keys for token verification, so that I can validate tokens without calling the Token Service.

#### Acceptance Criteria

1. THE Token_Service SHALL expose a GetJWKS endpoint returning current signing keys
2. THE Token_Service SHALL include both current and previous keys during rotation for graceful transition
3. THE Token_Service SHALL format keys per RFC 7517 (JWK format)
4. WHEN a key rotation occurs, THE Token_Service SHALL retain the previous key for a configurable period
5. FOR ALL key rotations, the JWKS response SHALL contain both old and new keys

### Requirement 8: KMS Integration

**User Story:** As a security engineer, I want HSM-backed signing via AWS KMS, so that private keys are never exposed.

#### Acceptance Criteria

1. THE Token_Service SHALL support AWS KMS for production signing
2. THE Token_Service SHALL support mock KMS for development and testing
3. THE Token_Service SHALL implement circuit breaker for KMS operations
4. WHEN KMS is unavailable and fallback is enabled, THE Token_Service SHALL use fallback signing with time limit
5. THE Token_Service SHALL log all KMS failures as security events
6. THE Token_Service SHALL map KMS algorithms to JWT algorithms (RSASSA_PSS_SHA_256 → PS256, ECDSA_SHA_256 → ES256)

### Requirement 9: Observability Integration

**User Story:** As an SRE, I want comprehensive observability for the Token Service, so that I can monitor health and debug issues.

#### Acceptance Criteria

1. THE Token_Service SHALL emit structured logs to Logging_Service via rust-common::LoggingClient
2. THE Token_Service SHALL include correlation_id, trace_id, and span_id in all log entries
3. THE Token_Service SHALL expose Prometheus metrics for token operations
4. THE Token_Service SHALL track metrics: tokens_issued_total, tokens_refreshed_total, tokens_revoked_total, dpop_validations_total
5. THE Token_Service SHALL track latency histograms for all gRPC methods
6. THE Token_Service SHALL integrate with OpenTelemetry for distributed tracing

### Requirement 10: Code Architecture

**User Story:** As a maintainer, I want clean architecture with zero redundancy, so that the codebase is maintainable and testable.

#### Acceptance Criteria

1. THE Token_Service SHALL separate source code from tests (src/ and tests/ directories)
2. THE Token_Service SHALL have no file exceeding 400 lines
3. THE Token_Service SHALL have no duplicated logic across modules
4. THE Token_Service SHALL use trait-based abstractions for testability
5. THE Token_Service SHALL centralize all configuration in a single Config struct
6. THE Token_Service SHALL centralize all error types in a single error module

### Requirement 11: Property-Based Testing

**User Story:** As a quality engineer, I want comprehensive property-based tests, so that correctness is verified across all valid inputs.

#### Acceptance Criteria

1. THE Token_Service SHALL have property tests for JWT round-trip consistency
2. THE Token_Service SHALL have property tests for DPoP validation rules
3. THE Token_Service SHALL have property tests for refresh token rotation
4. THE Token_Service SHALL have property tests for JWK thumbprint computation
5. THE Token_Service SHALL run minimum 100 iterations per property test
6. THE Token_Service SHALL use proptest 1.5 for all property-based tests
7. FOR ALL property tests, the test SHALL reference the design document property number

### Requirement 12: Security Hardening

**User Story:** As a security engineer, I want the Token Service hardened against common attacks, so that the authentication system is secure.

#### Acceptance Criteria

1. THE Token_Service SHALL use constant-time comparison for all secret comparisons
2. THE Token_Service SHALL validate all inputs before processing
3. THE Token_Service SHALL NOT log sensitive data (tokens, secrets, keys)
4. THE Token_Service SHALL use secure random generation for all tokens
5. THE Token_Service SHALL encrypt cached data using AES-256-GCM via rust-common::CacheClient
6. THE Token_Service SHALL implement rate limiting for token endpoints
