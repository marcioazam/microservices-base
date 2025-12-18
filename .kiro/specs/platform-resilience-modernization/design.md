# Design Document: Platform Resilience Service Modernization

## Overview

This design document describes the modernization of the `/platform/resilience-service` to state-of-the-art December 2024 standards. The modernization eliminates code redundancy, centralizes shared logic, upgrades dependencies, and applies Go 1.23 features including generics and iterators.

### Goals

1. Upgrade to Go 1.23+ with latest stable dependencies
2. Eliminate all duplicate code (6 instances of `generateEventID`, duplicate validation logic)
3. Centralize cross-cutting concerns (event emission, correlation, validation)
4. Apply generics where beneficial for type safety and reusability
5. Leverage Go 1.23 iterators for collection traversal
6. Enhance security and observability

### Non-Goals

1. Changing the gRPC API contract
2. Modifying the resilience pattern algorithms
3. Adding new resilience patterns

## Current State Analysis

### Redundancy Detected

| Location | Type | Instances | Issue |
|----------|------|-----------|-------|
| `circuitbreaker/breaker.go` | generateEventID | 1 | Duplicate function |
| `ratelimit/token_bucket.go` | generateEventID | 1 | Duplicate function |
| `ratelimit/sliding_window.go` | Uses token_bucket's | 0 | Indirect dependency |
| `retry/handler.go` | generateEventID | 1 | Duplicate function |
| `timeout/manager.go` | generateEventID | 1 | Duplicate function |
| `bulkhead/bulkhead.go` | generateEventID | 1 | Duplicate function |
| `health/aggregator.go` | generateEventID | 1 | Duplicate function |
| `policy/engine.go` | validateCircuitBreaker | 1 | Duplicate validation |
| `policy/engine.go` | validateRetry | 1 | Duplicate validation |
| `policy/engine.go` | validateTimeout | 1 | Duplicate validation |
| `policy/engine.go` | validateRateLimit | 1 | Duplicate validation |
| `policy/engine.go` | validateBulkhead | 1 | Duplicate validation |

### Dependency Versions (Current → Target)

| Dependency | Current | Target | Breaking Changes |
|------------|---------|--------|------------------|
| Go | 1.22 | 1.23 | iter package added, Timer/Ticker GC improvements |
| go-redis/v9 | 9.4.0 | 9.7.0 | Maintenance notifications support |
| otel | 1.24.0 | 1.32.0 | Semantic conventions v1.34.0 |
| otel/sdk | 1.24.0 | 1.32.0 | None |
| grpc-go | 1.62.0 | 1.68.0 | protobuf edition 2024 support |
| protobuf | 1.32.0 | 1.35.2 | None |

## Architecture

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

### 1. Domain Package Centralization

The domain package becomes the single source of truth for:

#### 1.1 Event ID Generation

```go
// domain/id.go
package domain

import (
    "crypto/rand"
    "encoding/hex"
    "time"
)

// GenerateEventID generates a unique event identifier.
// Format: timestamp-random (e.g., "20241216150405-a1b2c3d4")
func GenerateEventID() string {
    ts := time.Now().Format("20060102150405")
    b := make([]byte, 4)
    rand.Read(b)
    return ts + "-" + hex.EncodeToString(b)
}
```

#### 1.2 Correlation Function

```go
// domain/correlation.go
package domain

// CorrelationFunc returns a correlation ID for tracing.
type CorrelationFunc func() string

// DefaultCorrelationFunc returns an empty correlation ID.
func DefaultCorrelationFunc() string {
    return ""
}

// EnsureCorrelationFunc returns the provided function or default if nil.
func EnsureCorrelationFunc(fn CorrelationFunc) CorrelationFunc {
    if fn == nil {
        return DefaultCorrelationFunc
    }
    return fn
}
```

#### 1.3 Event Emission Helper

