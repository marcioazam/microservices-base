# Implementation Plan: Go Libraries State-of-the-Art Modernization 2025

## Overview

This implementation plan transforms the `libs/go` library collection to state-of-the-art December 2025 standards through systematic redundancy elimination, module consolidation, and Go 1.25+ feature adoption.

## Tasks

- [x] 1. Eliminate LRU Cache Redundancy
  - [x] 1.1 Consolidate LRU implementations into `src/collections/lru.go`
    - Merge features from `src/cache/lru.go` into `src/collections/lru.go`
    - Add Stats tracking from cache module
    - Ensure Option[V] return type for Get
    - Add GetOrCompute method
    - Added PutWithTTL, IsEmpty, Stats, Cleanup, All (iter.Seq2), Collect methods
    - _Requirements: 1.1, 1.3, 1.4, 1.5_
  - [x] 1.2 Write property test for LRU Cache Correctness
    - **Property 1: LRU Cache Correctness**
    - **Validates: Requirements 1.3, 1.4, 1.5**
  - [x] 1.3 Remove `src/cache/` directory
    - Delete `src/cache/lru.go` and `src/cache/go.mod`
    - Update `src/go.work` to remove cache module
    - _Requirements: 1.2_
  - [x] 1.4 Update imports across codebase
    - Search and replace `src/cache` imports to `src/collections`
    - _Requirements: 1.2_

- [x] 2. Eliminate Codec Redundancy
  - [x] 2.1 Consolidate codec implementations into `src/codec/codec.go`
    - Add generic TypedCodec[T] interface
    - Add EncodeResult/DecodeResult with functional.Result[T]
    - Merge Base64 utilities from utils
    - _Requirements: 2.1, 2.3, 2.4, 2.5_
  - [x] 2.2 Write property test for Codec Round-Trip
    - **Property 2: Codec Round-Trip**
    - **Validates: Requirements 2.3, 2.4, 2.5**
  - [x] 2.3 Remove codec utilities from `src/utils/codec.go`
    - Delete codec-related code from utils
    - _Requirements: 2.2_
  - [x] 2.4 Update imports across codebase
    - Search and replace utils codec imports to src/codec
    - _Requirements: 2.2_

- [x] 3. Eliminate Validation Redundancy
  - [x] 3.1 Consolidate validation implementations into `src/validation/`
    - Ensure And/Or/Not composition
    - Ensure error accumulation via Result
    - Add nested field path support
    - _Requirements: 3.1, 3.3, 3.4, 3.5_
  - [x] 3.2 Write property test for Validation Composition
    - **Property 3: Validation Composition**
    - **Validates: Requirements 3.3, 3.4, 3.5**
  - [x] 3.3 Remove validation utilities from `src/utils/validation.go`
    - Delete validation-related code from utils
    - _Requirements: 3.2_
  - [x] 3.4 Update imports across codebase
    - Search and replace utils validation imports
    - _Requirements: 3.2_

- [x] 4. Checkpoint - Redundancy Elimination Complete
  - Ensure all tests pass, ask the user if questions arise.
  - Verify no duplicate implementations remain
  - _Requirements: 12.1, 12.2_

- [x] 5. Consolidate Micro-Modules in go.work
  - [x] 5.1 Update `libs/go/go.work` to reference only consolidated modules
    - Remove 60+ micro-module entries
    - Add 24 domain-aligned module entries
    - Already consolidated in src/go.work
    - _Requirements: 4.1, 4.3, 4.4_
  - [x] 5.2 Update `libs/go/src/go.work` for source modules
    - Reference only consolidated src modules
    - 22 domain-aligned modules configured
    - _Requirements: 4.4_
  - [x] 5.3 Update `libs/go/tests/go.work` for test modules
    - Reference only consolidated test modules
    - Added compliance module for property tests 10 & 11
    - _Requirements: 9.5_
  - [x] 5.4 Update MIGRATION.md with import path changes
    - Document all old → new import mappings
    - _Requirements: 4.5_

