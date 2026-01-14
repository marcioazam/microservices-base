# Requirements Document

## Introduction

This document specifies the requirements for modernizing the File Upload Service to state-of-the-art December 2025 standards. The modernization eliminates legacy patterns, removes redundancy, integrates with platform services (logging-service, cache-service), and leverages shared libraries (libs/go) for centralized, reusable code.

## Glossary

- **File_Upload_Service**: The microservice responsible for handling file uploads, storage, validation, and retrieval
- **Logging_Service**: Platform centralized logging microservice (platform/logging-service) using gRPC for log ingestion
- **Cache_Service**: Platform distributed cache microservice (platform/cache-service) using Redis with gRPC/REST APIs
- **Libs_Go**: Shared Go libraries (libs/go/src) providing domain primitives, validation, resilience, and observability
- **S3_Provider**: AWS S3 storage backend for file persistence
- **Chunk_Manager**: Component managing chunked/multipart file uploads
- **Rate_Limiter**: Component enforcing request rate limits per tenant
- **Validator**: Component validating file types, sizes, and content
- **Metadata_Repository**: PostgreSQL-backed storage for file metadata
- **Audit_Logger**: Component recording file operation audit trails

## Requirements

### Requirement 1: Go Version and Dependency Modernization

**User Story:** As a platform engineer, I want the service to use Go 1.24+ with modern dependencies, so that we benefit from latest security patches and performance improvements.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use Go 1.24 or later as the minimum Go version
2. WHEN building the service THEN the File_Upload_Service SHALL use go-redis/redis/v9 instead of deprecated go-redis/v8
3. THE File_Upload_Service SHALL use AWS SDK Go v2 latest stable versions (2024.x+)
4. THE File_Upload_Service SHALL use OpenTelemetry Go SDK 1.28+ for observability
5. WHEN dependencies are updated THEN the File_Upload_Service SHALL remove all deprecated or vulnerable packages

### Requirement 2: Centralized Logging Integration

**User Story:** As a DevOps engineer, I want all logs sent to the centralized logging service, so that I can correlate logs across microservices.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL integrate with Logging_Service via gRPC client
2. WHEN logging events THEN the File_Upload_Service SHALL use libs/go/src/observability for context propagation
3. THE File_Upload_Service SHALL include correlation_id, tenant_id, and user_id in all log entries
4. WHEN Logging_Service is unavailable THEN the File_Upload_Service SHALL fallback to local structured JSON logging
5. THE File_Upload_Service SHALL use circuit breaker pattern for Logging_Service client resilience
6. THE File_Upload_Service SHALL remove internal observability/logger.go and use centralized implementation

### Requirement 3: Cache Service Integration

**User Story:** As a developer, I want file metadata cached via the platform cache service, so that repeated lookups are fast and consistent across instances.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL integrate with Cache_Service via gRPC client for metadata caching
2. WHEN retrieving file metadata THEN the File_Upload_Service SHALL check Cache_Service before querying database
3. WHEN file metadata is created or updated THEN the File_Upload_Service SHALL invalidate corresponding cache entries
4. THE File_Upload_Service SHALL use namespace "file-upload" for all cache keys
5. WHEN Cache_Service is unavailable THEN the File_Upload_Service SHALL fallback to direct database queries
6. THE File_Upload_Service SHALL use libs/go/src/fault for circuit breaker on cache client

### Requirement 4: Shared Library Integration

**User Story:** As a developer, I want to use centralized shared libraries, so that code is not duplicated across services.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/domain for UUID, ULID, and Email primitives
2. THE File_Upload_Service SHALL use libs/go/src/errors for typed error handling with HTTP/gRPC mapping
3. THE File_Upload_Service SHALL use libs/go/src/validation for composable input validation
4. THE File_Upload_Service SHALL use libs/go/src/http for HTTP client with retry and timeout
5. THE File_Upload_Service SHALL use libs/go/src/security for constant-time comparison and sanitization
6. THE File_Upload_Service SHALL use libs/go/src/workerpool for async task processing
7. THE File_Upload_Service SHALL use libs/go/src/pagination for cursor-based pagination
8. THE File_Upload_Service SHALL remove redundant internal implementations replaced by libs

### Requirement 5: Resilience Pattern Modernization

