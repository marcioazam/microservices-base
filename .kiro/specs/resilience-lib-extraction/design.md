# Design Document: Resilience Library Extraction

## Overview

This design document specifies the extraction of reusable code from `platform/resilience-service` into shared libraries under `libs/go/`. The extraction creates a comprehensive suite of **100 generic Go libraries** that can be consumed by all services in the monorepo.

### Goals
- **Maximum code reuse** across all services
- **Full generics support** using Go 1.18+ type parameters
- **Zero breaking changes** to existing resilience-service
- **Clean interfaces** with minimal dependencies
- **Comprehensive testing** with property-based tests

### Non-Goals
- Service-specific business logic extraction
- Breaking API changes to resilience-service
- External dependency additions beyond standard library

## Architecture

### Library Organization

```
libs/go/
├── uuid/                    # UUID v7 generation (Req 1)
├── resilience/
│   ├── errors/              # Resilience error types (Req 2)
│   ├── domain/              # Domain types & interfaces (Req 3)
│   ├── circuitbreaker/      # Generic circuit breaker (Req 21, 83)
│   ├── ratelimit/           # Generic rate limiter (Req 22, 84)
│   ├── retry/               # Generic retry (Req 20)
│   ├── bulkhead/            # Generic bulkhead (Req 23)
│   ├── timeout/             # Generic timeout (Req 24)
│   └── policy/              # Policy serialization (Req 50)
├── health/                  # Health aggregation (Req 4)
├── grpc/
│   └── errors/              # gRPC error mapping (Req 5)
├── server/
│   └── shutdown/            # Graceful shutdown (Req 6)
├── testutil/                # Test utilities (Req 7, 25)
├── events/                  # Event builder (Req 8, 18)
├── codec/                   # Serialization (Req 26)
├── result/                  # Result[T] monad (Req 11)
├── option/                  # Option[T] monad (Req 12)
├── either/                  # Either[L,R] type (Req 33)
├── registry/                # Generic registry (Req 13)
├── pool/                    # Object pool (Req 14)
├── pipeline/                # Processing pipeline (Req 15)
├── validator/               # Generic validator (Req 16)
├── cache/                   # TTL cache (Req 17)
├── lru/                     # LRU cache (Req 37)
├── eventbus/                # Event bus (Req 18)
├── fsm/                     # State machine (Req 19)
├── slices/                  # Slice utilities (Req 27)
├── maps/                    # Map utilities (Req 28)
├── correlation/             # Correlation ID (Req 29)
├── async/                   # Async utilities (Req 30)
├── lazy/                    # Lazy initialization (Req 31)
├── tuple/                   # Tuple types (Req 32)
├── set/                     # Set type (Req 34)
├── queue/                   # Queue & Stack (Req 35)
├── pqueue/                  # Priority queue (Req 36)
├── semaphore/               # Semaphore (Req 38)
├── workerpool/              # Worker pool (Req 39)
├── debounce/                # Debouncer/Throttler (Req 40)
├── batch/                   # Batch processor (Req 41)
├── pubsub/                  # Pub/Sub (Req 42)
├── ringbuffer/              # Ring buffer (Req 43)
├── trie/                    # Trie (Req 44)
├── bloom/                   # Bloom filter (Req 45)
├── middleware/              # Middleware chain (Req 46)
├── observer/                # Observer pattern (Req 47)
├── builder/                 # Builder pattern (Req 48)
├── spec/                    # Specification pattern (Req 49)
├── saga/                    # Saga pattern (Req 51)
├── command/                 # Command pattern (Req 52)
├── strategy/                # Strategy pattern (Req 53)
├── chain/                   # Chain of responsibility (Req 54)
├── visitor/                 # Visitor pattern (Req 55)
├── memento/                 # Memento pattern (Req 56)
├── flyweight/               # Flyweight pattern (Req 57)
├── proxy/                   # Proxy pattern (Req 58)
├── decorator/               # Decorator pattern (Req 59)
├── composite/               # Composite pattern (Req 60)
├── iterator/                # Iterator (Req 61)
├── stream/                  # Stream processing (Req 62)
├── rx/                      # Reactive extensions (Req 63)
├── monad/                   # Monad transformers (Req 64)
├── lens/                    # Lens optics (Req 65)
├── prism/                   # Prism optics (Req 66)
├── free/                    # Free monad (Req 67)
├── validated/               # Validated applicative (Req 68)
├── writer/                  # Writer monad (Req 69)
├── reader/                  # Reader monad (Req 70)
├── state/                   # State monad (Req 71)
├── io/                      # IO monad (Req 72)
├── task/                    # Task/Future (Req 73)
├── future/                  # Promise/Future (Req 74)
├── channels/                # Channel utilities (Req 75)
├── syncmap/                 # Concurrent map (Req 76)
├── atomic/                  # Atomic values (Req 77)
├── once/                    # Once with result (Req 78)
├── waitgroup/               # WaitGroup with results (Req 79)
├── errgroup/                # ErrGroup with results (Req 80)
├── singleflight/            # Singleflight (Req 81)
├── backpressure/            # Backpressure (Req 82)
├── metrics/                 # Typed metrics (Req 85)
├── tracer/                  # Typed tracer (Req 86)
├── logger/                  # Typed logger (Req 87)
├── config/                  # Config loader (Req 88)
├── repository/              # Repository pattern (Req 89)
├── uow/                     # Unit of Work (Req 90)
├── eventsourcing/           # Event sourcing (Req 91)
├── cqrs/                    # CQRS (Req 92)
├── domain/                  # Domain events (Req 93)
├── outbox/                  # Outbox pattern (Req 94)
├── inbox/                   # Inbox pattern (Req 95)
├── query/                   # Query specification (Req 96)
├── pagination/              # Pagination (Req 97)
├── sort/                    # Sorting utilities (Req 98)
├── diff/                    # Diff utilities (Req 99)
└── merge/                   # Merge utilities (Req 100)
```