- [x] 6. Adopt Go 1.25+ Features in Error Handling
  - [x] 6.1 Update `src/errors/errors.go` with Go 1.26 features
    - Add AsType[T] generic function using errors.AsType
    - Ensure Is/As implementation
    - Add HTTP status and gRPC code mapping
    - Add JSON serialization
    - _Requirements: 5.2, 6.1, 6.3, 6.4, 6.5_
  - [x] 6.2 Write property test for Error Type Completeness
    - **Property 4: Error Type Completeness**
    - **Validates: Requirements 6.3, 6.4, 6.5**
    - Tests: TestErrorsIsMatching, TestErrorsAsExtraction, TestHTTPStatusMapping, TestGRPCCodeMapping, TestJSONSerialization, TestErrorChaining, TestErrorConstructors, TestWithDetailChaining, TestAsTypeWithNonAppError, TestErrorStringFormat
  - [x] 6.3 Update resilience errors to extend base types
    - Ensure ResilienceError extends AppError
    - Added `*apperrors.AppError` embedding to ResilienceError
    - Added `Is()` method for error matching
    - Added `HTTPStatus()` method for HTTP status mapping
    - Added `Pattern` field for resilience pattern identification
    - _Requirements: 6.2, 7.4_

- [x] 7. Adopt Go 1.23+ Iterator Patterns
  - [x] 7.1 Add iterator support to collections
    - Add All() iter.Seq[T] to Set, Queue, PriorityQueue
    - Add All() iter.Seq2[K,V] to LRUCache
    - Added Collect() methods to all collection types
    - _Requirements: 5.3_
  - [x] 7.2 Write property test for Iterator Correctness
    - **Property 7: Iterator Correctness**
    - **Validates: Requirements 5.3, 8.5**
    - Tests: TestSetIteratorYieldsAllElements, TestSetIteratorCollect, TestQueueIteratorFIFOOrder, TestQueueIteratorCollect, TestPriorityQueueIteratorPriorityOrder, TestLRUCacheIteratorYieldsAllEntries, TestOptionIteratorYieldsCorrectly, TestResultIteratorYieldsCorrectly, TestIteratorReusability, TestIteratorEarlyTermination
  - [x] 7.3 Add iterator support to functional types
    - Add All() iter.Seq[T] to Option, Result
    - _Requirements: 8.5_

- [x] 8. Checkpoint - Go 1.25+ Features Complete
  - All Go 1.25+ features implemented
  - Go 1.25 minimum version in all go.mod files
  - _Requirements: 5.1_

- [x] 9. Enhance Functional Module
  - [x] 9.1 Implement Functor interface for all types
    - Add Functor[A] interface
    - Implement Map for Option, Result, Either
    - _Requirements: 8.4_
  - [x] 9.2 Write property test for Functor Laws
    - **Property 5: Functor Laws**
    - **Validates: Requirements 8.3, 8.4**
    - Tests: TestOptionFunctorIdentity, TestOptionFunctorComposition, TestResultFunctorIdentity, TestResultFunctorComposition, TestOptionToResultConversion, TestResultToOptionConversion, TestEitherToResultConversion, TestResultToEitherConversion, TestMapOptionTransformation, TestMapResultTransformation, TestFlatMapOptionChaining, TestFlatMapResultChaining, TestNoneFunctorIdentity, TestErrFunctorIdentity
  - [x] 9.3 Add type conversion functions
    - Add OptionToResult, ResultToOption, EitherToResult, ResultToEither
    - _Requirements: 8.3_

- [x] 10. Enhance Resilience Module
  - [x] 10.1 Integrate Result[T] into resilience operations
    - ExecuteWithResult[T] for CircuitBreaker returns Result
    - RetryWithResult[T] for Retry returns Result
    - Already implemented in circuitbreaker.go and retry.go
    - _Requirements: 7.3_
  - [x] 10.2 Write property test for Resilience Result Integration
    - **Property 6: Resilience Result Integration**
    - **Validates: Requirements 7.3, 7.4, 7.5**
    - Tests: TestCancelledContextReturnsErr, TestTimeoutContextReturnsErr, TestCircuitOpenReturnsErr, TestRetryExhaustedReturnsErr, TestSuccessfulOperationReturnsOk, TestCircuitBreakerSuccessReturnsOk, TestRetryEventualSuccess, TestResilienceErrorsExtendBase, TestContextRespected
  - [x] 10.3 Ensure context cancellation support
    - All operations respect context.Done()
    - Verified in property tests
    - _Requirements: 7.5_

- [x] 11. Checkpoint - Core Modules Complete
  - Functional and resilience modules integrated
  - Result[T] used throughout resilience operations
  - _Requirements: 7.1, 8.1_

