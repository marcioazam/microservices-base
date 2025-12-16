# Implementation Plan

## Estrutura de Diretórios (2025)

Este plano segue a arquitetura de monorepo aprovada:
- Resilience Service: `platform/resilience-service/`
- Protos: `api/proto/infra/`
- Deploy: `deploy/kubernetes/helm/resilience-service/`

---

## Fase 1: Estrutura do Projeto e Interfaces Core

- [x] 1. Set up project structure and core interfaces

  - [x] 1.1 Initialize Go module and directory structure
    - Create `platform/resilience-service/` with standard Go project layout
    - Set up `go.mod` with Go 1.22+
    - Create directories: `cmd/server/`, `internal/`, `pkg/`
    - _Requirements: 8.1_

  - [x] 1.2 Define core domain interfaces and types
    - Create `internal/domain/` with CircuitBreaker, RetryHandler, RateLimiter, Bulkhead, TimeoutManager interfaces
    - Define error types in `internal/domain/errors.go`
    - Define configuration types in `internal/config/`
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1_

  - [x] 1.3 Set up testing framework with gopter
    - Add gopter dependency for property-based testing
    - Create test utilities and generators in `internal/testutil/`
    - Configure test coverage reporting
    - _Requirements: 1.6, 2.6, 7.4_

---

## Fase 2: Circuit Breaker

- [x] 2. Implement Circuit Breaker component

  - [x] 2.1 Implement circuit breaker state machine
    - Create `internal/circuitbreaker/breaker.go` with state transitions
    - Implement Closed → Open → HalfOpen → Closed state machine
    - Add configurable thresholds and timeout
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 2.2 Write property test for circuit breaker state machine
    - **Property 1: Circuit Breaker State Machine Correctness**
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4**

  - [x] 2.3 Implement circuit breaker state serialization
    - Create JSON serialization for CircuitBreakerState
    - Implement state persistence interface
    - _Requirements: 1.6_

  - [x] 2.4 Write property test for state serialization round-trip
    - **Property 2: Circuit Breaker State Serialization Round-Trip**
    - **Validates: Requirements 1.6**

  - [x] 2.5 Implement circuit state change event emission
    - Create event emitter for state transitions
    - Include correlation ID and service metadata
    - _Requirements: 1.5_

  - [x] 2.6 Write property test for event emission
    - **Property 3: Circuit State Change Event Emission**
    - **Validates: Requirements 1.5**

- [x] 3. Checkpoint - Ensure all circuit breaker tests pass

---

## Fase 3: Retry Handler

- [x] 4. Implement Retry Handler component

  - [x] 4.1 Implement exponential backoff with jitter
    - Create `internal/retry/handler.go` with backoff calculation
    - Implement configurable base delay, multiplier, max delay
    - Add randomized jitter within configured percentage
    - _Requirements: 2.2, 2.3_

  - [x] 4.2 Write property test for backoff calculation
    - **Property 4: Retry Delay with Exponential Backoff and Jitter**
    - **Validates: Requirements 2.2, 2.3**

  - [x] 4.3 Implement retry exhaustion handling
    - Return final error with retry exhaustion metadata
    - Track total attempts made
    - _Requirements: 2.4_

  - [x] 4.4 Write property test for retry exhaustion
    - **Property 5: Retry Exhaustion Returns Final Error**
    - **Validates: Requirements 2.4**

  - [x] 4.5 Implement circuit breaker integration
    - Skip retries when circuit is open
    - Fail fast with circuit open error
    - _Requirements: 2.5_

  - [x] 4.6 Write property test for circuit breaker blocking retries
    - **Property 6: Open Circuit Blocks Retry Attempts**
    - **Validates: Requirements 2.5**

  - [x] 4.7 Implement retry policy configuration parsing
    - Create policy parser with validation
    - Implement pretty-printer for debugging
    - _Requirements: 2.6_

  - [x] 4.8 Write property test for policy round-trip
    - **Property 7: Retry Policy Configuration Round-Trip**
    - **Validates: Requirements 2.6**

- [x] 5. Checkpoint - Ensure all retry handler tests pass

---

## Fase 4: Timeout Manager

- [x] 6. Implement Timeout Manager component

  - [x] 6.1 Implement timeout enforcement with context cancellation
    - Create `internal/timeout/manager.go`
    - Use Go context for deadline propagation
    - Cancel operations exceeding timeout
    - _Requirements: 3.1_

  - [x] 6.2 Write property test for timeout enforcement
    - **Property 8: Timeout Enforcement**
    - **Validates: Requirements 3.1_

  - [x] 6.3 Implement operation-specific timeout precedence
    - Support per-operation timeout configuration
    - Override global default with operation-specific value
    - _Requirements: 3.2_

  - [x] 6.4 Write property test for timeout precedence
    - **Property 9: Operation-Specific Timeout Precedence**
    - **Validates: Requirements 3.2**

  - [x] 6.5 Implement timeout configuration validation
    - Validate positive durations within bounds
    - Reject invalid configurations with descriptive errors
    - _Requirements: 3.4_

  - [x] 6.6 Write property test for timeout validation
    - **Property 10: Timeout Configuration Validation**
    - **Validates: Requirements 3.4**

