# Requirements Document

## Introduction

This specification defines the modernization of the User Service microservice to state-of-the-art December 2025 standards. The modernization focuses on:
- Upgrading to Spring Boot 3.4+ with Java 21 virtual threads
- Integrating with platform services (cache-service, logging-service) via gRPC
- Eliminating redundant code and centralizing cross-cutting concerns
- Implementing distributed rate limiting via cache-service
- Enhancing security with OWASP 2025 compliance
- Restructuring test organization (separating unit/property/integration tests)

## Glossary

- **User_Service**: The microservice responsible for user registration, email verification, and profile management
- **Cache_Service**: Platform gRPC service for distributed caching (platform/cache-service)
- **Logging_Service**: Platform gRPC service for centralized audit and security logging (platform/logging-service)
- **Rate_Limiter**: Component that enforces request rate limits using distributed cache
- **Outbox_Publisher**: Component that publishes domain events via transactional outbox pattern
- **Token_Hasher**: Component that generates and hashes verification tokens using SHA-256
- **Password_Service**: Component that hashes and verifies passwords using Argon2id
- **Email_Normalizer**: Component that normalizes email addresses to lowercase
- **Validation_Service**: Centralized component for input validation (email, password, display name)
- **Virtual_Threads**: Java 21 lightweight threads for high-concurrency I/O operations
- **gRPC_Client**: Client for inter-service communication using Protocol Buffers

## Requirements

### Requirement 1: Platform Service Integration

**User Story:** As a platform architect, I want the User Service to integrate with centralized platform services, so that caching and logging are consistent across all microservices.

#### Acceptance Criteria

1. WHEN the User_Service starts, THE gRPC_Client SHALL establish connections to Cache_Service and Logging_Service
2. WHEN Cache_Service is unavailable, THE User_Service SHALL fall back to local Caffeine cache with circuit breaker protection
3. WHEN Logging_Service is unavailable, THE User_Service SHALL fall back to local structured JSON logging
4. WHEN an audit event occurs (registration, verification, profile update), THE Logging_Service_Client SHALL send the event asynchronously
5. WHEN a security event occurs (rate limit exceeded, invalid token), THE Logging_Service_Client SHALL send the event with IP masking

### Requirement 2: Distributed Rate Limiting

**User Story:** As a security engineer, I want rate limiting to be distributed across service instances, so that attackers cannot bypass limits by hitting different pods.

#### Acceptance Criteria

1. WHEN a registration request arrives, THE Rate_Limiter SHALL check the distributed cache for IP-based limits
2. WHEN a resend verification request arrives, THE Rate_Limiter SHALL check both email-based and IP-based limits in the distributed cache
3. WHEN rate limit data is stored, THE Cache_Service_Client SHALL use namespace "user-service:ratelimit" with appropriate TTL
4. IF Cache_Service is unavailable, THEN THE Rate_Limiter SHALL use local in-memory rate limiting as fallback
5. WHEN rate limit is exceeded, THE User_Service SHALL return HTTP 429 with Retry-After header

### Requirement 3: Virtual Threads Integration

**User Story:** As a platform engineer, I want the User Service to use Java 21 virtual threads, so that it can handle high concurrency with minimal resource usage.

#### Acceptance Criteria

1. THE User_Service SHALL enable virtual threads for all HTTP request handling
2. WHEN making gRPC calls to platform services, THE User_Service SHALL execute them on virtual threads
3. WHEN processing outbox events, THE Outbox_Dispatcher SHALL use virtual threads for Kafka publishing
4. THE User_Service SHALL configure HikariCP connection pool optimized for virtual threads

### Requirement 4: Centralized Validation

**User Story:** As a developer, I want all input validation logic centralized, so that validation rules are consistent and not duplicated.

#### Acceptance Criteria

1. THE Validation_Service SHALL provide a single entry point for all input validation
2. WHEN validating email, THE Validation_Service SHALL check format, disposable domains, and normalization
3. WHEN validating password, THE Validation_Service SHALL enforce minimum length, complexity, and maximum length
4. WHEN validating display name, THE Validation_Service SHALL check length limits and sanitize HTML/script content
5. WHEN validation fails, THE Validation_Service SHALL return structured error with field name and message

### Requirement 5: User Registration

**User Story:** As a new user, I want to register an account with email verification, so that I can access the platform securely.

#### Acceptance Criteria

