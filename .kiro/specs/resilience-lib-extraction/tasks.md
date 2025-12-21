# Implementation Plan

- [x] 1. Extract core types and interfaces to libs/go/resilience/



  - [x] 1.1 Add policy event types to libs/go/resilience/policy_events.go

    - Create PolicyEventType and PolicyEvent types
    - Add PolicyCreated, PolicyUpdated, PolicyDeleted constants
    - _Requirements: 8.1, 8.2_

  - [x] 1.2 Add health types to libs/go/resilience/health/types.go

    - Create HealthStatus, ServiceHealth, AggregatedHealth types
    - Create HealthChecker and HealthAggregator interfaces
    - Create HealthChangeEvent type
    - _Requirements: 2.2, 2.3_

  - [x] 1.3 Add resilience pattern interfaces to libs/go/resilience/

    - Add CircuitBreakerState and CircuitBreaker interface
    - Add RateLimitDecision, RateLimitHeaders, RateLimiter interface
    - Add BulkheadMetrics, Bulkhead, BulkheadManager interfaces
    - Add TimeoutManager interface
    - Add RetryHandler interface
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_



- [x] 2. Extract random source abstraction

  - [x] 2.1 Create libs/go/resilience/rand/rand.go

    - Extract RandSource interface
    - Extract CryptoRandSource implementation
    - Extract DeterministicRandSource implementation
    - Extract FixedRandSource implementation
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 2.2 Write property tests for random sources

    - **Property 13: Deterministic Random Source Reproducibility**
    - **Property 14: Random Source Value Range**
    - **Validates: Requirements 7.1, 7.2, 7.3**



- [x] 3. Extract circuit breaker implementation

  - [x] 3.1 Create libs/go/resilience/circuitbreaker/breaker.go

    - Extract Breaker struct and Config
    - Extract New, Execute, GetState, GetFullState methods
    - Extract RecordSuccess, RecordFailure, Reset methods
    - Extract state transition logic
    - _Requirements: 1.1_

  - [x] 3.2 Create libs/go/resilience/circuitbreaker/serialization.go

    - Extract MarshalState and UnmarshalState functions
    - Extract parseCircuitState helper
    - Extract StateStore interface
    - _Requirements: 5.1_
  - [x] 3.3 Write property tests for circuit breaker


    - **Property 1: Circuit Breaker State Transitions**
    - **Property 2: Circuit Breaker Half-Open Recovery**
    - **Property 11: Serialization Round-Trip**
    - **Validates: Requirements 1.1, 5.1, 5.3**



- [x] 4. Extract retry handler implementation

  - [x] 4.1 Create libs/go/resilience/retry/handler.go

    - Extract Handler struct and Config
    - Extract New, Execute, ExecuteWithCircuitBreaker methods
    - Extract CalculateDelay with exponential backoff and jitter
    - Update to use libs/go/resilience/rand for random source
    - _Requirements: 1.2_

  - [x] 4.2 Create libs/go/resilience/retry/policy.go

    - Extract PolicyDefinition struct
    - Extract ParsePolicy, ValidatePolicy, MarshalPolicy functions
    - Extract ToDefinition, FromDefinition converters
    - Extract PrettyPrint helper
    - _Requirements: 5.2_


  - [x] 4.3 Write property tests for retry handler


    - **Property 3: Retry Delay Bounds**
    - **Property 4: Retry Exponential Backoff**
    - **Property 12: Retry Policy Round-Trip**

    - **Validates: Requirements 1.2, 5.2, 5.3**




- [x] 5. Checkpoint - Ensure all tests pass

  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Extract rate limiter implementations

  - [x] 6.1 Create libs/go/resilience/ratelimit/token_bucket.go

    - Extract TokenBucket struct and TokenBucketConfig
    - Extract NewTokenBucket, Allow, GetHeaders methods
    - Extract refill, calculateResetTime, calculateRetryAfter helpers
    - _Requirements: 1.3_

  - [x] 6.2 Create libs/go/resilience/ratelimit/sliding_window.go

    - Extract SlidingWindow struct and SlidingWindowConfig
    - Extract NewSlidingWindow, Allow, GetHeaders methods
    - Extract pruneOldRequests, calculateResetTime, calculateRetryAfter helpers
    - _Requirements: 1.3_

  - [x] 6.3 Create libs/go/resilience/ratelimit/factory.go

    - Extract NewRateLimiter factory function
    - _Requirements: 1.3_

  - [x] 6.4 Write property tests for rate limiters

    - **Property 5: Token Bucket Capacity Invariant**
    - **Property 6: Sliding Window Request Count**
    - **Validates: Requirements 1.3**



- [x] 7. Extract bulkhead implementation

  - [x] 7.1 Create libs/go/resilience/bulkhead/bulkhead.go



    - Extract Bulkhead struct and Config
    - Extract New, Acquire, Release, GetMetrics methods
    - _Requirements: 1.4_
  - [x] 7.2 Create libs/go/resilience/bulkhead/manager.go

    - Extract Manager struct
    - Extract NewManager, GetBulkhead, GetAllMetrics methods
    - Extract Partitions iterator
    - _Requirements: 1.4_

  - [x] 7.3 Write property tests for bulkhead

    - **Property 7: Bulkhead Concurrency Limit**
    - **Validates: Requirements 1.4**

