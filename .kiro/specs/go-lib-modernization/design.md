# Design Document

## Overview

This design document specifies the modernization of the `libs/go` shared library collection to achieve state-of-the-art Go 1.25 standards. The modernization consolidates 45+ micro-modules into ~12 cohesive domain modules, separates source code from tests, eliminates redundancies, and adopts Go 1.25 features including generic type aliases, testing/synctest, and improved error handling.

## Architecture

### Current State

```
libs/go/
├── collections/     # 7 separate modules (lru, maps, pqueue, queue, set, slices, sort)
├── concurrency/     # 8 separate modules (async, atomic, channels, errgroup, once, pool, syncmap, waitgroup)
├── events/          # 3 separate modules (builder, eventbus, pubsub)
├── functional/      # 8 separate modules (either, iterator, lazy, option, pipeline, result, stream, tuple)
├── grpc/            # 1 module (errors)
├── optics/          # 2 separate modules (lens, prism)
├── patterns/        # 2 separate modules (registry, spec)
├── resilience/      # 7 separate modules (bulkhead, circuitbreaker, domain, errors, ratelimit, retry, timeout)
├── server/          # 3 separate modules (health, shutdown, tracing)
├── testing/         # 1 module (testutil)
├── utils/           # 9 separate modules (audit, cache, codec, diff, error, merge, uuid, validated, validator)
└── go.work          # Workspace file listing 45+ modules
```

### Target State

```
libs/go/
├── src/                          # Source code
│   ├── collections/              # Single module for all collections
│   │   ├── go.mod
│   │   ├── cache.go              # Unified cache (TTL + LRU)
│   │   ├── iterator.go           # Generic iterator interface
│   │   ├── lru.go
│   │   ├── maps.go
│   │   ├── pqueue.go
│   │   ├── queue.go
│   │   ├── set.go
│   │   ├── slices.go
│   │   └── sort.go
│   ├── concurrency/              # Single module for concurrency
│   │   ├── go.mod
│   │   ├── async.go              # Future[T] and parallel execution
│   │   ├── atomic.go
│   │   ├── channels.go
│   │   ├── errgroup.go
│   │   ├── pool.go               # Object pool
│   │   ├── syncmap.go
│   │   └── waitgroup.go
│   ├── events/                   # Unified event system
│   │   ├── go.mod
│   │   ├── bus.go                # Consolidated eventbus + pubsub
│   │   ├── builder.go
│   │   └── router.go             # Event filtering and routing
│   ├── functional/               # Unified functional types
│   │   ├── go.mod
│   │   ├── either.go
│   │   ├── functor.go            # Generic Functor interface
│   │   ├── iterator.go
│   │   ├── lazy.go
│   │   ├── option.go
│   │   ├── pipeline.go
│   │   ├── result.go
│   │   ├── stream.go
│   │   └── tuple.go
│   ├── grpc/                     # gRPC utilities
│   │   ├── go.mod
│   │   ├── errors.go
│   │   └── interceptors.go       # Error conversion interceptors
│   ├── optics/                   # Functional optics
│   │   ├── go.mod
│   │   ├── lens.go
│   │   └── prism.go
│   ├── patterns/                 # Design patterns
│   │   ├── go.mod
│   │   ├── registry.go
│   │   └── spec.go
│   ├── resilience/               # Fault tolerance
│   │   ├── go.mod
│   │   ├── bulkhead.go
│   │   ├── circuitbreaker.go
│   │   ├── config.go             # Unified configuration
│   │   ├── errors.go             # Centralized errors
│   │   ├── ratelimit.go
│   │   ├── retry.go
│   │   └── timeout.go
│   ├── server/                   # Server utilities
│   │   ├── go.mod
│   │   ├── health.go
│   │   ├── shutdown.go
│   │   └── tracing.go
│   ├── utils/                    # General utilities
│   │   ├── go.mod
│   │   ├── audit.go
│   │   ├── codec.go
│   │   ├── diff.go
│   │   ├── error.go              # HTTP error responses
│   │   ├── merge.go
│   │   ├── uuid.go
│   │   └── validation.go         # Consolidated validator + validated
│   └── testing/                  # Test utilities
│       ├── go.mod
│       ├── generators.go
│       ├── helpers.go
│       ├── mock_emitter.go
│       └── property.go           # Property test framework
├── tests/                        # Test code (mirrors src structure)
│   ├── collections/
│   │   ├── cache_test.go
│   │   ├── cache_property_test.go
│   │   ├── lru_test.go
│   │   └── ...
│   ├── functional/
│   │   ├── option_test.go
│   │   ├── result_test.go
│   │   ├── roundtrip_property_test.go
│   │   └── ...
│   └── ...
├── go.work                       # Workspace for 12 modules
└── README.md
```

