# Implementation Plan: File Upload Service Modernization 2025

## Overview

This implementation plan modernizes the File Upload Service to state-of-the-art December 2025 standards. Tasks are organized to build incrementally, with each task building on previous work. Property-based tests validate correctness properties from the design document.

## Tasks

- [x] 1. Modernize Go version and dependencies
  - [x] 1.1 Update go.mod to Go 1.24 and update all dependencies
    - Update go.mod to `go 1.24`
    - Update go-redis/redis to v9
    - Update AWS SDK to v2 latest stable
    - Update OpenTelemetry to 1.28+
    - Remove deprecated packages
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 2. Remove redundant internal packages and integrate libs/go
  - [x] 2.1 Remove internal/observability and integrate libs/go/src/observability
    - Delete internal/observability/logger.go
    - Delete internal/observability/metrics.go
    - Delete internal/observability/tracing.go
    - Update imports to use libs/go/src/observability
    - _Requirements: 2.2, 2.3, 19.1_
  - [x] 2.2 Remove internal/hash and integrate libs/go/src/security
    - Delete internal/hash/generator.go
    - Use libs/go/src/security for hashing functions
    - _Requirements: 19.2_
  - [x] 2.3 Remove internal/domain/errors.go and integrate libs/go/src/errors
    - Delete internal/domain/errors.go
    - Use libs/go/src/errors for typed error handling
    - _Requirements: 4.2, 19.3_
  - [x] 2.4 Update internal/validator to use libs/go/src/validation
    - Refactor validator.go to use composable validation from libs
    - Remove redundant validation logic
    - _Requirements: 4.3, 19.4_

- [x] 3. Implement Logging Service client integration
  - [x] 3.1 Create logging client with gRPC connection
    - Create internal/infrastructure/logging/client.go
    - Implement LoggingServiceClient with gRPC client
    - Add circuit breaker using libs/go/src/fault
    - Implement local fallback logger
    - _Requirements: 2.1, 2.4, 2.5, 2.6_
  - [x] 3.2 Write property test for logging fallback behavior
    - **Property 15: Logging Service Fallback**
    - **Validates: Requirements 2.4, 2.5**

- [x] 4. Implement Cache Service client integration
  - [x] 4.1 Create cache client with gRPC connection
    - Create internal/infrastructure/cache/client.go
    - Implement CacheServiceClient with gRPC client
    - Add circuit breaker using libs/go/src/fault
    - Use namespace "file-upload" for all keys
    - _Requirements: 3.1, 3.4, 3.5, 3.6_
  - [x] 4.2 Write property test for cache-aside pattern
    - **Property 2: Cache-Aside Pattern Correctness**
    - **Validates: Requirements 3.2, 3.3, 3.4**

- [x] 5. Checkpoint - Verify infrastructure clients
  - All tests pass ✓

- [x] 6. Implement Storage interface and S3 provider
  - [x] 6.1 Create Storage interface and S3 implementation
    - Create internal/infrastructure/storage/interface.go
    - Create internal/infrastructure/storage/s3/provider.go
    - Implement presigned URL generation
    - Add circuit breaker for S3 operations
    - Use tenant-isolated path structure
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_
  - [x] 6.2 Write property test for storage path tenant isolation
    - **Property 10: Storage Path Tenant Isolation**
    - **Validates: Requirements 9.5**
  - [x] 6.3 Write property test for presigned URL validity
    - **Property 11: Presigned URL Validity**
    - **Validates: Requirements 9.3, 9.4**

- [x] 7. Implement Metadata Repository with libs integration
  - [x] 7.1 Create MetadataRepository with sqlx and pagination
    - Create internal/infrastructure/repository/metadata.go
    - Use sqlx with named parameters
    - Implement soft delete with deleted_at
    - Use libs/go/src/pagination for cursor-based listing
    - Use libs/go/src/errors for error mapping
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_
  - [x] 7.2 Write property test for pagination cursor consistency
    - **Property 12: Pagination Cursor Consistency**
    - **Validates: Requirements 4.7, 10.2**
  - [x] 7.3 Write property test for soft delete correctness
    - **Property 13: Soft Delete Correctness**
    - **Validates: Requirements 10.3**
  - [x] 7.4 Write property test for database transaction atomicity
    - **Property 14: Database Transaction Atomicity**
    - **Validates: Requirements 10.5**

