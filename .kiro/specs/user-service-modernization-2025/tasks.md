# Implementation Plan: User Service Modernization 2025

## Overview

This implementation plan modernizes the User Service to December 2025 state-of-the-art standards. Tasks are organized to build incrementally, with property tests validating correctness at each stage. The implementation uses platform services (cache-service, logging-service) via gRPC and eliminates all code redundancy.

## Tasks

- [x] 1. Project Setup and Dependencies
  - [x] 1.1 Update build.gradle with Spring Boot 3.4+, Java 21, jqwik 1.9+, gRPC dependencies
    - Add spring-boot-starter-web, spring-boot-starter-data-jpa, spring-boot-starter-validation
    - Add grpc-netty-shaded, grpc-protobuf, grpc-stub for gRPC clients
    - Add jqwik, junit-jupiter, mockito, assertj, testcontainers for testing
    - Add caffeine for local cache fallback
    - Add resilience4j-circuitbreaker for circuit breaker
    - Configure virtual threads: spring.threads.virtual.enabled=true
    - _Requirements: 3.1, 3.4_

  - [x] 1.2 Create test directory structure
    - Create src/test/java/com/auth/userservice/unit/
    - Create src/test/java/com/auth/userservice/property/
    - Create src/test/java/com/auth/userservice/integration/
    - Mirror source package structure in each test directory
    - _Requirements: 11.1, 11.2_

- [x] 2. Centralized Security Utilities
  - [x] 2.1 Implement SecurityUtils component
    - Create src/main/java/com/auth/userservice/shared/security/SecurityUtils.java
    - Implement maskIp(String ip) - replace last octet with ***
    - Implement maskEmail(String email) - mask characters after first 2 before @
    - Implement getOrCreateCorrelationId(String provided)
    - Implement setMdcContext/clearMdcContext for MDC management
    - _Requirements: 9.1, 9.2, 12.1, 12.2, 12.3_

  - [x]* 2.2 Write property test for IP masking
    - **Property 2: Sensitive Data Masking Consistency (IP)**
    - *For any* valid IPv4 address, maskIp SHALL replace the last octet with ***
    - **Validates: Requirements 1.5, 9.2**

  - [x]* 2.3 Write property test for email masking
    - **Property 2: Sensitive Data Masking Consistency (Email)**
    - *For any* valid email, maskEmail SHALL mask characters after first 2 before @
    - **Validates: Requirements 1.5, 9.2**

- [x] 3. Centralized Validation Service
  - [x] 3.1 Implement ValidationService
    - Create src/main/java/com/auth/userservice/shared/validation/ValidationService.java
    - Create ValidationResult and FieldError records
    - Implement validateEmail - format, disposable domains, normalization
    - Implement validatePassword - min 8, max 128, complexity (upper, lower, digit, special)
    - Implement validateDisplayName - min 2, max 50, HTML/script sanitization
    - Implement validateRegistration - combines all validations
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x]* 3.2 Write property test for email validation
    - **Property 11: Validation Service Completeness (Email)**
    - *For any* string input, validateEmail SHALL return ValidationResult with valid=true for valid emails or valid=false with non-empty errors
    - **Validates: Requirements 4.2, 4.5**

  - [x]* 3.3 Write property test for password validation
    - **Property 11: Validation Service Completeness (Password)**
    - *For any* string input, validatePassword SHALL return ValidationResult with valid=true for compliant passwords or valid=false with non-empty errors
    - **Validates: Requirements 4.3, 4.5**

  - [x]* 3.4 Write property test for display name validation
    - **Property 11: Validation Service Completeness (DisplayName)**
    - *For any* string input, validateDisplayName SHALL return ValidationResult and sanitize HTML/script content
    - **Validates: Requirements 4.4, 4.5**