## Components and Interfaces

### Unified Functor Interface

```go
// Functor represents types that can be mapped over.
type Functor[F any, A any] interface {
    // Map applies a function to the wrapped value.
    Map(fn func(A) any) F
}

// Mappable is a type constraint for types that support Map operations.
type Mappable[T any] interface {
    Option[T] | Result[T] | Either[any, T]
}
```

### Unified Cache Interface

```go
// Cache provides a unified interface for caching with different eviction strategies.
type Cache[K comparable, V any] interface {
    Get(key K) (V, bool)
    Set(key K, value V)
    Delete(key K)
    GetOrCompute(key K, compute func() V) V
    Contains(key K) bool
    Len() int
    Clear()
    Close()
}

// EvictionStrategy defines the cache eviction behavior.
type EvictionStrategy int

const (
    EvictionTTL EvictionStrategy = iota
    EvictionLRU
    EvictionLFU
)

// CacheConfig configures cache behavior.
type CacheConfig[K comparable, V any] struct {
    Strategy    EvictionStrategy
    Capacity    int
    DefaultTTL  time.Duration
    OnEvict     func(K, V)
    Shards      int  // For concurrent access optimization
}
```

### Unified Iterator Interface

```go
// Iterator provides lazy iteration over collections.
type Iterator[T any] interface {
    Next() (T, bool)
    HasNext() bool
}

// IteratorOps provides functional operations on iterators.
type IteratorOps[T any] struct {
    iter Iterator[T]
}

func (i IteratorOps[T]) Map[U any](fn func(T) U) IteratorOps[U]
func (i IteratorOps[T]) Filter(predicate func(T) bool) IteratorOps[T]
func (i IteratorOps[T]) Reduce[U any](initial U, fn func(U, T) U) U
func (i IteratorOps[T]) ForEach(fn func(T))
func (i IteratorOps[T]) Collect() []T
```

### Unified Validation Framework

```go
// Validation represents a validation result that accumulates errors.
type Validation[E, A any] struct {
    value  A
    errors []E
    valid  bool
}

// Validator provides composable validation rules.
type Validator[T any] struct {
    rules []Rule[T]
}

// Rule represents a single validation rule.
type Rule[T any] struct {
    Name    string
    Check   func(T) bool
    Message string
    Path    string  // For nested field tracking
}

// Combine merges validators, accumulating all errors.
func Combine[E, A, B, C any](
    va Validation[E, A],
    vb Validation[E, B],
    fn func(A, B) C,
) Validation[E, C]
```

### Unified Event System

```go
// EventBus provides unified event publishing and subscription.
type EventBus[E any] interface {
    Publish(ctx context.Context, event E) error
    PublishAsync(ctx context.Context, event E) <-chan error
    Subscribe(handler func(E) error, filters ...Filter[E]) Subscription
    Unsubscribe(sub Subscription)
}

// Filter defines event filtering criteria.
type Filter[E any] func(E) bool

// Subscription represents an active subscription.
type Subscription interface {
    ID() string
    Cancel()
}

// DeliveryConfig configures event delivery behavior.
type DeliveryConfig struct {
    Mode        DeliveryMode  // Sync or Async
    RetryPolicy *RetryConfig  // Optional retry on failure
    Timeout     time.Duration
}
```

### Unified Resilience Configuration

```go
// ResilienceConfig provides unified configuration for all resilience patterns.
type ResilienceConfig struct {
    CircuitBreaker *CircuitBreakerConfig
    Retry          *RetryConfig
    RateLimit      *RateLimitConfig
    Bulkhead       *BulkheadConfig
    Timeout        *TimeoutConfig
}

// Validate checks all configuration values.
func (c *ResilienceConfig) Validate() error {
    var errs []error
    if c.CircuitBreaker != nil {
        if err := c.CircuitBreaker.Validate(); err != nil {
            errs = append(errs, err)
        }
    }
    // ... validate other configs
    if len(errs) > 0 {
        return NewInvalidPolicyError("resilience", "config", errs)
    }
    return nil
}
```

