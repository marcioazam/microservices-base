# Implementation Tasks: Resilience Service Generics Refactor

## Phase 1: Add Generic Interfaces to libs/go

### Task 1.1: Create Generic Resilience Executor Interface
- [x] Create `libs/go/src/resilience/executor.go`
- [x] Define `ResilienceExecutor[T any]` interface with:
  - `Execute(ctx, policyName, op func() error) error`
  - `ExecuteWithResult(ctx, policyName, op func() (T, error)) Result[T]`
- [x] Add `ExecutorConfig` struct for configuration
- [x] Create property tests in `libs/go/tests/resilience/executor_prop_test.go`
- [x] Update `libs/go/src/resilience/README.md`

**Acceptance:** Interface compiles, property tests pass (100+ iterations)

### Task 1.2: Create Generic Repository Pattern
- [x] Create `libs/go/src/patterns/repository.go`
- [x] Define `Repository[T any, ID comparable]` interface with:
  - `Get(ctx, id) Option[T]`
  - `Save(ctx, entity) Result[T]`
  - `Delete(ctx, id) error`
  - `List(ctx) Result[[]T]`
  - `Exists(ctx, id) bool`
- [x] Create `CachedRepository[T, ID]` wrapper using `LRUCache[ID, T]`
- [x] Create property tests in `libs/go/tests/patterns/repository_prop_test.go`
- [x] Update `libs/go/src/patterns/README.md`

**Acceptance:** Interface compiles, cached repository works with LRU, tests pass

### Task 1.3: Move ExecutionMetrics to libs/go
- [x] Create `libs/go/src/resilience/metrics.go`
- [x] Move `ExecutionMetrics` struct from resilience-service
- [x] Add builder methods (WithCircuitState, WithRetryAttempts, etc.)
- [x] Define `MetricsRecorder` interface
- [x] Create property tests in `libs/go/tests/resilience/metrics_prop_test.go`
- [x] Update imports in resilience-service

**Acceptance:** Metrics type is shared, no duplication, tests pass

### Task 1.4: Add Duration Validator to validation lib
- [x] Add `DurationRange(min, max time.Duration)` validator to `libs/go/src/validation/`
- [x] Add `FloatRange(min, max float64)` validator
- [x] Add property tests for new validators
- [x] Update `libs/go/src/validation/README.md`

**Acceptance:** Duration and float validators work, property tests pass

---

## Phase 2: Update Domain Layer

### Task 2.1: Refactor Policy Entity with Option Types
- [x] Update `platform/resilience-service/internal/domain/entities/policy.go`
- [x] Change nullable config pointers to `Option[*Config]`:
  - `circuitBreaker functional.Option[*CircuitBreakerConfig]`
  - `retry functional.Option[*RetryConfig]`
  - `timeout functional.Option[*TimeoutConfig]`
  - `rateLimit functional.Option[*RateLimitConfig]`
  - `bulkhead functional.Option[*BulkheadConfig]`
- [x] Update getter methods to return `Option[T]`
- [x] Update setter methods to return `Result[*Policy]`
- [x] Update unit tests

**Acceptance:** Policy uses Option types, nil checks replaced with IsSome()

### Task 2.2: Refactor Config Validation with Composable Validators
- [x] Update `platform/resilience-service/internal/domain/entities/configs.go`
- [x] Refactor `CircuitBreakerConfig.Validate()` to use `validation.ValidateAll()`
- [x] Refactor `RetryConfig.Validate()` to use composable validators
- [x] Refactor `TimeoutConfig.Validate()` to use `DurationRange()`
- [x] Refactor `RateLimitConfig.Validate()` to use `OneOf()`
- [x] Refactor `BulkheadConfig.Validate()` to use composable validators
- [x] Update all Validate() methods to return `Result[*Config]`
- [x] Update property tests

**Acceptance:** Validation uses libs/go validators, returns Result[T]

