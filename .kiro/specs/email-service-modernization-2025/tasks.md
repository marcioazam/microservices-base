# Implementation Plan: Email Service Modernization 2025

## Overview

This implementation plan modernizes the Email Service to December 2025 standards, integrating with platform services (logging-service, cache-service), eliminating redundancies, and ensuring all tests pass.

## Tasks

- [x] 1. Update dependencies and PHP version
  - [x] 1.1 Update composer.json with PHP 8.3 requirement and latest dependencies
    - Update php requirement to ^8.3
    - Update symfony/* packages to ^7.2
    - Update phpunit/phpunit to ^11.5
    - Add grpc/grpc ^1.65 and google/protobuf ^4.29
    - Add open-telemetry/sdk ^1.1 and open-telemetry/exporter-otlp ^1.1
    - Update doctrine/dbal to ^4.2
    - _Requirements: 3.1, 4.1, 4.2, 4.3, 4.4_

  - [x] 1.2 Update Dockerfile for PHP 8.3 with gRPC extension
    - Use php:8.3-fpm-alpine as base image
    - Install grpc and protobuf extensions
    - Configure OPcache for production
    - Run as non-root user
    - Add comprehensive health checks
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 2. Create Platform Client Layer
  - [x] 2.1 Create CacheClientInterface and CacheClient implementation
    - Create src/Infrastructure/Platform/CacheClientInterface.php
    - Create src/Infrastructure/Platform/CacheClient.php with gRPC integration
    - Create src/Infrastructure/Platform/CacheValue.php DTO
    - Create src/Infrastructure/Platform/InMemoryCacheClient.php fallback
    - _Requirements: 1.2, 1.5_

  - [x] 2.2 Write property test for cache integration round-trip
    - **Property 2: Cache Integration Round-Trip**
    - **Validates: Requirements 1.2**

  - [x] 2.3 Create LoggingClientInterface and LoggingClient implementation
    - Create src/Infrastructure/Platform/LoggingClientInterface.php
    - Create src/Infrastructure/Platform/LoggingClient.php with gRPC integration
    - Create src/Infrastructure/Platform/LogEntry.php DTO
    - Create src/Infrastructure/Platform/FallbackLogger.php for local JSON fallback
    - _Requirements: 1.1, 1.4_

  - [x] 2.4 Write property test for logging integration
    - **Property 1: Logging Integration Round-Trip**
    - **Validates: Requirements 1.1**

- [x] 3. Checkpoint - Platform clients complete
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. Centralize shared utilities
  - [x] 4.1 Create centralized PiiMasker utility
    - Create src/Application/Util/PiiMasker.php with maskEmail, maskPhone, maskName methods
    - Use match expressions for PHP 8.3 style
    - Make class final readonly
    - _Requirements: 2.2, 8.3_

  - [x] 4.2 Write property test for PII masking consistency
    - **Property 4: PII Masking Consistency**
    - **Validates: Requirements 2.2, 8.3**

  - [x] 4.3 Create centralized ErrorResponse factory
    - Create src/Api/Response/ErrorResponse.php with factory methods
    - Include validation, rateLimit, notFound, internal, provider error types
    - Use readonly class with constructor property promotion
    - _Requirements: 2.3, 2.5_

- [x] 5. Update RateLimiter to use CacheClient
  - [x] 5.1 Create CacheBackedRateLimiter implementation
    - Create src/Infrastructure/RateLimiter/CacheBackedRateLimiter.php
    - Use CacheClientInterface for distributed state
    - Implement fallback to InMemoryRateLimiter when cache unavailable
    - _Requirements: 1.3_

  - [x] 5.2 Write property test for rate limiting state consistency
    - **Property 3: Rate Limiting State Consistency**
    - **Validates: Requirements 1.3**

- [x] 6. Update AuditService to use LoggingClient
  - [x] 6.1 Refactor AuditService to use LoggingClient
    - Update src/Application/Service/AuditService.php
    - Inject LoggingClientInterface
    - Use PiiMasker for email masking
    - Include correlation IDs in all log entries
    - _Requirements: 1.1, 7.3_

  - [x] 6.2 Write property test for correlation ID presence
    - **Property 5: Correlation ID Presence**
    - **Validates: Requirements 7.3**

  - [x] 6.3 Write property test for structured error logging
    - **Property 6: Structured Error Logging**
    - **Validates: Requirements 7.4**

- [x] 7. Checkpoint - Core services updated
  - Ensure all tests pass, ask the user if questions arise.

- [x] 8. Update ValidationService
  - [x] 8.1 Refactor ValidationService to use CacheClient for domain validation caching
    - Update src/Application/Service/ValidationService.php
    - Cache MX record lookups via CacheClient
    - Use readonly class pattern
    - _Requirements: 2.1, 2.4_

  - [x] 8.2 Write property test for input validation allowlist
    - **Property 7: Input Validation Allowlist**
    - **Validates: Requirements 8.1**

- [x] 9. Add observability enhancements
  - [x] 9.1 Create OpenTelemetry tracing integration
    - Create src/Infrastructure/Observability/TracingService.php
    - Emit traces for email send operations
    - Include trace ID, span ID, operation name, timestamps, status
    - _Requirements: 7.1_

  - [x] 9.2 Write property test for OpenTelemetry trace completeness
    - **Property 10: OpenTelemetry Trace Completeness**
    - **Validates: Requirements 7.1**

  - [x] 9.3 Update health check endpoint with dependency status
    - Update src/Infrastructure/Observability/HealthCheck.php
    - Include CacheClient health status
    - Include LoggingClient health status
    - Include provider connectivity status
    - _Requirements: 7.5_

- [x] 10. Performance optimizations
  - [x] 10.1 Implement batch email processing
    - Create src/Application/Service/BatchEmailService.php
    - Process multiple emails in single operation
    - Return results maintaining input order
    - _Requirements: 9.2_

  - [x] 10.2 Write property test for batch processing completeness
    - **Property 8: Batch Email Processing Completeness**
    - **Validates: Requirements 9.2**

  - [x] 10.3 Implement template compilation caching
    - Update src/Application/Service/TemplateService.php
    - Cache compiled templates via CacheClient
    - _Requirements: 9.3_

  - [x] 10.4 Write property test for template cache effectiveness
    - **Property 9: Template Cache Effectiveness**
    - **Validates: Requirements 9.3**

- [x] 11. Checkpoint - Features complete
  - Ensure all tests pass, ask the user if questions arise.

- [x] 12. Architecture cleanup
  - [x] 12.1 Remove dead code and unused imports
    - Scan all PHP files for unused imports
    - Remove any dead/unreachable code
    - Remove deprecated methods
    - _Requirements: 6.1_

  - [x] 12.2 Remove .gitkeep files from directories with content
    - Remove tests/Integration/.gitkeep
    - Remove tests/Unit/.gitkeep
    - Remove tests/Property/.gitkeep
    - _Requirements: 6.2_

  - [x] 12.3 Ensure all files under 400 lines
    - Split any files exceeding 400 lines
    - Extract helper classes as needed
    - _Requirements: 6.3_

  - [x] 12.4 Apply PSR-12 coding standards
    - Run php_codesniffer with PSR-12 standard
    - Fix any violations
    - _Requirements: 6.4_

- [x] 13. Update existing property tests
  - [x] 13.1 Update EmailValidationPropertyTest to use new ValidationService
    - Ensure tests use CacheClient mock
    - Verify all existing tests pass
    - _Requirements: 5.4_

  - [x] 13.2 Update RateLimitPropertyTest to use CacheBackedRateLimiter
    - Add tests for CacheBackedRateLimiter
    - Verify fallback behavior
    - _Requirements: 5.4_

  - [x] 13.3 Update AuditLoggingPropertyTest to use LoggingClient
    - Ensure tests use LoggingClient mock
    - Verify PII masking via PiiMasker
    - _Requirements: 5.4_

  - [x] 13.4 Verify all existing property tests pass
    - Run full property test suite
    - Fix any failing tests
    - Ensure 100 iterations per property
    - _Requirements: 5.3, 5.4_

- [x] 14. Final validation
  - [x] 14.1 Run full test suite and verify 80%+ coverage
    - Run phpunit with coverage
    - Verify minimum 80% overall coverage
    - _Requirements: 5.2_

  - [x] 14.2 Run static analysis
    - Run phpstan at level 8
    - Fix any issues
    - _Requirements: 6.4_

  - [x] 14.3 Verify Docker build succeeds
    - Build Docker image
    - Verify health checks work
    - _Requirements: 10.1, 10.5_

- [x] 15. Final checkpoint - Production ready
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties with minimum 100 iterations
- All platform service integration uses gRPC clients from `platform/logging-service` and `platform/cache-service`
