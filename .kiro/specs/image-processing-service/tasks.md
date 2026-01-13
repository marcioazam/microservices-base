# Implementation Plan: Image Processing Service

## Overview

This plan implements the Image Processing Service as a Node.js microservice using Fastify, Sharp, BullMQ, and Redis. Tasks are organized to build incrementally, with core functionality first, then async processing, caching, and finally observability.

## Tasks

- [x] 1. Project Setup and Core Infrastructure
  - [x] 1.1 Initialize Node.js project with TypeScript configuration
    - Create package.json with dependencies: fastify, sharp, bullmq, ioredis, @aws-sdk/client-s3, pino, fast-check
    - Configure tsconfig.json with strict mode
    - Set up ESLint and Prettier
    - _Requirements: N/A (infrastructure)_

  - [x] 1.2 Create project directory structure
    - Create src/api/controllers/, src/api/middlewares/, src/api/routes/
    - Create src/services/image/, src/services/job/, src/services/cache/, src/services/storage/
    - Create src/domain/entities/, src/domain/types/, src/domain/errors/
    - Create src/infrastructure/, src/config/
    - Create tests/unit/, tests/integration/, tests/e2e/
    - _Requirements: N/A (infrastructure)_

  - [x] 1.3 Implement configuration module
    - Create src/config/index.ts with environment variable loading
    - Define config schema for server, redis, s3, auth settings
    - _Requirements: N/A (infrastructure)_

  - [x] 1.4 Implement error types and response utilities
    - Create src/domain/errors/errors.ts with ErrorCode enum
    - Create src/domain/errors/app-error.ts with custom error class
    - Create src/shared/utils/response.ts for consistent API responses
    - _Requirements: 12.1, 12.2_

- [x] 2. Image Processing Core (Sharp Integration)
  - [x] 2.1 Implement ImageService resize functionality
    - Create src/services/image/image.service.ts
    - Implement resize() with width, height, maintainAspectRatio, quality options
    - Use Sharp for image processing
    - _Requirements: 1.1, 1.2, 1.3, 1.6_

  - [x] 2.2 Write property tests for resize
    - **Property 1: Resize Dimensions Accuracy**
    - **Property 2: Aspect Ratio Preservation**
    - **Validates: Requirements 1.1, 1.2**

  - [x] 2.3 Implement ImageService format conversion
    - Implement convert() supporting jpeg, png, gif, webp, tiff
    - Handle transparency to opaque format conversion with background color
    - _Requirements: 2.1, 2.3, 2.5_

  - [x] 2.4 Write property tests for format conversion
    - **Property 7: Format Conversion Correctness**
    - **Property 8: Transparency Background Handling**
    - **Validates: Requirements 2.1, 2.3**

  - [x] 2.5 Implement ImageService adjustments
    - Implement adjust() for brightness, contrast, saturation (-100 to 100)
    - Apply adjustments in sequence when multiple specified
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [x] 2.6 Write property tests for adjustments
    - **Property 10: Brightness Adjustment Identity**
    - **Property 11: Contrast Adjustment Identity**
    - **Property 12: Saturation Desaturation**
    - **Validates: Requirements 3.1, 3.2, 3.3**

  - [x] 2.7 Implement ImageService rotation and flip
    - Implement rotate() with angle (0-360) and background color
    - Implement flip() with horizontal and vertical options
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [x] 2.8 Write property tests for rotation and flip
    - **Property 14: Rotation Identity**
    - **Property 15: Flip Idempotence**
    - **Validates: Requirements 4.1, 4.2, 4.3**

  - [x] 2.9 Implement ImageService watermark
    - Implement watermark() for text and image overlays
    - Support position presets and custom coordinates
    - Support opacity and font options
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [x] 2.10 Write property tests for watermark
    - **Property 16: Watermark Presence**
    - **Property 17: Watermark Opacity Zero**
    - **Validates: Requirements 5.1, 5.2, 5.3**

  - [x] 2.11 Implement ImageService compression
    - Implement compress() with lossy and lossless modes
    - Return compression statistics (original size, new size, ratio)
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [x] 2.12 Write property tests for compression
    - **Property 18: Lossy Compression Effectiveness**
    - **Property 19: Lossless Compression Integrity**
    - **Property 20: Compression Statistics Accuracy**
    - **Validates: Requirements 6.1, 6.2, 6.3**