### Task 2.3: Update Domain Interfaces with Generics
- [x] Update `platform/resilience-service/internal/domain/interfaces/resilience.go`
- [x] Change `PolicyRepository.Get()` to return `Option[*Policy]`
- [x] Change `PolicyRepository.Save()` to return `Result[*Policy]`
- [x] Update `ResilienceExecutor` to extend generic interface from libs/go
- [x] Update `MetricsRecorder` to use shared interface from libs/go
- [x] Update all implementations

**Acceptance:** Interfaces use generics, implementations compile

---

## Phase 3: Update Infrastructure Layer

### Task 3.1: Create Cached Policy Repository
- [x] Create `platform/resilience-service/internal/infrastructure/repositories/cached_repository.go`
- [x] Implement `CachedPolicyRepository` using `collections.LRUCache[string, *Policy]`
- [x] Implement cache-first `Get()` returning `Option[*Policy]`
- [x] Implement `Save()` with cache update returning `Result[*Policy]`
- [x] Implement `Delete()` with cache invalidation
- [x] Add `Stats()` method exposing cache statistics
- [x] Add eviction callback for logging
- [x] Create integration tests

**Acceptance:** Cache reduces Redis calls, stats exposed, tests pass

### Task 3.2: Create Typed Policy Codec
- [x] Create `platform/resilience-service/internal/infrastructure/repositories/codec.go`
- [x] Create `PolicyCodec` using `codec.TypedJSONCodec[*policyDTO]`
- [x] Implement `Encode(policy) Result[string]`
- [x] Implement `Decode(data) Result[*Policy]`
- [x] Create DTO types for serialization
- [x] Handle Option types in serialization (omitempty)
- [x] Create roundtrip property tests

**Acceptance:** Serialization is type-safe, roundtrip preserves data

### Task 3.3: Update Redis Repository to Use Codec
- [x] Update `platform/resilience-service/internal/infrastructure/repositories/redis_repository.go`
- [x] Replace manual JSON serialization with `PolicyCodec`
- [x] Update `Get()` to return `Option[*Policy]`
- [x] Update `Save()` to return `Result[*Policy]`
- [x] Remove `serializePolicy()` and `deserializePolicy()` methods
- [x] Update integration tests

**Acceptance:** Redis repository uses codec, returns Result/Option types

### Task 3.4: Update Failsafe Executor with Generics
- [x] Update `platform/resilience-service/internal/infrastructure/resilience/failsafe_executor.go`
- [x] Implement `ResilienceExecutor[T]` interface from libs/go
- [x] Change `ExecuteWithResult()` to use generic type parameter
- [x] Return `Result[T]` instead of `(any, error)`
- [x] Update metrics recording to use shared `ExecutionMetrics`
- [x] Update benchmark tests

**Acceptance:** Executor is generic, no type assertions needed

---

## Phase 4: Update Application Layer

### Task 4.1: Update Application Services
- [x] Update `platform/resilience-service/internal/application/services/`
- [x] Update service methods to use `Result[T]` return types
- [x] Update policy retrieval to handle `Option[*Policy]`
- [x] Use `FlatMapResult()` for operation chaining
- [x] Use `MapResult()` for transformations
- [x] Update unit tests

**Acceptance:** Services use functional error handling

### Task 4.2: Create Typed Event Publisher
- [x] Create `platform/resilience-service/internal/infrastructure/events/publisher.go`
- [x] Create `PolicyEventPublisher` using `events.EventBus[PolicyEvent]`
- [x] Implement `Publish(ctx, event) error`
- [x] Implement `Subscribe(handler func(PolicyEvent))`
- [x] Update event emission in services
- [x] Create integration tests

**Acceptance:** Events are typed, no interface{} in handlers

### Task 4.3: Update Metrics Recording
- [x] Update `platform/resilience-service/internal/infrastructure/observability/metrics_recorder.go`
- [x] Implement `MetricsRecorder` interface from libs/go
- [x] Use shared `ExecutionMetrics` type
- [x] Add cache statistics metrics (hits, misses, evictions)
- [x] Update Prometheus/OTel integration
- [x] Update tests

**Acceptance:** Metrics use shared types, cache stats exposed