## Data Models

### Functional Type Conversions

```go
// Either to Result conversion
func EitherToResult[T any](e Either[error, T]) Result[T] {
    if e.IsRight() {
        return Ok(e.RightValue())
    }
    return Err[T](e.LeftValue())
}

// Result to Either conversion
func ResultToEither[T any](r Result[T]) Either[error, T] {
    if r.IsOk() {
        return Right[error, T](r.Unwrap())
    }
    return Left[error, T](r.UnwrapErr())
}
```

### Error Hierarchy

```go
// ResilienceError is the base error type.
type ResilienceError struct {
    Code          ErrorCode
    Service       string
    Message       string
    Cause         error
    CorrelationID string
    Metadata      map[string]string
}

// Specific error types embed ResilienceError
type CircuitOpenError struct {
    ResilienceError
    ResetAt time.Time
}

type RateLimitError struct {
    ResilienceError
    RetryAfter time.Duration
    Limit      int
    Remaining  int
}

// JSON serialization structure
type ErrorJSON struct {
    Code          string            `json:"code"`
    Service       string            `json:"service"`
    Message       string            `json:"message"`
    CorrelationID string            `json:"correlation_id"`
    Metadata      map[string]string `json:"metadata,omitempty"`
    Details       interface{}       `json:"details,omitempty"`
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Either-Result Round Trip

*For any* Either[error, T] value, converting to Result[T] and back to Either[error, T] SHALL produce an equivalent value.

```go
// For all e: Either[error, T]
// ResultToEither(EitherToResult(e)) == e
```

**Validates: Requirements 3.2**

### Property 2: Resilience Error Type Hierarchy

*For any* error generated by a resilience component (circuit breaker, rate limiter, bulkhead, retry, timeout), the error SHALL be an instance of ResilienceError or one of its subtypes.

```go
// For all err from resilience components
// errors.As(err, &ResilienceError{}) == true
```

**Validates: Requirements 4.2**

### Property 3: Error Type Checking Functions

*For any* CircuitOpenError, IsCircuitOpen SHALL return true. *For any* RateLimitError, IsRateLimitExceeded SHALL return true. And so on for all error types.

```go
// For all err: CircuitOpenError
// IsCircuitOpen(err) == true
// For all err: RateLimitError
// IsRateLimitExceeded(err) == true
```

**Validates: Requirements 4.4**

### Property 4: Error JSON Round Trip

*For any* ResilienceError, serializing to JSON and deserializing SHALL produce an equivalent error with the same Code, Service, Message, and CorrelationID.

```go
// For all err: ResilienceError
// Deserialize(Serialize(err)).Code == err.Code
// Deserialize(Serialize(err)).Message == err.Message
```

**Validates: Requirements 4.5**

### Property 5: Invalid Configuration Detection

*For any* configuration with invalid values (negative thresholds, zero timeouts, etc.), Validate() SHALL return an InvalidPolicyError.

```go
// For all config with invalid values
// config.Validate() returns InvalidPolicyError
```

**Validates: Requirements 5.4**

### Property 6: Collection Map Identity

*For any* collection and the identity function, Map(collection, identity) SHALL return an equivalent collection.

```go
// For all c: Collection[T]
// Map(c, func(x T) T { return x }) == c
```

**Validates: Requirements 6.2**

### Property 7: Collection IsEmpty Invariant

*For any* collection, IsEmpty() SHALL return true if and only if Len() equals zero.

```go
// For all c: Collection[T]
// c.IsEmpty() == (c.Len() == 0)
```

**Validates: Requirements 6.4**

### Property 8: Collection Contains After Add

*For any* collection and element, after Add(element), Contains(element) SHALL return true.

```go
// For all c: Collection[T], e: T
// c.Add(e); c.Contains(e) == true
```

**Validates: Requirements 6.4**

### Property 9: Validation Error Accumulation

*For any* input that fails N validation rules, the validation result SHALL contain exactly N errors.

```go
// For all input failing rules r1, r2, ..., rN
// len(Validate(input).Errors) == N
```

**Validates: Requirements 7.2**

### Property 10: Validator And Composition

*For any* two validators v1 and v2, And(v1, v2) SHALL fail if either v1 or v2 fails.

```go
// For all v1, v2: Validator[T], input: T
// And(v1, v2).Validate(input).IsValid() == (v1.Validate(input).IsValid() && v2.Validate(input).IsValid())
```

**Validates: Requirements 7.3**

### Property 11: Validation Path Tracking

*For any* nested validation error, the error path SHALL correctly identify the field location.

```go
// For all nested struct with invalid field at path "a.b.c"
// validation.Errors[0].Path == "a.b.c"
```

**Validates: Requirements 7.4**

### Property 12: Validation Result Exclusivity

*For any* validation result, exactly one of IsValid() or HasErrors() SHALL be true.

```go
// For all v: Validation[E, A]
// v.IsValid() XOR len(v.Errors) > 0
```

**Validates: Requirements 7.5**

### Property 13: TTL Cache Expiry

*For any* TTL cache item, after the TTL duration has elapsed, Get SHALL return (zero, false).

```go
// For all cache with TTL t, key k, value v
// cache.Set(k, v); sleep(t + epsilon); cache.Get(k) == (zero, false)
```

**Validates: Requirements 8.2**

### Property 14: LRU Cache Eviction Order

*For any* LRU cache at capacity, adding a new item SHALL evict the least recently used item.

```go
// For all cache at capacity with LRU item k
// cache.Set(newKey, newValue); cache.Contains(k) == false
```

**Validates: Requirements 8.2**

### Property 15: Cache GetOrCompute Behavior

*For any* cache key, GetOrCompute SHALL return the cached value if present, or compute and cache the value if absent.

```go
// For all cache, key k not in cache, compute fn
// cache.GetOrCompute(k, fn) == fn()
// cache.Get(k) == (fn(), true)
```

**Validates: Requirements 8.4**

### Property 16: Cache Eviction Callback

*For any* cache with eviction callback, when an item is evicted, the callback SHALL be called with the correct key and value.

```go
// For all cache with callback, evicted item (k, v)
// callback was called with (k, v)
```

**Validates: Requirements 8.5**

### Property 17: Seeded Test Reproducibility

*For any* property test with a fixed seed, running the test twice SHALL produce identical results.

```go
// For all seed s, property p
// RunWithSeed(p, s) == RunWithSeed(p, s)
```

**Validates: Requirements 9.2**

### Property 18: Future Context Cancellation

*For any* Future with context, cancelling the context SHALL cause WaitContext to return context.Canceled.

```go
// For all future f with context ctx
// cancel(ctx); f.WaitContext(ctx) returns context.Canceled
```

**Validates: Requirements 9.4**

### Property 19: Future Result Integration

*For any* completed Future, Result() SHALL return Ok if the operation succeeded, Err if it failed.

```go
// For all future f that completed with (v, nil)
// f.Result().IsOk() == true && f.Result().Unwrap() == v
// For all future f that completed with (_, err)
// f.Result().IsErr() == true && f.Result().UnwrapErr() == err
```

**Validates: Requirements 9.5**

### Property 20: Generator Validity

*For any* generator for type T, all generated values SHALL be valid instances of T (non-nil for pointer types, within bounds for numeric types).

```go
// For all gen: Generator[T], sample from gen
// isValid(sample) == true
```

**Validates: Requirements 10.1**

### Property 21: gRPC Error Round Trip

*For any* ResilienceError, converting to gRPC status and back SHALL preserve the error code and message.

```go
// For all err: ResilienceError
// FromGRPCError(ToGRPCError(err)).Code == err.Code
```

**Validates: Requirements 14.1, 14.2**

### Property 22: Event Sync Delivery

*For any* event published with sync delivery, the handler SHALL complete before Publish returns.

```go
// For all event e, sync handler h
// handlerCompleted := false
// bus.Subscribe(func(e) { handlerCompleted = true })
// bus.Publish(e)
// handlerCompleted == true
```

**Validates: Requirements 15.2**

### Property 23: Event Filtering

*For any* subscription with filter, only events matching the filter SHALL be delivered.

```go
// For all filter f, event e
// if f(e) == false, handler is not called
// if f(e) == true, handler is called
```

**Validates: Requirements 15.4**

### Property 24: Event Retry on Failure

*For any* failed event delivery with retry policy, the event SHALL be retried according to the policy.

```go
// For all event e, handler that fails N times, retry policy with maxAttempts M
// handler is called min(N+1, M) times
```

**Validates: Requirements 15.5**

## Error Handling

### Centralized Error Strategy

All resilience components use the centralized error types from `resilience/errors`:

1. **CircuitOpenError**: Circuit breaker is open, includes ResetAt time
2. **RateLimitError**: Rate limit exceeded, includes RetryAfter duration
3. **TimeoutError**: Operation timed out, includes Timeout duration
4. **BulkheadFullError**: Bulkhead at capacity, includes partition info
5. **RetryExhaustedError**: All retries failed, includes attempt count and last error
6. **InvalidPolicyError**: Configuration validation failed, includes field and reason

### Error Conversion

```go
// HTTP error conversion
func ToHTTPStatus(err error) int {
    switch {
    case IsCircuitOpen(err):
        return http.StatusServiceUnavailable
    case IsRateLimitExceeded(err):
        return http.StatusTooManyRequests
    case IsTimeout(err):
        return http.StatusGatewayTimeout
    case IsBulkheadFull(err):
        return http.StatusServiceUnavailable
    default:
        return http.StatusInternalServerError
    }
}

