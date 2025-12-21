# Generics Migration Guide

Guide for migrating services to use the new generic patterns from `libs/go`.

## Overview

This guide covers migrating from traditional Go patterns to the new generic-based patterns:

- `Option[T]` for nullable values
- `Result[T]` for fallible operations
- `Repository[T, ID]` for data access
- `ResilienceExecutor[T]` for fault tolerance

## Migration Steps

### 1. Update Imports

Add the functional and patterns packages:

```go
import (
    "github.com/authcorp/libs/go/src/functional"
    "github.com/authcorp/libs/go/src/patterns"
    "github.com/authcorp/libs/go/src/fault"
    "github.com/authcorp/libs/go/src/validation"
)
```

### 2. Replace Nullable Pointers with Option[T]

**Before:**
```go
type Policy struct {
    circuitBreaker *CircuitBreakerConfig
}

func (p *Policy) CircuitBreaker() *CircuitBreakerConfig {
    return p.circuitBreaker
}

// Usage
if policy.CircuitBreaker() != nil {
    cb := policy.CircuitBreaker()
    // use cb
}
```

**After:**
```go
type Policy struct {
    circuitBreaker functional.Option[*CircuitBreakerConfig]
}

func (p *Policy) CircuitBreaker() functional.Option[*CircuitBreakerConfig] {
    return p.circuitBreaker
}

// Usage
if policy.CircuitBreaker().IsSome() {
    cb := policy.CircuitBreaker().Unwrap()
    // use cb
}

// Or with pattern matching
policy.CircuitBreaker().Match(
    func(cb *CircuitBreakerConfig) {
        // use cb
    },
    func() {
        // handle None case
    },
)
```

### 3. Replace (T, error) with Result[T]

**Before:**
```go
func (r *Repository) Save(ctx context.Context, entity *Entity) (*Entity, error) {
    // save logic
    if err != nil {
        return nil, err
    }
    return entity, nil
}

// Usage
entity, err := repo.Save(ctx, entity)
if err != nil {
    return err
}
```

**After:**
```go
func (r *Repository) Save(ctx context.Context, entity *Entity) functional.Result[*Entity] {
    // save logic
    if err != nil {
        return functional.Err[*Entity](err)
    }
    return functional.Ok(entity)
}

// Usage
result := repo.Save(ctx, entity)
if result.IsErr() {
    return result.UnwrapErr()
}
entity := result.Unwrap()

// Or with FlatMapResult for chaining
result := functional.FlatMapResult(repo.Save(ctx, entity), func(saved *Entity) functional.Result[*Entity] {
    return repo.Index(ctx, saved)
})
```

### 4. Update Repository Interfaces

**Before:**
```go
type Repository interface {
    Get(ctx context.Context, id string) (*Entity, error)
    Save(ctx context.Context, entity *Entity) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context) ([]*Entity, error)
}
```

**After:**
```go
type Repository interface {
    Get(ctx context.Context, id string) functional.Option[*Entity]
    Save(ctx context.Context, entity *Entity) functional.Result[*Entity]
    Delete(ctx context.Context, id string) error
    List(ctx context.Context) functional.Result[[]*Entity]
    Exists(ctx context.Context, id string) bool
}
```

### 5. Use Composable Validators

**Before:**
```go
func (c *Config) Validate() error {
    if c.Timeout < time.Second || c.Timeout > 5*time.Minute {
        return fmt.Errorf("timeout must be between 1s and 5m")
    }
    if c.MaxRetries < 1 || c.MaxRetries > 10 {
        return fmt.Errorf("max_retries must be between 1 and 10")
    }
    return nil
}
```

**After:**
```go
func (c *Config) ValidateResult() functional.Result[*Config] {
    result := validation.NewResult()

    result.Merge(validation.Field("timeout", c.Timeout,
        validation.DurationRange(time.Second, 5*time.Minute)))

    result.Merge(validation.Field("max_retries", c.MaxRetries,
        validation.InRange(1, 10)))

    if !result.IsValid() {
        return functional.Err[*Config](
            fmt.Errorf("validation failed: %v", result.ErrorMessages()))
    }
    return functional.Ok(c)
}
```

### 6. Add Caching with CachedRepository

```go
// Create inner repository
innerRepo := NewDatabaseRepository(db)

// Create LRU cache
cache := collections.NewLRUCache[string, *Entity](1000).
    WithTTL(5 * time.Minute).
    WithEvictCallback(func(key string, value *Entity) {
        logger.Debug("entity evicted", "key", key)
    })

// Wrap with caching
cachedRepo := patterns.NewCachedRepository(
    innerRepo,
    cache,
    func(e *Entity) string { return e.ID },
)

// Use like any repository
opt := cachedRepo.Get(ctx, "entity-123")

// Monitor cache performance
stats := cachedRepo.Stats()
logger.Info("cache stats",
    "hits", stats.Hits,
    "misses", stats.Misses,
    "hit_rate", stats.HitRate)
```

