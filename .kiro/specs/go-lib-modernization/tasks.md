# Implementation Plan: Go Library Modernization

## Overview

This implementation plan modernizes the `libs/go` shared library collection from 45+ micro-modules to ~12 cohesive domain modules, adopting Go 1.25 features and establishing clean source/test separation.

## Tasks

- [x] 1. Set up new directory structure and workspace
  - Create `libs/go/src/` and `libs/go/tests/` directories
  - Create new `go.work` file with 12 domain modules
  - Set up unified test runner script
  - _Requirements: 1.5, 2.1, 2.2, 2.4_

- [x] 2. Implement unified functional types module
  - [x] 2.1 Create `src/functional/` module with go.mod
    - Consolidate option.go, result.go, either.go from existing packages
    - Implement unified Functor interface
    - Add type conversion functions (EitherToResult, ResultToEither)
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_
  - [x] 2.2 Write property test for Either-Result round trip
    - **Property 1: Either-Result Round Trip**
    - **Validates: Requirements 3.2**
  - [x] 2.3 Migrate iterator.go, lazy.go, pipeline.go, stream.go, tuple.go
    - Ensure consistent Map/FlatMap/Match signatures
    - _Requirements: 3.3, 3.4_

- [x] 3. Implement centralized resilience errors module
  - [x] 3.1 Create `src/resilience/errors.go` with unified error types
    - Define ResilienceError base type with Code, Service, Message, CorrelationID
    - Implement CircuitOpenError, RateLimitError, TimeoutError, BulkheadFullError
    - Add error checking functions (IsCircuitOpen, IsRateLimited, etc.)
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  - [x] 3.2 Write property tests for error type hierarchy
    - **Property 2: Resilience Error Type Hierarchy**
    - **Property 3: Error Type Checking Functions**
    - **Validates: Requirements 4.2, 4.4**
  - [x] 3.3 Implement JSON serialization for errors
    - Add MarshalJSON/UnmarshalJSON methods
    - _Requirements: 4.5_
  - [x] 3.4 Write property test for error JSON round trip
    - **Property 4: Error JSON Round Trip**
    - **Validates: Requirements 4.5**

- [x] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement unified resilience configuration
  - [x] 5.1 Create `src/resilience/config.go` with unified Config pattern
    - Define CircuitBreakerConfig, RetryConfig, RateLimitConfig, BulkheadConfig, TimeoutConfig
    - Implement functional options pattern (WithThreshold, WithTimeout, etc.)
    - Add DefaultConfig() for each component
    - _Requirements: 5.1, 5.2, 5.3_
  - [x] 5.2 Implement configuration validation
    - Add Validate() method returning InvalidPolicyError for invalid configs
    - _Requirements: 5.4, 5.5_
  - [x] 5.3 Write property test for invalid configuration detection
    - **Property 5: Invalid Configuration Detection**
    - **Validates: Requirements 5.4**

- [x] 6. Consolidate resilience components
  - [x] 6.1 Migrate circuitbreaker.go to use centralized errors and config
    - Update to use ResilienceError types
    - Apply functional options pattern
    - _Requirements: 4.2, 5.1, 5.2_
  - [x] 6.2 Migrate retry.go with exponential backoff and jitter
    - Implement FullJitter, EqualJitter, DecorrelatedJitter strategies
    - Integrate with Result[T] type
    - _Requirements: 5.2, 5.3_
  - [x] 6.3 Migrate ratelimit.go, bulkhead.go, timeout.go
    - Apply unified config and error patterns
    - _Requirements: 4.2, 5.1_

- [x] 7. Implement unified collections module
  - [x] 7.1 Create `src/collections/` module with go.mod
    - Consolidate lru.go, maps.go, set.go, slices.go, queue.go, pqueue.go, sort.go
    - _Requirements: 1.1, 6.1_
  - [x] 7.2 Implement unified Iterator interface
    - Add Map, Filter, Reduce, ForEach operations
    - Use Go 1.23+ iterator pattern (func(yield func(T) bool))
    - _Requirements: 6.1, 6.2, 6.3_
  - [x] 7.3 Write property tests for collection operations
    - **Property 6: Collection Map Identity**
    - **Property 7: Collection IsEmpty Invariant**
    - **Property 8: Collection Contains After Add**
    - **Validates: Requirements 6.2, 6.4**
  - [x] 7.4 Implement unified Cache interface
    - Support TTL and LRU eviction strategies
    - Add GetOrCompute, sharded access for concurrency
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_
  - [x] 7.5 Write property tests for cache behavior
    - **Property 13: TTL Cache Expiry**
    - **Property 14: LRU Cache Eviction Order**
    - **Property 15: Cache GetOrCompute Behavior**
    - **Property 16: Cache Eviction Callback**
    - **Validates: Requirements 8.2, 8.4, 8.5**

- [x] 8. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Implement unified validation module
  - [x] 9.1 Create `src/utils/validation.go` consolidating validator and validated
    - Implement Validation[E, A] type with error accumulation
    - Add composable Rule[T] and Validator[T] types
    - Support field-level validation with path tracking
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  - [x] 9.2 Write property tests for validation
    - **Property 9: Validation Error Accumulation**
    - **Property 10: Validator And Composition**
    - **Property 11: Validation Path Tracking**
    - **Property 12: Validation Result Exclusivity**
    - **Validates: Requirements 7.2, 7.3, 7.4, 7.5**

