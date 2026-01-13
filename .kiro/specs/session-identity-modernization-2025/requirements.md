# Requirements Document

## Introduction

This document specifies the requirements for modernizing the Session Identity Core service to state-of-the-art December 2025 standards. The modernization focuses on eliminating redundancy, centralizing logic, integrating with platform services (Cache Service, Logging Service), leveraging shared libraries, and ensuring OAuth 2.1/RFC 9700 compliance with CAEP/SSF support.

## Glossary

- **Session_Identity_Service**: The Elixir-based microservice managing user sessions, OAuth 2.1 flows, and identity operations
- **Cache_Service**: The centralized platform cache service (`platform/cache-service`) providing Redis-backed caching with encryption
- **Logging_Service**: The centralized platform logging service (`platform/logging-service`) providing structured logging with OpenTelemetry
- **Auth_Platform_Libs**: Shared Elixir libraries (`libs/elixir/apps`) providing common functionality
- **OAuth21_Module**: The OAuth 2.1 implementation module compliant with RFC 9700 security best practices
- **PKCE_Module**: Proof Key for Code Exchange implementation using S256 method only
- **Event_Store**: Append-only event storage for session event sourcing
- **Risk_Scorer**: Adaptive authentication risk scoring engine
- **CAEP_Emitter**: Continuous Access Evaluation Protocol event emitter for SSF integration
- **Session_Manager**: GenServer managing session lifecycle and state
- **Session_Store**: Redis-backed session persistence layer
- **Session_Serializer**: JSON serialization with round-trip guarantee

## Requirements

### Requirement 1: Platform Service Integration

**User Story:** As a platform architect, I want the Session Identity Service to use centralized platform services, so that we eliminate redundant implementations and ensure consistent behavior across the platform.

#### Acceptance Criteria

1. WHEN the Session_Identity_Service stores session data, THE Session_Store SHALL use the Cache_Service client from Auth_Platform_Libs instead of direct Redix calls
2. WHEN the Session_Identity_Service logs events, THE service SHALL use the Logging_Service client from Auth_Platform_Libs instead of direct Logger calls
3. WHEN the Cache_Service is unavailable, THE Session_Store SHALL implement circuit breaker fallback using Auth_Platform resilience patterns
4. WHEN logging to the Logging_Service fails, THE service SHALL fallback to local Logger with structured JSON format
5. THE Session_Identity_Service SHALL use Auth_Platform.Security for all cryptographic operations including constant-time comparison and token generation
6. THE Session_Identity_Service SHALL use Auth_Platform.Validation for all input validation with error accumulation

### Requirement 2: Code Deduplication and Centralization

**User Story:** As a developer, I want all duplicated logic eliminated and centralized, so that the codebase is maintainable and changes propagate consistently.

#### Acceptance Criteria

1. THE Session_Identity_Service SHALL have exactly one implementation of session serialization logic
2. THE Session_Identity_Service SHALL have exactly one implementation of datetime parsing/formatting
3. THE Session_Identity_Service SHALL have exactly one implementation of UUID generation
4. THE Session_Identity_Service SHALL have exactly one implementation of Redis key prefixing
5. WHEN session data is converted to/from maps, THE Session_Serializer SHALL be the single source of truth
6. THE Session_Identity_Service SHALL centralize all TTL calculations in a single module
7. THE Session_Identity_Service SHALL centralize all error types in a dedicated errors module

### Requirement 3: OAuth 2.1 RFC 9700 Compliance

**User Story:** As a security engineer, I want the OAuth implementation to comply with RFC 9700 (OAuth 2.0 Security Best Current Practice), so that the service meets December 2025 security standards.

#### Acceptance Criteria

1. THE OAuth21_Module SHALL reject implicit grant (response_type=token) with error "unsupported_response_type"
2. THE OAuth21_Module SHALL reject resource owner password credentials grant with error "unsupported_grant_type"
3. THE OAuth21_Module SHALL require PKCE for all clients including confidential clients
4. THE PKCE_Module SHALL only support S256 code_challenge_method and reject "plain" method
5. THE OAuth21_Module SHALL enforce exact redirect_uri matching without pattern matching
6. THE OAuth21_Module SHALL implement refresh token rotation on every use
7. WHEN a code_verifier is provided, THE PKCE_Module SHALL verify using constant-time comparison
8. THE OAuth21_Module SHALL validate code_verifier length (43-128 characters) and character set

### Requirement 4: Session Management Security

**User Story:** As a security engineer, I want session management to follow 2025 security best practices, so that sessions are protected against common attacks.

#### Acceptance Criteria

1. THE Session_Manager SHALL generate session tokens with at least 256 bits of entropy using crypto.strong_rand_bytes
2. THE Session_Store SHALL encrypt sensitive session data before storage using Cache_Service encryption
3. THE Session_Manager SHALL bind sessions to device fingerprint and IP address
4. WHEN a session is created, THE Event_Store SHALL record a SessionCreated event with correlation_id
5. WHEN a session is terminated, THE Event_Store SHALL record a SessionInvalidated event with reason
6. THE Session_Manager SHALL implement session fixation protection by regenerating session ID on privilege escalation
7. THE Session_Store SHALL use namespaced Redis keys with "session:" prefix for isolation

### Requirement 5: Risk-Based Adaptive Authentication

**User Story:** As a security engineer, I want risk-based authentication to adapt to threat signals, so that high-risk sessions require step-up authentication.

#### Acceptance Criteria