- [x] 8. Checkpoint - Verify infrastructure layer
  - All tests pass ✓

- [x] 9. Implement File Validation Service
  - [x] 9.1 Create Validator with magic bytes and MIME detection
    - Refactor internal/validator/validator.go
    - Implement magic bytes detection
    - Validate extension matches MIME type
    - Use libs/go/src/validation for composition
    - Support tenant-specific allowlists
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6_
  - [x] 9.2 Write property test for file validation completeness
    - **Property 4: File Validation Completeness**
    - **Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6**

- [x] 10. Implement Rate Limiting with Cache Service
  - [x] 10.1 Refactor rate limiter to use Cache Service
    - Create internal/service/ratelimit/limiter.go
    - Use Cache_Service for distributed state
    - Implement sliding window algorithm
    - Add local fallback when cache unavailable
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_
  - [x] 10.2 Write property test for rate limiting correctness
    - **Property 9: Rate Limiting Correctness**
    - **Validates: Requirements 6.2, 6.3, 6.4, 6.5**

- [x] 11. Implement Chunked Upload with Cache Service
  - [x] 11.1 Refactor chunk manager to use Cache Service
    - Create internal/service/chunk/manager.go
    - Use Cache_Service for session state
    - Implement SHA-256 chunk verification
    - Support parallel chunk uploads
    - Use libs/go/src/workerpool for assembly
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_
  - [x] 11.2 Write property test for chunked upload integrity
    - **Property 5: Chunked Upload Integrity (Round-Trip)**
    - **Validates: Requirements 8.2, 8.3, 8.4, 8.5**

- [x] 12. Checkpoint - Verify service layer
  - All tests pass ✓

- [x] 13. Implement Security Hardening
  - [x] 13.1 Integrate security utilities from libs
    - Create internal/security/sanitizer.go
    - Implement constant-time token comparison
    - Sanitize filenames for path traversal
    - Implement sensitive data redaction in logs
    - _Requirements: 13.1, 13.2, 13.3, 13.6_
  - [x] 13.2 Write property test for security enforcement
    - **Property 6: Security Enforcement**
    - **Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2**

- [x] 14. Implement Resilience Patterns
  - [x] 14.1 Configure circuit breakers for all external dependencies
    - Create internal/resilience/circuit_breaker.go
    - Configure S3: 5 failure threshold
    - Configure Cache: 3 failure threshold
    - Configure Database: 3 failure threshold
    - Expose circuit state via Prometheus metrics
    - _Requirements: 5.1, 5.3, 5.4, 5.5, 5.6_
  - [x] 14.2 Configure retry with exponential backoff
    - Implemented in circuit_breaker.go
    - Configure max 3 attempts for database
    - _Requirements: 5.2_
  - [x] 14.3 Write property test for circuit breaker behavior
    - **Property 3: Circuit Breaker Behavior**
    - **Validates: Requirements 5.3, 5.4, 5.5, 2.5**

- [x] 15. Implement Async Processing with Worker Pool
  - [x] 15.1 Refactor async processor to use libs/go/src/workerpool
    - Create internal/async/processor.go
    - Implement virus scan task
    - Implement thumbnail generation task
    - Use retry with exponential backoff
    - Expose queue depth metrics
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6_
  - [x] 15.2 Write property test for async task retry behavior
    - **Property 18: Async Task Retry Behavior**
    - **Validates: Requirements 14.4, 5.2**

- [x] 16. Checkpoint - Verify resilience and async
  - All tests pass ✓

