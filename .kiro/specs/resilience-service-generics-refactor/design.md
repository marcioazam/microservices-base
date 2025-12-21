# Design Document: Resilience Service Generics Refactor

## Overview

This design document describes the refactoring of `platform/resilience-service` to maximize the use of Go generics and shared libraries from `libs/go`. The goal is to eliminate code duplication, improve type safety, and ensure reusable components are properly extracted to the shared library.

## Architecture

### Current State

```
platform/resilience-service/
├── internal/
│   ├── domain/
│   │   ├── entities/          # Policy, configs (duplicated logic)
│   │   ├── interfaces/        # Non-generic interfaces
│   │   └── valueobjects/      # Events, metrics (could be shared)
│   ├── infrastructure/
│   │   ├── repositories/      # Redis with manual serialization
│   │   └── resilience/        # Failsafe executor (any type)
│   └── presentation/
└── Uses: 4/24 libs (17%)
```

### Target State

```
libs/go/src/
├── resilience/
│   ├── executor.go            # ResilienceExecutor[T] interface
│   ├── metrics.go             # ExecutionMetrics (shared)
│   └── policy.go              # Generic policy types
├── patterns/
│   └── repository.go          # Repository[T, ID] interface
├── functional/
│   ├── result.go              # Result[T] (existing)
│   └── option.go              # Option[T] (existing)
├── collections/
│   └── lru.go                 # LRUCache[K, V] (existing)
├── validation/
│   └── validation.go          # Composable validators (existing)
├── codec/
│   └── codec.go               # TypedCodec[T] (existing)
└── events/
    └── eventbus.go            # EventBus[T] (existing)

platform/resilience-service/
├── internal/
│   ├── domain/
│   │   ├── entities/          # Service-specific policy configs
│   │   └── interfaces/        # Extends generic interfaces
│   ├── infrastructure/
│   │   ├── repositories/      # CachedRepository using LRUCache
│   │   └── resilience/        # Implements ResilienceExecutor[T]
│   └── presentation/
└── Uses: 12/24 libs (50%+)
```

## Component Design

### 1. Generic Resilience Executor

**Location:** `libs/go/src/resilience/executor.go`

```go
package resilience

import (
    "context"
    "github.com/authcorp/libs/go/src/functional"
)

// ResilienceExecutor applies resilience patterns to operations with type safety.
type ResilienceExecutor[T any] interface {
    // Execute runs an operation with resilience patterns.
    Execute(ctx context.Context, policyName string, op func() error) error
    
    // ExecuteWithResult runs an operation returning a typed result.
    ExecuteWithResult(ctx context.Context, policyName string, op func() (T, error)) functional.Result[T]
}

// ExecutorConfig configures the resilience executor.
type ExecutorConfig struct {
    DefaultTimeout    time.Duration
    MetricsEnabled    bool
    TracingEnabled    bool
}

// NewExecutor creates a new generic resilience executor.
func NewExecutor[T any](config ExecutorConfig) ResilienceExecutor[T] {
    return &executor[T]{config: config}
}
```

### 2. Generic Repository Pattern

**Location:** `libs/go/src/patterns/repository.go`

```go
package patterns

import (
    "context"
    "github.com/authcorp/libs/go/src/functional"
)

// Repository defines generic CRUD operations with type safety.
type Repository[T any, ID comparable] interface {
    // Get retrieves an entity by ID, returning Option for type-safe null handling.
    Get(ctx context.Context, id ID) functional.Option[T]
    
    // Save persists an entity.
    Save(ctx context.Context, entity T) functional.Result[T]
    
    // Delete removes an entity by ID.
    Delete(ctx context.Context, id ID) error
    
    // List returns all entities.
    List(ctx context.Context) functional.Result[[]T]
    
    // Exists checks if an entity exists.
    Exists(ctx context.Context, id ID) bool
}

// CachedRepository wraps a repository with LRU caching.
type CachedRepository[T any, ID comparable] struct {
    inner Repository[T, ID]
    cache *collections.LRUCache[ID, T]
}

// NewCachedRepository creates a cached repository wrapper.
func NewCachedRepository[T any, ID comparable](
    inner Repository[T, ID],
    cacheSize int,
    ttl time.Duration,
) *CachedRepository[T, ID] {
    return &CachedRepository[T, ID]{
        inner: inner,
        cache: collections.NewLRUCache[ID, T](cacheSize).WithTTL(ttl),
    }
}

func (r *CachedRepository[T, ID]) Get(ctx context.Context, id ID) functional.Option[T] {
    // Check cache first
    if opt := r.cache.Get(id); opt.IsSome() {
        return opt
    }
    // Fallback to inner repository
    opt := r.inner.Get(ctx, id)
    if opt.IsSome() {
        r.cache.Put(id, opt.Unwrap())
    }
    return opt
}
```

