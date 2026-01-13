# Implementation Plan: User Service

## Overview

This plan implements the User Service microservice in Java/Spring Boot following the design document. Tasks are organized to build incrementally, with property-based tests validating correctness properties as we go.

## Tasks

- [x] 1. Project Setup and Core Infrastructure
  - [x] 1.1 Initialize Spring Boot project with Gradle
    - Create `services/user-service/` directory structure
    - Configure `build.gradle` with dependencies (Spring Boot 3.x, jqwik, Testcontainers)
    - Set up `application.yml` with profiles (dev, prod)
    - _Requirements: 11.1, 11.2_

  - [x] 1.2 Create Flyway database migrations
    - V1: users table with indexes
    - V2: email_verification_tokens table
    - V3: outbox_events table
    - _Requirements: 1.6, 2.6, 7.1_

  - [x] 1.3 Create JPA entities and repositories
    - User entity with UserStatus enum
    - EmailVerificationToken entity
    - OutboxEvent entity
    - Spring Data JPA repositories
    - _Requirements: 1.6, 2.6, 7.1_

- [x] 2. Common Utilities and Validation
  - [x] 2.1 Implement EmailNormalizer utility
    - Lowercase and trim email
    - Handle edge cases (null, empty)
    - _Requirements: 1.2, 3.1_

  - [x] 2.2 Write property test for email normalization
    - **Property 1: Email Normalization Idempotence**
    - **Validates: Requirements 1.2, 3.1**

  - [x] 2.3 Implement TokenHasher utility
    - SHA-256 hashing for verification tokens
    - Secure random token generation (32 bytes)
    - _Requirements: 1.7, 1.8, 2.1_

  - [x] 2.4 Write property test for token hash consistency
    - **Property 3: Token Hash Consistency**
    - **Validates: Requirements 1.8, 2.1**

  - [x] 2.5 Implement PasswordValidator
    - Complexity rules (8+ chars, uppercase, lowercase, digit)
    - Disposable email domain blocklist
    - _Requirements: 6.2, 8.1, 8.2, 8.3_

  - [x] 2.6 Write property test for input validation
    - **Property 9: Input Validation Rejection**
    - **Validates: Requirements 1.1, 5.1, 8.1, 8.2, 8.3, 8.4**

- [x] 3. Checkpoint - Utilities Complete
  - Ensure all utility tests pass
  - Ask the user if questions arise

- [-] 4. Password Service with Argon2id
  - [x] 4.1 Implement PasswordService
    - Argon2id hashing with configurable parameters
    - Verify method with constant-time comparison
    - Integration with Crypto_Service (optional)
    - _Requirements: 1.5, 6.1, 6.4_

  - [x] 4.2 Write property test for password hashing
    - **Property 2: Password Hashing Round-Trip**
    - **Validates: Requirements 1.5, 6.1**

- [x] 5. User Registration Service
  - [x] 5.1 Implement UserRegistrationService
    - Validate input, normalize email
    - Check for existing user
    - Hash password, create User with PENDING_EMAIL status
    - Generate verification token, store hash
    - Publish EmailVerificationRequested event to outbox
    - _Requirements: 1.1-1.10_

  - [x] 5.2 Write property test for user registration initial state
    - **Property 4: User Registration Initial State**
    - **Validates: Requirements 1.6, 1.7**

  - [x] 5.3 Implement UserController
    - POST /v1/users endpoint
    - Request validation with Bean Validation
    - Response mapping to DTO
    - _Requirements: 1.10_

- [-] 6. Email Verification Service
  - [x] 6.1 Implement EmailVerificationService
    - Verify token: compute hash, lookup, validate expiry/used
    - Mark token as used, update user to ACTIVE
    - Publish UserEmailVerified event to outbox
    - _Requirements: 2.1-2.9_

  - [x] 6.2 Write property test for email verification state transition
    - **Property 5: Email Verification State Transition**
    - **Validates: Requirements 2.6, 2.7**

  - [x] 6.3 Implement resend verification logic
    - Rate limit check
    - Invalidate existing tokens
    - Generate new token
    - Always return 202 (anti-enumeration)
    - _Requirements: 3.1-3.5_

  - [x] 6.4 Write property test for resend anti-enumeration
    - **Property 11: Resend Anti-Enumeration**
    - **Validates: Requirements 3.5**

  - [x] 6.5 Implement EmailVerificationController
    - POST /v1/users/email/verify endpoint
    - POST /v1/users/email/resend endpoint
    - _Requirements: 2.9, 3.5_

- [x] 7. Checkpoint - Core Registration Flow Complete
  - Ensure registration and verification tests pass
  - Ask the user if questions arise

- [x] 8. Profile Service
  - [x] 8.1 Implement ProfileService
    - Get profile by user ID
    - Update allowed fields only (displayName)
    - Set updatedAt timestamp
    - _Requirements: 4.1-4.5, 5.1-5.4_

  - [x] 8.2 Write property test for profile update field restriction
    - **Property 10: Profile Update Field Restriction**
    - **Validates: Requirements 5.2, 5.3**

  - [x] 8.3 Implement MeController
    - GET /v1/users/me endpoint
    - PATCH /v1/users/me endpoint
    - JWT authentication required
    - _Requirements: 4.4, 5.4_

  - [x] 8.4 Write property test for sensitive data non-exposure
    - **Property 8: Sensitive Data Non-Exposure**
    - **Validates: Requirements 4.5, 6.3, 10.5**

