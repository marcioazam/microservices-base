# Implementation Plan: IAM Policy Service Modernization

## Overview

This implementation plan modernizes the IAM Policy Service to state-of-the-art December 2025 standards. The plan follows incremental development with property-based testing validation at each stage. All tasks use shared libraries from `libs/go/` and integrate with platform services (cache-service, logging-service).

## Tasks

- [x] 1. Project restructure and dependency setup
  - [x] 1.1 Update go.mod with Go 1.24+ and modern dependencies
    - Update module to Go 1.24
    - Add platform libs dependencies (cache, logging, config, errors, fault, grpc, observability, server, testing)
    - Update OPA to v1.0+, gRPC to v1.70+, OpenTelemetry to v1.35+
    - Add pgregory.net/rapid for property-based testing
    - _Requirements: 12.1, 12.4_
  - [x] 1.2 Create new directory structure
    - Create `tests/unit/`, `tests/property/`, `tests/integration/`, `tests/testutil/`
    - Move existing `internal/policy/engine_test.go` to `tests/unit/policy/`
    - Create empty test files for each component
    - _Requirements: 11.4, 12.1_

- [x] 2. Configuration management modernization
  - [x] 2.1 Implement centralized Config using libs/go/src/config
    - Create `internal/config/config.go` with all config structs
    - Use `libs/go/src/config` for loading from env and YAML
    - Implement validation for required keys
    - Support hot-reload for non-critical values
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6_
  - [x] 2.2 Write property test for configuration loading
    - **Property 1: Configuration Loading Consistency**
    - **Validates: Requirements 1.1, 1.2, 1.3**

- [x] 3. Logging client integration
  - [x] 3.1 Integrate libs/go/src/logging client
    - Create logging client initialization in main.go
    - Configure with service name, buffer size, flush interval
    - Implement local fallback when logging-service unavailable
    - Add context enrichment (correlation_id, trace_id, span_id)
    - _Requirements: 2.1, 2.3, 2.6_
  - [x] 3.2 Write property test for log entry enrichment
    - **Property 3: Log Entry Enrichment**
    - **Validates: Requirements 2.6**

- [x] 4. Cache client integration
  - [x] 4.1 Implement decision cache using libs/go/src/cache
    - Create `internal/cache/decision_cache.go`
    - Implement DecisionCache interface with Get/Set/Delete/Invalidate
    - Configure namespace `iam-policy` for all operations
    - Implement local fallback when cache-service unavailable
    - Generate deterministic cache keys from authorization input
    - _Requirements: 2.2, 2.4, 2.5, 3.6, 3.7_
  - [x] 4.2 Write property tests for cache operations
    - **Property 2: Cache Namespace Isolation**
    - **Property 5: Decision Cache Round-Trip**
    - **Validates: Requirements 2.5, 3.6, 3.7**

- [x] 5. Checkpoint - Ensure infrastructure integration works

- [x] 6. Policy engine modernization
  - [x] 6.1 Refactor policy engine with caching support
    - Update `internal/policy/engine.go` to use decision cache
    - Add metrics for policy evaluation
    - Implement cache invalidation on policy reload
    - Use libs/go/src/logging for structured logging
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.8_
  - [x] 6.2 Write property tests for policy evaluation
    - **Property 4: Policy Evaluation Determinism**
    - **Validates: Requirements 3.4, 3.5**

- [x] 7. RBAC module enhancement
  - [x] 7.1 Enhance role hierarchy with cycle detection
    - Update `internal/rbac/hierarchy.go` with HasCircularDependency
    - Implement permission caching per role
    - Add thread-safe operations with sync.RWMutex
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_
  - [x] 7.2 Write property tests for role hierarchy
    - **Property 6: Permission Inheritance Completeness**
    - **Property 7: Circular Dependency Detection**
    - **Validates: Requirements 4.2, 4.3, 4.4**

- [x] 8. Authorization service implementation
  - [x] 8.1 Create centralized authorization service
    - Create `internal/service/authorization.go`
    - Implement Authorize, BatchAuthorize, GetPermissions, GetRoles
    - Integrate policy engine, role hierarchy, and CAEP emitter
    - Add audit logging for all authorization decisions
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.8_
  - [x] 8.2 Write property tests for authorization service
    - **Property 8: gRPC Request-Response Consistency**
    - **Validates: Requirements 5.1, 5.2, 5.8**

- [x] 9. Checkpoint - Ensure core business logic works