```go
// domain/events.go (addition)

// EmitEvent safely emits an event, handling nil emitter.
func EmitEvent(emitter EventEmitter, event ResilienceEvent) {
    if emitter == nil {
        return
    }
    emitter.Emit(event)
}

// EmitAuditEvent safely emits an audit event, handling nil emitter.
func EmitAuditEvent(emitter EventEmitter, event AuditEvent) {
    if emitter == nil {
        return
    }
    emitter.EmitAudit(event)
}
```

#### 1.4 Config Validation Methods

```go
// domain/circuit_breaker.go (addition)

// Validate validates the circuit breaker configuration.
func (c *CircuitBreakerConfig) Validate() error {
    if c.FailureThreshold <= 0 {
        return NewInvalidPolicyError("circuit_breaker.failure_threshold must be positive")
    }
    if c.SuccessThreshold <= 0 {
        return NewInvalidPolicyError("circuit_breaker.success_threshold must be positive")
    }
    if c.Timeout <= 0 {
        return NewInvalidPolicyError("circuit_breaker.timeout must be positive")
    }
    return nil
}

// Similar Validate() methods for RetryConfig, TimeoutConfig, RateLimitConfig, BulkheadConfig
```

### 2. Rate Limiter Factory

```go
// ratelimit/factory.go
package ratelimit

import (
    "fmt"
    "github.com/auth-platform/platform/resilience-service/internal/domain"
)

// NewRateLimiter creates a rate limiter based on the algorithm configuration.
func NewRateLimiter(cfg domain.RateLimitConfig, emitter domain.EventEmitter) (domain.RateLimiter, error) {
    switch cfg.Algorithm {
    case domain.TokenBucket:
        return NewTokenBucket(TokenBucketConfig{
            Capacity:     cfg.BurstSize,
            RefillRate:   cfg.Limit,
            Window:       cfg.Window,
            EventEmitter: emitter,
        }), nil
    case domain.SlidingWindow:
        return NewSlidingWindow(SlidingWindowConfig{
            Limit:        cfg.Limit,
            Window:       cfg.Window,
            EventEmitter: emitter,
        }), nil
    default:
        return nil, fmt.Errorf("unknown rate limit algorithm: %s", cfg.Algorithm)
    }
}
```

### 3. Go 1.23 Iterator Support

```go
// policy/engine.go (addition)
import "iter"

// Policies returns an iterator over all policies.
func (e *Engine) Policies() iter.Seq[*domain.ResiliencePolicy] {
    return func(yield func(*domain.ResiliencePolicy) bool) {
        e.mu.RLock()
        defer e.mu.RUnlock()
        for _, p := range e.policies {
            if !yield(p) {
                return
            }
        }
    }
}

// health/aggregator.go (addition)

// Services returns an iterator over all registered services.
func (a *Aggregator) Services() iter.Seq[domain.ServiceHealth] {
    return func(yield func(domain.ServiceHealth) bool) {
        a.mu.RLock()
        defer a.mu.RUnlock()
        for name, entry := range a.services {
            sh := domain.ServiceHealth{
                Name:      name,
                Status:    entry.status,
                Message:   entry.message,
                LastCheck: entry.lastCheck,
            }
            if !yield(sh) {
                return
            }
        }
    }
}

// bulkhead/bulkhead.go (addition)

// Partitions returns an iterator over all partitions with their metrics.
func (m *Manager) Partitions() iter.Seq2[string, domain.BulkheadMetrics] {
    return func(yield func(string, domain.BulkheadMetrics) bool) {
        m.mu.RLock()
        defer m.mu.RUnlock()
        for name, b := range m.partitions {
            if !yield(name, b.GetMetrics()) {
                return
            }
        }
    }
}
```

### 4. Serialization Centralization

```go
// domain/serialization.go
package domain

import (
    "encoding/json"
    "time"
)

const TimeFormat = time.RFC3339Nano

// MarshalTime formats a time value for JSON serialization.
func MarshalTime(t time.Time) string {
    return t.Format(TimeFormat)
}

// UnmarshalTime parses a time value from JSON serialization.
func UnmarshalTime(s string) (time.Time, error) {
    return time.Parse(TimeFormat, s)
}
```