// gRPC error conversion
func ToGRPCCode(err error) codes.Code {
    switch {
    case IsCircuitOpen(err):
        return codes.Unavailable
    case IsRateLimitExceeded(err):
        return codes.ResourceExhausted
    case IsTimeout(err):
        return codes.DeadlineExceeded
    case IsBulkheadFull(err):
        return codes.ResourceExhausted
    default:
        return codes.Internal
    }
}
```

## Reusable Patterns (State-of-the-Art 2025)

### Pattern 1: Generic Middleware Chain

Composable middleware for HTTP/gRPC with type-safe context propagation.

```go
// Middleware represents a generic middleware function.
type Middleware[Ctx, Req, Res any] func(Handler[Ctx, Req, Res]) Handler[Ctx, Req, Res]

// Handler processes requests.
type Handler[Ctx, Req, Res any] func(ctx Ctx, req Req) (Res, error)

// Chain composes middlewares right-to-left.
func Chain[Ctx, Req, Res any](middlewares ...Middleware[Ctx, Req, Res]) Middleware[Ctx, Req, Res] {
    return func(next Handler[Ctx, Req, Res]) Handler[Ctx, Req, Res] {
        for i := len(middlewares) - 1; i >= 0; i-- {
            next = middlewares[i](next)
        }
        return next
    }
}