1. THE Risk_Scorer SHALL calculate risk scores in the range [0.0, 1.0]
2. WHEN risk_score >= 0.7, THE Risk_Scorer SHALL require step-up authentication
3. WHEN risk_score >= 0.9, THE Risk_Scorer SHALL require WebAuthn or TOTP factors
4. THE Risk_Scorer SHALL consider IP reputation, device fingerprint, failed attempts, time of day, and location
5. WHEN a known device is detected, THE Risk_Scorer SHALL reduce the device_risk factor to 0.0
6. WHEN failed_attempts >= 5, THE Risk_Scorer SHALL set behavior_risk to 0.9
7. THE Risk_Scorer SHALL emit RiskScoreUpdated events to the Event_Store

### Requirement 6: OIDC ID Token Generation

**User Story:** As an identity architect, I want ID tokens to comply with OpenID Connect Core 1.0, so that relying parties can verify user identity.

#### Acceptance Criteria

1. THE IdToken_Module SHALL include all required claims: sub, iss, aud, exp, iat
2. WHEN a nonce is provided in the authorization request, THE IdToken_Module SHALL include it in the ID token
3. THE IdToken_Module SHALL set exp claim based on configurable TTL (default 3600 seconds)
4. THE IdToken_Module SHALL support optional claims: auth_time, acr, amr, azp
5. THE IdToken_Module SHALL validate that all required claims are present before signing
6. THE IdToken_Module SHALL use Joken for JWT signing with RS256 algorithm

### Requirement 7: Event Sourcing and Audit Trail

**User Story:** As a compliance officer, I want complete audit trail of session events, so that we can investigate security incidents and meet compliance requirements.

#### Acceptance Criteria

1. THE Event_Store SHALL append events with monotonically increasing sequence numbers
2. THE Event_Store SHALL support event replay for aggregate reconstruction
3. THE Event_Store SHALL include correlation_id and causation_id for event tracing
4. THE Event_Store SHALL support schema versioning with migration capability
5. WHEN events are serialized, THE Event_Store SHALL use ISO 8601 UTC timestamps
6. THE Event_Store SHALL support loading aggregates from snapshots for performance
7. THE Event_Store SHALL emit events for: SessionCreated, SessionRefreshed, SessionInvalidated, DeviceBound, MfaVerified, RiskScoreUpdated

### Requirement 8: CAEP/SSF Integration

**User Story:** As a security architect, I want the service to emit CAEP events via SSF, so that relying parties can implement continuous access evaluation.

#### Acceptance Criteria

1. WHEN a user logs out, THE CAEP_Emitter SHALL emit a session-revoked event
2. WHEN an admin terminates a session, THE CAEP_Emitter SHALL emit a session-revoked event with admin context
3. WHEN a security policy violation occurs, THE CAEP_Emitter SHALL emit a session-revoked event with reason
4. THE CAEP_Emitter SHALL format events according to SSF specification with subject, event_timestamp, and reason_admin
5. WHEN CAEP transmission fails, THE CAEP_Emitter SHALL log the failure and continue operation
6. THE CAEP_Emitter SHALL support configurable enable/disable via application config

### Requirement 9: Architecture Modernization

**User Story:** As a platform architect, I want the service architecture to follow 2025 best practices, so that the codebase is maintainable and testable.

#### Acceptance Criteria

1. THE Session_Identity_Service SHALL separate source code from test code in distinct directories
2. THE Session_Identity_Service SHALL use Phoenix 1.8+ with LiveView 1.1+ patterns
3. THE Session_Identity_Service SHALL use Ecto 3.12+ for database operations
4. THE Session_Identity_Service SHALL use gRPC 0.7+ for inter-service communication
5. THE Session_Identity_Service SHALL implement health checks for Kubernetes readiness/liveness probes
6. THE Session_Identity_Service SHALL expose Prometheus metrics via telemetry
7. THE Session_Identity_Service SHALL support OpenTelemetry tracing with W3C Trace Context

### Requirement 10: Property-Based Testing

**User Story:** As a quality engineer, I want comprehensive property-based tests, so that correctness properties are verified across all valid inputs.

#### Acceptance Criteria

1. THE test suite SHALL include property tests for session serialization round-trip
2. THE test suite SHALL include property tests for PKCE S256 verification correctness
3. THE test suite SHALL include property tests for risk score bounds [0.0, 1.0]
4. THE test suite SHALL include property tests for ID token required claims completeness
5. THE test suite SHALL include property tests for event sequence number monotonicity
6. THE test suite SHALL use StreamData for property-based testing with minimum 100 iterations
7. THE test suite SHALL achieve minimum 80% code coverage on core modules

### Requirement 11: Error Handling Centralization

**User Story:** As a developer, I want centralized error handling, so that error responses are consistent and informative.

#### Acceptance Criteria

1. THE Session_Identity_Service SHALL define all error types in a centralized Errors module
2. WHEN an OAuth error occurs, THE service SHALL return RFC 6749 compliant error responses
3. WHEN a session error occurs, THE service SHALL return structured error with error_code and error_description
4. THE service SHALL use Result pattern ({:ok, value} | {:error, reason}) consistently
5. THE service SHALL log errors with correlation_id for traceability
6. THE service SHALL never expose internal error details to external clients

### Requirement 12: Configuration Management

**User Story:** As a DevOps engineer, I want externalized configuration, so that the service can be configured without code changes.

#### Acceptance Criteria

1. THE Session_Identity_Service SHALL read all configuration from environment variables
2. THE Session_Identity_Service SHALL provide sensible defaults for all optional configuration
3. THE Session_Identity_Service SHALL validate configuration at startup and fail fast on invalid config
4. THE Session_Identity_Service SHALL support configuration for: GRPC_PORT, REDIS_HOST, REDIS_PORT, DATABASE_URL, SESSION_TTL, CAEP_ENABLED, CAEP_TRANSMITTER_URL
5. THE Session_Identity_Service SHALL log configuration values (excluding secrets) at startup