## Components and Interfaces

### Core Generic Types

#### Result[T] (libs/go/result/)

```go
package result

// Result represents a computation that may fail.
type Result[T any] struct {
    value T
    err   error
    isOk  bool
}

// Ok creates a successful result.
func Ok[T any](value T) Result[T]

// Err creates a failed result.
func Err[T any](err error) Result[T]

// IsOk returns true if the result is successful.
func (r Result[T]) IsOk() bool

// IsErr returns true if the result is an error.
func (r Result[T]) IsErr() bool

// Unwrap returns the value or panics if error.
func (r Result[T]) Unwrap() T

// UnwrapOr returns the value or the default.
func (r Result[T]) UnwrapOr(defaultVal T) T

// UnwrapErr returns the error or panics if ok.
func (r Result[T]) UnwrapErr() error

// Map applies fn to the value if ok.
func Map[T, U any](r Result[T], fn func(T) U) Result[U]

// FlatMap applies fn to the value if ok, flattening the result.
func FlatMap[T, U any](r Result[T], fn func(T) Result[U]) Result[U]

// AndThen chains computations.
func (r Result[T]) AndThen(fn func(T) Result[T]) Result[T]
```

#### Option[T] (libs/go/option/)

```go
package option

// Option represents an optional value.
type Option[T any] struct {
    value   T
    present bool
}

// Some creates a present option.
func Some[T any](value T) Option[T]

// None creates an absent option.
func None[T any]() Option[T]

// FromPtr creates an option from a pointer.
func FromPtr[T any](ptr *T) Option[T]

// IsSome returns true if value is present.
func (o Option[T]) IsSome() bool

// IsNone returns true if value is absent.
func (o Option[T]) IsNone() bool

// Unwrap returns the value or panics.
func (o Option[T]) Unwrap() T

// UnwrapOr returns the value or default.
func (o Option[T]) UnwrapOr(defaultVal T) T

// ToPtr returns a pointer to the value or nil.
func (o Option[T]) ToPtr() *T

// Map applies fn if present.
func Map[T, U any](o Option[T], fn func(T) U) Option[U]

// Filter returns None if predicate fails.
func (o Option[T]) Filter(predicate func(T) bool) Option[T]
```

#### Registry[K, V] (libs/go/registry/)

```go
package registry

import "sync"

// Registry is a thread-safe key-value store.
type Registry[K comparable, V any] struct {
    mu    sync.RWMutex
    items map[K]V
}

// New creates a new registry.
func New[K comparable, V any]() *Registry[K, V]

// Register stores a value.
func (r *Registry[K, V]) Register(key K, value V)

// Get retrieves a value.
func (r *Registry[K, V]) Get(key K) (V, bool)

// GetOrDefault retrieves a value or returns default.
func (r *Registry[K, V]) GetOrDefault(key K, defaultVal V) V

// Has checks if key exists.
func (r *Registry[K, V]) Has(key K) bool

// Unregister removes a value.
func (r *Registry[K, V]) Unregister(key K) bool

// Keys returns all keys.
func (r *Registry[K, V]) Keys() []K

// Values returns all values.
func (r *Registry[K, V]) Values() []V

// ForEach iterates over all entries.
func (r *Registry[K, V]) ForEach(fn func(K, V))

// Clear removes all entries.
func (r *Registry[K, V]) Clear()

// Len returns the number of entries.
func (r *Registry[K, V]) Len() int
```