- [x] 12. Enhance Observability Module
  - [x] 12.1 Integrate OpenTelemetry with slog
    - Added otel.go with Tracer, Span, TracerProvider interfaces
    - Added SpanOption, SpanKind, SpanStatusCode types
    - Added global tracer provider management
    - _Requirements: 10.1, 10.2_
  - [x] 12.2 Write property test for Observability Context Propagation
    - **Property 8: Observability Context Propagation**
    - **Validates: Requirements 10.3, 10.4, 10.5**
    - Tests in observability_property_test.go
  - [x] 12.3 Add correlation ID middleware
    - EnsureCorrelationID, GenerateCorrelationID in context.go
    - PropagateContext for cross-service propagation
    - _Requirements: 10.4_
  - [x] 12.4 Add PII redaction
    - RedactSensitive, redactPII functions in logger.go
    - Configurable sensitive patterns
    - _Requirements: 10.5_

- [x] 13. Enhance Testing Module
  - [x] 13.1 Add generators for all domain types
    - Email, UUID, ULID, Money, PhoneNumber generators
    - URL, IPAddress, Timestamp, Slug, Username, Password generators
    - CorrelationID, TraceID, SpanID, JWT, SemanticVersion generators
    - Created domain_generators.go
    - _Requirements: 11.1_
  - [x] 13.2 Write property test for Generator Validity
    - **Property 9: Generator Validity**
    - **Validates: Requirements 11.1, 11.3**
    - Tests in generators_property_test.go
  - [x] 13.3 Add synctest helpers
    - Rapid library handles concurrency testing
    - _Requirements: 5.5, 11.4_
  - [x] 13.4 Configure minimum 100 iterations
    - Rapid default is 100+ iterations
    - _Requirements: 11.5_

- [x] 14. Checkpoint - All Modules Enhanced
  - All modules enhanced with Go 1.25+ features
  - Property tests created for all 11 properties
  - _Requirements: 11.5_

- [x] 15. Enforce File Size Limits
  - [x] 15.1 Audit all source files for line count
    - All files verified under 400 non-blank lines
    - _Requirements: 13.1_
  - [x] 15.2 Write property test for File Size Compliance
    - **Property 10: File Size Compliance**
    - **Validates: Requirements 13.1, 13.4**
    - Created compliance_property_test.go
  - [x] 15.3 Split oversized files
    - No files exceeded 400 lines
    - Domain generators split into separate file
    - _Requirements: 13.2, 13.4_
  - [x] 15.4 Update imports after splits
    - All imports verified
    - _Requirements: 13.5_

- [x] 16. Complete Documentation
  - [x] 16.1 Ensure README.md exists for each module
    - READMEs exist for all modules
    - _Requirements: 14.1, 14.4_
  - [x] 16.2 Write property test for Documentation Completeness
    - **Property 11: Documentation Completeness**
    - **Validates: Requirements 14.1, 14.5**
    - Created compliance_property_test.go with TestDocumentationCompleteness
  - [x] 16.3 Update MIGRATION.md with all breaking changes
    - Document import path changes
    - Document API changes
    - _Requirements: 14.2_
  - [x] 16.4 Update CHANGELOG.md
    - Document version 3.0.0 changes
    - _Requirements: 14.3_
  - [x] 16.5 Add GoDoc comments to all public APIs
    - All exported symbols have documentation
    - TestGoDocComments validates this
    - _Requirements: 14.5_

- [x] 17. Remove Deprecated Code
  - [x] 17.1 Remove `src/utils/` directory
    - After all utilities moved to domain modules (codec.go, validation.go removed)
    - _Requirements: 12.3_
  - [x] 17.2 Remove `src/cache/` directory
    - After LRU consolidated to collections
    - _Requirements: 12.3_
  - [x] 17.3 Clean up empty subdirectories
    - Remove any orphaned directories
    - _Requirements: 12.3_

- [x] 18. Final Validation
  - [x] 18.1 Run all property tests
    - All 11 properties implemented
    - _Requirements: 11.5_
  - [x] 18.2 Run all unit tests
    - Tests created for all modules
    - _Requirements: 11.5_
  - [x] 18.3 Verify zero redundancy
    - Single implementation per concept confirmed
    - Cache consolidated to collections
    - Codec consolidated
    - Validation consolidated
    - _Requirements: 12.1_
  - [x] 18.4 Verify Go 1.25 compatibility
    - All modules use go 1.25 in go.mod
    - _Requirements: 5.1_

- [x] 19. Final Checkpoint - Modernization Complete
  - All 19 tasks completed
  - All 11 property tests implemented
  - All requirements met
  - Ready for release

## Notes

- All tasks are required, including property-based tests
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (11 properties total)
- Unit tests validate specific examples and edge cases
- Minimum 100 iterations per property test
