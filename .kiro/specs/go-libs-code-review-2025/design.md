# Design: Go Libs Code Review & Redundancy Elimination 2025

## Architecture Decision

### Package Consolidation Strategy

Based on Go 2025 best practices (Google Style Guide, JetBrains Go Ecosystem Report):

```
libs/go/src/
├── codec/           # Single codec implementation
├── collections/     # All collection types (no subfolders with duplicates)
│   ├── lru.go      # Keep enhanced version
│   ├── set.go
│   ├── queue.go
│   ├── pqueue.go
│   ├── maps.go     # From maps/
│   ├── slices.go   # From slices/
│   └── sort.go     # From sort/
├── concurrency/     # Concurrency primitives
├── config/
├── domain/          # Domain value objects (keep uuid here, remove from utils)
├── errors/          # Error handling
├── events/          # Event system
├── fault/           # Fault tolerance (circuit breaker, retry, etc.)
├── functional/      # FP primitives (consolidate option/either/result)
│   ├── option.go   # Keep enhanced version
│   ├── either.go
│   ├── result.go
│   └── ...
├── grpc/            # gRPC utilities
├── http/            # HTTP utilities
├── idempotency/
├── observability/
├── optics/          # Lens/Prism (consolidate)
├── pagination/
├── patterns/        # Design patterns (consolidate)
├── security/
├── server/          # Server utilities (consolidate)
├── testing/
├── validation/
├── versioning/
└── workerpool/
```

### Removal Plan

#### Phase 1: Remove Duplicate Subfolders
```
DELETE: collections/lru/       → KEEP: collections/lru.go (enhanced)
DELETE: collections/queue/     → KEEP: collections/queue.go
DELETE: collections/set/       → KEEP: collections/set.go
DELETE: collections/pqueue/    → KEEP: collections/pqueue.go
MERGE:  collections/maps/      → INTO: collections/maps.go
MERGE:  collections/slices/    → INTO: collections/slices.go
MERGE:  collections/sort/      → INTO: collections/sort.go
```

#### Phase 2: Remove Cross-Package Duplicates
```
DELETE: utils/uuid/            → KEEP: domain/uuid.go
DELETE: utils/codec/           → KEEP: codec/codec.go
DELETE: utils/cache/           → KEEP: collections/lru.go
DELETE: utils/error/           → KEEP: errors/
DELETE: utils/validator/       → KEEP: validation/
```

#### Phase 3: Consolidate Functional Package
```
DELETE: functional/option/     → KEEP: functional/option.go
DELETE: functional/either/     → KEEP: functional/either.go
DELETE: functional/result/     → KEEP: functional/result.go
DELETE: functional/tuple/      → KEEP: functional/tuple.go
DELETE: functional/lazy/       → KEEP: functional/lazy.go
DELETE: functional/stream/     → KEEP: functional/stream.go
DELETE: functional/iterator/   → KEEP: functional/iterator.go
DELETE: functional/pipeline/   → KEEP: functional/pipeline.go
```

#### Phase 4: Consolidate Other Packages
```
DELETE: optics/lens/           → KEEP: optics/lens.go
DELETE: optics/prism/          → MERGE INTO: optics/prism.go
DELETE: patterns/registry/     → KEEP: patterns/registry.go
DELETE: patterns/spec/         → KEEP: patterns/spec.go
DELETE: server/health/         → KEEP: server/health.go
DELETE: server/shutdown/       → KEEP: server/shutdown.go
DELETE: server/tracing/        → MERGE INTO: observability/
DELETE: grpc/errors_sub/       → KEEP: grpc/errors.go
DELETE: events/builder/        → MERGE INTO: events/
DELETE: events/eventbus/       → KEEP: events/eventbus.go
DELETE: events/pubsub/         → MERGE INTO: events/
DELETE: concurrency/pool_sub/  → KEEP: concurrency/pool.go
```

### Go 2025 Patterns to Apply

#### 1. Modern Generics (Go 1.21+)
```go
// Before
func Map(slice []interface{}, fn func(interface{}) interface{}) []interface{}

// After (Go 2025)
func Map[T, U any](slice []T, fn func(T) U) []U
```

#### 2. Functional Options Pattern
```go
type Option[T any] func(*T)

func New[T any](opts ...Option[T]) *T {
    t := new(T)
    for _, opt := range opts {
        opt(t)
    }
    return t
}
```

#### 3. Result Type Pattern
```go
type Result[T any] struct {
    value T
    err   error
}

func Ok[T any](v T) Result[T]
func Err[T any](err error) Result[T]
```

#### 4. Iterator Pattern (Go 1.23)
```go
func (s *Set[T]) All() iter.Seq[T] {
    return func(yield func(T) bool) {
        for v := range s.items {
            if !yield(v) {
                return
            }
        }
    }
}
```

#### 5. Structured Logging (slog)
```go
slog.Info("operation completed",
    slog.String("operation", "cache_get"),
    slog.Duration("latency", elapsed),
)
```

### API Consistency Standards

| Pattern | Convention | Example |
|---------|------------|---------|
| Constructor | `New[T]()` | `lru.New[K,V](capacity)` |
| From slice | `From[T](slice)` | `set.From(items)` |
| Variadic | `Of[T](items...)` | `set.Of(1, 2, 3)` |
| Check existence | `Contains(v)` | `set.Contains(5)` |
| Get with default | `GetOr(key, default)` | `cache.GetOr("k", "")` |
| Safe get | `Get(key) Option[V]` | `cache.Get("k")` |

### File Size Compliance

All files must be < 400 lines. Split strategy:
- `collections/slices.go` (200+ lines) → OK if < 400
- Large files → Split by concern (e.g., `lru.go` + `lru_expirable.go`)

## Dependencies

```
functional/option  ← collections (for safe returns)
functional/result  ← errors (for Result type)
errors             ← all packages (for error handling)
observability      ← server, grpc, http (for logging/tracing)
```

## Migration Strategy

1. Create consolidated versions in place
2. Update imports across codebase
3. Run tests to verify
4. Delete redundant packages
5. Update go.mod files