### Resilience Patterns

#### CircuitBreaker[T] (libs/go/resilience/circuitbreaker/)

```go
package circuitbreaker

import (
    "context"
    "time"
)

// CircuitState represents the circuit breaker state.
type CircuitState int

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)

// Config holds circuit breaker configuration.
type Config struct {
    FailureThreshold int
    SuccessThreshold int
    Timeout          time.Duration
}

// CircuitBreaker protects operations with circuit breaker pattern.
type CircuitBreaker[T any] struct {
    // internal fields
}

// New creates a new circuit breaker.
func New[T any](name string, opts ...Option) *CircuitBreaker[T]

// Execute runs the operation with circuit breaker protection.
func (cb *CircuitBreaker[T]) Execute(ctx context.Context, op func() (T, error)) (T, error)

// State returns the current circuit state.
func (cb *CircuitBreaker[T]) State() CircuitState

// Reset forces the circuit to closed state.
func (cb *CircuitBreaker[T]) Reset()
```

#### RateLimiter[K] (libs/go/resilience/ratelimit/)

```go
package ratelimit

import (
    "context"
    "time"
)

// Decision represents a rate limit decision.
type Decision struct {
    Allowed    bool
    Remaining  int
    Limit      int
    ResetAt    time.Time
    RetryAfter time.Duration
}

// Headers contains rate limit response headers.
type Headers struct {
    Limit     int
    Remaining int
    Reset     int64
}

// RateLimiter controls request throughput.
type RateLimiter[K comparable] interface {
    Allow(ctx context.Context, key K) (Decision, error)
    Headers(ctx context.Context, key K) (Headers, error)
}

// NewTokenBucket creates a token bucket rate limiter.
func NewTokenBucket[K comparable](capacity int, refillRate float64) RateLimiter[K]

// NewSlidingWindow creates a sliding window rate limiter.
func NewSlidingWindow[K comparable](limit int, window time.Duration) RateLimiter[K]
```

#### Retry[T] (libs/go/resilience/retry/)

```go
package retry

import (
    "context"
    "time"
)

// Option configures retry behavior.
type Option func(*config)

// WithMaxAttempts sets maximum retry attempts.
func WithMaxAttempts(n int) Option

// WithBaseDelay sets base delay between retries.
func WithBaseDelay(d time.Duration) Option

// WithMaxDelay sets maximum delay between retries.
func WithMaxDelay(d time.Duration) Option

// WithMultiplier sets exponential backoff multiplier.
func WithMultiplier(m float64) Option

// WithJitter sets jitter percentage.
func WithJitter(percent float64) Option

// WithRetryIf sets retry predicate.
func WithRetryIf(predicate func(error) bool) Option

// Retry executes operation with retry policy.
func Retry[T any](ctx context.Context, op func() (T, error), opts ...Option) (T, error)
```

### Functional Utilities

#### Stream[T] (libs/go/stream/)

```go
package stream

// Stream represents a lazy sequence of elements.
type Stream[T any] struct {
    // internal iterator
}

// Of creates a stream from values.
func Of[T any](items ...T) Stream[T]

// FromSlice creates a stream from a slice.
func FromSlice[T any](items []T) Stream[T]

// Map transforms elements.
func (s Stream[T]) Map(fn func(T) T) Stream[T]

// MapTo transforms elements to different type.
func MapTo[T, U any](s Stream[T], fn func(T) U) Stream[U]

// Filter keeps elements matching predicate.
func (s Stream[T]) Filter(predicate func(T) bool) Stream[T]

// FlatMap transforms and flattens.
func FlatMap[T, U any](s Stream[T], fn func(T) Stream[U]) Stream[U]

// Reduce folds elements.
func (s Stream[T]) Reduce(initial T, fn func(T, T) T) T

// Collect materializes the stream.
func (s Stream[T]) Collect() []T

// FindFirst returns first element.
func (s Stream[T]) FindFirst() Option[T]

// AnyMatch checks if any element matches.
func (s Stream[T]) AnyMatch(predicate func(T) bool) bool

// AllMatch checks if all elements match.
func (s Stream[T]) AllMatch(predicate func(T) bool) bool

// Count returns element count.
func (s Stream[T]) Count() int

// Sorted sorts elements.
func (s Stream[T]) Sorted(less func(T, T) bool) Stream[T]

// Distinct removes duplicates (requires comparable).
func Distinct[T comparable](s Stream[T]) Stream[T]

// Limit takes first n elements.
func (s Stream[T]) Limit(n int) Stream[T]

// GroupBy groups elements by key.
func GroupBy[T any, K comparable](s Stream[T], keyFn func(T) K) map[K][]T
```