## Data Models

### Updated go.mod Dependencies

```go
module github.com/auth-platform/platform/resilience-service

go 1.23

require (
    github.com/leanovate/gopter v0.2.11
    github.com/redis/go-redis/v9 v9.7.0
    go.opentelemetry.io/otel v1.32.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.32.0
    go.opentelemetry.io/otel/sdk v1.32.0
    go.opentelemetry.io/otel/trace v1.32.0
    google.golang.org/grpc v1.68.0
    google.golang.org/protobuf v1.35.2
    gopkg.in/yaml.v3 v3.0.1
)
```

### Domain Types with Validation

All domain configuration types gain a `Validate() error` method that returns `*ResilienceError` on validation failure.

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Based on the prework analysis, the following correctness properties must be verified:

### Property 1: Event ID Uniqueness

*For any* sequence of N generated event IDs, all IDs in the sequence should be distinct from each other.

**Validates: Requirements 2.2**

### Property 2: Nil Emitter Safety

*For any* ResilienceEvent and nil EventEmitter, calling EmitEvent should not panic and should return without error.

**Validates: Requirements 4.2**

### Property 3: Rate Limiter Factory Correctness

*For any* valid RateLimitConfig with TokenBucket or SlidingWindow algorithm, the factory should return a non-nil RateLimiter that correctly enforces the configured limits.

**Validates: Requirements 5.2, 5.3**

### Property 4: Configuration Validation Correctness

*For any* configuration struct (CircuitBreakerConfig, RetryConfig, TimeoutConfig, RateLimitConfig, BulkheadConfig), the Validate() method should return nil for valid configurations and a non-nil error for invalid configurations.

**Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**

### Property 5: Serialization Round-Trip Consistency

*For any* valid CircuitBreakerState, ResiliencePolicy, or RetryConfig, serializing then deserializing should produce an equivalent value.

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 6: Time Format Consistency

*For any* time.Time value, MarshalTime followed by UnmarshalTime should produce an equivalent time value (within nanosecond precision).

**Validates: Requirements 7.5**

### Property 7: Generator Validity

*For any* configuration generated by testutil generators, the generated configuration should pass its Validate() method.

**Validates: Requirements 11.5**

### Property 8: Error Wrapping Preservation

*For any* error wrapped using ResilienceError with a Cause, calling errors.Unwrap should return the original cause error.

**Validates: Requirements 12.2**

### Property 9: gRPC Error Mapping Completeness

*For any* domain ErrorCode, ToGRPCCode should return a valid gRPC status code (not codes.Unknown).

**Validates: Requirements 12.3**

### Property 10: Shutdown Request Blocking

*For any* GracefulShutdown instance, after Shutdown is initiated, RequestStarted should return false.

**Validates: Requirements 14.1**

### Property 11: Shutdown Drain with Timeout

*For any* GracefulShutdown with in-flight requests, Shutdown should wait for requests to complete or return error after timeout.

**Validates: Requirements 14.2, 14.3**

### Property 12: Policy Reload Validation

*For any* invalid policy configuration, attempting to reload should preserve the existing valid policy.

**Validates: Requirements 15.2, 15.3**

### Property 13: Policy Format Parsing

*For any* valid ResiliencePolicy, both YAML and JSON serialization should parse correctly and produce equivalent policies.

**Validates: Requirements 15.5**

## Error Handling

### Error Types

All errors use the centralized `ResilienceError` type with appropriate error codes:

| Error Code | gRPC Code | Description |
|------------|-----------|-------------|
| ErrCircuitOpen | Unavailable | Circuit breaker is open |
| ErrRateLimitExceeded | ResourceExhausted | Rate limit exceeded |
| ErrTimeout | DeadlineExceeded | Operation timed out |
| ErrBulkheadFull | ResourceExhausted | Bulkhead capacity exceeded |
| ErrRetryExhausted | Unavailable | All retry attempts failed |
| ErrInvalidPolicy | InvalidArgument | Policy validation failed |
| ErrServiceUnavailable | Unavailable | Service is unavailable |