- [x] 4. Checkpoint - Shared Components
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Password and Token Services
  - [x] 5.1 Implement PasswordService with Argon2id
    - Create src/main/java/com/auth/userservice/shared/crypto/PasswordService.java
    - Use Bouncy Castle Argon2id with OWASP parameters (memory=19456KB, iterations=2, parallelism=1)
    - Implement hash(String password) returning Argon2id hash string
    - Implement verify(String password, String hash) returning boolean
    - _Requirements: 5.2_

  - [x]* 5.2 Write property test for password hash format
    - **Property 5: Password Hash Format Compliance**
    - *For any* password, hash SHALL return string starting with $argon2id$ with algorithm parameters
    - **Validates: Requirements 5.2**

  - [x]* 5.3 Write property test for password hash round-trip
    - **Property 6: Password Hash Round-Trip Verification**
    - *For any* password, hash then verify with same password SHALL return true, different password SHALL return false
    - **Validates: Requirements 5.2**

  - [x] 5.4 Implement TokenHasher with SHA-256
    - Create src/main/java/com/auth/userservice/shared/crypto/TokenHasher.java
    - Implement generateToken() returning secure random 32-byte token as hex
    - Implement hash(String token) returning 64-char hex SHA-256 hash
    - Implement verify(String token, String hash) returning boolean
    - _Requirements: 5.3_

  - [x]* 5.5 Write property test for token hash determinism
    - **Property 7: Token Hash Determinism**
    - *For any* token, hashing multiple times SHALL produce same 64-char hex hash, different tokens SHALL produce different hashes
    - **Validates: Requirements 5.3**

  - [x]* 5.6 Write property test for token round-trip
    - **Property 8: Token Verification Round-Trip**
    - *For any* generated token, hash then verify SHALL return true
    - **Validates: Requirements 5.3, 6.1**


- [x] 6. Platform Service Clients
  - [x] 6.1 Implement CacheServiceClient with circuit breaker
    - Create src/main/java/com/auth/userservice/infrastructure/cache/CacheServiceClient.java
    - Configure gRPC channel to platform/cache-service
    - Implement get(namespace, key), set(namespace, key, value, ttl), delete(namespace, key)
    - Add Caffeine local cache as fallback
    - Configure Resilience4j circuit breaker (failureThreshold=5, waitDuration=30s)
    - _Requirements: 1.1, 1.2, 2.3_

  - [x] 6.2 Implement LoggingServiceClient with fallback
    - Create src/main/java/com/auth/userservice/infrastructure/logging/LoggingServiceClient.java
    - Configure async gRPC channel to platform/logging-service
    - Implement logAudit(AuditEvent) and logSecurity(SecurityEvent)
    - Use SecurityUtils for IP/email masking in security events
    - Fallback to structured JSON local logging when unavailable
    - _Requirements: 1.1, 1.3, 1.4, 1.5_

  - [x]* 6.3 Write property test for platform service fallback
    - **Property 1: Platform Service Fallback Resilience**
    - *For any* cache/logging operation when service unavailable, operation SHALL complete using local fallback
    - **Validates: Requirements 1.2, 1.3, 2.4**

- [x] 7. Distributed Rate Limiting
  - [x] 7.1 Implement RateLimitService
    - Create src/main/java/com/auth/userservice/domain/ratelimit/RateLimitService.java
    - Use namespace "user-service:ratelimit" for all keys
    - Implement checkRegistrationLimit(ip) - 10 per hour per IP
    - Implement checkResendLimit(email, ip) - 3 per hour per email, 10 per hour per IP
    - Implement checkVerifyLimit(ip) - 20 per hour per IP
    - Throw RateLimitedException with retryAfter when exceeded
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 7.4_

  - [x]* 7.2 Write property test for rate limit namespace
    - **Property 4: Rate Limit Namespace Consistency**
    - *For any* rate limit key, it SHALL be prefixed with "user-service:ratelimit:"
    - **Validates: Requirements 2.3**

  - [x]* 7.3 Write property test for rate limit enforcement
    - **Property 3: Rate Limit Enforcement**
    - *For any* rate-limited operation when limit exceeded, SHALL throw RateLimitedException with valid retryAfter
    - **Validates: Requirements 2.5, 7.4**

- [x] 8. Checkpoint - Infrastructure Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Domain Entities and Repository
  - [x] 9.1 Implement User entity and UserStatus enum
    - Create src/main/java/com/auth/userservice/domain/user/User.java
    - Create src/main/java/com/auth/userservice/domain/user/UserStatus.java (PENDING_EMAIL, ACTIVE, SUSPENDED)
    - Use JPA annotations with UUID primary key
    - _Requirements: 5.1_

  - [x] 9.2 Implement EmailVerificationToken entity
    - Create src/main/java/com/auth/userservice/domain/verification/EmailVerificationToken.java
    - Store tokenHash (64-char hex), expiresAt, usedAt
    - _Requirements: 5.3, 6.2_

  - [x] 9.3 Implement OutboxEvent entity
    - Create src/main/java/com/auth/userservice/domain/outbox/OutboxEvent.java
    - Store aggregateType, aggregateId, eventType, payload (JSONB), processedAt
    - _Requirements: 5.4_

  - [x] 9.4 Implement UserRepository
    - Create src/main/java/com/auth/userservice/domain/user/UserRepository.java
    - Add findByEmail(String email) method
    - _Requirements: 5.5, 8.1_

