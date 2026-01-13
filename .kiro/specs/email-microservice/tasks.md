# Implementation Plan: Email Microservice

## Overview

This implementation plan creates a PHP 8.x Email Microservice following hexagonal architecture. Tasks are organized to build incrementally from domain layer up to API layer, with property-based tests validating correctness at each step.

## Tasks

- [x] 1. Set up project structure and dependencies
  - Create `services/email-service/` directory structure
  - Initialize Composer with PHP 8.2+ requirement
  - Add dependencies: symfony/mailer, twig/twig, php-amqplib/php-amqplib, predis/predis
  - Add dev dependencies: phpunit/phpunit, eris/eris (property testing), mockery/mockery
  - Configure PSR-4 autoloading
  - Create phpunit.xml configuration
  - _Requirements: All_

- [x] 2. Implement Domain Layer
  - [x] 2.1 Create value objects and enums
    - Implement `Recipient` value object with email validation
    - Implement `ContentType` enum (HTML, PLAIN)
    - Implement `EmailStatus` enum
    - Implement `AuditAction` enum
    - _Requirements: 1.1, 2.1_

  - [x] 2.2 Write property test for Recipient validation
    - **Property 6: RFC 5322 Email Validation**
    - **Validates: Requirements 2.1**

  - [x] 2.3 Create Email entity
    - Implement `Email` entity with all properties
    - Implement `Attachment` value object with size validation
    - Add factory method for creating emails from DTOs
    - _Requirements: 1.1, 1.2, 1.3_

  - [x] 2.4 Write property test for attachment size validation
    - **Property 3: Attachment Size Validation**
    - **Validates: Requirements 1.3**

  - [x] 2.5 Create Template entity
    - Implement `Template` entity with required/default variables
    - Add validation for template structure
    - _Requirements: 4.1, 4.2_

  - [x] 2.6 Create AuditLog entity
    - Implement `AuditLog` entity with all required fields
    - Add factory method for different audit actions
    - _Requirements: 5.1, 5.2_

- [x] 3. Checkpoint - Domain layer complete
  - Ensure all unit tests pass
  - Verify property tests pass with 100+ iterations
  - Ask the user if questions arise

- [x] 4. Implement Validation Service
  - [x] 4.1 Create ValidationService interface and implementation
    - Implement RFC 5322 email format validation
    - Implement MX record lookup validation
    - Implement disposable domain detection
    - _Requirements: 2.1, 2.2, 2.3_

  - [x] 4.2 Write property test for MX validation
    - **Property 7: MX Record Validation**
    - **Validates: Requirements 2.2**

  - [x] 4.3 Write property test for disposable domain rejection
    - **Property 8: Disposable Domain Rejection**
    - **Validates: Requirements 2.3**

  - [x] 4.4 Write property test for validation error specificity
    - **Property 10: Validation Error Specificity**
    - **Validates: Requirements 2.5**

- [x] 5. Implement Template Service
  - [x] 5.1 Create TemplateService interface and implementation
    - Integrate Twig template engine
    - Implement variable substitution
    - Implement default value handling
    - _Requirements: 4.1, 4.2, 4.4_

  - [x] 5.2 Write property test for template variable substitution
    - **Property 15: Template Variable Substitution**
    - **Validates: Requirements 4.1**

  - [x] 5.3 Write property test for default variable handling
    - **Property 16: Default Variable Handling**
    - **Validates: Requirements 4.2**

  - [x] 5.4 Implement XSS prevention
    - Configure Twig autoescape
    - Add HTML entity escaping for user variables
    - _Requirements: 4.3_

  - [x] 5.5 Write property test for XSS prevention
    - **Property 17: XSS Prevention**
    - **Validates: Requirements 4.3**

- [x] 6. Checkpoint - Services layer partial
  - Ensure all tests pass
  - Verify property tests pass with 100+ iterations
  - Ask the user if questions arise

