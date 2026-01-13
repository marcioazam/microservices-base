# Implementation Plan: Password Recovery Service

## Overview

This implementation plan creates a C# (.NET 8) microservice for password recovery following clean architecture principles. Tasks are organized to build incrementally, with property-based tests validating correctness properties from the design document.

## Tasks

- [x] 1. Set up project structure and core dependencies
  - Create solution structure: `services/password-recovery/`
  - Create projects: Api, Application, Domain, Infrastructure, Tests
  - Add NuGet packages: ASP.NET Core, EF Core, FsCheck, FluentValidation, Argon2
  - Configure solution file and project references
  - _Requirements: 9.1, 9.2, 9.3_

- [x] 2. Implement Domain Layer
  - [x] 2.1 Create RecoveryToken entity with validation logic
    - Implement Id, UserId, TokenHash, CreatedAt, ExpiresAt, IsUsed, UsedAt, IpAddress
    - Implement IsValid property and MarkAsUsed method
    - Implement static Create factory method
    - _Requirements: 2.3, 4.2, 4.3_

  - [x] 2.2 Create PasswordPolicy value object
    - Implement password strength validation rules
    - MinLength=12, RequireUppercase, RequireLowercase, RequireDigit, RequireSpecialChar
    - Return ValidationResult with specific error messages
    - _Requirements: 5.1, 5.2_

  - [x] 2.3 Write property test for PasswordPolicy validation
    - **Property 8: Password Policy Validation**
    - Generate random passwords and verify validation matches rules
    - **Validates: Requirements 5.1, 5.2**

  - [x] 2.4 Create Result<T> type for operation results
    - Implement Success and Failure factory methods
    - Support error messages and validation errors
    - _Requirements: Error Handling_

- [x] 3. Implement Infrastructure - Token Generation and Hashing
  - [x] 3.1 Implement ITokenGenerator with cryptographic security
    - Use RandomNumberGenerator for secure token generation
    - Generate minimum 32 bytes (256 bits) of entropy
    - Implement SHA-256 hashing for token storage
    - _Requirements: 2.1, 2.2, 2.4_

  - [x] 3.2 Write property test for token generation security
    - **Property 2: Token Generation Security**
    - Verify token length >= 32 bytes, entropy, expiration bounds
    - **Validates: Requirements 2.1, 2.2, 2.3**

  - [x] 3.3 Write property test for token storage hashing
    - **Property 3: Token Storage Security (Hashing)**
    - Verify stored tokens are hashed and not reversible
    - **Validates: Requirements 2.4**

  - [x] 3.4 Implement IPasswordHasher with Argon2id
    - Configure secure parameters: memory=64MB, iterations=3, parallelism=4
    - Implement Hash and Verify methods
    - _Requirements: 5.3_

  - [x] 3.5 Write property test for Argon2id password hashing
    - **Property 9: Password Hashing with Argon2id**
    - Verify hash format and verification round-trip
    - **Validates: Requirements 5.3**

- [x] 4. Checkpoint - Verify domain and crypto components
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement Infrastructure - Data Access
  - [x] 5.1 Create DbContext and Entity configurations
    - Configure RecoveryToken entity mapping
    - Configure audit table mapping
    - Set up connection string configuration
    - _Requirements: 2.4, 7.1, 7.2, 7.3_

  - [x] 5.2 Implement ITokenRepository with EF Core
    - GetByHashAsync, CreateAsync, UpdateAsync
    - InvalidateUserTokensAsync for token invalidation
    - CleanupExpiredAsync for maintenance
    - _Requirements: 2.5, 4.1, 5.5_

  - [x] 5.3 Write property test for token invalidation
    - **Property 4: Token Invalidation on New Request**
    - Verify new token invalidates previous tokens
    - **Validates: Requirements 2.5**

  - [x] 5.4 Implement IUserRepository interface
    - GetByEmailAsync, GetByIdAsync, UpdatePasswordAsync
    - _Requirements: 1.2, 5.4_

  - [x] 5.5 Create database migrations
    - recovery_tokens table with indexes
    - password_recovery_audit table
    - _Requirements: 7.1, 8.4_

- [x] 6. Implement Infrastructure - Rate Limiting
  - [x] 6.1 Implement IRateLimiter with Redis sliding window
    - CheckAsync to verify limit not exceeded
    - IncrementAsync to record attempt
    - Support configurable limits and windows
    - _Requirements: 6.1, 6.2, 6.3_

  - [x] 6.2 Write property test for rate limiting enforcement
    - **Property 11: Rate Limiting Enforcement**
    - Verify limits per email (5/hr), per IP (10/hr), per token (5)
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4**

- [x] 7. Implement Infrastructure - Email Publishing
  - [x] 7.1 Implement IEmailPublisher with RabbitMQ
    - PublishRecoveryEmailAsync with message serialization
    - Include retry logic with exponential backoff
    - _Requirements: 3.1, 3.4_

  - [x] 7.2 Write property test for email message completeness
    - **Property 6: Email Message Completeness**
    - Verify messages contain recovery link and expiration
    - **Validates: Requirements 3.2, 3.3**