- [x] 17. Implement API Layer with Error Standardization
  - [x] 17.1 Implement RFC 7807 error responses
    - Create internal/api/errors/problem.go
    - Use libs/go/src/errors for HTTP mapping
    - Include correlation_id in all responses
    - _Requirements: 11.1, 11.2, 11.3, 11.4_
  - [x] 17.2 Write property test for error response consistency
    - **Property 1: Error Response Consistency**
    - **Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4**
  - [x] 17.3 Implement API versioning
    - Support /api/v1/ prefix in main.go
    - _Requirements: 20.1, 20.2, 20.3, 20.5_

- [x] 18. Implement Observability Enhancement
  - [x] 18.1 Configure OpenTelemetry tracing
    - Use OpenTelemetry Go SDK 1.28+
    - Propagate W3C Trace Context
    - Create spans for all operations
    - _Requirements: 12.1, 12.2_
  - [x] 18.2 Configure Prometheus metrics
    - Expose /metrics endpoint
    - Record upload latency, size, error rate
    - _Requirements: 12.3, 12.4_
  - [x] 18.3 Write property test for observability completeness
    - **Property 7: Observability Completeness**
    - **Validates: Requirements 2.2, 2.3, 12.1, 12.2, 12.4**

- [x] 19. Implement Health Checks and Graceful Shutdown
  - [x] 19.1 Implement health endpoints
    - Create internal/health/checker.go
    - Implement /health/live endpoint
    - Implement /health/ready endpoint
    - Check database and cache availability
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_
  - [x] 19.2 Write property test for health check accuracy
    - **Property 17: Health Check Accuracy**
    - **Validates: Requirements 16.3, 16.4**
  - [x] 19.3 Implement graceful shutdown
    - Create internal/server/shutdown.go
    - Stop accepting new requests on SIGTERM
    - Wait for in-flight requests
    - Configure 30s timeout
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5_
  - [x] 19.4 Write property test for graceful shutdown behavior
    - **Property 8: Graceful Shutdown Behavior**
    - **Validates: Requirements 17.2, 17.3, 17.4, 17.5**

- [x] 20. Implement Configuration Modernization
  - [x] 20.1 Refactor configuration to use libs/go/src/config
    - Create internal/config/config.go
    - Support environment variable overrides
    - Validate configuration at startup
    - Fail fast on missing required config
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_
  - [x] 20.2 Write property test for configuration validation
    - **Property 16: Configuration Validation**
    - **Validates: Requirements 15.2, 15.3, 15.4**

- [x] 21. Checkpoint - Verify API and configuration
  - All tests pass ✓

- [x] 22. Wire all components in main.go
  - [x] 22.1 Update cmd/server/main.go with all integrations
    - Initialize logging client
    - Initialize cache client
    - Initialize storage provider
    - Initialize metadata repository
    - Wire all middleware
    - Configure graceful shutdown
    - _Requirements: All_

- [x] 23. Final cleanup and test organization
  - [x] 23.1 Organize tests in tests/ directory
    - Property tests in tests/property/
    - Test utilities in tests/testutil/
    - _Requirements: 18.1, 18.2, 18.3, 18.4, 18.5, 18.6_
  - [x] 23.2 Remove all dead code and unused files
    - Removed old gopter-based property tests
    - Removed old files referencing deleted packages
    - Cleaned up unused imports
    - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5_

- [x] 24. Final checkpoint - All tests pass
  - All 18 property tests pass ✓
  - All correctness properties have tests ✓

## Summary

All 24 tasks completed. The file-upload service has been modernized to December 2025 standards with:
- Go 1.24 with modern dependencies
- Integration with platform services (logging-service, cache-service)
- Adoption of shared libs/go libraries
- 18 property-based tests validating correctness properties
- Resilience patterns (circuit breakers, retry, graceful shutdown)
- RFC 7807 error responses
- Health checks and observability

## Notes

- All tasks are required for comprehensive implementation
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (18 total)
- Unit tests validate specific examples and edge cases
- All redundant internal packages are removed in favor of libs/go
- Platform services (logging-service, cache-service) are integrated via gRPC clients