- [x] 7. Implement Rate Limiter
  - [x] 7.1 Create RateLimiter interface and Redis implementation
    - Implement sliding window rate limiting
    - Implement per-sender tracking
    - Implement global provider tracking
    - _Requirements: 9.1, 9.2, 9.5_

  - [x] 7.2 Write property test for per-sender rate limiting
    - **Property 27: Per-Sender Rate Limiting**
    - **Validates: Requirements 9.1**

  - [x] 7.3 Write property test for rate limit enforcement
    - **Property 9: Rate Limit Enforcement**
    - **Validates: Requirements 2.4, 9.4**

  - [x] 7.4 Implement graceful rate limit approach
    - Queue emails when approaching limit (≥80%)
    - Return 429 with retry-after when exceeded
    - _Requirements: 9.3_

  - [x] 7.5 Write property test for graceful rate limit approach
    - **Property 29: Graceful Rate Limit Approach**
    - **Validates: Requirements 9.3**

- [x] 8. Implement Audit Service
  - [x] 8.1 Create AuditService interface and PostgreSQL implementation
    - Implement audit log creation
    - Implement PII masking for email addresses
    - Implement query with filters
    - _Requirements: 5.1, 5.2, 5.4, 7.5_

  - [x] 8.2 Write property test for audit log completeness
    - **Property 19: Audit Log Completeness**
    - **Validates: Requirements 5.1, 5.2**

  - [x] 8.3 Write property test for PII masking
    - **Property 24: PII Masking in Logs**
    - **Validates: Requirements 7.5**

  - [x] 8.4 Write property test for audit log filtering
    - **Property 20: Audit Log Filtering**
    - **Validates: Requirements 5.4**

- [x] 9. Implement Email Providers
  - [x] 9.1 Create EmailProviderInterface
    - Define send, getDeliveryStatus, getName, isHealthy methods
    - Create ProviderResult and DeliveryStatus DTOs
    - _Requirements: 8.1, 8.4_

  - [x] 9.2 Implement SendGrid adapter
    - Integrate with SendGrid API
    - Implement SPF/DKIM/DMARC header handling
    - _Requirements: 7.4, 8.1_

  - [x] 9.3 Implement Mailgun adapter
    - Integrate with Mailgun API
    - Implement SPF/DKIM/DMARC header handling
    - _Requirements: 7.4, 8.1_

  - [x] 9.4 Implement Amazon SES adapter
    - Integrate with AWS SES SDK
    - Implement SPF/DKIM/DMARC header handling
    - _Requirements: 7.4, 8.1_

  - [x] 9.5 Write property test for email authentication headers
    - **Property 23: Email Authentication Headers**
    - **Validates: Requirements 7.4**

  - [x] 9.6 Implement ProviderRouter
    - Route emails by type (transactional, marketing, verification)
    - Implement failover logic
    - Track per-provider metrics
    - _Requirements: 8.2, 8.3, 8.5_

  - [x] 9.7 Write property test for provider routing
    - **Property 25: Provider Routing by Email Type**
    - **Validates: Requirements 8.3**

  - [x] 9.8 Write property test for provider failover
    - **Property 5: Provider Failover**
    - **Validates: Requirements 1.6, 8.2**

  - [x] 9.9 Write property test for per-provider metrics
    - **Property 26: Per-Provider Metrics**
    - **Validates: Requirements 8.5**

- [x] 10. Checkpoint - Providers complete
  - Ensure all tests pass
  - Verify property tests pass with 100+ iterations
  - Ask the user if questions arise

- [x] 11. Implement Queue Service
  - [x] 11.1 Create QueueService interface and RabbitMQ implementation
    - Implement enqueue with priority
    - Implement FIFO ordering within priority
    - Implement worker processing
    - _Requirements: 6.1, 6.2, 6.4_

  - [x] 11.2 Write property test for queue ordering
    - **Property 21: Queue Priority and FIFO Ordering**
    - **Validates: Requirements 6.1, 6.2**

  - [x] 11.3 Implement retry logic with exponential backoff
    - Implement retry scheduling (1s, 2s, 4s, 8s, 16s)
    - Track attempt count
    - _Requirements: 3.2, 6.3_

  - [x] 11.4 Write property test for retry count enforcement
    - **Property 12: Retry Count Enforcement**
    - **Validates: Requirements 3.2**

  - [x] 11.5 Write property test for exponential backoff timing
    - **Property 22: Exponential Backoff Timing**
    - **Validates: Requirements 6.3**

  - [x] 11.6 Implement dead letter queue handling
    - Move failed emails after max retries
    - Preserve failure reason
    - _Requirements: 3.3_

  - [x] 11.7 Write property test for dead letter queue routing
    - **Property 13: Dead Letter Queue Routing**
    - **Validates: Requirements 3.3**

