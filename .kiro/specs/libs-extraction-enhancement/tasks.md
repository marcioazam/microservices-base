# Implementation Plan: Library Extraction Enhancement

## Overview

This implementation plan breaks down the library enhancement into discrete, incremental tasks. Each task builds on previous work and includes property-based tests to validate correctness. The plan prioritizes core libraries that other components depend on.

## Tasks

- [ ] 1. Set up testing infrastructure and property-based testing framework
  - Configure gopter for Go property-based testing
  - Create test helper utilities and common generators
  - Set up test organization structure
  - _Requirements: 9.1, 9.8, 9.9_

- [ ] 2. Implement Domain Primitives Library (Go)
  - [ ] 2.1 Implement Email type with validation
    - Create Email struct with private value field
    - Implement NewEmail constructor with RFC 5322 validation
    - Implement String(), MarshalJSON(), UnmarshalJSON()
    - _Requirements: 2.1, 2.8_

  - [ ] 2.2 Write property tests for Email type
    - **Property 1: Domain Primitive Validation Consistency (Email)**
    - **Property 2: Domain Primitive Serialization Round-Trip (Email)**
    - **Validates: Requirements 2.1, 2.8, 2.9**

  - [ ] 2.3 Implement UUID and ULID types
    - Create UUID struct with generation and parsing
    - Create ULID struct with time-ordered generation
    - Implement String(), MarshalJSON(), UnmarshalJSON() for both
    - _Requirements: 2.2_

  - [ ] 2.4 Write property tests for UUID/ULID types
    - **Property 1: Domain Primitive Validation Consistency (UUID/ULID)**
    - **Property 2: Domain Primitive Serialization Round-Trip (UUID/ULID)**
    - **Validates: Requirements 2.2, 2.9**

  - [ ] 2.5 Implement Money type with currency handling
    - Create Money struct with big.Int amount and Currency
    - Implement arithmetic operations (Add, Subtract, Multiply)
    - Implement currency validation and precision handling
    - _Requirements: 2.3_

  - [ ] 2.6 Write property tests for Money type
    - **Property 1: Domain Primitive Validation Consistency (Money)**
    - **Property 2: Domain Primitive Serialization Round-Trip (Money)**
    - **Validates: Requirements 2.3, 2.9**

  - [ ] 2.7 Implement PhoneNumber, URL, Timestamp, Duration types
    - Create PhoneNumber with E.164 validation
    - Create URL with scheme validation
    - Create Timestamp with ISO 8601 parsing
    - Create Duration with human-readable parsing
    - _Requirements: 2.4, 2.5, 2.6, 2.7_

  - [ ] 2.8 Write property tests for remaining domain types
    - **Property 1: Domain Primitive Validation Consistency (all types)**
    - **Property 2: Domain Primitive Serialization Round-Trip (all types)**
    - **Validates: Requirements 2.4, 2.5, 2.6, 2.7, 2.9**

- [ ] 3. Checkpoint - Domain Primitives Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 4. Enhance Error Handling Library (Go)
  - [ ] 4.1 Implement typed error codes and AppError struct
    - Define ErrorCode constants for all categories
    - Create AppError struct with Code, Message, Details, Cause
    - Implement Error() and Unwrap() methods
    - _Requirements: 3.1, 3.2_

  - [ ] 4.2 Implement error wrapping and chain traversal
    - Implement Wrap() function with context preservation
    - Implement RootCause() for chain traversal
    - Implement Is() and As() compatibility
    - _Requirements: 3.2, 3.3_

  - [ ] 4.3 Write property tests for error chain preservation
    - **Property 3: Error Chain Preservation**
    - **Validates: Requirements 3.2, 3.3, 3.7**

  - [ ] 4.4 Implement HTTP and gRPC status code mapping
    - Implement HTTPStatus() method on AppError
    - Implement GRPCCode() method on AppError
    - Create mapping tables for all error codes
    - _Requirements: 3.4, 3.5_

  - [ ] 4.5 Write property tests for status code mapping
    - **Property 4: Error Code to Status Mapping Consistency**
    - **Validates: Requirements 3.4, 3.5**

  - [ ] 4.6 Implement error serialization with redaction
    - Implement ToAPIResponse() with internal detail redaction
    - Implement structured logging format
    - _Requirements: 3.6, 3.8_

  - [ ] 4.7 Write property tests for error serialization
    - **Property 5: Error Serialization Redaction**
    - **Validates: Requirements 3.8**

