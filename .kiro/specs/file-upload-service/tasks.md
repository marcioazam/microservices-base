# Implementation Plan: File Upload Service

## Overview

This implementation plan breaks down the File Upload Service into discrete, incremental tasks. Each task builds on previous work and includes testing requirements. The service will be implemented in Go using Gin framework, with AWS S3 as the primary storage provider and PostgreSQL for metadata.

## Tasks

- [x] 1. Project Setup and Core Structure
  - [x] 1.1 Initialize Go module and project structure
    - Create `services/file-upload/` directory structure
    - Initialize `go.mod` with module name `github.com/auth-platform/file-upload`
    - Create directories: `cmd/`, `internal/`, `pkg/`, `api/`, `configs/`, `migrations/`
    - Add `.gitignore` and `Makefile`
    - _Requirements: Project structure_

  - [x] 1.2 Set up configuration management
    - Create `internal/config/config.go` with Viper integration
    - Define configuration struct for storage, database, auth, rate limiting
    - Support environment variables and config files
    - _Requirements: Configuration management_

  - [x] 1.3 Set up logging and observability foundation
    - Create `internal/observability/logger.go` with structured JSON logging (zerolog)
    - Create `internal/observability/metrics.go` with Prometheus metrics
    - Create `internal/observability/tracing.go` with OpenTelemetry setup
    - _Requirements: 11.2, 11.3_

  - [x] 1.4 Write property test for structured logging format
    - **Property 13: Structured Logging Format**
    - **Validates: Requirements 11.2**

- [x] 2. Data Models and Database Layer
  - [x] 2.1 Create domain models
    - Create `internal/domain/file.go` with FileMetadata, FileStatus, ScanStatus
    - Create `internal/domain/chunk.go` with ChunkSession, ChunkData
    - Create `internal/domain/errors.go` with domain error types
    - _Requirements: Data models_

  - [x] 2.2 Create database migrations
    - Create `migrations/001_create_files_table.sql`
    - Create `migrations/002_create_chunk_sessions_table.sql`
    - Create `migrations/003_create_audit_logs_table.sql`
    - Add indexes for performance
    - _Requirements: Database schema_

  - [x] 2.3 Implement metadata store
    - Create `internal/storage/metadata/repository.go` implementing MetadataStore interface
    - Implement CRUD operations with sqlx
    - Implement pagination and search
    - _Requirements: 13.1, 13.2, 13.3, 13.4_

  - [x] 2.4 Write property test for metadata persistence
    - **Property 16: Metadata Persistence**
    - **Validates: Requirements 13.1**

  - [x] 2.5 Write property test for file listing and search
    - **Property 17: File Listing and Search**
    - **Validates: Requirements 13.3, 13.4**

- [x] 3. Checkpoint - Database Layer Complete
  - Ensure all database tests pass
  - Verify migrations run successfully
  - Ask the user if questions arise

- [x] 4. File Validation Components
  - [x] 4.1 Implement MIME type detection
    - Create `internal/validator/mimetype.go` with magic byte detection
    - Support JPEG, PNG, GIF, PDF, MP4, MOV, DOCX, XLSX
    - Use `github.com/h2non/filetype` for detection
    - _Requirements: 2.1, 2.4_

  - [x] 4.2 Implement file validator
    - Create `internal/validator/validator.go` implementing Validator interface
    - Implement type validation, size validation, extension matching
    - Make allowlist configurable
    - _Requirements: 2.1, 2.2, 2.3, 3.1, 3.2_

  - [x] 4.3 Write property test for file type validation
    - **Property 2: File Type Validation Correctness**
    - **Validates: Requirements 2.1, 2.2, 2.3**

  - [x] 4.4 Write property test for file size validation
    - **Property 3: File Size Validation**
    - **Validates: Requirements 3.1, 3.2**