- [x] 10. Error Handling
  - [x] 10.1 Implement exception hierarchy
    - Create src/main/java/com/auth/userservice/shared/exception/UserServiceException.java (sealed)
    - Create EmailExistsException, InvalidTokenException, ExpiredTokenException, AlreadyUsedException
    - Create UserNotFoundException, RateLimitedException, ValidationException
    - Each exception provides errorCode and httpStatus
    - _Requirements: 10.1, 10.2_

  - [x] 10.2 Implement ProblemDetail record and GlobalExceptionHandler
    - Create src/main/java/com/auth/userservice/api/error/ProblemDetail.java
    - Create src/main/java/com/auth/userservice/api/error/GlobalExceptionHandler.java
    - Map all UserServiceException subclasses to RFC 7807 responses
    - Include correlationId, timestamp, errorCode in all responses
    - Mask internal details for 500 errors
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

  - [x]* 10.3 Write property test for RFC 7807 compliance
    - **Property 14: RFC 7807 Error Response Compliance**
    - *For any* UserServiceException, mapped ProblemDetail SHALL contain type, title, status, detail, instance, timestamp, correlationId, errorCode
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.4**

- [x] 11. Correlation ID Filter
  - [x] 11.1 Implement CorrelationIdFilter
    - Create src/main/java/com/auth/userservice/api/filter/CorrelationIdFilter.java
    - Extract X-Correlation-ID from request or generate new UUID
    - Set MDC context using SecurityUtils
    - Add X-Correlation-ID to response headers
    - Clear MDC context after request
    - _Requirements: 9.1_

  - [x] 11.2 Write property test for correlation ID propagation
    - **Property 15: Correlation ID Propagation**
    - *For any* request, response SHALL include X-Correlation-ID header matching MDC context
    - **Validates: Requirements 9.1**


- [x] 12. Checkpoint - Domain and Error Handling
  - Ensure all tests pass, ask the user if questions arise.

- [x] 13. Registration Service
  - [x] 13.1 Implement RegistrationService
    - Create src/main/java/com/auth/userservice/domain/registration/RegistrationService.java
    - Delegate validation to ValidationService
    - Check rate limit via RateLimitService
    - Check email uniqueness, throw EmailExistsException if exists
    - Hash password via PasswordService
    - Create User with PENDING_EMAIL status
    - Generate token, hash via TokenHasher, create EmailVerificationToken
    - Publish UserRegistered and EmailVerificationRequested events to outbox
    - Log audit event via LoggingServiceClient
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 12.4, 12.5_

  - [x]* 13.2 Write property test for registration state transition
    - **Property 9: User Registration State Transition**
    - *For any* valid registration, user SHALL have PENDING_EMAIL status, emailVerified=false, and both events published
    - **Validates: Requirements 5.1, 5.4, 5.6**

  - [x]* 13.3 Write property test for duplicate email rejection
    - **Property 10: Duplicate Email Rejection**
    - *For any* registration with existing email, SHALL throw EmailExistsException
    - **Validates: Requirements 5.5**

- [x] 14. Email Verification Service
  - [x] 14.1 Implement EmailVerificationService
    - Create src/main/java/com/auth/userservice/domain/verification/EmailVerificationService.java
    - Check rate limit via RateLimitService
    - Hash provided token, lookup by tokenHash
    - Validate token: throw InvalidTokenException if not found, ExpiredTokenException if expired, AlreadyUsedException if used
    - Mark token as used with timestamp
    - Update user to ACTIVE status, emailVerified=true
    - Publish UserEmailVerified event to outbox
    - Log audit event via LoggingServiceClient
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6_

  - [x]* 14.2 Write property test for email verification state transition
    - **Property 12: Email Verification State Transition**
    - *For any* valid token, user SHALL transition to ACTIVE, token marked used, event published
    - **Validates: Requirements 6.1, 6.2, 6.6**

