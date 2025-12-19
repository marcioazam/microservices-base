# Changelog

All notable changes to the Go library collection.

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
