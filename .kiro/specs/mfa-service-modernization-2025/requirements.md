# Requirements Document

## Introduction

This document specifies the requirements for modernizing the MFA (Multi-Factor Authentication) Service to state-of-the-art December 2025 standards. The modernization focuses on eliminating redundancy, centralizing logic, integrating with platform services (logging-service, cache-service), leveraging the Elixir libs (auth_platform, auth_platform_clients), and ensuring production-ready code with comprehensive test coverage.

## Glossary

- **MFA_Service**: The Multi-Factor Authentication microservice providing TOTP, WebAuthn/Passkeys, and device fingerprinting capabilities
- **TOTP**: Time-based One-Time Password algorithm (RFC 6238)
- **WebAuthn**: Web Authentication API for passwordless authentication (W3C Level 2)
- **Passkey**: Discoverable WebAuthn credentials enabling passwordless login
- **CAEP**: Continuous Access Evaluation Protocol for real-time security events
- **Cache_Service**: Platform centralized cache service (Go-based gRPC service)
- **Logging_Service**: Platform centralized logging service (.NET-based gRPC service)
- **Auth_Platform_Lib**: Elixir library providing resilience patterns, validation, security utilities
- **Auth_Platform_Clients**: Elixir library providing gRPC clients for platform services
- **Challenge**: Cryptographic random value used in WebAuthn ceremonies
- **Credential**: WebAuthn public key credential stored for authentication
- **Sign_Count**: Counter preventing authenticator cloning attacks
- **Device_Fingerprint**: Hash of device attributes for change detection

## Requirements

### Requirement 1: Platform Service Integration

**User Story:** As a platform architect, I want the MFA service to use centralized platform services for logging and caching, so that observability and caching are consistent across all microservices.

#### Acceptance Criteria

1. WHEN the MFA_Service starts, THE Application SHALL initialize Auth_Platform_Clients for Cache_Service and Logging_Service connections
2. WHEN storing WebAuthn challenges, THE MFA_Service SHALL use Cache_Service instead of direct Redis/ETS access
3. WHEN logging MFA events, THE MFA_Service SHALL use Logging_Service client with structured JSON format
4. WHEN Cache_Service is unavailable, THE MFA_Service SHALL fallback gracefully using circuit breaker pattern from Auth_Platform_Lib
5. WHEN Logging_Service is unavailable, THE MFA_Service SHALL fallback to local Logger with correlation ID preservation

### Requirement 2: Elixir Libs Integration

**User Story:** As a developer, I want the MFA service to leverage shared Elixir libraries, so that common patterns are reused and code duplication is eliminated.

#### Acceptance Criteria

1. THE MFA_Service SHALL use AuthPlatform.Security for constant-time comparison and token generation
2. THE MFA_Service SHALL use AuthPlatform.Validation for input validation with error accumulation
3. THE MFA_Service SHALL use AuthPlatform.Resilience.CircuitBreaker for external service calls
4. THE MFA_Service SHALL use AuthPlatform.Resilience.Retry for transient failure handling
5. THE MFA_Service SHALL use AuthPlatform.Observability.Logger for structured logging with correlation IDs
6. THE MFA_Service SHALL use AuthPlatform.Observability.Telemetry for metrics emission

### Requirement 3: TOTP Module Modernization

**User Story:** As a user, I want TOTP authentication to be secure and reliable, so that I can use authenticator apps for two-factor authentication.

#### Acceptance Criteria

1. WHEN generating a TOTP secret, THE Generator SHALL use :crypto.strong_rand_bytes with 160-bit minimum entropy
2. WHEN validating a TOTP code, THE Validator SHALL accept codes within ±1 time window (30 seconds each)
3. WHEN comparing TOTP codes, THE Validator SHALL use AuthPlatform.Security.constant_time_compare to prevent timing attacks
4. WHEN encrypting TOTP secrets for storage, THE Generator SHALL use AES-256-GCM authenticated encryption
5. WHEN generating provisioning URIs, THE Generator SHALL follow otpauth:// URI format per RFC 6238
6. FOR ALL valid TOTP secrets, generating a code and validating it within the time window SHALL succeed (round-trip property)

### Requirement 4: WebAuthn/Passkeys Module Modernization

**User Story:** As a user, I want to use passkeys for passwordless authentication, so that I can log in securely without remembering passwords.

#### Acceptance Criteria

1. WHEN generating WebAuthn challenges, THE Challenge module SHALL produce 32 bytes of cryptographic randomness
2. WHEN storing challenges, THE Registration and Authentication modules SHALL use Cache_Service with 5-minute TTL
3. WHEN verifying attestation, THE Registration module SHALL validate client data type, challenge, and origin
4. WHEN verifying assertion, THE Authentication module SHALL validate signature using stored public key
5. WHEN checking sign count, THE Authentication module SHALL reject counts not strictly greater than stored value
6. WHEN parsing authenticator data, THE modules SHALL correctly extract RP ID hash, flags, and sign count
7. FOR ALL WebAuthn challenges, encoding then decoding SHALL produce the original challenge (round-trip property)
8. FOR ALL valid credentials, authentication with correct signature SHALL succeed

### Requirement 5: Device Fingerprint Module Modernization

**User Story:** As a security administrator, I want to detect significant device changes, so that suspicious authentication attempts can be flagged for re-authentication.

#### Acceptance Criteria