- [x] 5. Hash Generation
  - [x] 5.1 Implement hash generator
    - Create `internal/hash/generator.go` implementing HashGenerator interface
    - Implement streaming SHA256 computation
    - Implement hash verification
    - _Requirements: 5.1, 5.3_

  - [x] 5.2 Write property test for hash computation
    - **Property 5: Hash Computation Correctness**
    - **Validates: Requirements 5.1**

- [-] 6. Cloud Storage Integration
  - [x] 6.1 Implement S3 storage provider
    - Create `internal/storage/s3/provider.go` implementing StorageProvider interface
    - Implement upload, download, delete operations
    - Implement signed URL generation
    - Use AWS SDK v2
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [x] 6.2 Implement storage path generator
    - Create `internal/storage/path.go` for path generation
    - Implement `/{tenant_id}/{year}/{month}/{day}/{file_hash}/{filename}` format
    - _Requirements: 4.5_

  - [x] 6.3 Write property test for storage path structure
    - **Property 4: Storage Path Structure**
    - **Validates: Requirements 4.5**

- [x] 7. Checkpoint - Core Storage Complete
  - Ensure all storage tests pass
  - Verify S3 integration works (with localstack or minio)
  - Ask the user if questions arise

- [-] 8. Authentication and Authorization
  - [x] 8.1 Implement JWT validation
    - Create `internal/auth/jwt.go` implementing AuthHandler interface
    - Implement JWKS-based token validation
    - Extract user context from claims
    - _Requirements: 8.1, 8.3, 8.4_

  - [x] 8.2 Implement authorization middleware
    - Create `internal/auth/middleware.go` with Gin middleware
    - Implement tenant isolation checks
    - _Requirements: 8.2_

  - [x] 8.3 Write property test for authentication enforcement
    - **Property 9: Authentication Enforcement**
    - **Validates: Requirements 8.1, 8.4**

  - [x] 8.4 Write property test for tenant isolation
    - **Property 10: Tenant Isolation**
    - **Validates: Requirements 8.2, 13.2**

- [-] 9. Rate Limiting
  - [-] 9.1 Implement rate limiter
    - Create `internal/ratelimit/limiter.go` with Redis-based rate limiting
    - Implement per-tenant rate limiting
    - Use sliding window algorithm
    - _Requirements: 10.2_

  - [ ] 9.2 Implement rate limit middleware
    - Create `internal/ratelimit/middleware.go` with Gin middleware
    - Return 429 with Retry-After header when exceeded
    - _Requirements: 10.3_

  - [ ] 9.3 Write property test for rate limiting
    - **Property 11: Rate Limiting Enforcement**
    - **Validates: Requirements 10.2, 10.3**

- [ ] 10. Audit Logging
  - [ ] 10.1 Implement audit logger
    - Create `internal/audit/logger.go` for audit log creation
    - Implement file operation logging (upload, access, delete)
    - Ensure no PII in logs
    - _Requirements: 12.1, 12.2, 12.3_

  - [ ] 10.2 Write property test for audit log completeness
    - **Property 14: Audit Log Completeness**
    - **Validates: Requirements 12.1, 12.2**

  - [ ] 10.3 Write property test for audit log security
    - **Property 15: Audit Log Security**
    - **Validates: Requirements 12.3**

- [ ] 11. Checkpoint - Security Layer Complete
  - Ensure all auth and audit tests pass
  - Verify JWT validation works with test tokens
  - Ask the user if questions arise

- [ ] 12. Upload API Implementation
  - [ ] 12.1 Implement upload handler
    - Create `internal/api/handlers/upload.go` with upload endpoint
    - Implement multipart form parsing
    - Wire validation, hashing, storage, metadata
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [ ] 12.2 Implement deduplication logic
    - Add hash-based deduplication in upload handler
    - Return existing file reference if duplicate
    - _Requirements: 5.2_

  - [ ] 12.3 Write property test for upload response completeness
    - **Property 1: Upload Response Completeness**
    - **Validates: Requirements 1.2, 5.4**

  - [ ] 12.4 Write property test for file deduplication
    - **Property 6: File Deduplication**
    - **Validates: Requirements 5.2**