- [x] 3. Checkpoint - Core Image Processing
  - Ensure all image processing tests pass
  - Verify Sharp integration works correctly
  - Ask the user if questions arise

- [x] 4. Request Validation
  - [x] 4.1 Implement request validators
    - Create src/api/validators/resize.validator.ts
    - Create src/api/validators/convert.validator.ts
    - Create src/api/validators/adjust.validator.ts
    - Create src/api/validators/watermark.validator.ts
    - Validate dimensions, formats, ranges
    - _Requirements: 1.5, 2.4, 3.5_

  - [x] 4.2 Write property tests for validation
    - **Property 5: Invalid Dimension Rejection**
    - **Property 9: Unsupported Format Rejection**
    - **Property 13: Adjustment Range Validation**
    - **Validates: Requirements 1.5, 2.4, 3.5**

- [x] 5. Storage Service
  - [x] 5.1 Implement StorageService with S3
    - Create src/services/storage/storage.service.ts
    - Implement upload(), download(), getSignedUrl(), delete()
    - Support configurable S3 endpoint (AWS S3 or MinIO)
    - _Requirements: 7.3, 7.4_

  - [x] 5.2 Write property tests for storage
    - **Property 23: Storage Round-Trip**
    - **Validates: Requirements 7.3, 7.4**

- [x] 6. API Layer
  - [x] 6.1 Implement Fastify server setup
    - Create src/infrastructure/server.ts
    - Configure multipart upload support
    - Set up request logging with correlation IDs
    - _Requirements: 12.3_

  - [x] 6.2 Implement upload controller
    - Create src/api/controllers/upload.controller.ts
    - Handle multipart/form-data uploads
    - Handle URL-based image fetching
    - Validate file size and image format
    - _Requirements: 7.1, 7.2, 7.5, 7.6_

  - [x] 6.3 Write property tests for upload validation
    - **Property 21: Upload Validation**
    - **Property 22: Invalid File Rejection**
    - **Validates: Requirements 7.1, 7.6**

  - [x] 6.4 Implement image processing controller
    - Create src/api/controllers/image.controller.ts
    - Implement endpoints: resize, convert, adjust, rotate, watermark, compress
    - Support both sync and async modes
    - _Requirements: 1.1-1.6, 2.1-2.5, 3.1-3.5, 4.1-4.5, 5.1-5.6, 6.1-6.5_

  - [x] 6.5 Implement response formatting
    - Support binary and base64 JSON responses based on Accept header
    - Include metadata in all responses
    - _Requirements: 12.4, 12.5_

  - [x] 6.6 Write property tests for response format
    - **Property 32: Response Format Consistency**
    - **Property 33: Request ID Presence**
    - **Property 34: HTTP Status Code Correctness**
    - **Property 35: Response Format Options**
    - **Validates: Requirements 12.1, 12.2, 12.3, 12.4, 12.5**

- [x] 7. Checkpoint - Sync API Complete
  - Ensure all API endpoints work synchronously
  - Verify request validation and error responses
  - Ask the user if questions arise

- [x] 8. Async Processing (BullMQ)
  - [x] 8.1 Implement JobService
    - Create src/services/job/job.service.ts
    - Implement enqueue(), getStatus(), getResult(), cancel()
    - Use BullMQ for job queue management
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

  - [x] 8.2 Implement queue worker
    - Create src/infrastructure/worker.ts
    - Process image operations from queue
    - Update job status and store results
    - Handle failures with error recording
    - _Requirements: 8.2, 8.4, 8.5_

  - [x] 8.3 Implement job controller
    - Create src/api/controllers/job.controller.ts
    - Implement getStatus, getResult, cancel endpoints
    - _Requirements: 8.3, 8.6_

  - [x] 8.4 Write property tests for async processing
    - **Property 24: Async Job ID Return**
    - **Property 25: Job Status Validity**
    - **Property 26: Completed Job Result Availability**
    - **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.6**