- [ ] 5. Checkpoint - Error Handling Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 6. Enhance Validation Library (Go)
  - [ ] 6.1 Implement core validator types and composition
    - Create Validator[T] function type
    - Create ValidationResult and ValidationError types
    - Implement And(), Or(), Not() composition functions
    - _Requirements: 4.6_

  - [ ] 6.2 Implement string and numeric validators
    - Implement MinLength, MaxLength, MatchesRegex
    - Implement Range, Positive, NonZero for numbers
    - _Requirements: 4.2, 4.3_

  - [ ] 6.3 Implement collection and struct validators
    - Implement MinSize, MaxSize, UniqueElements
    - Implement struct field validation with paths
    - _Requirements: 4.4, 4.5_

  - [ ] 6.4 Write property tests for validation completeness
    - **Property 6: Validation Error Completeness**
    - **Property 7: Nested Validation Field Paths**
    - **Validates: Requirements 4.7, 4.8**

- [ ] 7. Enhance Codec Library (Go)
  - [ ] 7.1 Implement JSON serialization with options
    - Create JSONCodec with pretty, compact, omit empty options
    - Implement Encode() and Decode() methods
    - _Requirements: 5.1_

  - [ ] 7.2 Implement YAML and Base64 codecs
    - Create YAMLCodec for configuration files
    - Create Base64Codec with URL-safe variant
    - _Requirements: 5.2, 5.3_

  - [ ] 7.3 Write property tests for codec round-trip
    - **Property 8: Codec Round-Trip Consistency**
    - **Validates: Requirements 5.6**

- [ ] 8. Checkpoint - Validation and Codec Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 9. Enhance Observability Library (Go)
  - [ ] 9.1 Implement structured logger with JSON output
    - Create Logger struct with context and fields
    - Implement With() for adding fields
    - Implement Info(), Warn(), Error(), Debug() methods
    - _Requirements: 6.1, 6.2_

  - [ ] 9.2 Implement correlation ID and trace context propagation
    - Create context helpers for correlation ID
    - Integrate with OpenTelemetry trace context
    - _Requirements: 6.3, 6.4_

  - [ ] 9.3 Implement PII redaction
    - Create RedactSensitive() function
    - Define sensitive field patterns
    - _Requirements: 6.9_

  - [ ] 9.4 Write property tests for observability
    - **Property 9: Log Entry Timestamp Format**
    - **Property 10: Trace Context Propagation**
    - **Property 11: PII Redaction in Logs**
    - **Validates: Requirements 6.7, 6.8, 6.9**

- [ ] 10. Implement Security Library (Go)
  - [ ] 10.1 Implement constant-time comparison and random generation
    - Create ConstantTimeCompare() function
    - Create GenerateRandomBytes(), GenerateRandomHex(), GenerateRandomBase64()
    - _Requirements: 7.2, 7.3_

  - [ ] 10.2 Implement input sanitization
    - Create SanitizeHTML(), SanitizeSQL(), SanitizeShell()
    - _Requirements: 7.1_

  - [ ] 10.3 Write property tests for security utilities
    - **Property 12: Random Token Uniqueness**
    - **Validates: Requirements 7.8**

- [ ] 11. Checkpoint - Observability and Security Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 12. Implement HTTP/gRPC Transport Library (Go)
  - [ ] 12.1 Implement HTTP middleware
    - Create LoggingMiddleware with request/response logging
    - Create TracingMiddleware with span creation
    - Create ErrorHandlerMiddleware
    - _Requirements: 8.1, 8.3_

  - [ ] 12.2 Implement health check handlers
    - Create HealthHandler with liveness and readiness probes
    - Implement check registration
    - _Requirements: 8.4, 8.5_

  - [ ] 12.3 Implement graceful shutdown
    - Create ShutdownHandler with request draining
    - Implement timeout enforcement
    - _Requirements: 8.6, 8.7_

  - [ ] 12.4 Write property tests for transport
    - **Property 13: Request Timeout Enforcement**
    - **Validates: Requirements 8.8**

- [ ] 13. Implement Configuration Library (Go)
  - [ ] 13.1 Implement configuration loader
    - Create Loader with defaults, file, env sources
    - Implement LoadFile() and LoadEnv()
    - _Requirements: 10.1, 10.2, 10.3_

  - [ ] 13.2 Implement configuration validation
    - Create Validate() with required keys
    - Implement typed accessors
    - _Requirements: 10.4, 10.5_

  - [ ] 13.3 Write property tests for configuration
    - **Property 15: Configuration Error Completeness**
    - **Validates: Requirements 10.7, 10.8**