#### Lens[S, A] (libs/go/lens/)

```go
package lens

// Lens provides access to nested immutable structures.
type Lens[S, A any] struct {
    get func(S) A
    set func(S, A) S
}

// NewLens creates a lens from get and set functions.
func NewLens[S, A any](get func(S) A, set func(S, A) S) Lens[S, A]

// Get retrieves the focused value.
func (l Lens[S, A]) Get(source S) A

// Set returns a new structure with the focused value replaced.
func (l Lens[S, A]) Set(source S, value A) S

// Modify applies a function to the focused value.
func (l Lens[S, A]) Modify(source S, fn func(A) A) S

// Compose creates a lens focusing deeper.
func Compose[S, A, B any](outer Lens[S, A], inner Lens[A, B]) Lens[S, B]
```

## Data Models

### UUID v7 Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         unix_ts_ms (32 bits)                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          unix_ts_ms (16 bits) |  ver  |   rand_a (12 bits)    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|var|                       rand_b (62 bits)                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       rand_b (continued)                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Resilience Policy Schema

```go
type ResiliencePolicy struct {
    Name           string                `json:"name" yaml:"name"`
    Version        int                   `json:"version" yaml:"version"`
    CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
    Retry          *RetryConfig          `json:"retry,omitempty"`
    Timeout        *TimeoutConfig        `json:"timeout,omitempty"`
    RateLimit      *RateLimitConfig      `json:"rate_limit,omitempty"`
    Bulkhead       *BulkheadConfig       `json:"bulkhead,omitempty"`
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: UUID v7 Format Compliance
*For any* generated UUID v7, the string SHALL be 36 characters with hyphens at positions 8, 13, 18, 23, version nibble SHALL be 7, and variant bits SHALL be 10xx.
**Validates: Requirements 1.1, 1.4**

### Property 2: UUID v7 Timestamp Round-Trip
*For any* generated UUID v7, parsing the timestamp and comparing to generation time SHALL differ by at most 1 millisecond.
**Validates: Requirements 1.2, 1.5**

### Property 3: UUID v7 Validation Consistency
*For any* string, IsValidUUIDv7() SHALL return true if and only if the string matches UUID v7 format with correct version and variant bits.
**Validates: Requirements 1.3**

### Property 4: Error String Contains Required Fields
*For any* ResilienceError with non-empty service, Error() SHALL return a string containing the error code, service name, and message.
**Validates: Requirements 2.6**

### Property 5: Error Unwrap Returns Cause
*For any* ResilienceError with a non-nil Cause, Unwrap() SHALL return that exact cause error.
**Validates: Requirements 2.7**

### Property 6: Domain Type JSON Round-Trip
*For any* valid domain type (CircuitBreakerConfig, RetryConfig, etc.), JSON marshal followed by unmarshal SHALL produce an equivalent value.
**Validates: Requirements 3.8, 3.9, 9.1, 9.2, 9.3**

### Property 7: Health Status Aggregation Returns Worst
*For any* non-empty list of health statuses, aggregation SHALL return the worst status where Unhealthy > Degraded > Healthy.
**Validates: Requirements 4.2**

### Property 8: Result Map Preserves Structure
*For any* Result[T] and function fn, Map(result, fn) SHALL return Ok(fn(value)) if result is Ok, or Err(error) if result is Err.
**Validates: Requirements 11.6**

### Property 9: Result FlatMap Monad Law
*For any* Result[T], FlatMap with identity function SHALL return equivalent result (left identity), and FlatMap SHALL be associative.
**Validates: Requirements 11.7**

### Property 10: Option Map Preserves Structure
*For any* Option[T] and function fn, Map(option, fn) SHALL return Some(fn(value)) if option is Some, or None if option is None.
**Validates: Requirements 12.7**

### Property 11: Option Pointer Round-Trip
*For any* non-nil pointer ptr, FromPtr(ptr).ToPtr() SHALL return a pointer to an equal value. For nil pointer, FromPtr(nil).ToPtr() SHALL return nil.
**Validates: Requirements 12.9, 12.10**

### Property 12: Slice Map Preserves Length
*For any* slice and function fn, Map(slice, fn) SHALL return a slice of the same length with fn applied to each element.
**Validates: Requirements 27.1**

### Property 13: Slice Filter Subset Property
*For any* slice and predicate, Filter(slice, predicate) SHALL return a slice where all elements satisfy the predicate and all satisfying elements from original are present.
**Validates: Requirements 27.2**

### Property 14: Slice Chunk-Flatten Identity
*For any* slice and positive chunk size, Flatten(Chunk(slice, size)) SHALL equal the original slice.
**Validates: Requirements 27.9, 27.10**

### Property 15: Set Union Contains All
*For any* two sets A and B, Union(A, B) SHALL contain all elements from A and all elements from B.
**Validates: Requirements 34.8**

### Property 16: Set Intersection Contains Common
*For any* two sets A and B, Intersection(A, B) SHALL contain only elements present in both A and B.
**Validates: Requirements 34.9**

### Property 17: Set Difference Excludes Second
*For any* two sets A and B, Difference(A, B) SHALL contain elements in A that are not in B.
**Validates: Requirements 34.10**

### Property 18: Stream Collect Materializes All
*For any* stream created from slice, Collect() SHALL return a slice with all elements in order.
**Validates: Requirements 62.7**

### Property 19: Stream GroupBy Partitions Correctly
*For any* stream and key function, GroupBy SHALL partition elements such that all elements with same key are in same group.
**Validates: Requirements 62.15**

### Property 20: Lens Get-Set Identity
*For any* lens, source, and value, Get(Set(source, value)) SHALL equal value.
**Validates: Requirements 65.5**

### Property 21: Lens Set-Get Identity
*For any* lens and source, Set(source, Get(source)) SHALL equal source.
**Validates: Requirements 65.5**

### Property 22: Diff-Patch Round-Trip
*For any* two slices old and new, Patch(old, Diff(old, new)) SHALL equal new.
**Validates: Requirements 99.1, 99.2**

### Property 23: Registry Thread-Safety
*For any* concurrent operations on Registry, all operations SHALL complete without data races and maintain consistency.
**Validates: Requirements 13.11**

### Property 24: Circuit Breaker State Transitions
*For any* circuit breaker, state transitions SHALL follow: Closed→Open (on failure threshold), Open→HalfOpen (on timeout), HalfOpen→Closed (on success threshold), HalfOpen→Open (on failure).
**Validates: Requirements 21.4, 21.5, 21.6, 21.7**

## Error Handling

### Error Types

All libraries use the Result[T] pattern for fallible operations:

```go
// Preferred: Return Result[T]
func Parse[T any](data []byte) Result[T]

