# Design Document

## Overview

This design document describes the State of the Art modernization for `platform/resilience-service`. The primary focus is eliminating redundancy by fixing re-export patterns and removing duplicate implementations.

## Architecture

The resilience-service follows a clean architecture with:
- `internal/domain/` - Domain types and re-exports from shared libraries
- `internal/*/` - Implementation packages (circuitbreaker, ratelimit, retry, timeout, bulkhead, health, policy)
- `libs/go/resilience/` - Shared resilience primitives
- `libs/go/error/` - Shared error types
- `libs/go/validation/` - Shared validators

## Components and Interfaces

### Re-export Pattern Fix

The issue: Go variable assignments of functions cannot be called like functions in some contexts.

**Before (broken):**
```go
var GenerateEventID = resilience.GenerateEventID
// domain.GenerateEventID() fails in some contexts
```

**After (fixed):**
```go
func GenerateEventID() string {
    return resilience.GenerateEventID()
}
// domain.GenerateEventID() works correctly
```

### Affected Files

1. `internal/domain/id.go` - GenerateEventID, GenerateEventIDWithPrefix
2. `internal/domain/correlation.go` - EnsureCorrelationFunc
3. `internal/domain/serialization.go` - MarshalTime, UnmarshalTime, NowUTC, etc.
4. `internal/timeout/manager.go` - Remove duplicate generateEventID

## Data Models

No changes to data models required.

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Re-export Function Callability

*For any* re-exported function in the domain package, calling it SHALL produce the same result as calling the original library function directly.

**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: Zero Redundancy

*For any* functionality (event ID generation, correlation handling, time formatting), there SHALL be exactly one implementation in the codebase.

**Validates: Requirements 2.1, 2.2, 2.3, 2.4**

### Property 3: Compilation Success

*For any* valid Go code in the service, the compiler SHALL produce no errors related to function calls on re-exported symbols.

**Validates: Requirements 1.1**

## Error Handling

No changes to error handling patterns.

## Testing Strategy

### Unit Tests
- Verify re-exported functions produce correct results
- Verify no compilation errors

### Property-Based Tests
- Already covered by existing property tests in tests/property/

## Redundancy Elimination Map

| Location | Type | Instances | Target | Strategy |
|----------|------|-----------|--------|----------|
| timeout/manager.go | generateEventID | 1 | domain.GenerateEventID | Remove local, use domain |
| domain/id.go | var assignment | 2 | function wrapper | Convert to function |
| domain/correlation.go | var assignment | 1 | function wrapper | Convert to function |
| domain/serialization.go | var assignment | 5 | function wrapper | Convert to function |

## Technology Stack

| Component | Current | Modern | Status |
|-----------|---------|--------|--------|
| Go | 1.23 | 1.23 | ✅ Current |
| go-redis | 9.7.0 | 9.7.0 | ✅ Current |
| OpenTelemetry | 1.32.0 | 1.32.0 | ✅ Current |
| grpc-go | 1.68.0 | 1.68.0 | ✅ Current |
| gopter | 0.2.11 | 0.2.11 | ✅ Current |