- [ ] 14. Checkpoint - Transport and Configuration Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 15. Enhance Resilience Library (Go)
  - [ ] 15.1 Enhance circuit breaker with Result integration
    - Update CircuitBreaker to return Result[T]
    - Implement CircuitOpenError with reset time
    - _Requirements: 11.1, 11.7_

  - [ ] 15.2 Write property tests for circuit breaker
    - **Property 16: Circuit Breaker State Transitions**
    - **Validates: Requirements 11.8**

  - [ ] 15.3 Enhance rate limiter with retry-after
    - Update RateLimiter to return RateLimitError
    - Include retry-after duration in error
    - _Requirements: 11.3_

  - [ ] 15.4 Write property tests for rate limiter
    - **Property 17: Rate Limiter Error Information**
    - **Validates: Requirements 11.9**

  - [ ] 15.5 Implement configuration validation for all resilience components
    - Add Validate() to all config types
    - Reject invalid parameters
    - _Requirements: 11.10_

  - [ ] 15.6 Write property tests for resilience configuration
    - **Property 18: Resilience Configuration Validation**
    - **Validates: Requirements 11.10**

- [ ] 16. Implement Pagination Library (Go)
  - [ ] 16.1 Implement pagination types and cursor encoding
    - Create Page, Cursor, PageResult types
    - Implement EncodeCursor() and DecodeCursor()
    - _Requirements: 12.1, 12.2, 12.3, 12.5_

  - [ ] 16.2 Implement pagination validation
    - Create NewPage() with validation
    - Validate offset, limit parameters
    - _Requirements: 12.6_

  - [ ] 16.3 Write property tests for pagination
    - **Property 19: Pagination Parameter Validation**
    - **Property 20: Cursor Encoding Round-Trip**
    - **Validates: Requirements 12.6, 12.7**

- [ ] 17. Implement Context Propagation Library (Go)
  - [ ] 17.1 Implement context value helpers
    - Create WithCorrelationID(), CorrelationID()
    - Create WithUserContext(), UserContext()
    - Create WithDeadline propagation
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5_

  - [ ] 17.2 Write property tests for context propagation
    - **Property 21: Context Value Preservation**
    - **Validates: Requirements 13.6**

- [ ] 18. Checkpoint - Resilience, Pagination, Context Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 19. Enhance Cache Library (Go)
  - [ ] 19.1 Implement LRU cache with TTL
    - Create LRUCache[K, V] with generics
    - Implement Get(), Set(), Delete()
    - Implement TTL expiration
    - _Requirements: 14.1, 14.2, 14.5_

  - [ ] 19.2 Implement cache statistics and invalidation
    - Add Stats struct with hits, misses, evictions
    - Implement pattern-based invalidation
    - _Requirements: 14.3, 14.4_

  - [ ] 19.3 Write property tests for cache
    - **Property 22: Cache LRU Eviction Policy**
    - **Property 23: Cache TTL Expiration**
    - **Validates: Requirements 14.7, 14.8**

- [ ] 20. Enhance Testing Utilities Library (Go)
  - [ ] 20.1 Implement property-based test generators
    - Create generators for all domain primitives
    - Create generators for common types (strings, numbers, collections)
    - _Requirements: 9.1_

  - [ ] 20.2 Implement fixture builders and mock factories
    - Create builder pattern for domain objects
    - Create mock factories for common interfaces
    - _Requirements: 9.2, 9.3_

  - [ ] 20.3 Write property tests for test generators
    - **Property 14: Test Data Validity**
    - **Validates: Requirements 9.8**

- [ ] 21. Checkpoint - Cache and Testing Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 22. Implement Worker Pool Library (Go)
  - [ ] 22.1 Implement worker pool with priority queue
    - Create WorkerPool[T] with configurable worker count
    - Implement priority-based job queue
    - Implement job status tracking
    - _Requirements: 19.1, 19.2, 19.6_

  - [ ] 22.2 Implement graceful shutdown and dead letter queue
    - Implement Shutdown() with job draining
    - Implement dead letter queue for failed jobs
    - Implement retry with exponential backoff
    - _Requirements: 19.3, 19.4, 19.5_

  - [ ] 22.3 Write property tests for worker pool
    - **Property 24: Worker Pool Panic Recovery**
    - **Property 25: Worker Pool Graceful Shutdown**
    - **Validates: Requirements 19.8, 19.9**