- [x] 8. Extract timeout manager implementation


  - [x] 8.1 Create libs/go/resilience/timeout/manager.go

    - Extract Manager struct and Config
    - Extract New, Execute, GetTimeout, WithTimeout methods
    - _Requirements: 1.5_

  - [x] 8.2 Write property tests for timeout manager

    - Test timeout enforcement behavior
    - **Validates: Requirements 1.5**


- [x] 9. Checkpoint - Ensure all tests pass


  - Ensure all tests pass, ask the user if questions arise.


- [x] 10. Extract health aggregator implementation


  - [x] 10.1 Create libs/go/resilience/health/aggregator.go

    - Extract Aggregator struct and Config
    - Extract NewAggregator, GetAggregatedHealth methods
    - Extract RegisterService, UnregisterService, UpdateHealth methods
    - Extract CheckAll, checkService helpers
    - Extract aggregateStatus, AggregateStatuses functions
    - Extract Services iterator
    - _Requirements: 2.1_


  - [x] 10.2 Write property tests for health aggregator

    - **Property 8: Health Status Aggregation**


    - **Validates: Requirements 2.1**

- [x] 11. Extract graceful shutdown implementation

  - [x] 11.1 Create libs/go/resilience/shutdown/graceful.go

    - Extract GracefulShutdown struct
    - Extract NewGracefulShutdown constructor
    - Extract RequestStarted, RequestFinished, InFlightCount methods
    - Extract Shutdown, IsShutdown, ShutdownCh methods
    - _Requirements: 3.1, 3.2, 3.3_



  - [x] 11.2 Write property tests for graceful shutdown


    - **Property 9: Graceful Shutdown Request Tracking**
    - **Validates: Requirements 3.1, 3.2, 3.3**

- [x] 12. Extract test utilities

  - [x] 12.1 Create libs/go/resilience/testutil/generators.go

    - Extract GenCircuitState, GenCircuitBreakerConfig, GenCircuitBreakerState
    - Extract GenRetryConfig, GenTimeoutConfig
    - Extract GenRateLimitConfig, GenBulkheadConfig
    - Extract GenHealthStatus, GenResiliencePolicy
    - Extract GenCorrelationID, GenServiceName
    - _Requirements: 4.1_
  - [x] 12.2 Create libs/go/resilience/testutil/emitter.go


    - Extract MockEventEmitter struct
    - Extract NewMockEventEmitter, Emit, EmitAudit methods
    - Extract GetEvents, GetAuditEvents, Clear methods
    - Extract GetStateChangeEvents helper
    - _Requirements: 4.3_



  - [x] 12.3 Write property tests for test utilities
    - **Property 10: Generated Configs Pass Validation**


    - **Property 15: Mock Event Emitter Recording**
    - **Validates: Requirements 4.1, 4.2, 4.3**




- [x] 13. Checkpoint - Ensure all tests pass

  - Ensure all tests pass, ask the user if questions arise.


- [x] 14. Update resilience-service to use extracted libraries
  - [x] 14.1 Update internal/domain/ to import from libs

    - Update circuit_breaker.go to import CircuitBreakerState, CircuitBreaker from libs
    - Update ratelimit.go to import RateLimitDecision, RateLimitHeaders, RateLimiter from libs
    - Update bulkhead.go to import BulkheadMetrics, Bulkhead, BulkheadManager from libs
    - Update timeout.go to import TimeoutManager from libs
    - Update retry.go to import RetryHandler from libs
    - Update health.go to import health types from libs
    - Update policy_events.go to import from libs
    - _Requirements: 6.1, 6.3_
  - [x] 14.2 Update internal/circuitbreaker/ to use libs

    - Update breaker.go to import from libs/go/resilience/circuitbreaker
    - Update serialization.go to re-export from libs

    - Remove duplicated implementation code
    - _Requirements: 6.1, 6.4_

  - [x] 14.3 Update internal/retry/ to use libs


    - Update handler.go to import from libs/go/resilience/retry
    - Update policy.go to re-export from libs
    - Update rand.go to import from libs/go/resilience/rand
    - Remove duplicated implementation code
    - _Requirements: 6.1, 6.4_
  - [x] 14.4 Update internal/ratelimit/ to use libs


    - Update token_bucket.go to import from libs
    - Update sliding_window.go to import from libs

    - Update factory.go to re-export from libs
    - Remove duplicated implementation code
    - _Requirements: 6.1, 6.4_

  - [x] 14.5 Update internal/bulkhead/ to use libs

    - Update bulkhead.go to import from libs/go/resilience/bulkhead
    - Remove duplicated implementation code

    - _Requirements: 6.1, 6.4_
  - [x] 14.6 Update internal/timeout/ to use libs

    - Update manager.go to import from libs/go/resilience/timeout
    - Remove duplicated implementation code
    - _Requirements: 6.1, 6.4_

  - [x] 14.7 Update internal/health/ to use libs

    - Update aggregator.go to import from libs/go/resilience/health
    - Remove duplicated implementation code
    - _Requirements: 6.1, 6.4_
  - [x] 14.8 Update internal/server/ to use libs


    - Update shutdown.go to import from libs/go/resilience/shutdown
    - Remove duplicated implementation code
    - _Requirements: 6.1, 6.4_

  - [x] 14.9 Update tests/testutil/ to use libs

    - Update generators.go to import from libs/go/resilience/testutil
    - Remove duplicated generator code
    - _Requirements: 6.1, 6.4_



- [x] 15. Final Checkpoint - Ensure all tests pass

  - Ensure all tests pass, ask the user if questions arise.
