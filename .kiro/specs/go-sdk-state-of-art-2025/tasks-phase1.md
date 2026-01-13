# Phase 1: Foundation - Architecture and Error Handling

## Task 1.1: Create New Directory Structure
**Requirement:** REQ-1 (Architecture Reorganization)
**Acceptance Criteria:** 1.1, 1.2, 1.3

Create the target directory structure:
```
sdk/go/
├── src/
│   ├── sdk.go
│   ├── client/
│   ├── auth/
│   ├── token/
│   ├── middleware/
│   ├── errors/
│   ├── types/
│   ├── retry/
│   └── internal/observability/
├── tests/
│   ├── client/
│   ├── auth/
│   ├── token/
│   ├── middleware/
│   ├── errors/
│   ├── types/
│   ├── retry/
│   ├── integration/
│   └── property/
└── examples/
```

**Deliverables:**
- [ ] Create `src/` directory with package subdirectories
- [ ] Create `tests/` directory mirroring `src/` structure
- [ ] Create `examples/` directory
- [ ] Update `go.mod` module path if needed

---

## Task 1.2: Implement Unified Error Package
**Requirement:** REQ-2 (Error Handling Modernization)
**Acceptance Criteria:** 2.1, 2.2, 2.3, 2.4, 2.5, 2.6

Consolidate error handling into `src/errors/`:

**Files to create:**
- `src/errors/errors.go` - SDKError type and error codes
- `src/errors/sanitize.go` - Error sanitization utilities
- `tests/errors/errors_test.go` - Unit tests
- `tests/errors/errors_prop_test.go` - Property tests

**Property Tests (Properties 1-4):**
```go
// Property 1: Error Structure Completeness
func TestProperty_ErrorStructureCompleteness(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        code := rapid.SampledFrom(allErrorCodes).Draw(t, "code")
        msg := rapid.StringN(1, 100, 200).Draw(t, "message")
        err := NewError(code, msg)
        
        assert.NotEmpty(t, err.Code)
        assert.NotEmpty(t, err.Message)
    })
}

// Property 2: Error Helper Functions Correctness
func TestProperty_ErrorHelperCorrectness(t *rapid.T) { ... }

// Property 3: Error Message Sanitization
func TestProperty_ErrorSanitization(t *rapid.T) { ... }

// Property 4: Error Chain Preservation
func TestProperty_ErrorChainPreservation(t *rapid.T) { ... }
```

**Deliverables:**
- [ ] Implement `SDKError` with Code, Message, Cause
- [ ] Implement all `ErrorCode` constants
- [ ] Implement `Is*` helper functions (IsTokenExpired, IsRateLimited, etc.)
- [ ] Implement `SanitizeError` with sensitive pattern detection
- [ ] Implement `WrapError` for error chain support
- [ ] Remove sentinel errors (backward compatibility via Is* functions)
- [ ] Property tests for Properties 1-4 (100+ iterations each)
- [ ] Unit tests for edge cases

---

## Task 1.3: Implement Result and Option Types
**Requirement:** REQ-3 (Result Pattern Enhancement)
**Acceptance Criteria:** 3.1, 3.2, 3.3, 3.4, 3.5

Implement functional types in `src/types/`:

**Files to create:**
- `src/types/result.go` - Result[T] type
- `src/types/option.go` - Option[T] type
- `tests/types/result_test.go` - Unit tests
- `tests/types/result_prop_test.go` - Property tests
- `tests/types/option_test.go` - Unit tests
- `tests/types/option_prop_test.go` - Property tests

**Property Tests (Properties 5-6):**
```go
// Property 5: Result/Option Functor Laws
func TestProperty_ResultFunctorLaws(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        v := rapid.Int().Draw(t, "value")
        f := func(x int) int { return x * 2 }
        g := func(x int) int { return x + 1 }
        
        r := Ok(v)
        // Map(Ok(v), f) == Ok(f(v))
        assert.Equal(t, Map(r, f).Unwrap(), f(v))
        // Map(Map(r, f), g) == Map(r, compose(g, f))
        assert.Equal(t, Map(Map(r, f), g).Unwrap(), g(f(v)))
    })
}

// Property 6: Result/Option Conversion Round-Trip
func TestProperty_ResultOptionConversion(t *rapid.T) { ... }
```

**Deliverables:**
- [ ] Implement `Result[T]` with Ok, Err constructors
- [ ] Implement `Map`, `FlatMap`, `MapErr` for Result
- [ ] Implement `Option[T]` with Some, None constructors
- [ ] Implement `MapOption`, `FlatMapOption`, `Filter` for Option
- [ ] Implement `ToOption`, `OkOr` conversion functions
- [ ] Property tests for Properties 5-6 (100+ iterations each)
- [ ] Unit tests for panic cases and edge conditions
