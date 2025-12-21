# Requirements Document

## Introduction

This document defines the requirements for refactoring the `platform/resilience-service` to maximize the use of Go generics (`[T any]`) and shared libraries from `libs/go`. The goal is to eliminate code duplication, improve type safety, and ensure reusable components are properly extracted to the shared library.

## Glossary

- **Resilience_Service**: The microservice that provides centralized resilience patterns (circuit breaker, retry, rate limiting, timeout, bulkhead)
- **Libs_Go**: The shared Go library at `libs/go/src/` containing reusable components
- **Generics**: Go type parameters (`[T any]`) that enable type-safe reusable code
- **Result_T**: A generic type `Result[T]` representing success or failure with type safety
- **Option_T**: A generic type `Option[T]` representing optional values (Some/None)
- **LRU_Cache**: Least Recently Used cache with generic key-value types `LRUCache[K, V]`
- **Policy**: A resilience policy configuration containing circuit breaker, retry, timeout, rate limit, and bulkhead settings
- **Executor**: Component that applies resilience patterns to operations

## Requirements

### Requirement 1: Generic Resilience Executor

**User Story:** As a developer, I want the resilience executor to use generics, so that I get compile-time type safety instead of runtime `any` type assertions.

#### Acceptance Criteria

1. WHEN executing an operation with result, THE Executor SHALL use generic type parameter `ExecuteWithResult[T any]` instead of `func() (any, error)`
2. WHEN an operation succeeds, THE Executor SHALL return `Result[T]` with the typed value instead of `(any, error)`
3. WHEN an operation fails, THE Executor SHALL return `Result[T]` with the error wrapped in the Result type
4. THE Executor interface SHALL be defined in `libs/go/src/resilience/` for reuse across services

### Requirement 2: Option Type for Nullable Fields

**User Story:** As a developer, I want optional configuration fields to use `Option[T]`, so that null handling is explicit and type-safe.

#### Acceptance Criteria

1. WHEN a Policy has optional configurations (circuit breaker, retry, etc.), THE Policy entity SHALL use `Option[*Config]` instead of nullable pointers
2. WHEN retrieving a policy that doesn't exist, THE Repository SHALL return `Option[*Policy]` instead of `(*Policy, error)` with nil
3. THE Option type from `libs/go/src/functional/option.go` SHALL be used for all optional values
4. WHEN checking if a configuration exists, THE System SHALL use `IsSome()` method instead of nil checks

### Requirement 3: LRU Cache for Policy Caching

**User Story:** As a developer, I want policies to be cached in memory using the shared LRU cache, so that Redis lookups are minimized and performance is improved.

#### Acceptance Criteria

1. THE Resilience_Service SHALL use `LRUCache[string, *Policy]` from `libs/go/src/collections/lru.go` for policy caching
2. WHEN a policy is requested, THE CachedRepository SHALL check the LRU cache before querying Redis
3. WHEN a policy is saved or updated, THE CachedRepository SHALL invalidate the cache entry
4. THE LRU cache SHALL be configured with TTL for automatic expiration
5. THE cache statistics (hits, misses, evictions) SHALL be exposed via metrics

### Requirement 4: Composable Validation

**User Story:** As a developer, I want configuration validation to use the shared validation library, so that validation logic is consistent and composable.

#### Acceptance Criteria

1. THE CircuitBreakerConfig validation SHALL use `validation.ValidateAll()` from `libs/go/src/validation/`
2. THE RetryConfig validation SHALL use composable validators like `InRange()`, `Min()`, `Max()`
3. THE TimeoutConfig validation SHALL use `DurationRange()` validator for duration fields
4. THE RateLimitConfig validation SHALL use `OneOf()` validator for algorithm field
5. WHEN validation fails, THE System SHALL return accumulated errors with field names

### Requirement 5: Generic Event Bus

**User Story:** As a developer, I want domain events to use the shared event bus with generics, so that event handling is type-safe.

#### Acceptance Criteria

1. THE PolicyEvent SHALL be published using `EventBus[PolicyEvent]` from `libs/go/src/events/`
2. WHEN a policy is created, updated, or deleted, THE System SHALL emit typed events
3. THE event subscribers SHALL receive strongly-typed events instead of `interface{}`
4. THE EventEmitter interface SHALL use generics for type-safe event emission

### Requirement 6: Shared Codec for Serialization

**User Story:** As a developer, I want policy serialization to use the shared codec library, so that JSON encoding/decoding is consistent and type-safe.

#### Acceptance Criteria

1. THE RedisRepository SHALL use `TypedJSONCodec[Policy]` from `libs/go/src/codec/` for serialization
2. WHEN serializing a policy, THE Codec SHALL return `Result[string]` instead of `(string, error)`
3. WHEN deserializing a policy, THE Codec SHALL return `Result[*Policy]` instead of `(*Policy, error)`
4. THE Codec SHALL handle all nested configuration types (CircuitBreaker, Retry, etc.)

### Requirement 7: Export Reusable Components to Libs

**User Story:** As a developer, I want resilience-specific components that are reusable to be exported to libs/go, so that other services can use them.

#### Acceptance Criteria

1. THE generic `ResilienceExecutor[T]` interface SHALL be defined in `libs/go/src/resilience/executor.go`
2. THE `ExecutionMetrics` value object SHALL be moved to `libs/go/src/resilience/metrics.go`
3. THE `HealthStatus` value object SHALL use the existing `libs/go/src/server/health.go`
4. THE policy configuration types SHALL remain in the service (service-specific)
5. WHEN a component is used by multiple services, THE Component SHALL be in libs/go

### Requirement 8: Generic Repository Pattern

**User Story:** As a developer, I want the repository pattern to use generics, so that CRUD operations are type-safe and reusable.

#### Acceptance Criteria

1. THE Repository interface SHALL be generic: `Repository[T any, ID comparable]`
2. THE Get method SHALL return `Option[T]` for type-safe optional results
3. THE List method SHALL return `[]T` with proper type inference
4. THE generic repository interface SHALL be defined in `libs/go/src/patterns/repository.go`
5. THE PolicyRepository SHALL implement `Repository[*Policy, string]`

### Requirement 9: Functional Error Handling

**User Story:** As a developer, I want error handling to use Result type consistently, so that error propagation is explicit and composable.

#### Acceptance Criteria

1. WHEN an operation can fail, THE Function SHALL return `Result[T]` instead of `(T, error)`
2. THE Result type from `libs/go/src/functional/result.go` SHALL be used for all fallible operations
3. WHEN chaining operations, THE System SHALL use `FlatMapResult()` for composition
4. WHEN transforming success values, THE System SHALL use `MapResult()` for transformation
5. THE existing `Try()` and `TryFunc()` helpers SHALL be used for wrapping standard Go errors

### Requirement 10: Metrics Recorder with Generics

**User Story:** As a developer, I want the metrics recorder to use generics for type-safe metric recording, so that metric types are validated at compile time.

#### Acceptance Criteria

1. THE MetricsRecorder SHALL use generic histogram type `Histogram[T numeric]` for duration metrics
2. THE counter metrics SHALL use generic `Counter[T numeric]` type
3. THE metrics types SHALL be defined in `libs/go/src/observability/metrics.go`
4. WHEN recording execution time, THE Recorder SHALL accept `time.Duration` with type safety