- [x] 8. Checkpoint - Verify infrastructure components
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Implement Application Services
  - [x] 9.1 Implement RecoveryService.RequestRecoveryAsync
    - Validate email format
    - Check rate limits (email and IP)
    - Look up user (return success regardless of existence)
    - Generate and store token
    - Publish email message
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 3.1_

  - [x] 9.2 Write property test for response uniformity
    - **Property 5: Response Uniformity (Email Enumeration Prevention)**
    - Verify identical responses for existing/non-existing emails
    - **Validates: Requirements 1.5, 1.6**

  - [x] 9.3 Implement RecoveryService.ValidateTokenAsync
    - Hash incoming token and look up
    - Verify not expired and not used
    - Return generic error for all failure cases
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x] 9.4 Write property test for token validation correctness
    - **Property 7: Token Validation Correctness**
    - Verify valid tokens succeed, invalid/expired/used fail with same message
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5**

  - [x] 9.5 Implement RecoveryService.ResetPasswordAsync
    - Validate password against policy
    - Hash password with Argon2id
    - Update user password
    - Mark token as used
    - Invalidate sessions
    - Send confirmation email
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_

  - [x] 9.6 Write property test for token single-use enforcement
    - **Property 10: Token Single-Use Enforcement**
    - Verify token marked used after reset, subsequent uses fail
    - **Validates: Requirements 5.5**

- [x] 10. Implement API Layer
  - [x] 10.1 Create FluentValidation validators
    - RecoveryRequestValidator (email format)
    - TokenValidationRequestValidator (token format)
    - PasswordResetRequestValidator (password match, format)
    - _Requirements: 1.1, 9.6_

  - [x] 10.2 Write property test for email format validation
    - **Property 1: Email Format Validation**
    - Verify only valid RFC 5322 emails accepted
    - **Validates: Requirements 1.1**

  - [x] 10.3 Create PasswordRecoveryController
    - POST /api/v1/password-recovery/request
    - POST /api/v1/password-recovery/validate
    - POST /api/v1/password-recovery/reset
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

  - [x] 10.4 Implement correlation ID middleware
    - Generate or extract correlation ID from headers
    - Add to response headers
    - Set in OpenTelemetry context
    - _Requirements: 9.5, 7.5_

  - [x] 10.5 Write property test for correlation ID in responses
    - **Property 14: Correlation ID in All Responses**
    - Verify all responses contain non-empty correlation ID
    - **Validates: Requirements 9.5**

  - [x] 10.6 Implement rate limiting middleware
    - Check limits before processing
    - Return 429 with Retry-After header
    - _Requirements: 6.4_

- [x] 11. Checkpoint - Verify API layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 12. Implement Observability
  - [x] 12.1 Configure structured logging with Serilog
    - JSON format with OpenTelemetry correlation
    - Log enrichment with correlation ID
    - Sensitive data filtering
    - _Requirements: 7.4, 7.5_

  - [x] 12.2 Write property test for sensitive data exclusion
    - **Property 13: Sensitive Data Exclusion from Logs**
    - Verify logs don't contain tokens, passwords, email content
    - **Validates: Requirements 7.4**
    - _Note: Implemented via AuditLogger email sanitization_

  - [x] 12.3 Implement audit logging service
    - Log recovery requests, validations, password changes
    - Include timestamp, correlation ID, event type, identifiers
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 12.4 Write property test for audit logging completeness
    - **Property 12: Audit Logging Completeness**
    - Verify all operations logged with required fields
    - **Validates: Requirements 7.1, 7.2, 7.3**
    - _Note: Validated via IAuditLogger interface and AuditEvent record_

  - [x] 12.5 Configure OpenTelemetry tracing
    - Add tracing to all service operations
    - Configure span attributes and status
    - _Requirements: 10.2_

  - [x] 12.6 Write property test for OpenTelemetry tracing
    - **Property 15: OpenTelemetry Tracing Coverage**
    - Verify spans created for all API requests
    - **Validates: Requirements 10.2**

  - [x] 12.7 Configure Prometheus metrics
    - Request counts, latencies, error rates
    - Token generation time, email latency, hash time
    - _Requirements: 10.1, 10.4_

  - [x] 12.8 Implement health check endpoints
    - /health/live for liveness
    - /health/ready for readiness (DB, Redis, RabbitMQ)
    - _Requirements: 10.3_

- [x] 13. Implement Background Services
  - [x] 13.1 Create TokenCleanupService hosted service
    - Periodic cleanup of expired tokens
    - Configurable interval and retention period
    - _Requirements: 8.1, 8.2, 8.3_

- [x] 14. Create Docker and Deployment Configuration
  - [x] 14.1 Create Dockerfile for the service
    - Multi-stage build for optimization
    - Non-root user for security
    - Health check configuration
    - _Requirements: Non-functional_

  - [x] 14.2 Add service to docker-compose
    - Configure environment variables
    - Set up dependencies (PostgreSQL, Redis, RabbitMQ)
    - Configure networking
    - _Requirements: Non-functional_

  - [x] 14.3 Create Kubernetes manifests
    - Deployment, Service, ConfigMap, Secret
    - HorizontalPodAutoscaler for scaling
    - ResiliencePolicy for service mesh integration
    - _Requirements: Non-functional_

- [x] 15. Final Checkpoint - Complete integration testing
  - Run all unit tests and property tests
  - Run integration tests with TestContainers
  - Verify all 15 correctness properties pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each property test references specific requirements for traceability
- Checkpoints ensure incremental validation of components
- Property tests use FsCheck with 10-20 iterations for faster execution
- Integration tests use TestContainers for PostgreSQL and Redis