// WithLogging creates logging middleware.
func WithLogging[Ctx, Req, Res any](logger Logger) Middleware[Ctx, Req, Res] {
    return func(next Handler[Ctx, Req, Res]) Handler[Ctx, Req, Res] {
        return func(ctx Ctx, req Req) (Res, error) {
            start := time.Now()
            res, err := next(ctx, req)
            logger.Info("request", "duration", time.Since(start), "error", err)
            return res, err
        }
    }
}
```

**Validates: Requirements 5.2, 14.4**

### Pattern 2: Generic Retry with Exponential Backoff

Full jitter backoff following AWS best practices.

```go
// RetryConfig configures retry behavior.
type RetryConfig struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Jitter      JitterStrategy
}

// JitterStrategy defines jitter calculation.
type JitterStrategy int

const (
    NoJitter JitterStrategy = iota
    FullJitter
    EqualJitter
    DecorrelatedJitter
)

// Retry executes fn with exponential backoff.
func Retry[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) Result[T] {
    var lastErr error
    delay := cfg.BaseDelay
    
    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        result, err := fn()
        if err == nil {
            return Ok(result)
        }
        if !IsRetryable(err) {
            return Err[T](err)
        }
        lastErr = err
        
        sleep := calculateBackoff(delay, cfg.MaxDelay, cfg.Jitter)
        select {
        case <-ctx.Done():
            return Err[T](ctx.Err())
        case <-time.After(sleep):
        }
        delay = min(delay*2, cfg.MaxDelay)
    }
    return Err[T](NewRetryExhaustedError(cfg.MaxAttempts, lastErr))
}

func calculateBackoff(delay, maxDelay time.Duration, jitter JitterStrategy) time.Duration {
    switch jitter {
    case FullJitter:
        return time.Duration(rand.Int63n(int64(min(delay, maxDelay))))
    case EqualJitter:
        half := delay / 2
        return half + time.Duration(rand.Int63n(int64(half)))
    default:
        return delay
    }
}
```

**Validates: Requirements 5.3, 15.5**

### Pattern 3: Generic Worker Pool

Context-aware worker pool with backpressure.

```go
// WorkerPool manages concurrent task execution.
type WorkerPool[T, R any] struct {
    workers   int
    taskQueue chan Task[T, R]
    results   chan Result[R]
    wg        sync.WaitGroup
}