---

## Fase 5: Rate Limiter

- [x] 7. Implement Rate Limiter component

  - [x] 7.1 Implement token bucket algorithm
    - Create `internal/ratelimit/token_bucket.go`
    - Implement token refill at configured rate
    - Enforce capacity limit for bursting
    - _Requirements: 4.2_

  - [x] 7.2 Write property test for token bucket invariants
    - **Property 12: Token Bucket Invariants**
    - **Validates: Requirements 4.2**

  - [x] 7.3 Implement sliding window algorithm
    - Create `internal/ratelimit/sliding_window.go`
    - Track request counts within sliding time window
    - _Requirements: 4.3_

  - [x] 7.4 Write property test for sliding window counting
    - **Property 13: Sliding Window Request Counting**
    - **Validates: Requirements 4.3**

  - [x] 7.5 Implement rate limit enforcement and response
    - Return 429 status with Retry-After header
    - Include rate limit headers in all responses
    - _Requirements: 4.1, 4.5_

  - [x] 7.6 Write property test for rate limit enforcement
    - **Property 11: Rate Limit Enforcement**
    - **Validates: Requirements 4.1**

  - [x] 7.7 Write property test for rate limit headers
    - **Property 14: Rate Limit Response Headers**
    - **Validates: Requirements 4.5**

- [x] 8. Checkpoint - Ensure all rate limiter tests pass

---

## Fase 6: Bulkhead

- [x] 9. Implement Bulkhead component

  - [x] 9.1 Implement semaphore-based concurrency limiting
    - Create `internal/bulkhead/bulkhead.go`
    - Use semaphore for max concurrent requests
    - Implement queue with configurable capacity
    - _Requirements: 5.1, 5.2_

  - [x] 9.2 Write property test for bulkhead enforcement
    - **Property 15: Bulkhead Concurrent Request Enforcement**
    - **Validates: Requirements 5.1, 5.2**

  - [x] 9.3 Implement partition isolation
    - Support partitioning by service name, operation type, or custom key
    - Ensure independent limits per partition
    - _Requirements: 5.3_

  - [x] 9.4 Write property test for partition isolation
    - **Property 16: Bulkhead Partition Isolation**
    - **Validates: Requirements 5.3**

  - [x] 9.5 Implement bulkhead metrics reporting
    - Report active count, queued count, rejected count
    - Expose metrics per partition
    - _Requirements: 5.4_

  - [x] 9.6 Write property test for metrics accuracy
    - **Property 17: Bulkhead Metrics Accuracy**
    - **Validates: Requirements 5.4**

---

## Fase 7: Health Aggregator

- [x] 10. Implement Health Aggregator component

  - [x] 10.1 Implement health aggregation logic
    - Create `internal/health/aggregator.go`
    - Aggregate health from all protected services
    - Calculate overall status (healthy/degraded/unhealthy)
    - _Requirements: 6.1_

  - [x] 10.2 Write property test for health aggregation
    - **Property 18: Health Aggregation Logic**
    - **Validates: Requirements 6.1**

  - [x] 10.3 Implement health change event emission
    - Emit CAEP event on health status change
    - Include service name, previous and new status
    - _Requirements: 6.2_

  - [x] 10.4 Write property test for health change events
    - **Property 19: Health Change Event Emission**
    - **Validates: Requirements 6.2**

- [x] 11. Checkpoint - Ensure all health aggregator tests pass

---

## Fase 8: Policy Engine

- [x] 12. Implement Policy Engine component

  - [x] 12.1 Implement policy validation
    - Create `internal/policy/engine.go`
    - Validate policy configurations
    - Reject invalid policies with descriptive errors
    - _Requirements: 7.1_

  - [x] 12.2 Write property test for policy validation
    - **Property 20: Policy Validation Rejects Invalid Configurations**
    - **Validates: Requirements 7.1**

  - [x] 12.3 Implement policy serialization and pretty-printing
    - JSON serialization for ResiliencePolicy
    - Pretty-printer for human-readable output
    - _Requirements: 7.4_

  - [x] 12.4 Write property test for policy round-trip
    - **Property 21: Policy Definition Round-Trip**
    - **Validates: Requirements 7.4**

  - [x] 12.5 Implement policy hot-reload
    - Watch configuration file for changes
    - Apply updates without service restart
    - _Requirements: 7.2_

---

## Fase 9: gRPC Service Layer