1. WHEN computing a device fingerprint, THE Fingerprint module SHALL hash normalized attributes using SHA-256
2. WHEN comparing fingerprints, THE Fingerprint module SHALL calculate similarity as matching attributes ratio
3. WHEN similarity drops below 70%, THE Fingerprint module SHALL flag as significant change requiring re-authentication
4. WHEN extracting attributes, THE Fingerprint module SHALL handle missing headers gracefully with empty defaults
5. FOR ALL identical attribute sets, computing fingerprints SHALL produce identical hashes (determinism property)
6. FOR ALL attribute sets, comparing a fingerprint with itself SHALL return 100% similarity (reflexivity property)

### Requirement 6: CAEP Event Emission Modernization

**User Story:** As a security system, I want MFA credential changes to emit CAEP events, so that downstream systems can react to security-relevant changes.

#### Acceptance Criteria

1. WHEN a passkey is added, THE Emitter SHALL emit credential-change event with change_type "create"
2. WHEN a passkey is removed, THE Emitter SHALL emit credential-change event with change_type "delete"
3. WHEN TOTP is enabled or disabled, THE Emitter SHALL emit credential-change event with appropriate change_type
4. WHEN emitting events, THE Emitter SHALL use Logging_Service for audit logging
5. IF CAEP service is unavailable, THEN THE Emitter SHALL log the failure and continue without blocking

### Requirement 7: Passkey Management Modernization

**User Story:** As a user, I want to manage my passkeys (list, rename, delete), so that I can maintain control over my authentication methods.

#### Acceptance Criteria

1. WHEN listing passkeys, THE Management module SHALL return all credentials with device name, created date, and last used date
2. WHEN renaming a passkey, THE Management module SHALL validate the new name length (max 255 characters)
3. WHEN deleting a passkey, THE Management module SHALL require recent authentication (within 5 minutes)
4. WHEN deleting the last passkey, THE Management module SHALL verify alternative authentication method exists
5. WHEN checking deletion eligibility, THE Management module SHALL return clear reason if deletion is blocked

### Requirement 8: Cross-Device Authentication Modernization

**User Story:** As a user, I want to authenticate using my phone when logging in on a desktop, so that I can use passkeys across devices.

#### Acceptance Criteria

1. WHEN generating QR code, THE CrossDevice module SHALL encode CTAP hybrid transport data in FIDO:// URI format
2. WHEN creating cross-device session, THE CrossDevice module SHALL store session in Cache_Service with 5-minute TTL
3. WHEN completing authentication, THE CrossDevice module SHALL verify session not expired before processing
4. WHEN cross-device authentication fails, THE CrossDevice module SHALL return available fallback methods
5. WHEN checking session status, THE CrossDevice module SHALL return pending, completed, or expired state

### Requirement 9: Code Architecture Modernization

**User Story:** As a maintainer, I want the codebase to follow state-of-the-art patterns, so that it is maintainable, testable, and production-ready.

#### Acceptance Criteria

1. THE MFA_Service SHALL separate source code from test code in distinct directories
2. THE MFA_Service SHALL have no files exceeding 400 lines of code
3. THE MFA_Service SHALL eliminate all redundant challenge storage implementations (centralize in Cache_Service)
4. THE MFA_Service SHALL use consistent error handling with AuthPlatform.Errors.AppError
5. THE MFA_Service SHALL emit telemetry events for all MFA operations using AuthPlatform.Observability.Telemetry
6. THE MFA_Service SHALL use Elixir 1.17+ with OTP 27+ for latest language features

### Requirement 10: Test Coverage and Quality

**User Story:** As a quality engineer, I want comprehensive test coverage, so that the service is reliable and regressions are caught early.

#### Acceptance Criteria

1. THE MFA_Service SHALL have unit tests for all public functions
2. THE MFA_Service SHALL have property-based tests for cryptographic operations (TOTP, WebAuthn challenges)
3. THE MFA_Service SHALL have property-based tests for round-trip operations (encode/decode, encrypt/decrypt)
4. THE MFA_Service SHALL have integration tests for platform service clients
5. THE MFA_Service SHALL achieve minimum 80% code coverage
6. THE MFA_Service SHALL pass all tests before deployment (zero failing tests)

### Requirement 11: Security Hardening

**User Story:** As a security officer, I want the MFA service to follow security best practices, so that authentication is protected against attacks.

#### Acceptance Criteria

1. THE MFA_Service SHALL use constant-time comparison for all secret comparisons
2. THE MFA_Service SHALL use cryptographically secure random number generation for all tokens and challenges
3. THE MFA_Service SHALL validate all inputs using AuthPlatform.Validation before processing
4. THE MFA_Service SHALL sanitize all log output to prevent PII leakage using AuthPlatform.Observability.Logger
5. THE MFA_Service SHALL use parameterized queries for all database operations
6. IF invalid input is detected, THEN THE MFA_Service SHALL return appropriate error without exposing internal details

### Requirement 12: Performance Requirements

**User Story:** As a platform operator, I want the MFA service to meet latency SLOs, so that authentication does not degrade user experience.

#### Acceptance Criteria

1. WHEN generating registration options, THE MFA_Service SHALL complete within 200ms at p99 latency
2. WHEN generating authentication options, THE MFA_Service SHALL complete within 100ms at p99 latency
3. WHEN validating TOTP codes, THE MFA_Service SHALL complete within 50ms at p99 latency
4. WHEN verifying WebAuthn assertions, THE MFA_Service SHALL complete within 150ms at p99 latency
5. THE MFA_Service SHALL emit latency metrics via telemetry for monitoring