// Task represents work to be done.
type Task[T, R any] struct {
    Input   T
    Execute func(context.Context, T) (R, error)
}

// NewWorkerPool creates a bounded worker pool.
func NewWorkerPool[T, R any](workers, queueSize int) *WorkerPool[T, R] {
    return &WorkerPool[T, R]{
        workers:   workers,
        taskQueue: make(chan Task[T, R], queueSize),
        results:   make(chan Result[R], queueSize),
    }
}

// Start launches workers.
func (p *WorkerPool[T, R]) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(ctx)
    }
}

func (p *WorkerPool[T, R]) worker(ctx context.Context) {
    defer p.wg.Done()
    for {
        select {
        case <-ctx.Done():
            return
        case task, ok := <-p.taskQueue:
            if !ok {
                return
            }
            result, err := task.Execute(ctx, task.Input)
            if err != nil {
                p.results <- Err[R](err)
            } else {
                p.results <- Ok(result)
            }
        }
    }
}
```

**Validates: Requirements 9.3, 9.4**

### Pattern 4: Generic State Machine

Type-safe FSM with transition validation.

```go
// State represents a state in the machine.
type State string

// Event triggers state transitions.
type Event string

// Transition defines a valid state change.
type Transition[S ~string, E ~string] struct {
    From   S
    Event  E
    To     S
    Guard  func() bool
    Action func() error
}

// StateMachine manages state transitions.
type StateMachine[S ~string, E ~string, D any] struct {
    current     S
    data        D
    transitions map[S]map[E]Transition[S, E]
    mu          sync.RWMutex
}

// Fire attempts a state transition.
func (sm *StateMachine[S, E, D]) Fire(event E) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    events, ok := sm.transitions[sm.current]
    if !ok {
        return NewInvalidTransitionError(sm.current, event)
    }
    
    trans, ok := events[event]
    if !ok {
        return NewInvalidTransitionError(sm.current, event)
    }
    
    if trans.Guard != nil && !trans.Guard() {
        return NewGuardFailedError(sm.current, event)
    }
    
    if trans.Action != nil {
        if err := trans.Action(); err != nil {
            return err
        }
    }
    
    sm.current = trans.To
    return nil
}
```

**Validates: Requirements 15.4**

### Pattern 5: Generic Iterator with Lazy Evaluation

Go 1.23+ compatible iterator pattern.

```go
// Iterator provides lazy iteration.
type Iterator[T any] func(yield func(T) bool)

// Map transforms iterator elements.
func Map[T, U any](iter Iterator[T], fn func(T) U) Iterator[U] {
    return func(yield func(U) bool) {
        iter(func(t T) bool {
            return yield(fn(t))
        })
    }
}

// Filter keeps elements matching predicate.
func Filter[T any](iter Iterator[T], pred func(T) bool) Iterator[T] {
    return func(yield func(T) bool) {
        iter(func(t T) bool {
            if pred(t) {
                return yield(t)
            }
            return true
        })
    }
}

// Reduce accumulates iterator values.
func Reduce[T, U any](iter Iterator[T], initial U, fn func(U, T) U) U {
    acc := initial
    iter(func(t T) bool {
        acc = fn(acc, t)
        return true
    })
    return acc
}

// Take limits iterator to n elements.
func Take[T any](iter Iterator[T], n int) Iterator[T] {
    return func(yield func(T) bool) {
        count := 0
        iter(func(t T) bool {
            if count >= n {
                return false
            }
            count++
            return yield(t)
        })
    }
}

// Collect materializes iterator to slice.
func Collect[T any](iter Iterator[T]) []T {
    var result []T
    iter(func(t T) bool {
        result = append(result, t)
        return true
    })
    return result
}
```

**Validates: Requirements 6.2, 6.3**

### Pattern 6: Generic Repository

Type-safe data access with specification pattern.

```go
// Entity constraint for repository items.
type Entity[ID comparable] interface {
    GetID() ID
}

// Specification defines query criteria.
type Specification[T any] interface {
    IsSatisfiedBy(T) bool
    And(Specification[T]) Specification[T]
    Or(Specification[T]) Specification[T]
    Not() Specification[T]
}