- [x] 9. Outbox Pattern Implementation
  - [x] 9.1 Implement OutboxPublisher
    - Create OutboxEvent in same transaction as domain operation
    - JSON serialization of event payload
    - _Requirements: 7.1, 7.2_

  - [x] 9.2 Write property test for outbox event completeness
    - **Property 6: Outbox Event Completeness**
    - **Validates: Requirements 1.9, 2.8, 7.1, 7.2**

  - [x] 9.3 Implement OutboxDispatcher
    - Poll for unprocessed events
    - Publish to Kafka
    - Mark as processed on success
    - Retry with exponential backoff on failure
    - _Requirements: 7.3-7.6_

- [x] 10. Rate Limiting
  - [x] 10.1 Implement RateLimitService
    - Integration with Cache_Service
    - Configurable limits per endpoint
    - Sliding window algorithm
    - _Requirements: 9.1-9.5_

  - [x] 10.2 Write property test for rate limiting enforcement
    - **Property 7: Rate Limiting Enforcement**
    - **Validates: Requirements 3.2, 9.1, 9.2, 9.3, 9.4**

  - [x] 10.3 Add rate limit interceptor to controllers
    - Apply to registration, verify, resend endpoints
    - Return 429 with Retry-After header
    - _Requirements: 9.4_

- [x] 11. Checkpoint - Core Features Complete
  - Ensure all core feature tests pass
  - Ask the user if questions arise

- [x] 12. Platform Service Integration
  - [x] 12.1 Implement CryptoServiceClient
    - gRPC client for Crypto_Service
    - Circuit breaker with Resilience4j
    - Fallback to local Argon2id
    - _Requirements: 12.1, 12.4, 12.5_

  - [x] 12.2 Implement CacheServiceClient
    - gRPC client for Cache_Service
    - Circuit breaker with Resilience4j
    - Fallback to local cache (Caffeine)
    - _Requirements: 12.2, 12.4, 12.5_

  - [x] 12.3 Implement LoggingServiceClient
    - gRPC client for Logging_Service
    - Circuit breaker with Resilience4j
    - Fallback to local logging
    - _Requirements: 12.3, 12.4, 12.5_

  - [x] 12.4 Write property test for circuit breaker state transitions
    - **Property 12: Circuit Breaker State Transitions**
    - **Validates: Requirements 12.4, 12.5**

- [x] 13. Security Configuration
  - [x] 13.1 Implement SecurityConfig
    - JWT validation with JWKS
    - Public endpoints: /v1/users, /v1/users/email/*, /health/*
    - Protected endpoints: /v1/users/me
    - _Requirements: 4.1_

  - [x] 13.2 Implement JwtAuthConverter
    - Extract user ID from JWT claims
    - Map to Spring Security principal
    - _Requirements: 4.1_

- [x] 14. Observability
  - [x] 14.1 Implement structured logging
    - JSON format with correlationId, userId, timestamp
    - MDC propagation
    - Sensitive data filtering
    - _Requirements: 10.1, 10.5_

  - [x] 14.2 Configure Prometheus metrics
    - Registration, verification, profile update counters
    - Outbox event metrics
    - Platform client metrics
    - _Requirements: 10.2_

  - [x] 14.3 Configure OpenTelemetry tracing
    - W3C Trace Context propagation
    - Span creation for service methods
    - _Requirements: 10.3_

- [x] 15. Health Checks and Graceful Shutdown
  - [x] 15.1 Implement health endpoints
    - /health/live - liveness probe
    - /health/ready - readiness probe (DB, dependencies)
    - _Requirements: 11.1, 11.2_

  - [x] 15.2 Configure graceful shutdown
    - SIGTERM handling
    - In-flight request completion
    - Configurable timeout
    - _Requirements: 11.3, 11.4, 11.5_

- [x] 16. Error Handling
  - [x] 16.1 Implement GlobalExceptionHandler
    - RFC 7807 Problem Details format
    - Map exceptions to HTTP status codes
    - Correlation ID in error responses
    - _Requirements: 1.4, 8.5_

  - [x] 16.2 Implement error codes and messages
    - VALIDATION_ERROR, EMAIL_ALREADY_EXISTS, etc.
    - Generic messages for security-sensitive errors
    - _Requirements: 1.4, 2.3, 2.4, 2.5_

- [x] 17. API Documentation
  - [x] 17.1 Configure OpenAPI/Swagger
    - springdoc-openapi integration
    - Request/response examples
    - Error response documentation
    - _Requirements: N/A (best practice)_

- [x] 18. Docker and Deployment
  - [x] 18.1 Create Dockerfile
    - Multi-stage build
    - JRE 21 base image
    - Health check configuration
    - _Requirements: 11.1, 11.2_

  - [x] 18.2 Create Kubernetes manifests
    - Deployment, Service, ConfigMap
    - Liveness and readiness probes
    - Resource limits
    - _Requirements: 11.1, 11.2_

  - [x] 18.3 Create ResiliencePolicy for Service Mesh
    - Circuit breaker configuration
    - Retry and timeout settings
    - _Requirements: 12.4_

- [x] 19. Integration Tests
  - [x] 19.1 Write integration tests with Testcontainers
    - PostgreSQL container
    - Kafka container
    - Full registration flow test
    - Full verification flow test
    - _Requirements: All_

- [x] 20. Final Checkpoint
  - Ensure all tests pass
  - Verify OpenAPI documentation
  - Review security configuration
  - Ask the user if questions arise

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (100+ iterations with jqwik)
- Unit tests validate specific examples and edge cases