---

## Phase 5: Update Presentation Layer

### Task 5.1: Update gRPC Handlers
- [x] Update `platform/resilience-service/internal/presentation/grpc/`
- [x] Handle `Option[*Policy]` in GetPolicy handler
- [x] Handle `Result[T]` in CreatePolicy/UpdatePolicy handlers
- [x] Map Result errors to gRPC status codes
- [x] Update error responses
- [x] Update integration tests

**Acceptance:** gRPC handlers work with new types

---

## Phase 6: Cleanup and Documentation

### Task 6.1: Remove Deprecated Code
- [x] Remove old non-generic interfaces
- [x] Remove duplicate type definitions
- [x] Remove manual serialization code
- [x] Remove nil checks replaced by Option
- [x] Run linter to find dead code

**Acceptance:** No deprecated code remains

### Task 6.2: Update Dependencies
- [x] Update `platform/resilience-service/go.mod` with new libs/go imports
- [x] Remove unused replace directives
- [x] Run `go mod tidy`
- [x] Verify all imports resolve

**Acceptance:** go.mod is clean, all imports work

### Task 6.3: Update Documentation
- [x] Update `platform/resilience-service/README.md` with new patterns
- [x] Update `libs/go/README.md` with new packages
- [x] Add migration guide for other services
- [x] Update API documentation
- [x] Add code examples

**Acceptance:** Documentation reflects new architecture

### Task 6.4: Run Full Test Suite
- [x] Run all property tests (100+ iterations each)
- [x] Run all unit tests
- [x] Run all integration tests
- [x] Run all benchmark tests
- [x] Verify 90%+ code coverage
- [x] Fix any regressions

**Acceptance:** All tests pass, coverage >= 90%

---

## Summary

| Phase | Tasks | Estimated Effort |
|-------|-------|------------------|
| Phase 1: libs/go | 4 tasks | 2-3 days |
| Phase 2: Domain | 3 tasks | 1-2 days |
| Phase 3: Infrastructure | 4 tasks | 2-3 days |
| Phase 4: Application | 3 tasks | 1-2 days |
| Phase 5: Presentation | 1 task | 0.5 day |
| Phase 6: Cleanup | 4 tasks | 1 day |
| **Total** | **19 tasks** | **8-12 days** |

## Dependencies Graph

```
Phase 1 (libs/go)
    ├── Task 1.1 (Executor) ──────────────────────┐
    ├── Task 1.2 (Repository) ───────────────────┐│
    ├── Task 1.3 (Metrics) ─────────────────────┐││
    └── Task 1.4 (Validators) ─────────────────┐│││
                                               ││││
Phase 2 (Domain)                               ││││
    ├── Task 2.1 (Policy) ◄────────────────────┘│││
    ├── Task 2.2 (Validation) ◄─────────────────┘││
    └── Task 2.3 (Interfaces) ◄──────────────────┘│
                                                  │
Phase 3 (Infrastructure)                          │
    ├── Task 3.1 (Cache) ◄────────────────────────┤
    ├── Task 3.2 (Codec) ◄────────────────────────┤
    ├── Task 3.3 (Redis) ◄────────────────────────┤
    └── Task 3.4 (Executor) ◄─────────────────────┘
                │
Phase 4 (Application)
    ├── Task 4.1 (Services) ◄─────────────────────┘
    ├── Task 4.2 (Events)
    └── Task 4.3 (Metrics)
                │
Phase 5 (Presentation)
    └── Task 5.1 (gRPC) ◄─────────────────────────┘
                │
Phase 6 (Cleanup)
    ├── Task 6.1 (Remove deprecated)
    ├── Task 6.2 (Dependencies)
    ├── Task 6.3 (Documentation)
    └── Task 6.4 (Tests)
```

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking changes | Feature flags, gradual rollout |
| Performance regression | Benchmark tests before/after |
| Type inference issues | Explicit type parameters where needed |
| Serialization compatibility | Roundtrip property tests |
| Cache invalidation bugs | Integration tests with Redis |