// Repository provides generic data access.
type Repository[T Entity[ID], ID comparable] interface {
    FindByID(ctx context.Context, id ID) (Option[T], error)
    FindAll(ctx context.Context) ([]T, error)
    FindBySpec(ctx context.Context, spec Specification[T]) ([]T, error)
    Save(ctx context.Context, entity T) error
    Delete(ctx context.Context, id ID) error
}

// BaseSpec implements Specification combinators.
type BaseSpec[T any] struct {
    predicate func(T) bool
}

func (s BaseSpec[T]) IsSatisfiedBy(t T) bool { return s.predicate(t) }

func (s BaseSpec[T]) And(other Specification[T]) Specification[T] {
    return BaseSpec[T]{func(t T) bool {
        return s.IsSatisfiedBy(t) && other.IsSatisfiedBy(t)
    }}
}

func (s BaseSpec[T]) Or(other Specification[T]) Specification[T] {
    return BaseSpec[T]{func(t T) bool {
        return s.IsSatisfiedBy(t) || other.IsSatisfiedBy(t)
    }}
}

func (s BaseSpec[T]) Not() Specification[T] {
    return BaseSpec[T]{func(t T) bool { return !s.IsSatisfiedBy(t) }}
}
```

**Validates: Requirements 12.2**

See `design-patterns.md` for additional patterns (Pagination, Health Check, Outbox, Builder, Saga).

## Testing Strategy

### Dual Testing Approach

The library uses both unit tests and property-based tests:

1. **Unit Tests**: Verify specific examples, edge cases, and error conditions
2. **Property Tests**: Verify universal properties across many generated inputs

### Property-Based Testing Framework

Using Go's testing/quick package enhanced with custom generators:

```go
// Property test configuration
const PropertyTestIterations = 100

// Property test annotation format
// Feature: go-lib-modernization, Property N: [property description]
// Validates: Requirements X.Y

func TestProperty_EitherResultRoundTrip(t *testing.T) {
    // Feature: go-lib-modernization, Property 1: Either-Result Round Trip
    // Validates: Requirements 3.2
    
    gen := EitherGen[error, int](ErrorGen(), IntGen(0, 1000))
    
    err := quick.Check(func(e Either[error, int]) bool {
        roundTripped := ResultToEither(EitherToResult(e))
        return e.IsRight() == roundTripped.IsRight() &&
               (e.IsLeft() || e.RightValue() == roundTripped.RightValue())
    }, &quick.Config{MaxCount: PropertyTestIterations})
    
    if err != nil {
        t.Errorf("Property failed: %v", err)
    }
}
```

### Test Organization

```
tests/
├── collections/
│   ├── cache_test.go           # Unit tests
│   ├── cache_property_test.go  # Property tests for cache
│   ├── lru_test.go
│   └── set_property_test.go
├── functional/
│   ├── option_test.go
│   ├── result_test.go
│   ├── either_test.go
│   └── roundtrip_property_test.go  # Property 1
├── resilience/
│   ├── circuitbreaker_test.go
│   ├── errors_test.go
│   ├── errors_property_test.go     # Properties 2, 3, 4
│   └── config_property_test.go     # Property 5
├── validation/
│   └── validation_property_test.go # Properties 9-12
└── events/
    └── bus_property_test.go        # Properties 22-24
```

### Synctest Integration

For deterministic concurrency testing (Go 1.25):

```go
import "testing/synctest"

func TestFuture_ContextCancellation(t *testing.T) {
    synctest.Run(func() {
        ctx, cancel := context.WithCancel(context.Background())
        
        f := GoContext(ctx, func(ctx context.Context) (int, error) {
            <-ctx.Done()
            return 0, ctx.Err()
        })
        
        cancel()
        
        _, err := f.WaitContext(ctx)
        if err != context.Canceled {
            t.Errorf("expected context.Canceled, got %v", err)
        }
    })
}
```

### Benchmark Requirements

Critical operations must have benchmarks:

```go
func BenchmarkCache_GetOrCompute(b *testing.B) {
    cache := NewCache[string, int](WithCapacity(1000))
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cache.GetOrCompute("key", func() int { return 42 })
    }
}

func BenchmarkCircuitBreaker_Execute(b *testing.B) {
    cb := NewCircuitBreaker[int]("test")
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cb.Execute(ctx, func() (int, error) { return 42, nil })
    }
}
```
