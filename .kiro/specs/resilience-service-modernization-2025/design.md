# Design Document

## Overview

This design document describes the comprehensive State of the Art modernization for `platform/resilience-service` targeting December 2024/2025 standards. Building on the previous phase that fixed re-export patterns, this phase focuses on:

1. **Dependency Upgrades** - Latest stable versions of grpc-go, go-redis, OpenTelemetry
2. **Timestamp Consistency** - Eliminating time.Now() in favor of domain.NowUTC()
3. **Iterator Patterns** - Consistent use of Go 1.23 iter.Seq/iter.Seq2
4. **Error Handling** - Centralized error constructors and type checking
5. **Property Testing** - Comprehensive coverage of correctness properties
6. **Security Hardening** - Secure random number generation

## Architecture

The resilience-service maintains its clean architecture:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           gRPC Service Layer                                │
│  ExecuteWithResilience │ GetCircuitState │ UpdatePolicy │ Health            │
├─────────────────────────────────────────────────────────────────────────────┤
│                         Policy Engine (Hot-Reload)                          │
│  Parser │ Validator (delegates to domain) │ Store │ Iter[Policy]            │
├─────────────────────────────────────────────────────────────────────────────┤
│                      Resilience Patterns Layer                              │
│  CircuitBreaker │ Retry │ Timeout │ RateLimiter[T] │ Bulkhead               │
├─────────────────────────────────────────────────────────────────────────────┤
│                         Domain Layer (Centralized)                          │
│  Configs + Validate() │ EventID │ CorrelationFunc │ EmitEvent │ Errors      │
├─────────────────────────────────────────────────────────────────────────────┤
│                      Infrastructure Clients                                 │
│  Redis (TLS) │ OpenTelemetry 1.32+ │ slog │ MetricsRecorder                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Timestamp Consistency

**Current State:** Several files use `time.Now()` directly instead of `domain.NowUTC()`.

**Files Requiring Changes:**
- `internal/circuitbreaker/breaker.go` - Uses `time.Now()` in `transitionTo()` and `emitStateChangeEvent()`
- `internal/retry/handler.go` - Uses `time.Now()` in `emitRetryEvent()`
- `internal/bulkhead/bulkhead.go` - Uses `time.Now()` in `emitRejectionEvent()`
- `internal/health/aggregator.go` - Uses `time.Now()` in multiple places

**Solution:** Replace all `time.Now()` calls with `domain.NowUTC()` for consistency.

### 2. Dependency Updates

**Current Versions (from go.mod):**
| Package | Current | Target | Status |
|---------|---------|--------|--------|
| grpc-go | 1.68.0 | 1.77.0 | Upgrade needed |
| go-redis/v9 | 9.7.0 | 9.7.0+ | ✅ Current |
| OpenTelemetry | 1.32.0 | 1.32.0+ | ✅ Current |
| protobuf | 1.35.2 | 1.35.2+ | ✅ Current |
| gopter | 0.2.11 | 0.2.11 | ✅ Current |

**grpc-go 1.77.0 Changes:**
- Performance improvements in transport layer
- Bug fixes for xdsclient race conditions
- Memory optimization for large buffers

### 3. Iterator Pattern Consistency

**Current Implementation:** Already uses `iter.Seq` and `iter.Seq2` in:
- `internal/bulkhead/bulkhead.go` - `Partitions() iter.Seq2[string, domain.BulkheadMetrics]`
- `internal/health/aggregator.go` - `Services() iter.Seq[domain.ServiceHealth]`
- `internal/policy/engine.go` - `Policies() iter.Seq[domain.ResiliencePolicy]`

**Status:** ✅ Already following Go 1.23 patterns

### 4. Secure Random Number Generation

**Current State:** `internal/retry/handler.go` uses:
```go
randSource: rand.New(rand.NewSource(time.Now().UnixNano()))
```

**Issue:** Time-based seeding is predictable and not suitable for production.

**Solution:** Use crypto/rand for seeding and support injection for testing:
```go
type Handler struct {
    // ... existing fields
    randSource RandSource // Interface for testability
}

type RandSource interface {
    Float64() float64
}
```

## Data Models

No changes to data models required. Existing models are well-structured.

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Configuration Validation Consistency

*For any* configuration type (CircuitBreakerConfig, RetryConfig, TimeoutConfig, RateLimitConfig, BulkheadConfig), calling Validate() on a valid configuration SHALL return nil, and calling Validate() on an invalid configuration SHALL return a non-nil error.

**Validates: Requirements 3.2**

### Property 2: Error Constructor Type Preservation

*For any* error created using domain.New*Error() constructors, the corresponding domain.Is*() function SHALL return true for that error.

**Validates: Requirements 4.1, 4.3**

### Property 3: Serialization Round-Trip