**User Story:** As a platform engineer, I want consistent resilience patterns across all external dependencies, so that the service degrades gracefully under failure.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/fault for circuit breaker configuration
2. THE File_Upload_Service SHALL use libs/go/src/resilience for retry with exponential backoff
3. WHEN S3 operations fail THEN the File_Upload_Service SHALL apply circuit breaker with 5 failure threshold
4. WHEN Redis operations fail THEN the File_Upload_Service SHALL apply circuit breaker with 3 failure threshold
5. WHEN database operations fail THEN the File_Upload_Service SHALL apply retry with max 3 attempts
6. THE File_Upload_Service SHALL expose circuit breaker state via Prometheus metrics

### Requirement 6: Rate Limiting Modernization

**User Story:** As a security engineer, I want rate limiting to use the cache service, so that limits are enforced consistently across service instances.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use Cache_Service for distributed rate limit state
2. WHEN rate limit is exceeded THEN the File_Upload_Service SHALL return HTTP 429 with Retry-After header
3. THE File_Upload_Service SHALL support per-tenant and per-user rate limits
4. THE File_Upload_Service SHALL use sliding window algorithm for rate limiting
5. WHEN Cache_Service is unavailable THEN the File_Upload_Service SHALL use local in-memory rate limiting as fallback

### Requirement 7: File Validation Enhancement

**User Story:** As a security engineer, I want comprehensive file validation, so that malicious files are rejected before storage.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL validate file content using magic bytes detection
2. THE File_Upload_Service SHALL validate file extension matches detected MIME type
3. THE File_Upload_Service SHALL enforce configurable file size limits per tenant
4. THE File_Upload_Service SHALL use libs/go/src/validation for validation composition
5. WHEN validation fails THEN the File_Upload_Service SHALL return specific error codes with details
6. THE File_Upload_Service SHALL support configurable allowlist of MIME types per tenant

### Requirement 8: Chunked Upload Modernization

**User Story:** As a user, I want to upload large files in chunks, so that uploads can resume after network interruptions.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use Cache_Service for chunk session state management
2. WHEN a chunk is uploaded THEN the File_Upload_Service SHALL verify chunk checksum using SHA-256
3. THE File_Upload_Service SHALL support parallel chunk uploads within a session
4. WHEN all chunks are uploaded THEN the File_Upload_Service SHALL assemble and verify final file hash
5. THE File_Upload_Service SHALL automatically cleanup expired sessions after configurable TTL
6. THE File_Upload_Service SHALL use libs/go/src/workerpool for parallel chunk assembly

### Requirement 9: Storage Provider Abstraction

**User Story:** As a platform engineer, I want storage provider abstraction, so that we can switch between S3-compatible backends.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL define a Storage interface for provider abstraction
2. THE File_Upload_Service SHALL implement S3 provider using AWS SDK Go v2
3. THE File_Upload_Service SHALL support presigned URLs for direct client uploads
4. THE File_Upload_Service SHALL support presigned URLs for time-limited downloads
5. WHEN generating storage paths THEN the File_Upload_Service SHALL use tenant-isolated hierarchical structure
6. THE File_Upload_Service SHALL use libs/go/src/fault for storage operation resilience

### Requirement 10: Metadata Repository Modernization

**User Story:** As a developer, I want type-safe database operations, so that SQL errors are caught at compile time.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use sqlx with named parameters for all queries
2. THE File_Upload_Service SHALL use libs/go/src/pagination for cursor-based file listing
3. THE File_Upload_Service SHALL implement soft delete with deleted_at timestamp
4. WHEN querying files THEN the File_Upload_Service SHALL support filtering by status, date range, and MIME type
5. THE File_Upload_Service SHALL use database transactions for multi-step operations
6. THE File_Upload_Service SHALL use libs/go/src/errors for database error mapping

### Requirement 11: API Response Standardization

**User Story:** As an API consumer, I want consistent response formats, so that error handling is predictable.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/errors for HTTP status code mapping
2. WHEN returning errors THEN the File_Upload_Service SHALL include error code, message, and request_id
3. THE File_Upload_Service SHALL use RFC 7807 Problem Details format for error responses
4. THE File_Upload_Service SHALL include X-Correlation-ID header in all responses
5. THE File_Upload_Service SHALL use libs/go/src/codec for JSON serialization

### Requirement 12: Observability Enhancement

**User Story:** As a DevOps engineer, I want comprehensive observability, so that I can monitor service health and performance.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use OpenTelemetry for distributed tracing
2. THE File_Upload_Service SHALL propagate W3C Trace Context headers
3. THE File_Upload_Service SHALL expose Prometheus metrics at /metrics endpoint
4. THE File_Upload_Service SHALL record upload latency, size, and error rate metrics
5. THE File_Upload_Service SHALL use libs/go/src/observability for context propagation
6. THE File_Upload_Service SHALL integrate with Logging_Service for centralized log aggregation