1. WHEN a valid registration request is received, THE User_Service SHALL create a user with PENDING_EMAIL status
2. WHEN a user is created, THE Password_Service SHALL hash the password using Argon2id with OWASP-recommended parameters
3. WHEN a user is created, THE User_Service SHALL generate a verification token and store its SHA-256 hash
4. WHEN registration completes, THE Outbox_Publisher SHALL publish UserRegistered and EmailVerificationRequested events
5. IF email already exists, THEN THE User_Service SHALL return HTTP 409 with EMAIL_EXISTS error code
6. WHEN registration succeeds, THE User_Service SHALL return HTTP 201 with user ID and status

### Requirement 6: Email Verification

**User Story:** As a registered user, I want to verify my email address, so that my account becomes active.

#### Acceptance Criteria

1. WHEN a valid verification token is provided, THE User_Service SHALL activate the user account
2. WHEN a token is verified, THE User_Service SHALL mark the token as used with timestamp
3. IF the token is expired, THEN THE User_Service SHALL return HTTP 400 with EXPIRED_TOKEN error
4. IF the token is already used, THEN THE User_Service SHALL return HTTP 400 with ALREADY_USED error
5. IF the token is invalid, THEN THE User_Service SHALL return HTTP 400 with INVALID_TOKEN error
6. WHEN email is verified, THE Outbox_Publisher SHALL publish UserEmailVerified event

### Requirement 7: Resend Verification Email

**User Story:** As a user who didn't receive verification email, I want to request a new one, so that I can complete registration.

#### Acceptance Criteria

1. WHEN a resend request is received, THE User_Service SHALL always return HTTP 202 to prevent email enumeration
2. WHEN the email exists and is unverified, THE User_Service SHALL invalidate previous tokens and create a new one
3. WHEN a new token is created, THE Outbox_Publisher SHALL publish EmailVerificationRequested event
4. THE Rate_Limiter SHALL enforce stricter limits on resend requests (3 per hour per email)

### Requirement 8: Profile Management

**User Story:** As an authenticated user, I want to view and update my profile, so that I can manage my account information.

#### Acceptance Criteria

1. WHEN an authenticated user requests their profile, THE User_Service SHALL return profile data from database
2. WHEN an authenticated user updates their display name, THE Validation_Service SHALL validate and sanitize the input
3. WHEN profile is updated, THE User_Service SHALL update the updatedAt timestamp
4. IF the user is not found, THEN THE User_Service SHALL return HTTP 404 with USER_NOT_FOUND error
5. WHEN profile is retrieved, THE User_Service SHALL cache the result in Cache_Service with 5-minute TTL

### Requirement 9: Security and Observability

**User Story:** As a security engineer, I want comprehensive audit logging and security controls, so that I can monitor and investigate incidents.

#### Acceptance Criteria

1. WHEN any API request is processed, THE User_Service SHALL include correlation ID in logs and responses
2. WHEN logging user data, THE User_Service SHALL mask email addresses and IP addresses
3. WHEN a security event occurs, THE Logging_Service_Client SHALL log with SECURITY level and event metadata
4. THE User_Service SHALL expose Prometheus metrics for request latency, error rates, and rate limit hits
5. THE User_Service SHALL propagate OpenTelemetry trace context to platform services

### Requirement 10: Error Handling

**User Story:** As an API consumer, I want consistent error responses, so that I can handle errors programmatically.

#### Acceptance Criteria

1. WHEN an error occurs, THE User_Service SHALL return RFC 7807 Problem Detail format
2. WHEN returning errors, THE User_Service SHALL include error code, timestamp, and correlation ID
3. WHEN validation fails, THE User_Service SHALL return HTTP 400 with field-level error details
4. WHEN an unexpected error occurs, THE User_Service SHALL return HTTP 500 without exposing internal details
5. THE User_Service SHALL log all errors with appropriate severity level

### Requirement 11: Test Organization

**User Story:** As a developer, I want tests organized by type and mirroring source structure, so that I can easily find and maintain tests.

#### Acceptance Criteria

1. THE test directory SHALL separate unit tests, property tests, and integration tests into distinct packages
2. WHEN a source file exists, THE corresponding unit test file SHALL mirror its package path
3. THE property tests SHALL use jqwik with minimum 100 iterations per property
4. THE integration tests SHALL use Testcontainers for PostgreSQL and Kafka
5. WHEN running tests, THE build system SHALL support running each test type independently

### Requirement 12: Code Deduplication

**User Story:** As a maintainer, I want zero code duplication, so that changes only need to be made in one place.

#### Acceptance Criteria

1. THE User_Service SHALL have a single IP masking utility used by all components
2. THE User_Service SHALL have a single correlation ID management utility
3. THE User_Service SHALL have a single MDC (Mapped Diagnostic Context) management utility
4. WHEN validation logic is needed, THE component SHALL delegate to Validation_Service
5. WHEN cache operations are needed, THE component SHALL delegate to Cache_Service_Client
