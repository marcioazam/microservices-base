# Changelog

All notable changes to the Go library collection.

## [3.0.0] - 2025-12-19

### Added

- **Go 1.25+ Features**
  - `errors.AsType[T]` generic error type assertion
  - Go 1.23+ iterator support (`iter.Seq`, `iter.Seq2`) for all collections
  - `All()` method on Set, Queue, PriorityQueue, LRUCache, Option, Result

- **LRU Cache Enhancements** (`src/collections/lru.go`)
  - `GetOrCompute` for lazy initialization
  - `Stats` tracking (hits, misses, evictions, expirations, hit rate)
  - `All()` iterator returning `iter.Seq2[K, V]`
  - `Collect()` returning slice of key-value pairs

- **Codec Enhancements** (`src/codec/`)
  - `TypedCodec[T]` interface for type-safe encoding/decoding
  - `EncodeResult`/`DecodeResult` returning `functional.Result`
  - `TypedJSONCodec[T]` and `TypedYAMLCodec[T]`

- **Validation Enhancements** (`src/validation/`)
  - `NestedField` for nested field path tracking
  - `ValidateAll` for error accumulation
  - `AddFieldError` convenience method

- **Functor Enhancements** (`src/functional/`)
  - `OptionToResult` and `ResultToOption` conversion functions
  - `Identity` and `Compose` functions for functor law testing

- **Property-Based Tests**
  - 11 comprehensive property tests covering all correctness properties
  - Minimum 100 iterations per property test
  - Tests for LRU, Codec, Validation, Errors, Functor, Iterator, Resilience

### Changed

- **BREAKING**: Removed `src/cache/` module - consolidated to `src/collections/`
- **BREAKING**: Removed `src/utils/codec.go` - consolidated to `src/codec/`
- **BREAKING**: Removed `src/utils/validation.go` - consolidated to `src/validation/`
- **BREAKING**: LRU Cache `Get` now returns `functional.Option[V]` instead of `(V, bool)`
- Updated all module paths to `github.com/authcorp/libs/go/src/`
- Minimum Go version: 1.25.5

### Removed

- `src/cache/` directory (use `src/collections/` for LRU cache)
- `src/utils/codec.go` (use `src/codec/`)
- `src/utils/validation.go` (use `src/validation/`)
- `tests/cache/` directory (tests consolidated to `tests/collections/`)

### Migration

See [MIGRATION.md](MIGRATION.md) for detailed migration instructions.

---

## [2.0.0] - 2025-12-19

### Added

- **Unified Functional Module** (`src/functional/`)
  - Consolidated Option, Result, Either, Iterator, Stream, Lazy, Pipeline, Tuple
  - Common Functor interface for all types
  - Type conversion functions (EitherToResult, ResultToEither)
  - Go 1.23+ iterator pattern support

- **Unified Resilience Module** (`src/resilience/`)
  - Centralized error types with ResilienceError base
  - Unified configuration with functional options pattern
  - Circuit breaker, retry, rate limit, bulkhead, timeout
  - JSON serialization for all error types

- **Unified Collections Module** (`src/collections/`)
  - Generic Set, Queue, Stack, PriorityQueue
  - LRU Cache with TTL and eviction callbacks
  - Unified Iterator interface

- **Unified Concurrency Module** (`src/concurrency/`)
  - Generic Future[T] with context support
  - WorkerPool[T, R] with backpressure
  - ErrGroup for coordinated goroutines

- **Unified Events Module** (`src/events/`)
  - Generic EventBus[E] with filtering
  - Sync and async delivery modes
  - Subscription management

- **Unified Testing Module** (`src/testing/`)
  - Property-based test generators for all types
  - Synctest integration helpers
  - Seeded reproducibility support

- **gRPC Integration** (`src/grpc/`)
  - Bidirectional error conversion
  - Unary and stream interceptors
  - Correlation ID support

### Changed

- **BREAKING**: Consolidated 45+ micro-modules into 12 domain modules
- **BREAKING**: All resilience errors now extend ResilienceError
- **BREAKING**: Configuration uses functional options pattern
- **BREAKING**: Source code moved to `src/`, tests to `tests/`
- Minimum Go version: 1.25

### Deprecated

- Old import paths (see MIGRATION.md for mapping)
- Individual micro-module packages

### Migration

See [MIGRATION.md](MIGRATION.md) for detailed migration instructions.

## [1.x.x] - Previous Releases

Legacy micro-module structure. See git history for details.
