# Implementation Plan

## Phase 1: Go Version and Dependency Modernization

- [x] 1. Upgrade Go version and dependencies

  - [x] 1.1 Update go.mod to Go 1.23 and upgrade all dependencies


    - Update Go version from 1.22 to 1.23
    - Upgrade go-redis/v9 from 9.4.0 to 9.7.0
    - Upgrade otel from 1.24.0 to 1.32.0
    - Upgrade grpc-go from 1.62.0 to 1.68.0
    - Upgrade protobuf from 1.32.0 to 1.35.2
    - Run `go mod tidy` to resolve dependencies
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

## Phase 2: Domain Package Centralization


- [x] 2. Centralize event ID generation

  - [x] 2.1 Create domain/id.go with centralized GenerateEventID function

    - Implement GenerateEventID() using timestamp-random format
    - Format: "20060102150405-a1b2c3d4" (timestamp + 4 random bytes hex)
    - _Requirements: 2.1, 2.2_


  - [x] 2.2 Write property test for event ID uniqueness


    - **Property 1: Event ID Uniqueness**
    - **Validates: Requirements 2.2**


  - [x] 2.3 Update circuitbreaker/breaker.go to use centralized GenerateEventID

    - Remove local generateEventID function


    - Import and use domain.GenerateEventID
    - _Requirements: 2.3, 2.4_



  - [x] 2.4 Update ratelimit/token_bucket.go to use centralized GenerateEventID
    - Remove local generateEventID function
    - Import and use domain.GenerateEventID
    - _Requirements: 2.3, 2.4_

  - [x] 2.5 Update retry/handler.go to use centralized GenerateEventID


    - Remove local generateEventID function
    - Import and use domain.GenerateEventID
    - _Requirements: 2.3, 2.4_



  - [x] 2.6 Update bulkhead/bulkhead.go to use centralized GenerateEventID
    - Remove local generateEventID function
    - Import and use domain.GenerateEventID
    - _Requirements: 2.3, 2.4_

  - [x] 2.7 Update health/aggregator.go to use centralized GenerateEventID
    - Remove local generateEventID function
    - Import and use domain.GenerateEventID
    - _Requirements: 2.3, 2.4_


- [x] 3. Centralize correlation function handling
  - [x] 3.1 Create domain/correlation.go with CorrelationFunc type and helpers
    - Define CorrelationFunc type alias
    - Implement DefaultCorrelationFunc returning empty string
    - Implement EnsureCorrelationFunc helper
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [x] 3.2 Update all components to use centralized correlation functions
    - Update circuitbreaker, ratelimit, retry, bulkhead, health packages
    - Replace inline nil checks with EnsureCorrelationFunc
    - _Requirements: 3.1, 3.2_

- [x] 4. Centralize event emission pattern
  - [x] 4.1 Add EmitEvent and EmitAuditEvent helpers to domain/events.go
    - Implement EmitEvent with nil emitter safety
    - Implement EmitAuditEvent with nil emitter safety
    - _Requirements: 4.1, 4.2, 4.3_

  - [x] 4.2 Write property test for nil emitter safety
    - **Property 2: Nil Emitter Safety**
    - **Validates: Requirements 4.2**

  - [x] 4.3 Update all components to use centralized EmitEvent helper
    - Update circuitbreaker, ratelimit, retry, bulkhead, health packages
    - Remove duplicate nil-check patterns
    - _Requirements: 4.4_

- [x] 5. Checkpoint - Ensure all tests pass

  - Ensure all tests pass, ask the user if questions arise.




## Phase 3: Configuration Validation Centralization

- [x] 6. Add Validate methods to domain config types



  - [x] 6.1 Add Validate() method to CircuitBreakerConfig
    - Validate FailureThreshold > 0
    - Validate SuccessThreshold > 0
    - Validate Timeout > 0
    - _Requirements: 6.1_

  - [x] 6.2 Add Validate() method to RetryConfig
    - Validate MaxAttempts > 0
    - Validate BaseDelay > 0
    - Validate MaxDelay > 0
    - Validate Multiplier >= 1.0
    - Validate JitterPercent in [0, 1]
    - _Requirements: 6.2_

  - [x] 6.3 Add Validate() method to TimeoutConfig
    - Validate Default > 0
    - _Requirements: 6.3_
  - [x] 6.4 Add Validate() method to RateLimitConfig


    - Validate Limit > 0
    - Validate Window > 0
    - _Requirements: 6.4_


  - [x] 6.5 Add Validate() method to BulkheadConfig


    - Validate MaxConcurrent > 0

    - Validate MaxQueue >= 0

    - _Requirements: 6.5_


  - [x] 6.6 Write property test for configuration validation correctness
    - **Property 4: Configuration Validation Correctness**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**

  - [x] 6.7 Update policy/engine.go to delegate validation to domain methods
    - Remove duplicate validateCircuitBreaker, validateRetry, validateTimeout, validateRateLimit, validateBulkhead functions
    - Call config.Validate() methods instead
    - _Requirements: 6.6_