*For any* valid time value, calling MarshalTime() then UnmarshalTime() SHALL produce an equivalent time value (within nanosecond precision).

**Validates: Requirements 7.2**

### Property 4: Iterator Completeness

*For any* collection exposed via iter.Seq or iter.Seq2, iterating over the collection SHALL yield all elements exactly once.

**Validates: Requirements 5.1, 5.3**

### Property 5: gRPC Error Mapping Completeness

*For any* ResilienceError with a known error code, mapping to gRPC status SHALL produce a valid, non-Unknown status code.

**Validates: Requirements 10.1**

### Property 6: gRPC Error Code Correctness

*For any* ErrCircuitOpen error, gRPC mapping SHALL produce codes.Unavailable.
*For any* ErrRateLimitExceeded error, gRPC mapping SHALL produce codes.ResourceExhausted.
*For any* ErrTimeout error, gRPC mapping SHALL produce codes.DeadlineExceeded.
*For any* ErrBulkheadFull error, gRPC mapping SHALL produce codes.ResourceExhausted.

**Validates: Requirements 10.2, 10.3, 10.4, 10.5**

### Property 7: Correlation ID Propagation

*For any* event created with a non-nil correlation function, the event's CorrelationID field SHALL contain the value returned by that function.

**Validates: Requirements 9.1**

### Property 8: Event JSON Serialization

*For any* ResilienceEvent or AuditEvent, JSON marshaling SHALL succeed and produce valid JSON.

**Validates: Requirements 9.2**

### Property 9: Retry Delay Bounds

*For any* retry attempt, the calculated delay SHALL be non-negative and SHALL not exceed MaxDelay.

**Validates: Requirements 8.1**

### Property 10: Deterministic Retry with Fixed Seed

*For any* retry handler with an injected deterministic random source, calling CalculateDelay() with the same attempt number SHALL produce the same delay.

**Validates: Requirements 8.2**

## Error Handling

Error handling follows the existing centralized pattern:
- All resilience errors created via `domain.New*Error()` constructors
- Error type checking via `domain.Is*()` functions
- gRPC mapping via `internal/grpc/errors.go`

## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests and property-based tests:

1. **Unit Tests** - Verify specific examples and edge cases
2. **Property-Based Tests** - Verify universal properties across many inputs

### Property-Based Testing Framework

- **Library:** gopter v0.2.11
- **Minimum Iterations:** 100 per property
- **Annotation Format:** `**Feature: resilience-service-modernization-2025, Property N: Description**`

### Unit Testing Requirements

- Test specific error conditions
- Test edge cases (empty inputs, boundary values)
- Test integration points between components

### Property-Based Testing Requirements

Each correctness property MUST be implemented as a property-based test:

| Property | Test File | Description |
|----------|-----------|-------------|
| Property 1 | validation_prop_test.go | Config validation consistency |
| Property 2 | error_prop_test.go | Error constructor type preservation |
| Property 3 | serialization_prop_test.go | Time serialization round-trip |
| Property 4 | structure_prop_test.go | Iterator completeness |
| Property 5-6 | grpc_errors_prop_test.go | gRPC error mapping |
| Property 7-8 | centralization_prop_test.go | Correlation and JSON |
| Property 9-10 | retry_handler_prop_test.go | Retry delay bounds |

## Redundancy Elimination Map

| Location | Type | Current | Target | Strategy |
|----------|------|---------|--------|----------|
| circuitbreaker/breaker.go | time.Now() | 3 calls | domain.NowUTC() | Replace |
| retry/handler.go | time.Now() | 1 call | domain.NowUTC() | Replace |
| bulkhead/bulkhead.go | time.Now() | 1 call | domain.NowUTC() | Replace |
| health/aggregator.go | time.Now() | 3 calls | domain.NowUTC() | Replace |
| retry/handler.go | rand seed | time-based | crypto/rand | Replace |

## Technology Stack

| Component | Current | Target | Source |
|-----------|---------|--------|--------|
| Go | 1.23 | 1.23 | go.dev/blog/go1.23 |
| grpc-go | 1.68.0 | 1.77.0 | github.com/grpc/grpc-go/releases |
| go-redis/v9 | 9.7.0 | 9.7.0+ | github.com/redis/go-redis |
| OpenTelemetry | 1.32.0 | 1.32.0+ | opentelemetry.io |
| gopter | 0.2.11 | 0.2.11 | github.com/leanovate/gopter |
| protobuf | 1.35.2 | 1.35.2+ | google.golang.org/protobuf |

## Security Considerations

1. **Secure Randomness** - Replace time-based seeding with crypto/rand
2. **Dependency Scanning** - Verify no known vulnerabilities in updated dependencies
3. **Error Information Leakage** - Ensure gRPC errors don't expose internal details