### 7. Use Shared ExecutionMetrics

**Before:**
```go
type ExecutionMetrics struct {
    PolicyName    string
    ExecutionTime time.Duration
    Success       bool
    // ... custom fields
}
```

**After:**
```go
import "github.com/authcorp/libs/go/src/fault"

// Use shared type
metrics := fault.NewExecutionMetrics(policyName, duration, success).
    WithCircuitState("closed").
    WithRetryAttempts(2).
    WithCorrelationID(correlationID)

// Record with shared interface
recorder.RecordExecution(ctx, metrics)
```

## Common Patterns

### Chaining Operations with FlatMapResult

```go
func (s *Service) CreateAndIndex(ctx context.Context, data CreateData) functional.Result[*Entity] {
    return functional.FlatMapResult(
        s.validator.Validate(data),
        func(validated CreateData) functional.Result[*Entity] {
            return functional.FlatMapResult(
                s.repo.Save(ctx, validated.ToEntity()),
                func(saved *Entity) functional.Result[*Entity] {
                    return s.indexer.Index(ctx, saved)
                },
            )
        },
    )
}
```

### Converting Option to Result

```go
func (s *Service) GetRequired(ctx context.Context, id string) functional.Result[*Entity] {
    opt := s.repo.Get(ctx, id)
    return functional.FromOption(opt, fmt.Errorf("entity %s not found", id))
}
```

### Handling Option in gRPC Handlers

```go
func (h *Handler) GetEntity(ctx context.Context, req *pb.GetRequest) (*pb.Entity, error) {
    opt := h.service.Get(ctx, req.Id)
    
    if !opt.IsSome() {
        return nil, status.Errorf(codes.NotFound, "entity %s not found", req.Id)
    }
    
    return toProto(opt.Unwrap()), nil
}
```

### Mapping Result Errors to gRPC Status

```go
func mapErrorToStatus(err error) error {
    errStr := err.Error()
    
    switch {
    case strings.Contains(errStr, "not found"):
        return status.Error(codes.NotFound, err.Error())
    case strings.Contains(errStr, "already exists"):
        return status.Error(codes.AlreadyExists, err.Error())
    case strings.Contains(errStr, "validation"):
        return status.Error(codes.InvalidArgument, err.Error())
    default:
        return status.Error(codes.Internal, err.Error())
    }
}
```

## Testing

### Property Tests for Option/Result

```go
func TestOptionSomeNoneConsistency(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        value := rapid.Int().Draw(t, "value")
        
        some := functional.Some(value)
        none := functional.None[int]()
        
        if !some.IsSome() {
            t.Error("Some should be Some")
        }
        if none.IsSome() {
            t.Error("None should not be Some")
        }
        if some.Unwrap() != value {
            t.Errorf("Unwrap = %d, want %d", some.Unwrap(), value)
        }
    })
}
```

### Testing Repository with Option Returns

```go
func TestRepositoryGetReturnsOption(t *testing.T) {
    repo := NewInMemoryRepository()
    ctx := context.Background()
    
    // Non-existent returns None
    opt := repo.Get(ctx, "non-existent")
    if opt.IsSome() {
        t.Error("Expected None for non-existent entity")
    }
    
    // After save returns Some
    entity := &Entity{ID: "test-1"}
    repo.Save(ctx, entity)
    
    opt = repo.Get(ctx, "test-1")
    if !opt.IsSome() {
        t.Error("Expected Some for existing entity")
    }
}
```

## Checklist

- [ ] Update imports to include functional, patterns, validation packages
- [ ] Replace nullable pointers with `Option[T]`
- [ ] Replace `(T, error)` returns with `Result[T]` where appropriate
- [ ] Update repository interfaces to use Option/Result
- [ ] Migrate validation to composable validators
- [ ] Add caching layer using CachedRepository if needed
- [ ] Use shared ExecutionMetrics type
- [ ] Update tests to handle Option/Result types
- [ ] Add property tests for new patterns
- [ ] Update documentation

## See Also

- [libs/go/src/functional/README.md](../libs/go/src/functional/README.md)
- [libs/go/src/patterns/README.md](../libs/go/src/patterns/README.md)
- [libs/go/src/validation/README.md](../libs/go/src/validation/README.md)
- [libs/go/src/fault/README.md](../libs/go/src/fault/README.md)