- [x] 7. Checkpoint - Ensure all tests pass

  - Ensure all tests pass, ask the user if questions arise.

## Phase 4: Rate Limiter Factory


- [x] 8. Implement generic rate limiter factory


  - [x] 8.1 Create ratelimit/factory.go with NewRateLimiter factory function

    - Accept domain.RateLimitConfig and domain.EventEmitter
    - Return TokenBucket for TokenBucket algorithm

    - Return SlidingWindow for SlidingWindow algorithm


    - Return error for unknown algorithm
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  - [x] 8.2 Write property test for rate limiter factory correctness

    - **Property 3: Rate Limiter Factory Correctness**


    - **Validates: Requirements 5.2, 5.3**




## Phase 5: Serialization Centralization



- [x] 9. Centralize time serialization
  - [x] 9.1 Create domain/serialization.go with time format helpers
    - Define TimeFormat constant as time.RFC3339Nano
    - Implement MarshalTime function
    - Implement UnmarshalTime function
    - _Requirements: 7.5_

  - [x] 9.2 Write property test for time format round-trip consistency
    - **Property 6: Time Format Consistency**
    - **Validates: Requirements 7.5**

  - [x] 9.3 Update circuitbreaker/serialization.go to use centralized time format
    - Import and use domain.TimeFormat, domain.MarshalTime, domain.UnmarshalTime
    - _Requirements: 7.5_



- [x] 10. Write property tests for serialization round-trip
  - [x] 10.1 Write property test for ResiliencePolicy serialization round-trip
    - **Property 5: Serialization Round-Trip Consistency (Policy)**
    - **Validates: Requirements 7.2**

  - [x] 10.2 Write property test for RetryConfig serialization round-trip
    - **Property 5: Serialization Round-Trip Consistency (RetryConfig)**
    - **Validates: Requirements 7.3**
  - [x] 10.3 Write property test for YAML/JSON policy format parsing

    - **Property 13: Policy Format Parsing**
    - **Validates: Requirements 15.5**

- [x] 11. Checkpoint - Ensure all tests pass

  - Ensure all tests pass, ask the user if questions arise.

## Phase 6: Go 1.23 Iterator Support

- [x] 12. Add iterator support to policy engine
  - [x] 12.1 Add Policies() iter.Seq method to policy/engine.go
    - Import iter package from standard library
    - Return iterator over all policies
    - _Requirements: 8.1_

- [x] 13. Add iterator support to health aggregator
  - [x] 13.1 Add Services() iter.Seq method to health/aggregator.go
    - Import iter package from standard library
    - Return iterator over all registered services
    - _Requirements: 8.2_

- [x] 14. Add iterator support to bulkhead manager
  - [x] 14.1 Add Partitions() iter.Seq2 method to bulkhead/bulkhead.go
    - Import iter package from standard library
    - Return iterator over partition name and metrics pairs
    - _Requirements: 8.3_

## Phase 7: Error Handling Enhancement

- [x] 15. Enhance error handling consistency
  - [x] 15.1 Write property test for error wrapping preservation
    - **Property 8: Error Wrapping Preservation**
    - **Validates: Requirements 12.2**
  - [x] 15.2 Write property test for gRPC error mapping completeness
    - **Property 9: gRPC Error Mapping Completeness**
    - **Validates: Requirements 12.3**

## Phase 8: Shutdown and Policy Reload Enhancement

- [x] 16. Write property tests for shutdown behavior
  - [x] 16.1 Write property test for shutdown request blocking
    - **Property 10: Shutdown Request Blocking**
    - **Validates: Requirements 14.1**
  - [x] 16.2 Write property test for shutdown drain with timeout
    - **Property 11: Shutdown Drain with Timeout**
    - **Validates: Requirements 14.2, 14.3**

- [x] 17. Write property test for policy reload validation
  - [x] 17.1 Write property test for policy reload preserving valid policy on invalid input
    - **Property 12: Policy Reload Validation**
    - **Validates: Requirements 15.2, 15.3**

## Phase 9: Test Generator Enhancement

- [x] 18. Enhance test generators for validation
  - [x] 18.1 Write property test for generator validity
    - **Property 7: Generator Validity**
    - **Validates: Requirements 11.5**
    - Verify all generated configs pass their Validate() methods

- [x] 19. Final Checkpoint - Ensure all tests pass
  - All property tests passing (88.978s total)