### 3. Policy Entity with Option Types

**Location:** `platform/resilience-service/internal/domain/entities/policy.go`

```go
package entities

import (
    "github.com/authcorp/libs/go/src/functional"
    "github.com/authcorp/libs/go/src/validation"
)

// Policy represents a resilience policy with type-safe optional configs.
type Policy struct {
    name           string
    version        int
    circuitBreaker functional.Option[*CircuitBreakerConfig]
    retry          functional.Option[*RetryConfig]
    timeout        functional.Option[*TimeoutConfig]
    rateLimit      functional.Option[*RateLimitConfig]
    bulkhead       functional.Option[*BulkheadConfig]
    createdAt      time.Time
    updatedAt      time.Time
}

// CircuitBreaker returns the circuit breaker config as Option.
func (p *Policy) CircuitBreaker() functional.Option[*CircuitBreakerConfig] {
    return p.circuitBreaker
}

// SetCircuitBreaker sets the circuit breaker config.
func (p *Policy) SetCircuitBreaker(config *CircuitBreakerConfig) functional.Result[*Policy] {
    result := config.Validate()
    if result.IsErr() {
        return functional.Err[*Policy](result.UnwrapErr())
    }
    p.circuitBreaker = functional.Some(config)
    p.updatedAt = time.Now().UTC()
    return functional.Ok(p)
}
```

### 4. Composable Validation

**Location:** `platform/resilience-service/internal/domain/entities/configs.go`

```go
package entities

import (
    "github.com/authcorp/libs/go/src/functional"
    "github.com/authcorp/libs/go/src/validation"
)

// CircuitBreakerConfig with composable validation.
type CircuitBreakerConfig struct {
    FailureThreshold int
    SuccessThreshold int
    Timeout          time.Duration
    ProbeCount       int
}

// Validate uses composable validators from libs/go.
func (c *CircuitBreakerConfig) Validate() functional.Result[*CircuitBreakerConfig] {
    result := validation.ValidateAll(
        validation.Field("failure_threshold").Int(c.FailureThreshold,
            validation.InRange(1, 100)),
        validation.Field("success_threshold").Int(c.SuccessThreshold,
            validation.InRange(1, 10)),
        validation.Field("timeout").Duration(c.Timeout,
            validation.DurationRange(time.Second, 5*time.Minute)),
        validation.Field("probe_count").Int(c.ProbeCount,
            validation.InRange(1, 10)),
    )
    
    if result.HasErrors() {
        return functional.Err[*CircuitBreakerConfig](result.Error())
    }
    return functional.Ok(c)
}

// RetryConfig with composable validation.
type RetryConfig struct {
    MaxAttempts   int
    BaseDelay     time.Duration
    MaxDelay      time.Duration
    Multiplier    float64
    JitterPercent float64
}

func (r *RetryConfig) Validate() functional.Result[*RetryConfig] {
    result := validation.ValidateAll(
        validation.Field("max_attempts").Int(r.MaxAttempts,
            validation.InRange(1, 10)),
        validation.Field("base_delay").Duration(r.BaseDelay,
            validation.DurationRange(time.Millisecond, 10*time.Second)),
        validation.Field("max_delay").Duration(r.MaxDelay,
            validation.DurationRange(time.Second, 5*time.Minute)),
        validation.Field("multiplier").Float(r.Multiplier,
            validation.FloatRange(1.0, 10.0)),
        validation.Field("jitter_percent").Float(r.JitterPercent,
            validation.FloatRange(0.0, 1.0)),
    )
    
    // Custom cross-field validation
    if r.BaseDelay > r.MaxDelay {
        result.AddError("base_delay", "cannot be greater than max_delay")
    }
    
    if result.HasErrors() {
        return functional.Err[*RetryConfig](result.Error())
    }
    return functional.Ok(r)
}

// RateLimitConfig with OneOf validator.
type RateLimitConfig struct {
    Algorithm string
    Limit     int
    Window    time.Duration
    BurstSize int
}

func (r *RateLimitConfig) Validate() functional.Result[*RateLimitConfig] {
    result := validation.ValidateAll(
        validation.Field("algorithm").String(r.Algorithm,
            validation.OneOf("token_bucket", "sliding_window")),
        validation.Field("limit").Int(r.Limit,
            validation.InRange(1, 100000)),
        validation.Field("window").Duration(r.Window,
            validation.DurationRange(time.Second, time.Hour)),
        validation.Field("burst_size").Int(r.BurstSize,
            validation.InRange(1, 10000)),
    )
    
    if r.BurstSize > r.Limit {
        result.AddError("burst_size", "cannot be greater than limit")
    }
    
    if result.HasErrors() {
        return functional.Err[*RateLimitConfig](result.Error())
    }
    return functional.Ok(r)
}
```