- [x] 10. Implement unified concurrency module
  - [x] 10.1 Create `src/concurrency/` module with go.mod
    - Consolidate async.go, pool.go, errgroup.go, atomic.go, channels.go
    - _Requirements: 9.3_
  - [x] 10.2 Implement generic Future[T] with context support
    - Add WaitContext, Result() integration
    - _Requirements: 9.4, 9.5_
  - [x] 10.3 Write property tests for Future
    - **Property 18: Future Context Cancellation**
    - **Property 19: Future Result Integration**
    - **Validates: Requirements 9.4, 9.5**
  - [x] 10.4 Implement generic WorkerPool[T, R]
    - Add backpressure support via bounded channels
    - _Requirements: 9.3_

- [x] 11. Implement unified events module
  - [x] 11.1 Create `src/events/` module consolidating eventbus and pubsub
    - Implement generic EventBus[E] interface
    - Support sync and async delivery modes
    - Add event filtering and routing
    - _Requirements: 15.1, 15.2, 15.3, 15.4_
  - [x] 11.2 Write property tests for event system
    - **Property 22: Event Sync Delivery**
    - **Property 23: Event Filtering**
    - **Property 24: Event Retry on Failure**
    - **Validates: Requirements 15.2, 15.4, 15.5**
  - [x] 11.3 Implement retry policy for failed deliveries
    - Integrate with RetryConfig from resilience module
    - _Requirements: 15.5_

- [x] 12. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 13. Implement gRPC integration
  - [x] 13.1 Create `src/grpc/errors.go` with bidirectional error conversion
    - Implement ToGRPCError, FromGRPCError functions
    - Preserve error details and metadata
    - _Requirements: 14.1, 14.2, 14.3_
  - [x] 13.2 Write property test for gRPC error round trip
    - **Property 21: gRPC Error Round Trip**
    - **Validates: Requirements 14.1, 14.2**
  - [x] 13.3 Implement error conversion interceptors
    - Add unary and stream interceptors
    - Include correlation ID logging
    - _Requirements: 14.4, 14.5_

- [x] 14. Implement testing infrastructure
  - [x] 14.1 Create `src/testing/` module with generators
    - Implement generators for all domain types (Option, Result, Either, etc.)
    - Add shrinking support for minimal counterexamples
    - _Requirements: 10.1, 10.2_
  - [x] 14.2 Write property test for generator validity
    - **Property 20: Generator Validity**
    - **Validates: Requirements 10.1**
  - [x] 14.3 Implement synctest integration helpers
    - Add deterministic concurrency test utilities
    - _Requirements: 9.1, 9.2_
  - [x] 14.4 Write property test for seeded reproducibility
    - **Property 17: Seeded Test Reproducibility**
    - **Validates: Requirements 9.2**

- [x] 15. Migrate remaining modules
  - [x] 15.1 Migrate `src/optics/` (lens.go, prism.go)
    - Update imports to use new functional module
    - _Requirements: 1.1, 1.4_
  - [x] 15.2 Migrate `src/patterns/` (registry.go, spec.go)
    - Update to use unified collection interfaces
    - _Requirements: 1.1, 1.4_
  - [x] 15.3 Migrate `src/server/` (health.go, shutdown.go, tracing.go)
    - Integrate with unified error handling
    - _Requirements: 1.1, 1.4_
  - [x] 15.4 Migrate `src/utils/` remaining files
    - audit.go, codec.go, diff.go, error.go, merge.go, uuid.go
    - _Requirements: 1.1, 1.4_

- [x] 16. Create backward compatibility aliases
  - [x] 16.1 Create alias packages in old locations
    - Add type aliases pointing to new consolidated modules
    - Mark as deprecated with migration instructions
    - _Requirements: 1.2_
  - [x] 16.2 Update all internal cross-references
    - Ensure all packages use new import paths internally
    - _Requirements: 1.4_

- [x] 17. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 18. Documentation and benchmarks
  - [x] 18.1 Create README.md for each domain module
    - Include usage examples and API documentation
    - _Requirements: 12.1, 12.2, 12.4_
  - [x] 18.2 Create migration guide
    - Document old to new import path mappings
    - Include code migration examples
    - _Requirements: 12.3_
  - [x] 18.3 Implement benchmarks for critical operations
    - Cache GetOrCompute, CircuitBreaker Execute, Iterator operations
    - _Requirements: 13.3_
  - [x] 18.4 Update CHANGELOG with breaking changes
    - Document all API changes and deprecations
    - _Requirements: 12.5_

- [x] 19. Final validation
  - [x] 19.1 Run full test suite with coverage
    - Verify all property tests pass with 100+ iterations
    - _Requirements: 10.3, 10.4_
  - [x] 19.2 Verify Go 1.25 feature adoption
    - Confirm generic type aliases, synctest, DWARF v5 usage
    - _Requirements: 11.1, 11.2, 11.3_
  - [x] 19.3 Run benchmarks and compare with baseline
    - Ensure no performance regressions
    - _Requirements: 13.1, 13.2_

- [x] 20. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All tasks are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (24 total)
- Unit tests validate specific examples and edge cases
- Go 1.25 features: generic type aliases, testing/synctest, DWARF v5
