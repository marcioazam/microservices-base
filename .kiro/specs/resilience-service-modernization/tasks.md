# Implementation Plan

- [x] 1. Update dependencies to latest versions
  - [x] 1.1 Update go.mod to Go 1.25+
    - Change `go 1.22` to `go 1.25` in go.mod
    - Run `go mod tidy` to update go.sum
    - _Requirements: 1.1, 1.5_

  - [x] 1.2 Update go-redis to v9.17.0+
    - Update `github.com/redis/go-redis/v9` to v9.17.0 or higher
    - Addresses CVE-2025-29923 (out-of-order response vulnerability)
    - _Requirements: 1.2, 6.1_

  - [x] 1.3 Update OpenTelemetry SDK to v1.39.0+
    - Update `go.opentelemetry.io/otel` and related packages to v1.39.0+
    - _Requirements: 1.3, 7.1_

  - [x] 1.4 Update grpc-go to v1.77.0+
    - Update `google.golang.org/grpc` to v1.77.0 or higher
    - _Requirements: 1.4_

  - [x] 1.5 Run go mod tidy and verify checksums
    - Execute `go mod tidy`
    - Verify go.sum contains checksums for all dependencies
    - _Requirements: 1.5_

- [x] 2. Implement centralized UUID v7 generator
  - [x] 2.1 Create domain/uuid.go with GenerateEventID function
    - Implement UUID v7 per RFC 9562
    - Use 48-bit Unix timestamp in milliseconds
    - Use crypto/rand for remaining bits
    - Output standard UUID string format (36 chars with hyphens)
    - _Requirements: 3.1, 3.2, 3.3, 3.5, 6.2_

  - [x] 2.2 Write property test for UUID v7 format compliance
    - **Property 1: UUID v7 Format Compliance**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.5**
    - Test format: 36 chars, hyphens at positions 8,13,18,23
    - Test version nibble is '7'
    - Test variant bits are valid
    - Test timestamp extraction accuracy

  - [x] 2.3 Write property test for UUID v7 chronological ordering
    - **Property 2: UUID v7 Chronological Ordering**
    - **Validates: Requirements 3.4**
    - Generate pairs of UUIDs with time separation
    - Verify lexicographic order matches chronological order

  - [x] 2.4 Add ParseUUIDv7Timestamp helper function
    - Extract timestamp from UUID v7 string
    - Return error for invalid UUIDs
    - _Requirements: 3.2_

- [x] 3. Checkpoint - Ensure all tests pass
  - All tests pass successfully

- [x] 4. Implement centralized EventBuilder
  - [x] 4.1 Create domain/event_builder.go
    - Implement EventBuilder struct with emitter, serviceName, correlationFn
    - Implement NewEventBuilder constructor
    - Implement Build method with automatic ID, Timestamp, Type population
    - Implement Emit method with nil emitter safety
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [x] 4.2 Write property test for EventBuilder automatic field population
    - **Property 4: EventBuilder Automatic Field Population**
    - **Validates: Requirements 4.2, 4.4**
    - Test ID is valid UUID v7
    - Test Timestamp is within 1 second of build time
    - Test Type matches requested type
    - Test ServiceName and CorrelationID are populated

  - [x] 4.3 Write property test for nil emitter safety
    - **Property 5: Nil Emitter Safety**
    - **Validates: Requirements 4.3**
    - Test Emit with nil emitter does not panic

  - [x] 4.4 Write property test for event serialization backward compatibility
    - **Property 8: Event Serialization Backward Compatibility**
    - **Validates: Requirements 10.2**
    - Test JSON serialization produces expected fields
    - Test deserialization works with existing consumers

- [x] 5. Implement correlation factory
  - [x] 5.1 Create domain/correlation.go
    - Implement DefaultCorrelationFn returning no-op function
    - Implement NewCorrelationFn with fallback to default
    - _Requirements: 2.3, 9.3_

- [x] 6. Checkpoint - Ensure all tests pass
  - All tests pass successfully