### 5. Cached Repository with LRU

**Location:** `platform/resilience-service/internal/infrastructure/repositories/cached_repository.go`

```go
package repositories

import (
    "context"
    "github.com/authcorp/libs/go/src/collections"
    "github.com/authcorp/libs/go/src/functional"
    "github.com/auth-platform/platform/resilience-service/internal/domain/entities"
)

// CachedPolicyRepository wraps Redis with LRU caching.
type CachedPolicyRepository struct {
    redis  *RedisRepository
    cache  *collections.LRUCache[string, *entities.Policy]
    logger *slog.Logger
}

// NewCachedPolicyRepository creates a cached policy repository.
func NewCachedPolicyRepository(
    redis *RedisRepository,
    cacheSize int,
    ttl time.Duration,
    logger *slog.Logger,
) *CachedPolicyRepository {
    cache := collections.NewLRUCache[string, *entities.Policy](cacheSize).
        WithTTL(ttl).
        WithEvictCallback(func(key string, value *entities.Policy) {
            logger.Debug("policy evicted from cache", slog.String("policy", key))
        })
    
    return &CachedPolicyRepository{
        redis:  redis,
        cache:  cache,
        logger: logger,
    }
}

// Get retrieves a policy, checking cache first.
func (r *CachedPolicyRepository) Get(ctx context.Context, name string) functional.Option[*entities.Policy] {
    // Check cache first
    if opt := r.cache.Get(name); opt.IsSome() {
        r.logger.DebugContext(ctx, "cache hit", slog.String("policy", name))
        return opt
    }
    
    r.logger.DebugContext(ctx, "cache miss", slog.String("policy", name))
    
    // Fallback to Redis
    policy, err := r.redis.Get(ctx, name)
    if err != nil || policy == nil {
        return functional.None[*entities.Policy]()
    }
    
    // Populate cache
    r.cache.Put(name, policy)
    return functional.Some(policy)
}

// Save stores a policy and invalidates cache.
func (r *CachedPolicyRepository) Save(ctx context.Context, policy *entities.Policy) functional.Result[*entities.Policy] {
    if err := r.redis.Save(ctx, policy); err != nil {
        return functional.Err[*entities.Policy](err)
    }
    
    // Update cache
    r.cache.Put(policy.Name(), policy)
    return functional.Ok(policy)
}

// Delete removes a policy and invalidates cache.
func (r *CachedPolicyRepository) Delete(ctx context.Context, name string) error {
    r.cache.Remove(name)
    return r.redis.Delete(ctx, name)
}

// Stats returns cache statistics.
func (r *CachedPolicyRepository) Stats() collections.Stats {
    return r.cache.Stats()
}
```

### 6. Typed Codec for Serialization

**Location:** `platform/resilience-service/internal/infrastructure/repositories/codec.go`