// Alternative: Return (T, error) for Go idiom compatibility
func ParseWithError[T any](data []byte) (T, error)
```

### Error Codes

Resilience errors use typed error codes:

```go
type ErrorCode string

const (
    ErrCircuitOpen        ErrorCode = "CIRCUIT_OPEN"
    ErrRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
    ErrTimeout            ErrorCode = "TIMEOUT"
    ErrBulkheadFull       ErrorCode = "BULKHEAD_FULL"
    ErrRetryExhausted     ErrorCode = "RETRY_EXHAUSTED"
    ErrInvalidPolicy      ErrorCode = "INVALID_POLICY"
)
```

## Testing Strategy

### Dual Testing Approach

1. **Unit Tests**: Verify specific examples and edge cases
2. **Property-Based Tests**: Verify universal properties across all inputs

### Property-Based Testing Framework

All libraries use `github.com/leanovate/gopter` for property-based testing with minimum 100 iterations per property.

### Test Annotation Format

```go
// **Feature: resilience-lib-extraction, Property 1: UUID v7 Format Compliance**
// **Validates: Requirements 1.1, 1.4**
func TestUUIDv7FormatCompliance(t *testing.T) {
    properties := gopter.NewProperties(gopter.DefaultTestParameters())
    properties.Property("generated UUID v7 is RFC 9562 compliant", prop.ForAll(
        func() bool {
            id := uuid.GenerateEventID()
            return uuid.IsValidUUIDv7(id)
        },
    ))
    properties.TestingRun(t)
}
```

### Test Coverage Requirements

- All public functions must have unit tests
- All correctness properties must have property-based tests
- Round-trip properties for all serialization
- Thread-safety tests for concurrent types