### Requirement 13: Security Hardening

**User Story:** As a security engineer, I want security best practices enforced, so that the service is protected against common attacks.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/security for input sanitization
2. THE File_Upload_Service SHALL use constant-time comparison for token validation
3. THE File_Upload_Service SHALL sanitize filenames to prevent path traversal attacks
4. THE File_Upload_Service SHALL validate JWT tokens using JWKS endpoint
5. THE File_Upload_Service SHALL enforce tenant isolation for all file operations
6. WHEN logging THEN the File_Upload_Service SHALL redact sensitive data (tokens, passwords)

### Requirement 14: Async Processing Modernization

**User Story:** As a developer, I want async task processing using shared worker pool, so that background jobs are handled efficiently.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/workerpool for async task processing
2. THE File_Upload_Service SHALL support virus scanning as async task
3. THE File_Upload_Service SHALL support thumbnail generation as async task
4. WHEN async task fails THEN the File_Upload_Service SHALL retry with exponential backoff
5. THE File_Upload_Service SHALL expose async queue depth via Prometheus metrics
6. THE File_Upload_Service SHALL use libs/go/src/resilience for task retry logic

### Requirement 15: Configuration Modernization

**User Story:** As a DevOps engineer, I want configuration from environment variables, so that the service is 12-factor compliant.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/config for configuration management
2. THE File_Upload_Service SHALL support environment variable overrides for all settings
3. THE File_Upload_Service SHALL validate configuration at startup
4. WHEN required configuration is missing THEN the File_Upload_Service SHALL fail fast with clear error message
5. THE File_Upload_Service SHALL support hot-reload for non-critical configuration changes

### Requirement 16: Health Check Enhancement

**User Story:** As a platform engineer, I want comprehensive health checks, so that Kubernetes can properly manage service lifecycle.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL expose /health/live endpoint for liveness probe
2. THE File_Upload_Service SHALL expose /health/ready endpoint for readiness probe
3. WHEN database is unavailable THEN the readiness probe SHALL return unhealthy
4. WHEN Cache_Service is unavailable THEN the readiness probe SHALL return degraded status
5. THE File_Upload_Service SHALL use libs/go/src/server for health endpoint implementation

### Requirement 17: Graceful Shutdown

**User Story:** As a platform engineer, I want graceful shutdown, so that in-flight requests complete before termination.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/server for graceful shutdown handling
2. WHEN SIGTERM is received THEN the File_Upload_Service SHALL stop accepting new requests
3. WHEN SIGTERM is received THEN the File_Upload_Service SHALL wait for in-flight requests to complete
4. THE File_Upload_Service SHALL have configurable shutdown timeout (default 30 seconds)
5. WHEN shutdown timeout is exceeded THEN the File_Upload_Service SHALL force terminate

### Requirement 18: Test Architecture Separation

**User Story:** As a developer, I want tests separated from source code, so that the codebase is organized and maintainable.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL organize tests in tests/ directory separate from internal/
2. THE File_Upload_Service SHALL have property-based tests using rapid library
3. THE File_Upload_Service SHALL have unit tests for all business logic
4. THE File_Upload_Service SHALL have integration tests for external dependencies
5. THE File_Upload_Service SHALL achieve minimum 80% code coverage on core paths
6. THE File_Upload_Service SHALL use libs/go/src/testing for test utilities and generators

### Requirement 19: Code Deduplication

**User Story:** As a developer, I want zero code duplication, so that maintenance is simplified and bugs are fixed in one place.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL remove internal/observability/ and use libs/go/src/observability
2. THE File_Upload_Service SHALL remove internal/hash/ and use libs/go/src/security for hashing
3. THE File_Upload_Service SHALL remove redundant error types and use libs/go/src/errors
4. THE File_Upload_Service SHALL remove internal validation logic duplicated in libs/go/src/validation
5. THE File_Upload_Service SHALL centralize all Redis operations through Cache_Service client

### Requirement 20: API Versioning

**User Story:** As an API consumer, I want versioned APIs, so that breaking changes don't affect existing integrations.

#### Acceptance Criteria

1. THE File_Upload_Service SHALL use libs/go/src/versioning for API version handling
2. THE File_Upload_Service SHALL support /api/v1/ prefix for current API version
3. WHEN API version is deprecated THEN the File_Upload_Service SHALL return Deprecation header
4. THE File_Upload_Service SHALL document breaking changes in CHANGELOG
5. THE File_Upload_Service SHALL support version extraction from URL path or Accept header
