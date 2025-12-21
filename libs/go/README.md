# Go Libraries

Production-ready Go libraries for building microservices with state-of-the-art patterns.

**Total: 24 packages | 94 test files | 100% coverage on core paths**

## Quick Start

```go
import (
    "github.com/authcorp/libs/go/src/domain"
    "github.com/authcorp/libs/go/src/errors"
    "github.com/authcorp/libs/go/src/validation"
    "github.com/authcorp/libs/go/src/http"
    "github.com/authcorp/libs/go/src/workerpool"
)

// Domain primitives with built-in validation
email, err := domain.NewEmail("user@example.com")
uuid := domain.NewUUID()
ulid := domain.NewULID()
money, _ := domain.NewMoney(1000, domain.USD)

// Typed errors with HTTP/gRPC mapping
err := errors.NotFound("user")
err = errors.Validation("invalid input").WithDetail("field", "email")
status := err.HTTPStatus() // 404

// Composable validation
result := validation.ValidateAll(
    validation.Field("name").String(name, validation.Required(), validation.MinLength(2)),
    validation.Field("age").Int(age, validation.InRange(0, 150)),
)

// Resilient HTTP client
client := http.NewClient().
    WithTimeout(30 * time.Second).
    WithRetry(3, 100 * time.Millisecond)
resp, _ := client.Get(ctx, "https://api.example.com/users")

// Worker pool with generics
pool := workerpool.NewPool(4, func(ctx context.Context, data int) (int, error) {
    return data * 2, nil
})
pool.Start()
pool.Submit(workerpool.Job[int]{ID: "job-1", Data: 42})
```

## Packages

### Domain Primitives (`domain/`)
Type-safe value objects with validation:
- `Email` - RFC 5322 compliant email addresses
- `UUID` - RFC 4122 UUID v4 identifiers
- `ULID` - Time-ordered lexicographically sortable IDs
- `Money` - Monetary values with currency handling
- `PhoneNumber` - E.164 formatted phone numbers
- `URL` - Validated URLs with scheme restrictions
- `Timestamp` - ISO 8601 timestamps
- `Duration` - Human-readable duration parsing

### Error Handling (`errors/`)
Typed errors with HTTP/gRPC mapping (Go 1.25+):
- `AppError` - Standard error type with code, message, details
- Error wrapping with `Wrap()`, `RootCause()`, `Chain()`
- HTTP status mapping via `HTTPStatus()`
- gRPC code mapping via `GRPCCode()`
- API response with PII redaction via `ToAPIResponse()`
- `AsType[T]()` - Generic error type assertion (Go 1.25+)
- `Must[T]()` - Panic-on-error helper for initialization
- Re-exported `Is`, `As`, `Unwrap`, `Join` from standard library

### Validation (`validation/`)
Composable validation with error accumulation:
- `Validator[T]` function type
- Composition: `And()`, `Or()`, `Not()`
- String: `Required()`, `MinLength()`, `MaxLength()`, `MatchesRegex()`, `OneOf()`
- Numeric: `Positive()`, `NonZero()`, `InRange()`, `Min()`, `Max()`, `IntRange()`, `IntPositive()`
- Float: `FloatRange()`, `FloatPositive()`, `FloatNonNegative()`, `FloatMin()`, `FloatMax()`
- Duration: `DurationRange()`, `DurationMin()`, `DurationMax()`, `DurationPositive()`, `DurationNonZero()`
- Collections: `MinSize()`, `MaxSize()`, `UniqueElements()`
- Field validation: `Field()`, `NestedField()`, `ValidateAll()`
- Fluent API with method chaining on `Result`

### Codec (`codec/`)
Encoding/decoding with multiple formats:
- `JSONCodec` - JSON with pretty/compact options
- `YAMLCodec` - YAML for configuration
- `Base64Codec` - Standard and URL-safe variants
- `TypedCodec[T]` - Generic type-safe interface
- `TypedJSONCodec[T]`, `TypedYAMLCodec[T]` - Type-safe implementations
- `EncodeResult`, `DecodeResult` - `functional.Result[T]` integration

### Observability (`observability/`)
Structured logging and tracing:
- JSON structured logging with levels
- Correlation ID propagation
- Trace context (W3C format)
- PII redaction in logs
- User context preservation

### Security (`security/`)
Security utilities:
- `ConstantTimeCompare()` - Timing-safe comparison
- `GenerateToken()` - Secure random tokens
- `SanitizeHTML()`, `SanitizeSQL()`, `SanitizeShell()`
- `DetectSQLInjection()` - SQL injection detection
- `MaskSensitive()` - Data masking for logs