- [ ] 13. Chunked Upload Implementation
  - [ ] 13.1 Implement chunk manager
    - Create `internal/chunk/manager.go` implementing ChunkManager interface
    - Use Redis for session state
    - Implement chunk storage and assembly
    - _Requirements: 6.1, 6.2, 6.3, 6.5_

  - [ ] 13.2 Implement chunked upload endpoints
    - Create `internal/api/handlers/chunk.go` with init, upload, complete endpoints
    - Implement session management
    - _Requirements: 6.1, 6.2, 6.4_

  - [ ] 13.3 Write property test for chunked upload round-trip
    - **Property 7: Chunked Upload Round-Trip**
    - **Validates: Requirements 6.3**

- [ ] 14. Async Processing
  - [ ] 14.1 Implement async processor
    - Create `internal/async/processor.go` implementing AsyncProcessor interface
    - Use goroutine worker pool
    - Implement task queuing and execution
    - _Requirements: 7.1, 7.2, 7.4_

  - [ ] 14.2 Implement malware scanner integration
    - Create `internal/scanner/clamav.go` implementing MalwareScanner interface
    - Integrate with ClamAV via clamd protocol
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

  - [ ] 14.3 Write property test for async processing independence
    - **Property 8: Async Processing Independence**
    - **Validates: Requirements 7.3**

- [ ] 15. Checkpoint - Upload Flow Complete
  - Ensure all upload tests pass
  - Verify end-to-end upload flow works
  - Ask the user if questions arise

- [ ] 16. File Management APIs
  - [ ] 16.1 Implement file retrieval endpoints
    - Create `internal/api/handlers/files.go` with get, list, search endpoints
    - Implement pagination and filtering
    - _Requirements: 13.1, 13.2, 13.3, 13.4_

  - [ ] 16.2 Implement file deletion
    - Add soft-delete endpoint
    - Implement retention period cleanup job
    - _Requirements: 14.1, 14.2, 14.3, 14.4_

  - [ ] 16.3 Implement download URL generation
    - Add endpoint for signed download URLs
    - Support configurable expiration
    - _Requirements: 4.2_

  - [ ] 16.4 Write property test for soft-delete behavior
    - **Property 18: Soft-Delete Behavior**
    - **Validates: Requirements 14.1**

  - [ ] 16.5 Write property test for delete authorization
    - **Property 19: Delete Authorization**
    - **Validates: Requirements 14.3**

- [ ] 17. Concurrency and Performance
  - [ ] 17.1 Implement concurrent upload handling
    - Add worker pool for storage operations
    - Implement connection pooling for database
    - _Requirements: 10.1, 10.4_

  - [ ] 17.2 Write property test for concurrent upload handling
    - **Property 12: Concurrent Upload Handling**
    - **Validates: Requirements 10.1**

- [ ] 18. API Router and Server
  - [ ] 18.1 Set up Gin router
    - Create `internal/api/router.go` with route definitions
    - Wire all handlers and middleware
    - Add health check endpoint
    - _Requirements: 11.4_

  - [ ] 18.2 Create main entry point
    - Create `cmd/server/main.go` with server initialization
    - Implement graceful shutdown
    - Wire dependency injection
    - _Requirements: Server setup_

- [ ] 19. Docker and Deployment
  - [ ] 19.1 Create Dockerfile
    - Create `services/file-upload/Dockerfile` with multi-stage build
    - Optimize for small image size
    - _Requirements: Deployment_

  - [ ] 19.2 Add to docker-compose
    - Update `deploy/docker/docker-compose.yml` with file-upload service
    - Add environment configuration
    - _Requirements: Deployment_

- [ ] 20. Final Checkpoint - Service Complete
  - Ensure all tests pass (unit, property, integration)
  - Verify Docker build succeeds
  - Run full end-to-end test
  - Ask the user if questions arise

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests use `github.com/leanovate/gopter` with 100+ iterations
- Unit tests target 80% coverage minimum