- [ ] 23. Implement Distributed Lock Library (Go)
  - [ ] 23.1 Implement lock interface and Redis backend
    - Create Lock interface with Acquire, Release, Renew
    - Implement RedisLock with TTL and fencing tokens
    - _Requirements: 20.1, 20.2, 20.3, 20.5, 20.7_

  - [ ] 23.2 Implement etcd backend and try-lock
    - Implement EtcdLock with lease-based locking
    - Implement TryLock with timeout
    - _Requirements: 20.4, 20.6_

  - [ ] 23.3 Write property tests for distributed lock
    - **Property 26: Distributed Lock TTL Expiration**
    - **Validates: Requirements 20.8, 20.9**

- [ ] 24. Implement Feature Flags Library (Go)
  - [ ] 24.1 Implement flag evaluation with targeting
    - Create Flag struct with percentage rollout
    - Implement user/tenant targeting
    - Implement IsEnabled() with context
    - _Requirements: 21.1, 21.2, 21.3_

  - [ ] 24.2 Implement flag sources and caching
    - Implement local file configuration
    - Implement remote HTTP configuration
    - Implement fallback to cached values
    - _Requirements: 21.6, 21.7_

  - [ ] 24.3 Write property tests for feature flags
    - **Property 27: Feature Flag Default Fallback**
    - **Property 28: Feature Flag Remote Fallback**
    - **Validates: Requirements 21.8, 21.9**

- [ ] 25. Implement Metrics Library (Go)
  - [ ] 25.1 Implement metric types
    - Create Counter, Gauge, Histogram interfaces
    - Implement thread-safe metric storage
    - Implement labels/tags support
    - _Requirements: 22.1, 22.2, 22.3, 22.5_

  - [ ] 25.2 Implement exporters and HTTP handler
    - Implement Prometheus exposition format
    - Implement OpenTelemetry OTLP export
    - Create /metrics HTTP handler
    - _Requirements: 22.6, 22.7, 22.8_

  - [ ] 25.3 Write property tests for metrics
    - **Property 29: Metrics Thread Safety**
    - **Validates: Requirements 22.9**

- [ ] 26. Checkpoint - Worker Pool, Lock, Flags, Metrics Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 27. Implement HTTP Client Library (Go)
  - [ ] 27.1 Implement resilient HTTP client
    - Create Client with timeout and connection pooling
    - Implement automatic retry with backoff
    - Implement circuit breaker integration
    - _Requirements: 23.2, 23.3, 23.4, 23.7_

  - [ ] 27.2 Implement observability features
    - Implement request/response logging
    - Implement trace context propagation
    - Implement correlation ID propagation
    - _Requirements: 23.1, 23.5, 23.6_

  - [ ] 27.3 Write property tests for HTTP client
    - **Property 30: HTTP Client Error Typing**
    - **Validates: Requirements 23.9**

- [ ] 28. Implement Database Utilities Library (Go)
  - [ ] 28.1 Implement transaction management
    - Create DB wrapper with tracing
    - Implement WithTransaction with auto-rollback
    - Implement retry for transient errors
    - _Requirements: 24.1, 24.3, 24.7_

  - [ ] 28.2 Implement query builder and health check
    - Create QueryBuilder with SQL injection prevention
    - Implement health check for connectivity
    - Implement query tracing with OpenTelemetry
    - _Requirements: 24.2, 24.4, 24.5_

  - [ ] 28.3 Write property tests for database utilities
    - **Property 31: Database Transaction Auto-Rollback**
    - **Validates: Requirements 24.8**

- [ ] 29. Implement Event Bus Library (Go)
  - [ ] 29.1 Implement pub/sub with topic routing
    - Create Event struct with metadata
    - Implement Subscribe and Publish
    - Implement async event delivery
    - _Requirements: 25.1, 25.2, 25.3_

  - [ ] 29.2 Implement reliability features
    - Implement dead letter handling
    - Implement event ordering per topic
    - Implement retry with backoff
    - _Requirements: 25.5, 25.7_

  - [ ] 29.3 Write property tests for event bus
    - **Property 32: Event Bus Subscriber Recovery**
    - **Validates: Requirements 25.8**

- [ ] 30. Checkpoint - HTTP Client, Database, Event Bus Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 31. Implement Outbox Pattern Library (Go)
  - [ ] 31.1 Implement outbox storage and publisher
    - Create OutboxEntry with idempotency key
    - Implement Store() within transaction
    - Implement background ProcessPending()
    - _Requirements: 28.1, 28.2, 28.4_

  - [ ] 31.2 Implement delivery guarantees
    - Implement at-least-once delivery
    - Implement event ordering within aggregate
    - Implement cleanup of processed events
    - _Requirements: 28.3, 28.5, 28.6_

  - [ ] 31.3 Write property tests for outbox
    - **Property 33: Outbox At-Least-Once Delivery**
    - **Validates: Requirements 28.7**