### Pagination (`pagination/`)
Cursor and offset pagination:
- `Page` - Offset/limit pagination
- `Cursor` - Opaque cursor encoding
- `PageResult[T]` - Generic paginated results
- `PageInfo` - Pagination metadata

### Functional (`functional/`)
Functional programming primitives:
- `Option[T]` - Optional values (Some/None)
- `Result[T]` - Success/failure (Ok/Err)
- `Either[L, R]` - Left/Right values
- `Validated[E, A]` - Error accumulation (applicative functor)
- `Pair[A, B]`, `Triple[A, B, C]`, `Quad[A, B, C, D]` - Tuples with `Unpack()` methods
- `Zip`, `Zip3`, `ZipWith`, `Unzip`, `Unzip3` - Slice combining/splitting
- `ZipIter`, `EnumerateIter` - Go 1.23+ iterator support
- `EnumerateSlice` - Index-value pairs from slices
- `MapPairFirst`, `MapPairSecond`, `MapPairBoth` - Tuple transformations
- `Pipeline` - Function composition

### Fault Tolerance (`fault/`)
Generic resilience patterns with full type safety (Go 1.25+):
- `ResilienceExecutor[T]` - Generic interface for type-safe resilience execution
- `PolicyConfig` - Unified policy configuration for all patterns
- `CircuitBreakerPolicyConfig` - Circuit breaker parameters
- `RetryPolicyConfig` - Retry with exponential backoff parameters
- `TimeoutPolicyConfig` - Operation timeout parameters
- `RateLimitPolicyConfig` - Rate limiting parameters
- `BulkheadPolicyConfig` - Bulkhead isolation parameters
- `ExecutorConfig` - Executor behavior configuration
- `ExecuteFunc[T]` - Helper for Result-based execution
- `ExecuteSimple[T]` - Helper for simple error-based execution

### Resilience (`resilience/`)
Fault tolerance pattern implementations:
- `CircuitBreaker` - Fail-fast with recovery
- `RateLimiter` - Token bucket rate limiting
- `Retry` - Exponential backoff retry
- `Bulkhead` - Concurrency limiting
- `Timeout` - Operation timeouts
- `ExecutionMetrics` - Shared metrics type for observability
- `MetricsRecorder` - Interface for recording execution metrics

### Collections (`collections/`)
Generic collections with Go 1.23+ iterator support:
- `LRUCache[K, V]` - Thread-safe LRU cache with TTL, eviction callbacks, and statistics
  - `Get()` returns `Option[V]` for type-safe access
  - `GetOrCompute()` for lazy initialization
  - `PutWithTTL()` for per-entry TTL
  - `Stats()` for monitoring (hits, misses, evictions, hit rate)
  - `All()` returns `iter.Seq2[K, V]` iterator
  - `Cleanup()` for expired entry removal
- `lru.Cache[K, V]` - Lightweight LRU cache (no TTL, no dependencies)
  - `Get()`, `Set()`, `Peek()` for basic operations
  - `GetOrSet()` for atomic get-or-insert
  - `Resize()` for dynamic capacity changes
  - `Keys()`, `Values()` for iteration
- `Set[T]` - Hash set
- `PriorityQueue[T]` - Heap-based priority queue
- `Queue[T]` - FIFO queue

### Concurrency (`concurrency/`)
Concurrency utilities:
- `Future[T]` - Async computation
- `WorkerPool[T, R]` - Job processing pool
- `ErrGroup` - Error-propagating goroutine group

### HTTP (`http/`)
Resilient HTTP client and middleware:
- `Client` - HTTP client with retry, timeout, headers
- `Middleware` - Chainable middleware (logging, recovery, CORS, timeout)
- `HealthHandler` - Liveness/readiness probes

### Worker Pool (`workerpool/`)
Generic worker pool with priority queue:
- `Pool[T, R]` - Generic worker pool
- `Job[T]` - Job with ID, data, priority
- `Result[R]` - Job result with error handling
- Panic recovery and statistics

### Idempotency (`idempotency/`)
Idempotency key handling:
- `Store` interface - Pluggable storage
- `MemoryStore` - In-memory implementation
- Lock/unlock for concurrent requests
- TTL-based expiration and cleanup

### Versioning (`versioning/`)
API versioning utilities:
- `Version` - API version with deprecation
- `Router` - Version-aware request routing
- `PathVersionExtractor` - Extract from URL path
- `HeaderVersionExtractor` - Extract from header
- Deprecation and sunset headers