- [x] 10. gRPC handler modernization
  - [x] 10.1 Refactor gRPC handler with interceptors
    - Update `internal/grpc/handler.go` to use authorization service
    - Create `internal/grpc/interceptors.go` with custom interceptors
    - Use libs/go/src/grpc for error mapping and logging interceptors
    - Implement ReloadPolicies RPC
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_
  - [x] 10.2 Write unit tests for gRPC handler
    - Test all RPC methods with valid and invalid inputs
    - Test error mapping to gRPC status codes
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [x] 11. CAEP emitter modernization
  - [x] 11.1 Refactor CAEP emitter with logging integration
    - Update `internal/caep/emitter.go` to use libs/go/src/logging
    - Add structured logging for all events
    - Implement graceful handling when CAEP disabled
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_
  - [x] 11.2 Write property tests for CAEP events
    - **Property 9: CAEP Event Structure Completeness**
    - **Validates: Requirements 6.1, 6.2, 6.4**

- [x] 12. Health and observability
  - [x] 12.1 Implement health manager
    - Create `internal/health/manager.go`
    - Implement liveness and readiness handlers
    - Register health checks for cache and logging clients
    - Handle shutdown state for readiness
    - _Requirements: 7.7, 7.8, 7.9_
  - [x] 12.2 Implement OpenTelemetry tracing
    - Add otelgrpc interceptors for trace propagation
    - Configure trace context extraction and injection
    - Integrate with libs/go/src/observability
    - _Requirements: 7.1, 7.2_
  - [x] 12.3 Implement Prometheus metrics
    - Add metrics for authorization decisions, cache hits/misses, policy evaluations
    - Expose metrics at /metrics endpoint
    - _Requirements: 7.3, 7.4, 7.5, 7.6_
  - [x] 12.4 Write property tests for observability
    - **Property 10: Trace Context Propagation**
    - **Property 11: Metrics Recording Accuracy**
    - **Validates: Requirements 7.2, 7.4, 7.5, 7.6**

- [x] 13. Checkpoint - Ensure observability works

- [x] 14. Error handling and resilience
  - [x] 14.1 Implement circuit breaker for cache client
    - Configure circuit breaker using libs/go/src/fault
    - Implement fallback to local cache when circuit open
    - Add metrics for circuit breaker state
    - _Requirements: 9.2, 9.3_
  - [x] 14.2 Implement consistent error handling
    - Use libs/go/src/errors for error construction
    - Map internal errors to gRPC status codes
    - Include correlation IDs in all error responses
    - _Requirements: 9.1, 9.5, 9.6_
  - [x] 14.3 Write property tests for error handling
    - **Property 12: Circuit Breaker State Transitions**
    - **Property 13: Error Response Consistency**
    - **Validates: Requirements 9.3, 9.5, 9.6**

- [x] 15. Security hardening
  - [x] 15.1 Implement input validation
    - Validate all authorization request fields
    - Sanitize log output to prevent injection
    - Ensure internal errors not exposed in responses
    - _Requirements: 10.1, 10.2, 10.3_
  - [x] 15.2 Implement rate limiting
    - Add per-client rate limiting for authorization requests
    - Configure limits via environment variables
    - Return appropriate error when rate limited
    - _Requirements: 10.5_
  - [x] 15.3 Write property tests for security
    - **Property 14: Input Validation and Sanitization**
    - **Property 15: Rate Limiting Enforcement**
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.5**

- [x] 16. Graceful shutdown
  - [x] 16.1 Implement graceful shutdown using libs/go/src/server
    - Handle SIGTERM and SIGINT signals
    - Stop accepting new requests immediately
    - Wait for in-flight requests with configurable timeout
    - Flush logging buffers before shutdown
    - Close cache and logging client connections
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_
  - [x] 16.2 Write unit tests for graceful shutdown
    - Test signal handling
    - Test in-flight request completion
    - Test resource cleanup
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [x] 17. Main entry point modernization
  - [x] 17.1 Refactor main.go with dependency injection
    - Initialize all components with proper dependency injection
    - Wire up interceptor chain
    - Start gRPC server, health HTTP server, metrics HTTP server
    - Implement graceful shutdown coordination
    - _Requirements: 12.6_

- [x] 18. Final checkpoint - Full test suite

- [x] 19. Documentation and cleanup
  - [x] 19.1 Update README.md
    - Document new architecture and configuration
    - Add API reference for gRPC endpoints
    - Document environment variables
    - Add testing instructions
  - [x] 19.2 Remove legacy code
    - Remove any deprecated patterns
    - Ensure no file exceeds 400 lines
    - Verify no code duplication
    - _Requirements: 12.2, 12.3_

- [x] 20. Integration tests
  - [x] 20.1 Write integration tests for platform services
    - Test cache-service integration with fallback
    - Test logging-service integration with fallback
    - Test end-to-end authorization flow
    - _Requirements: 11.3_

## Notes

- All tasks are required for comprehensive testing and production readiness
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (100+ iterations each)
- Unit tests validate specific examples and edge cases
- All code uses shared libraries from `libs/go/src/`
- Integration with platform services (cache-service, logging-service) is mandatory