- [x] 15. Resend Verification Service
  - [x] 15.1 Implement ResendVerificationService
    - Create src/main/java/com/auth/userservice/domain/verification/ResendVerificationService.java
    - Check rate limit via RateLimitService (stricter: 3 per hour per email)
    - Always return success (HTTP 202) to prevent email enumeration
    - If email exists and unverified: invalidate previous tokens, create new token, publish event
    - Log audit event via LoggingServiceClient
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [x]* 15.2 Write property test for resend idempotency
    - **Property 13: Resend Verification Idempotency**
    - *For any* resend request, SHALL return success regardless of email existence
    - **Validates: Requirements 7.1, 7.2, 7.3**

- [x] 16. Profile Service
  - [x] 16.1 Implement ProfileService
    - Create src/main/java/com/auth/userservice/domain/profile/ProfileService.java
    - Implement getProfile(userId) - check cache first, then database, cache result with 5-min TTL
    - Implement updateDisplayName(userId, displayName) - validate via ValidationService, update, invalidate cache
    - Throw UserNotFoundException if user not found
    - Log audit event via LoggingServiceClient
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 12.4, 12.5_

  - [x]* 16.2 Write property test for profile cache consistency
    - **Property 16: Profile Cache Consistency**
    - *For any* profile retrieval, result SHALL be cached; updates SHALL invalidate cache
    - **Validates: Requirements 8.3, 8.5**

- [x] 17. Checkpoint - Domain Services
  - Ensure all tests pass, ask the user if questions arise.

- [x] 18. API Controllers
  - [x] 18.1 Implement UserController
    - Create src/main/java/com/auth/userservice/api/controller/UserController.java
    - POST /api/v1/users - registration endpoint
    - Delegate to RegistrationService
    - Return 201 with user ID and status
    - _Requirements: 5.6_

  - [x] 18.2 Implement EmailVerificationController
    - Create src/main/java/com/auth/userservice/api/controller/EmailVerificationController.java
    - POST /api/v1/users/verify - verify token endpoint
    - POST /api/v1/users/resend-verification - resend endpoint
    - Delegate to EmailVerificationService and ResendVerificationService
    - _Requirements: 6.1, 7.1_

  - [x] 18.3 Implement MeController
    - Create src/main/java/com/auth/userservice/api/controller/MeController.java
    - GET /api/v1/me - get profile endpoint
    - PATCH /api/v1/me - update profile endpoint
    - Delegate to ProfileService
    - _Requirements: 8.1, 8.2_

  - [x] 18.4 Implement HealthController
    - Create src/main/java/com/auth/userservice/api/controller/HealthController.java
    - GET /health - basic health check
    - GET /health/ready - readiness check (database, cache, logging service)
    - _Requirements: 1.1_

- [x] 19. Outbox Publisher
  - [x] 19.1 Implement OutboxPublisher
    - Create src/main/java/com/auth/userservice/infrastructure/outbox/OutboxPublisher.java
    - Implement publish(aggregateType, aggregateId, eventType, payload)
    - Store event in outbox table within same transaction
    - _Requirements: 5.4, 6.6, 7.3_

  - [x] 19.2 Implement OutboxDispatcher with virtual threads
    - Create src/main/java/com/auth/userservice/infrastructure/outbox/OutboxDispatcher.java
    - Poll unprocessed events from outbox table
    - Publish to Kafka using virtual threads
    - Mark events as processed
    - _Requirements: 3.3_

- [x] 20. Integration Tests
  - [x] 20.1 Write integration tests for registration flow
    - Test successful registration returns 201
    - Test duplicate email returns 409
    - Test validation errors return 400
    - Test rate limiting returns 429
    - Use Testcontainers for PostgreSQL
    - _Requirements: 5.1, 5.5, 5.6, 2.5_

  - [x] 20.2 Write integration tests for email verification flow
    - Test successful verification activates user
    - Test expired token returns 400
    - Test invalid token returns 400
    - Test already used token returns 400
    - _Requirements: 6.1, 6.3, 6.4, 6.5_

  - [x] 20.3 Write integration tests for profile flow
    - Test get profile returns user data
    - Test update profile validates input
    - Test not found returns 404
    - _Requirements: 8.1, 8.2, 8.4_

- [x] 21. Final Checkpoint
  - Ensure all tests pass, ask the user if questions arise.
  - Verify all 16 correctness properties are implemented
  - Verify test coverage meets requirements (80% for domain/shared)
  - Verify no code duplication (single utilities for masking, correlation ID, MDC)

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (jqwik with 100+ iterations)
- Unit tests validate specific examples and edge cases
- Integration tests use Testcontainers for PostgreSQL and Kafka
- All platform service calls use gRPC clients with circuit breaker fallbacks