- [x] 9. Caching Layer
  - [x] 9.1 Implement CacheService
    - Create src/services/cache/cache.service.ts
    - Implement generateKey() with input hash and operation parameters
    - Implement get(), set(), invalidate()
    - Use Redis for cache storage
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

  - [x] 9.2 Integrate caching into image processing flow
    - Check cache before processing
    - Store results in cache after processing
    - _Requirements: 9.2_

  - [x] 9.3 Write property tests for caching
    - **Property 27: Cache Key Determinism**
    - **Property 28: Cache Hit Consistency**
    - **Validates: Requirements 9.1, 9.2**

- [x] 10. Checkpoint - Async and Caching Complete
  - Ensure async job processing works end-to-end
  - Verify cache hit/miss behavior
  - Ask the user if questions arise

- [x] 11. Authentication and Authorization
  - [x] 11.1 Implement auth middleware
    - Create src/api/middlewares/auth.middleware.ts
    - Validate JWT tokens
    - Extract user identity and permissions
    - _Requirements: 10.1, 10.2, 10.3_

  - [x] 11.2 Implement rate limiting middleware
    - Create src/api/middlewares/rate-limit.middleware.ts
    - Use Redis for distributed rate limiting
    - Return 429 with retry-after header
    - _Requirements: 10.4_

  - [x] 11.3 Write property tests for auth
    - **Property 29: JWT Extraction**
    - **Property 30: Authentication Enforcement**
    - **Property 31: Authorization Enforcement**
    - **Validates: Requirements 10.1, 10.2, 10.3**

- [x] 12. Batch Processing
  - [x] 12.1 Implement batch resize endpoint
    - Accept array of images with shared options
    - Process in parallel with concurrency limit
    - Return results array matching input order
    - _Requirements: 1.4_

  - [x] 12.2 Write property tests for batch processing
    - **Property 4: Batch Processing Completeness**
    - **Validates: Requirements 1.4**

- [x] 13. Observability
  - [x] 13.1 Implement Prometheus metrics
    - Create src/infrastructure/metrics.ts
    - Expose request count, latency histograms, error rates
    - Add /metrics endpoint
    - _Requirements: 11.1_

  - [x] 13.2 Implement health check endpoint
    - Create /health endpoint
    - Check Redis, S3, and queue connectivity
    - _Requirements: 11.3_

  - [x] 13.3 Configure structured logging
    - Use Pino for JSON logging
    - Include correlation IDs in all logs
    - Configure log levels
    - _Requirements: 11.2_

  - [x] 13.4 Implement OpenTelemetry tracing
    - Configure trace context propagation
    - Add spans for image processing operations
    - _Requirements: 11.4_

- [x] 14. Docker and Deployment
  - [x] 14.1 Create Dockerfile
    - Multi-stage build for production
    - Include Sharp native dependencies
    - Configure health check
    - _Requirements: N/A (infrastructure)_

  - [x] 14.2 Create docker-compose configuration
    - Add to deploy/docker/image-processing-service/
    - Configure Redis, MinIO dependencies
    - Set up networking with other services
    - _Requirements: N/A (infrastructure)_

  - [x] 14.3 Create Kubernetes manifests
    - Create deployment, service, configmap
    - Configure resource limits and HPA
    - Add to deploy/kubernetes/
    - _Requirements: N/A (infrastructure)_

- [x] 15. Final Checkpoint
  - Run full test suite (unit, property, integration)
  - Verify all endpoints work with authentication
  - Test async processing end-to-end
  - Verify metrics and health endpoints
  - All implementation complete

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using fast-check (100+ iterations)
- Unit tests validate specific examples and edge cases
- The service follows the monorepo structure defined in steering rules