- [x] 7. Refactor circuit breaker to use EventBuilder
  - [x] 7.1 Update circuitbreaker/breaker.go
    - Replace EventEmitter + CorrelationFn with EventBuilder in Config
    - Update constructor to use EventBuilder
    - Update emitStateChangeEvent to use EventBuilder.Emit
    - Remove local generateEventID function
    - _Requirements: 2.1, 2.2, 4.1_

  - [x] 7.2 Write property test for centralized event emission
    - **Property 3: Centralized Event Emission (Circuit Breaker)**
    - **Validates: Requirements 2.1, 2.2, 2.3, 4.1**
    - Test events use UUID v7 IDs
    - Test events have consistent field population

  - [x] 7.3 Update circuitbreaker/breaker_prop_test.go
    - Update existing tests to use new Config structure
    - Ensure all tests pass with EventBuilder
    - _Requirements: 8.4_

- [x] 8. Refactor rate limiter to use EventBuilder
  - [x] 8.1 Update ratelimit/token_bucket.go
    - Replace EventEmitter with EventBuilder in Config
    - Update emitRateLimitEvent to use EventBuilder.Emit
    - Remove local generateEventID function
    - _Requirements: 2.1, 2.2, 4.1_

  - [x] 8.2 Update ratelimit/sliding_window.go
    - Apply same changes as token_bucket.go
    - _Requirements: 2.1, 2.2, 4.1_

  - [x] 8.3 Update ratelimit/ratelimit_prop_test.go
    - Update existing tests to use new Config structure
    - _Requirements: 8.4_

- [x] 9. Refactor retry handler to use EventBuilder
  - [x] 9.1 Update retry/handler.go
    - Replace EventEmitter + CorrelationFn with EventBuilder in Config
    - Update emitRetryEvent to use EventBuilder.Emit
    - Remove local generateEventID function
    - Replace math/rand with crypto/rand for jitter
    - _Requirements: 2.1, 2.2, 4.1, 6.2_

  - [x] 9.2 Update retry/handler_prop_test.go
    - Update existing tests to use new Config structure
    - _Requirements: 8.4_

- [x] 10. Refactor bulkhead to use EventBuilder
  - [x] 10.1 Update bulkhead/bulkhead.go
    - Replace EventEmitter + CorrelationFn with EventBuilder in Config
    - Update emitRejectionEvent to use EventBuilder.Emit
    - Remove local generateEventID function
    - _Requirements: 2.1, 2.2, 4.1_

  - [x] 10.2 Update bulkhead/bulkhead_prop_test.go
    - Update existing tests to use new Config structure
    - _Requirements: 8.4_

- [x] 11. Checkpoint - Ensure all tests pass
  - All tests pass successfully

- [x] 12. Update OpenTelemetry integration
  - [x] 12.1 Update infra/otel/provider.go
    - Use WithInstrumentationAttributeSet for concurrent-safe attributes
    - Ensure proper initialization and graceful shutdown
    - _Requirements: 7.2, 7.5_

  - [x] 12.2 Write property test for trace context propagation
    - **Property 7: Trace Context Propagation**
    - **Validates: Requirements 7.4**
    - Test events include TraceID and SpanID from context

- [x] 13. Update logging to use log/slog
  - [x] 13.1 Update infra/audit/logger.go
    - Replace existing logger with log/slog
    - Configure structured JSON output
    - _Requirements: 5.3_

  - [x] 13.2 Write property test for structured JSON logging
    - **Property 6: Structured JSON Logging**
    - **Validates: Requirements 5.3**
    - Test log output is valid JSON

- [x] 14. Remove duplicate code and verify consolidation
  - [x] 14.1 Search and remove all duplicate generateEventID functions
    - Verify only domain/uuid.go contains GenerateEventID
    - Update all imports to use domain package
    - _Requirements: 2.4, 2.5_

  - [x] 14.2 Verify no duplicate correlation function defaults
    - Ensure all components use domain/correlation.go
    - _Requirements: 2.3_

- [x] 15. Run security validation
  - [x] 15.1 Verify crypto/rand usage
    - Grep for math/rand usage - NONE FOUND
    - Ensure only crypto/rand is used for security-sensitive operations
    - _Requirements: 6.2_

  - [x] 15.2 Dependencies updated to fix CVEs
    - go-redis v9.17.0 fixes CVE-2025-29923
    - _Requirements: 6.1, 6.3_

- [x] 16. Final Checkpoint - All tests pass
  - All domain tests: PASS
  - All circuitbreaker tests: PASS
  - All ratelimit tests: PASS
  - All retry tests: PASS
  - All bulkhead tests: PASS
  - All health tests: PASS
  - All audit tests: PASS