- [ ] 32. Implement Idempotency Library (Go)
  - [ ] 32.1 Implement idempotency store and middleware
    - Create Store interface with Redis backend
    - Implement response caching with TTL
    - Implement HTTP middleware
    - _Requirements: 29.1, 29.2, 29.3, 29.5_

  - [ ] 32.2 Implement concurrent request handling
    - Implement lock while processing
    - Handle missing idempotency key
    - _Requirements: 29.4_

  - [ ] 32.3 Write property tests for idempotency
    - **Property 34: Idempotency Duplicate Response**
    - **Validates: Requirements 29.6**

- [ ] 33. Implement API Versioning Library (Go)
  - [ ] 33.1 Implement version extraction and routing
    - Implement URL path versioning (/v1/, /v2/)
    - Implement header-based versioning
    - Implement version routing middleware
    - _Requirements: 27.1, 27.2, 27.5_

  - [ ] 33.2 Implement deprecation handling
    - Implement deprecation warnings
    - Add Deprecation and Sunset headers
    - _Requirements: 27.3, 27.4_

  - [ ] 33.3 Write property tests for API versioning
    - **Property 35: API Version Deprecation Headers**
    - **Validates: Requirements 27.6**

- [ ] 34. Implement Structured Concurrency Library (Go)
  - [ ] 34.1 Implement task group with error propagation
    - Create TaskGroup with context cancellation
    - Implement Go() with panic recovery
    - Implement Wait() with error aggregation
    - _Requirements: 30.1, 30.2, 30.3_

  - [ ] 34.2 Implement fan-out/fan-in patterns
    - Implement FanOut with rate limiting
    - Implement timeout enforcement
    - _Requirements: 30.4, 30.5, 30.6_

  - [ ] 34.3 Write property tests for structured concurrency
    - **Property 36: Task Group Error Cancellation**
    - **Property 37: Task Group Panic Recovery**
    - **Validates: Requirements 30.7, 30.3**

- [ ] 35. Implement DTO Library (Go)
  - [ ] 35.1 Implement struct validation and binding
    - Implement struct tag-based validation
    - Implement JSON binding with custom errors
    - Implement field transformation
    - _Requirements: 26.1, 26.2, 26.3_

  - [ ] 35.2 Implement nested validation and error formatting
    - Implement nested struct validation
    - Implement validation error formatting for API
    - _Requirements: 26.4, 26.6_

- [ ] 36. Checkpoint - Outbox, Idempotency, Versioning, Concurrency, DTO Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 37. Update README Documentation (Go)
  - [ ] 37.1 Update main libs/go/README.md
    - Add comprehensive overview of all 25+ library categories
    - Add quick start examples for each library
    - Add migration guide for new libraries
    - _Requirements: 15.8_

  - [ ] 37.2 Update individual library READMEs
    - Add 3+ usage examples per package
    - Add API reference for public interfaces
    - Add error handling patterns
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7_

- [ ] 38. Implement Rust Library Parity (libs/rust/)
  - [ ] 38.1 Implement Rust error handling library
    - Create error types with thiserror
    - Implement error mapping to HTTP/gRPC
    - _Requirements: 16.1_

  - [ ] 38.2 Implement Rust domain primitives
    - Create Email, UUID, Money types with validation
    - Implement serde serialization
    - _Requirements: 16.3_

  - [ ] 38.3 Implement Rust observability library
    - Create structured logging with tracing
    - Implement correlation ID propagation
    - _Requirements: 16.4_

  - [ ] 38.4 Implement Rust resilience patterns
    - Create circuit breaker, retry, rate limiter
    - _Requirements: 16.5_

  - [ ] 38.5 Write property tests for Rust libraries
    - Use proptest for property-based testing
    - Test round-trip serialization
    - Test validation consistency
    - _Requirements: 16.1-16.7_

- [ ] 39. Update Rust README Documentation
  - [ ] 39.1 Create libs/rust/README.md
    - Add comprehensive overview of all crates
    - Add usage examples
    - _Requirements: 15.9_

- [ ] 40. Final Checkpoint - All Libraries Complete
  - Ensure all tests pass, ask the user if questions arise.
  - Run full test suite across all libraries
  - Verify documentation completeness

## Notes

- All tasks are required for comprehensive 200% production-ready library
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (37 properties total)
- Unit tests validate specific examples and edge cases
- Go libraries are prioritized as they have more existing code to enhance
- Rust libraries follow Go patterns for consistency
- Total: 30 requirements, 37 correctness properties, 40 implementation tasks