```go
package repositories

import (
    "github.com/authcorp/libs/go/src/codec"
    "github.com/authcorp/libs/go/src/functional"
    "github.com/auth-platform/platform/resilience-service/internal/domain/entities"
)

// PolicyCodec provides type-safe JSON serialization for policies.
type PolicyCodec struct {
    codec codec.TypedCodec[*policyDTO]
}

// policyDTO is the serialization format for Policy.
type policyDTO struct {
    Name           string                    `json:"name"`
    Version        int                       `json:"version"`
    CircuitBreaker *circuitBreakerDTO        `json:"circuit_breaker,omitempty"`
    Retry          *retryDTO                 `json:"retry,omitempty"`
    Timeout        *timeoutDTO               `json:"timeout,omitempty"`
    RateLimit      *rateLimitDTO             `json:"rate_limit,omitempty"`
    Bulkhead       *bulkheadDTO              `json:"bulkhead,omitempty"`
    CreatedAt      time.Time                 `json:"created_at"`
    UpdatedAt      time.Time                 `json:"updated_at"`
}

// NewPolicyCodec creates a new policy codec.
func NewPolicyCodec() *PolicyCodec {
    return &PolicyCodec{
        codec: codec.NewTypedJSONCodec[*policyDTO](),
    }
}

// Encode serializes a policy to JSON.
func (c *PolicyCodec) Encode(policy *entities.Policy) functional.Result[string] {
    dto := c.toDTO(policy)
    return c.codec.Encode(dto)
}

// Decode deserializes JSON to a policy.
func (c *PolicyCodec) Decode(data string) functional.Result[*entities.Policy] {
    result := c.codec.Decode(data)
    if result.IsErr() {
        return functional.Err[*entities.Policy](result.UnwrapErr())
    }
    return c.fromDTO(result.Unwrap())
}

func (c *PolicyCodec) toDTO(policy *entities.Policy) *policyDTO {
    dto := &policyDTO{
        Name:      policy.Name(),
        Version:   policy.Version(),
        CreatedAt: policy.CreatedAt(),
        UpdatedAt: policy.UpdatedAt(),
    }
    
    policy.CircuitBreaker().Match(
        func(cb *entities.CircuitBreakerConfig) { dto.CircuitBreaker = toCircuitBreakerDTO(cb) },
        func() {},
    )
    
    policy.Retry().Match(
        func(r *entities.RetryConfig) { dto.Retry = toRetryDTO(r) },
        func() {},
    )
    
    // ... other configs
    
    return dto
}

func (c *PolicyCodec) fromDTO(dto *policyDTO) functional.Result[*entities.Policy] {
    policy, err := entities.NewPolicy(dto.Name)
    if err != nil {
        return functional.Err[*entities.Policy](err)
    }
    
    if dto.CircuitBreaker != nil {
        cb := fromCircuitBreakerDTO(dto.CircuitBreaker)
        if result := policy.SetCircuitBreaker(cb); result.IsErr() {
            return functional.Err[*entities.Policy](result.UnwrapErr())
        }
    }
    
    // ... other configs
    
    return functional.Ok(policy)
}
```

### 7. Generic Event Bus Integration

**Location:** `platform/resilience-service/internal/infrastructure/events/publisher.go`

```go
package events

import (
    "context"
    "github.com/authcorp/libs/go/src/events"
    "github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
)

// PolicyEventPublisher publishes typed policy events.
type PolicyEventPublisher struct {
    bus    *events.EventBus[valueobjects.PolicyEvent]
    logger *slog.Logger
}

// NewPolicyEventPublisher creates a new policy event publisher.
func NewPolicyEventPublisher(logger *slog.Logger) *PolicyEventPublisher {
    return &PolicyEventPublisher{
        bus:    events.NewEventBus[valueobjects.PolicyEvent](),
        logger: logger,
    }
}

// Publish emits a typed policy event.
func (p *PolicyEventPublisher) Publish(ctx context.Context, event valueobjects.PolicyEvent) error {
    p.logger.InfoContext(ctx, "publishing policy event",
        slog.String("event_type", string(event.Type)),
        slog.String("policy_name", event.PolicyName))
    
    return p.bus.Publish(ctx, event)
}

// Subscribe registers a typed event handler.
func (p *PolicyEventPublisher) Subscribe(handler func(valueobjects.PolicyEvent)) {
    p.bus.Subscribe(handler)
}
```

### 8. Execution Metrics (Shared)

**Location:** `libs/go/src/resilience/metrics.go`