### Cache (`cache/`)
> **Note:** Being consolidated into `collections/` - use `collections.LRUCache` instead.
- `LRUCache[K, V]` - Generic LRU cache with TTL (deprecated, use `collections.LRUCache`)

### Config (`config/`)
Configuration management:
- `Config` - Type-safe configuration
- Environment variable binding
- Default values and validation

## Testing

All libraries include property-based tests using [rapid](https://github.com/flyingmutant/rapid).

The `testing/` module provides domain-specific generators:

```go
import testutil "github.com/authcorp/libs/go/src/testing"

// Domain generators for property tests
email := testutil.EmailGen().Draw(t, "email")       // RFC 5322 emails
uuid := testutil.UUIDGen().Draw(t, "uuid")          // UUID v4
ulid := testutil.ULIDGen().Draw(t, "ulid")          // ULID
money := testutil.MoneyGen().Draw(t, "money")       // Money{Amount, Currency}
phone := testutil.PhoneNumberGen().Draw(t, "phone") // E.164 format
url := testutil.URLGen().Draw(t, "url")             // HTTP/HTTPS URLs
ip := testutil.IPAddressGen().Draw(t, "ip")         // IPv4 addresses
ts := testutil.RecentTimestampGen().Draw(t, "ts")   // Recent timestamps
```

Tests are organized in `libs/go/tests/` mirroring the `src/` structure:

```bash
# Run all tests (Windows PowerShell)
cd libs/go/tests
Get-ChildItem -Directory | ForEach-Object { Push-Location $_.FullName; go test -v ./...; Pop-Location }

# Run all tests (Linux/Mac)
cd libs/go/tests
for dir in */; do (cd "$dir" && go test -v ./...); done

# Run specific package tests
cd libs/go/tests/resilience && go test -v ./...
cd libs/go/tests/collections && go test -v ./...
```

## Project Structure

```
libs/go/
├── src/                    # Source packages (25 modules)
│   ├── cache/              # LRU cache with TTL
│   ├── codec/              # JSON/YAML/Base64 codecs
│   ├── collections/        # Generic collections (LRU, Set, Queue, PriorityQueue)
│   ├── concurrency/        # Async, Atomic, Channels, ErrGroup, Pool
│   ├── config/             # Configuration management
│   ├── domain/             # Domain primitives (Email, UUID, Money)
│   ├── errors/             # Enhanced error handling
│   ├── events/             # Event bus, pub/sub, builder
│   ├── fault/              # Generic resilience executor interface
│   ├── functional/         # Option, Result, Either, Pipeline, Stream
│   ├── grpc/               # gRPC utilities and error mapping
│   ├── http/               # HTTP client & middleware
│   ├── idempotency/        # Idempotency key handling
│   ├── observability/      # Structured logging, tracing
│   ├── optics/             # Lens, Prism (functional optics)
│   ├── pagination/         # Cursor/offset pagination
│   ├── patterns/           # Repository pattern, CachedRepository, Pagination
│   ├── resilience/         # Circuit breaker, retry, bulkhead, rate limit
│   ├── security/           # Security utilities, sanitization
│   ├── server/             # Health endpoints, graceful shutdown
│   ├── testing/            # Test utilities, generators, mocks
│   ├── utils/              # General utilities (audit, merge, uuid)
│   ├── validation/         # Composable validation
│   ├── versioning/         # API versioning
│   ├── workerpool/         # Generic worker pool
│   └── go.work
│
└── tests/                  # All tests (mirrors src/ structure)
    ├── cache/
    ├── codec/
    ├── collections/        # lru/, maps/, pqueue/, queue/, set/, slices/, sort/
    ├── concurrency/        # async/, atomic/, channels/, errgroup/, once/, pool/
    ├── config/
    ├── domain/
    ├── errors/
    ├── events/             # builder/, eventbus/, pubsub/
    ├── functional/         # either/, iterator/, lazy/, option/, pipeline/
    ├── grpc/
    ├── http/
    ├── idempotency/
    ├── observability/
    ├── optics/             # lens/, prism/
    ├── pagination/
    ├── patterns/           # registry/, spec/
    ├── resilience/         # bulkhead/, circuitbreaker/, ratelimit/, retry/
    ├── security/
    ├── server/             # health/, shutdown/, tracing/
    ├── testing/
    ├── utils/              # audit/, cache/, codec/, diff/, error/, merge/
    ├── validation/
    ├── versioning/
    ├── workerpool/
    └── go.work
```

## License

Internal use only.