- [x] 13. Implement gRPC Service Layer

  - [x] 13.1 Expand protobuf service definitions
    - Update `api/proto/infra/resilience.proto`
    - Add ExecuteWithResilience, UpdatePolicy RPCs
    - Define request/response messages per design
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

  - [x] 13.2 Implement gRPC server with resilience operations
    - Create `internal/grpc/server.go`
    - Implement all RPC handlers
    - Wire up resilience components
    - _Requirements: 8.1_

  - [x] 13.3 Implement circuit state retrieval
    - Return current circuit breaker state for specified service
    - _Requirements: 8.2_

  - [x] 13.4 Write property test for state retrieval consistency
    - **Property 22: Circuit State Retrieval Consistency**
    - **Validates: Requirements 8.2**

  - [x] 13.5 Implement error to gRPC status mapping
    - Map internal errors to appropriate gRPC status codes
    - Include detailed error metadata
    - _Requirements: 8.4_

  - [x] 13.6 Write property test for error mapping
    - **Property 23: Error to gRPC Status Code Mapping**
    - **Validates: Requirements 8.4**

- [x] 14. Checkpoint - Ensure all gRPC service tests pass

---

## Fase 10: Infrastructure Layer

- [x] 15. Implement Infrastructure Layer

  - [x] 15.1 Implement Redis client for distributed state
    - Create `internal/infra/redis/client.go`
    - Support rate limit state synchronization
    - Handle connection failures gracefully
    - _Requirements: 4.4_

  - [x] 15.2 Implement OpenTelemetry integration
    - Create `internal/infra/otel/provider.go`
    - Configure tracing with W3C Trace Context
    - Export metrics to Prometheus
    - _Requirements: 6.3, 6.4_

  - [x] 15.3 Implement audit event logging
    - Create structured audit events
    - Include correlation ID and SPIFFE identity
    - _Requirements: 9.3_

  - [x] 15.4 Write property test for audit event fields
    - **Property 24: Audit Event Required Fields**
    - **Validates: Requirements 9.3**

---

## Fase 11: Server Lifecycle

- [x] 16. Implement Server Lifecycle

  - [x] 16.1 Implement main server entry point
    - Create `cmd/server/main.go`
    - Wire all components together
    - Load configuration from environment/files
    - _Requirements: 8.1_

  - [x] 16.2 Implement graceful shutdown
    - Add signal handling (SIGTERM, SIGINT)
    - Drain in-flight requests before termination
    - _Requirements: 10.4_

  - [x] 16.3 Write property test for graceful shutdown
    - **Property 25: Graceful Shutdown Request Draining**
    - **Validates: Requirements 10.4**

  - [x] 16.4 Implement health check endpoints
    - gRPC health check service
    - Kubernetes liveness and readiness probes
    - _Requirements: 6.1_

---

## Fase 12: Kubernetes Deployment

- [x] 17. Finalize Kubernetes Deployment

  - [x] 17.1 Create Dockerfile for resilience-service
    - Create `deploy/docker/resilience-service/Dockerfile`
    - Multi-stage build for minimal image
    - Non-root user, read-only filesystem
    - _Requirements: 10.1_

  - [x] 17.2 Verify Helm chart configuration
    - Ensure `deploy/kubernetes/helm/resilience-service/` is complete
    - Verify Linkerd injection annotation
    - Confirm HPA configuration
    - _Requirements: 10.1, 10.3_

- [x] 18. Final Checkpoint - All tests pass ✓

---

## Summary

| Fase | Componente | Tasks | Property Tests | Status |
|------|------------|-------|----------------|--------|
| 1 | Setup | 3 | 0 | ✅ |
| 2 | Circuit Breaker | 6 | 3 | ✅ |
| 3 | Retry Handler | 8 | 4 | ✅ |
| 4 | Timeout Manager | 6 | 3 | ✅ |
| 5 | Rate Limiter | 7 | 4 | ✅ |
| 6 | Bulkhead | 6 | 3 | ✅ |
| 7 | Health Aggregator | 4 | 2 | ✅ |
| 8 | Policy Engine | 5 | 2 | ✅ |
| 9 | gRPC Service | 6 | 2 | ✅ |
| 10 | Infrastructure | 4 | 1 | ✅ |
| 11 | Server Lifecycle | 4 | 1 | ✅ |
| 12 | Kubernetes | 2 | 0 | ✅ |
| **Total** | | **61** | **25** | **✅ COMPLETE** |

---

## Test Results

All 25 property tests pass with 100+ iterations each:

```
ok  github.com/auth-platform/platform/resilience-service/internal/bulkhead       16.635s
ok  github.com/auth-platform/platform/resilience-service/internal/circuitbreaker 5.966s
ok  github.com/auth-platform/platform/resilience-service/internal/grpc           1.371s
ok  github.com/auth-platform/platform/resilience-service/internal/health         0.932s
ok  github.com/auth-platform/platform/resilience-service/internal/infra/audit    1.057s
ok  github.com/auth-platform/platform/resilience-service/internal/policy         0.699s
ok  github.com/auth-platform/platform/resilience-service/internal/ratelimit      35.913s
ok  github.com/auth-platform/platform/resilience-service/internal/retry          1.363s
ok  github.com/auth-platform/platform/resilience-service/internal/server         22.814s
ok  github.com/auth-platform/platform/resilience-service/internal/timeout        5.871s
```