```go
package resilience

import (
    "time"
    "github.com/authcorp/libs/go/src/functional"
)

// ExecutionMetrics captures resilience execution statistics.
type ExecutionMetrics struct {
    PolicyName     string        `json:"policy_name"`
    ExecutionTime  time.Duration `json:"execution_time"`
    Success        bool          `json:"success"`
    CircuitState   string        `json:"circuit_state,omitempty"`
    RetryAttempts  int           `json:"retry_attempts,omitempty"`
    RateLimited    bool          `json:"rate_limited,omitempty"`
    BulkheadQueued bool          `json:"bulkhead_queued,omitempty"`
    Timestamp      time.Time     `json:"timestamp"`
}

// NewExecutionMetrics creates new execution metrics.
func NewExecutionMetrics(policyName string, executionTime time.Duration, success bool) ExecutionMetrics {
    return ExecutionMetrics{
        PolicyName:    policyName,
        ExecutionTime: executionTime,
        Success:       success,
        Timestamp:     time.Now().UTC(),
    }
}

// Builder methods for fluent API
func (e ExecutionMetrics) WithCircuitState(state string) ExecutionMetrics {
    e.CircuitState = state
    return e
}

func (e ExecutionMetrics) WithRetryAttempts(attempts int) ExecutionMetrics {
    e.RetryAttempts = attempts
    return e
}

func (e ExecutionMetrics) WithRateLimit(limited bool) ExecutionMetrics {
    e.RateLimited = limited
    return e
}

func (e ExecutionMetrics) WithBulkheadQueue(queued bool) ExecutionMetrics {
    e.BulkheadQueued = queued
    return e
}

// MetricsRecorder records execution metrics with type safety.
type MetricsRecorder interface {
    RecordExecution(ctx context.Context, metrics ExecutionMetrics)
    RecordCircuitState(ctx context.Context, policyName string, state string)
    RecordRetryAttempt(ctx context.Context, policyName string, attempt int)
}
```

## Migration Strategy

### Phase 1: Add New Generic Interfaces to libs/go

1. Create `libs/go/src/resilience/executor.go` with `ResilienceExecutor[T]`
2. Create `libs/go/src/patterns/repository.go` with `Repository[T, ID]`
3. Move `ExecutionMetrics` to `libs/go/src/resilience/metrics.go`
4. Add property tests for new generic types

### Phase 2: Update Domain Layer

1. Refactor `Policy` entity to use `Option[T]` for optional configs
2. Update config validation to use composable validators
3. Update interfaces to use `Result[T]` return types
4. Add unit tests for new validation logic

### Phase 3: Update Infrastructure Layer

1. Create `CachedPolicyRepository` using `LRUCache[K, V]`
2. Create `PolicyCodec` using `TypedJSONCodec[T]`
3. Update `FailsafeExecutor` to implement `ResilienceExecutor[T]`
4. Add integration tests for cached repository

### Phase 4: Update Application Layer

1. Update services to use new generic interfaces
2. Update event publishing to use typed `EventBus[T]`
3. Update metrics recording to use shared types
4. Add end-to-end tests

### Phase 5: Cleanup

1. Remove deprecated non-generic interfaces
2. Update go.mod dependencies
3. Update documentation
4. Run full test suite

## Dependencies

### New libs/go Imports

```go
import (
    "github.com/authcorp/libs/go/src/functional"      // Result[T], Option[T]
    "github.com/authcorp/libs/go/src/collections"     // LRUCache[K, V]
    "github.com/authcorp/libs/go/src/validation"      // Composable validators
    "github.com/authcorp/libs/go/src/codec"           // TypedJSONCodec[T]
    "github.com/authcorp/libs/go/src/events"          // EventBus[T]
    "github.com/authcorp/libs/go/src/resilience"      // ExecutionMetrics, ResilienceExecutor[T]
    "github.com/authcorp/libs/go/src/patterns"        // Repository[T, ID]
)
```

### Updated go.mod

```go
require (
    github.com/authcorp/libs/go/src/functional v0.0.0
    github.com/authcorp/libs/go/src/collections v0.0.0
    github.com/authcorp/libs/go/src/validation v0.0.0
    github.com/authcorp/libs/go/src/codec v0.0.0
    github.com/authcorp/libs/go/src/events v0.0.0
    github.com/authcorp/libs/go/src/resilience v0.0.0
    github.com/authcorp/libs/go/src/patterns v0.0.0
)
```

## Testing Strategy

### Property-Based Tests

1. `executor_prop_test.go` - Generic executor properties
2. `repository_prop_test.go` - Generic repository properties
3. `cache_prop_test.go` - LRU cache integration properties
4. `validation_prop_test.go` - Composable validation properties

### Integration Tests

1. `cached_repository_test.go` - Cache + Redis integration
2. `codec_test.go` - Serialization roundtrip
3. `eventbus_test.go` - Event publishing/subscribing

### Benchmark Tests

1. `cache_bench_test.go` - LRU cache performance
2. `executor_bench_test.go` - Generic executor overhead

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| libs/go usage | 4/24 (17%) | 12/24 (50%+) |
| Generic interfaces | 0 | 5+ |
| Type assertions (`any`) | 10+ | 0 |
| Code duplication | High | Minimal |
| Test coverage | 80% | 90%+ |
