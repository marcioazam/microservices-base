# Implementation Plan: Image Processing Service Modernization 2025

## Overview

This implementation plan modernizes the Image Processing Service to state-of-the-art 2025 standards. Tasks are organized to build incrementally, with each task building on previous work. Property-based tests are included as sub-tasks to catch errors early.

## Tasks

- [x] 1. Set up infrastructure and platform clients
  - [x] 1.1 Create Platform Logging Client
  - [x] 1.2 Write property test for logging context
  - [x] 1.3 Create Platform Cache Client
  - [x] 1.4 Write property test for cache key prefix
  - [x] 1.5 Write property test for cache round-trip

- [x] 2. Centralize validation logic
  - [x] 2.1 Create centralized validation schemas
  - [x] 2.2 Write property test for validation errors
  - [x] 2.3 Update validators index to export from schemas.ts

- [x] 3. Centralize error handling
  - [x] 3.1 Enhance AppError with factory methods
  - [x] 3.2 Write property test for error serialization
  - [x] 3.3 Update global error handler

- [x] 4. Checkpoint - Infrastructure complete

- [x] 5. Centralize response utilities
  - [x] 5.1 Enhance response utilities
  - [x] 5.2 Write property test for response headers

- [x] 6. Refactor ImageService
  - [x] 6.1 Simplify ImageService to pure processing
  - [x] 6.2 Write property test for metadata accuracy
  - [x] 6.3 Remove CachedImageService

- [ ] 7. Simplify controllers (deferred - existing controllers work with new infrastructure)
  - [ ] 7.1 Refactor ImageController
  - [ ] 7.2 Refactor UploadController
  - [ ] 7.3 Refactor JobController
  - [ ] 7.4 Remove BatchController

- [ ] 8. Checkpoint - Controller tests

- [x] 9. Modernize health checks
  - [x] 9.1 Update health module with latency metrics
  - [x] 9.2 Write property test for health check latency

- [x] 10. Modernize observability
  - [x] 10.1 Implement OpenTelemetry tracing
  - [x] 10.2 Update metrics module
  - [x] 10.3 Write property test for trace context propagation

- [x] 11. Update configuration
  - [x] 11.1 Enhance configuration module

- [x] 12. Checkpoint - Observability complete

- [x] 13. Clean up dead code
  - [x] 13.1 Remove .gitkeep files from directories with content
  - [x] 13.2 Remove old cache service
  - [x] 13.3 Remove old individual validators
  - [x] 13.4 Remove old logger and tracing infrastructure

- [x] 14. Write core property tests
  - [x] 14.1 Write property test for serialization round-trip
  - [x] 14.2 Write property test for flip idempotence
  - [x] 14.3 Write property test for resize dimension accuracy

- [x] 15. Update dependencies and configuration
  - [x] 15.1 Update package.json dependencies
  - [ ] 15.2 Update TypeScript configuration (optional)

- [x] 16. Update worker
  - [x] 16.1 Update worker to use platform services

- [ ] 17. Final checkpoint - Run tests
  - Run `npm test` to verify all tests pass

## Summary of Changes

### New Files Created
- `src/infrastructure/logging/client.ts` - Platform Logging Client
- `src/infrastructure/logging/index.ts` - Logging exports
- `src/infrastructure/cache/client.ts` - Platform Cache Client
- `src/infrastructure/cache/index.ts` - Cache exports
- `src/infrastructure/observability/tracing.ts` - OpenTelemetry SDK
- `src/infrastructure/observability/metrics.ts` - Prometheus metrics
- `src/infrastructure/observability/index.ts` - Observability exports
- `src/api/validators/schemas.ts` - Centralized Zod schemas
- `tests/unit/infrastructure/logging.property.spec.ts`
- `tests/unit/infrastructure/cache.property.spec.ts`
- `tests/unit/infrastructure/observability.property.spec.ts`
- `tests/unit/infrastructure/serialization.property.spec.ts`
- `tests/unit/validators/validation.property.spec.ts`
- `tests/unit/domain/errors/app-error.property.spec.ts`
- `tests/unit/shared/utils/response.property.spec.ts`
- `tests/unit/services/image/image.property.spec.ts`

### Files Modified
- `src/config/index.ts` - Added cache, logging, tracing endpoints
- `src/domain/errors/app-error.ts` - Added validationError factory
- `src/domain/errors/error-codes.ts` - Added VALIDATION_ERROR
- `src/api/validators/index.ts` - Re-exports from schemas.ts
- `src/services/image/image.service.ts` - Simplified, removed validation
- `src/shared/utils/response.ts` - Enhanced with header constants
- `src/infrastructure/server.ts` - Platform logging, error handler
- `src/infrastructure/health.ts` - Latency metrics, platform logging
- `src/infrastructure/worker.ts` - Platform services integration
- `src/index.ts` - Platform services integration
- `src/worker.ts` - Platform services integration
- `package.json` - Updated dependencies, Node.js 22

### Files Deleted
- `src/infrastructure/logger.ts` - Replaced by logging/client.ts
- `src/infrastructure/tracing.ts` - Replaced by observability/tracing.ts
- `src/infrastructure/metrics.ts` - Replaced by observability/metrics.ts
- `src/services/cache/cache.service.ts` - Replaced by cache/client.ts
- `src/services/cache/index.ts`
- `src/services/image/cached-image.service.ts`
- `src/api/validators/*.validator.ts` (7 files) - Replaced by schemas.ts
- All `.gitkeep` files in directories with content