### Error Wrapping

All errors that wrap underlying errors use `fmt.Errorf` with `%w` to preserve the error chain:

```go
return fmt.Errorf("load policy from file: %w", err)
```

## Testing Strategy

### Dual Testing Approach

The modernization maintains both unit tests and property-based tests:

1. **Unit Tests**: Verify specific examples, edge cases, and integration points
2. **Property-Based Tests**: Verify universal properties using gopter with 100+ iterations

### Property-Based Testing Framework

- **Library**: github.com/leanovate/gopter v0.2.11
- **Minimum Iterations**: 100 per property
- **Generators**: Centralized in `tests/testutil/generators.go`

### Test Annotation Format

All property-based tests must be annotated with:

```go
// **Feature: platform-resilience-modernization, Property N: Property Name**
// **Validates: Requirements X.Y**
```

### Test Organization

```
tests/
├── property/
│   ├── centralization_prop_test.go    # Properties 1, 2, 7
│   ├── factory_prop_test.go           # Property 3
│   ├── validation_prop_test.go        # Property 4
│   ├── serialization_prop_test.go     # Properties 5, 6, 13
│   ├── error_prop_test.go             # Properties 8, 9
│   ├── shutdown_prop_test.go          # Properties 10, 11
│   └── policy_prop_test.go            # Property 12
├── testutil/
│   ├── generators.go                  # Centralized generators
│   └── helpers.go                     # Test utilities
└── unit/
    └── ...                            # Unit tests
```

## File Changes Summary

### Files to Modify

| File | Changes |
|------|---------|
| `go.mod` | Upgrade Go version and dependencies |
| `internal/domain/events.go` | Add EmitEvent, EmitAuditEvent helpers |
| `internal/domain/circuit_breaker.go` | Add Validate() method |
| `internal/domain/retry.go` | Add Validate() method |
| `internal/domain/timeout.go` | Add Validate() method |
| `internal/domain/ratelimit.go` | Add Validate() method |
| `internal/domain/bulkhead.go` | Add Validate() method |
| `internal/circuitbreaker/breaker.go` | Use centralized GenerateEventID, EmitEvent |
| `internal/ratelimit/token_bucket.go` | Use centralized GenerateEventID, EmitEvent |
| `internal/ratelimit/sliding_window.go` | Use centralized GenerateEventID, EmitEvent |
| `internal/retry/handler.go` | Use centralized GenerateEventID, EmitEvent |
| `internal/timeout/manager.go` | Use centralized GenerateEventID, EmitEvent |
| `internal/bulkhead/bulkhead.go` | Use centralized GenerateEventID, EmitEvent |
| `internal/health/aggregator.go` | Use centralized GenerateEventID, EmitEvent |
| `internal/policy/engine.go` | Delegate validation to domain, add iterator |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/domain/id.go` | Centralized event ID generation |
| `internal/domain/correlation.go` | Centralized correlation function |
| `internal/domain/serialization.go` | Centralized time serialization |
| `internal/ratelimit/factory.go` | Rate limiter factory |
| `tests/property/centralization_prop_test.go` | Centralization properties |
| `tests/property/factory_prop_test.go` | Factory properties |
| `tests/property/validation_prop_test.go` | Validation properties |

### Files to Delete

None - all changes are modifications or additions.

## Migration Strategy

1. **Phase 1**: Add new centralized functions without removing old ones
2. **Phase 2**: Update all components to use centralized functions
3. **Phase 3**: Remove duplicate functions
4. **Phase 4**: Add property-based tests for new properties
5. **Phase 5**: Upgrade dependencies

This approach ensures backward compatibility during migration.