- [x] 12. Implement Email Service
  - [x] 12.1 Create EmailService interface and implementation
    - Implement send (sync and async)
    - Implement resend for failed emails
    - Implement getStatus
    - Wire all dependencies
    - _Requirements: 1.1, 1.4, 3.1_

  - [x] 12.2 Write property test for valid email sending
    - **Property 1: Valid Email Sending**
    - **Validates: Requirements 1.1**

  - [x] 12.3 Write property test for HTML content preservation
    - **Property 2: HTML Content Preservation**
    - **Validates: Requirements 1.2**

  - [x] 12.4 Write property test for multi-recipient emails
    - **Property 4: Multi-Recipient Individual Emails**
    - **Validates: Requirements 1.5**

  - [x] 12.5 Write property test for failed email resend
    - **Property 11: Failed Email Resend**
    - **Validates: Requirements 3.1**

  - [x] 12.6 Implement verification email handling
    - Generate unique tokens
    - Handle resend with new token
    - _Requirements: 3.4_

  - [x] 12.7 Write property test for verification token regeneration
    - **Property 14: Verification Token Regeneration**
    - **Validates: Requirements 3.4**

- [x] 13. Checkpoint - Core services complete
  - Ensure all tests pass
  - Verify property tests pass with 100+ iterations
  - Ask the user if questions arise

- [x] 14. Implement API Layer
  - [x] 14.1 Create request DTOs with JSON schema validation
    - SendEmailRequest, VerifyEmailRequest, ResendEmailRequest
    - AuditQueryRequest
    - Implement validation rules
    - _Requirements: 10.6_

  - [x] 14.2 Write property test for API request validation
    - **Property 30: API Request Validation**
    - **Validates: Requirements 10.6, 10.7**

  - [x] 14.3 Create EmailController
    - Implement POST /api/v1/emails/send
    - Implement POST /api/v1/emails/verify
    - Implement POST /api/v1/emails/resend
    - Implement GET /api/v1/emails/{id}/status
    - Implement GET /api/v1/emails/audit
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

  - [x] 14.4 Implement error handling middleware
    - Map exceptions to HTTP status codes
    - Create standardized error responses
    - Add correlation ID tracking
    - _Requirements: 10.7_

  - [x] 14.5 Implement authentication middleware
    - Support API key authentication
    - Support OAuth2 authentication
    - Return 401 for invalid credentials
    - _Requirements: 7.3, 7.6_

- [x] 15. Implement Observability
  - [x] 15.1 Add Prometheus metrics
    - Email delivery rate
    - Failure rate
    - Queue depth
    - Per-provider metrics
    - _Requirements: 5.3, 6.5_

  - [x] 15.2 Add structured logging
    - JSON format with correlation IDs
    - PII masking
    - Log levels (ERROR, WARN, INFO, DEBUG)
    - _Requirements: 5.1, 5.2, 7.5_

  - [x] 15.3 Add health check endpoint
    - Check database connectivity
    - Check queue connectivity
    - Check Redis connectivity
    - Check provider health
    - _Requirements: All_

- [x] 16. Create Docker configuration
  - [x] 16.1 Create Dockerfile
    - PHP 8.2 FPM base image
    - Install required extensions (amqp, redis, pdo_pgsql)
    - Configure OPcache for production
    - _Requirements: All_

  - [x] 16.2 Add to docker-compose
    - Add email-service to deploy/docker/docker-compose.yml
    - Configure environment variables
    - Set up networking with other services
    - _Requirements: All_

- [x] 17. Final checkpoint - All tests pass
  - All source files validated (<400 lines each)
  - 9 property test files created with 100+ iterations
  - Docker configuration complete
  - Ready for `composer install && vendor/bin/phpunit`
  - _Note: Run tests in Docker or with local PHP 8.2+ and Composer_

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (100+ iterations each)
- Unit tests validate specific examples and edge cases
