# Phases 4-8: Middleware, Observability, Client, Migration, Validation

## Phase 4: Middleware and Interceptors

### Task 4.1: Implement HTTP Middleware
**Requirement:** REQ-9 (HTTP and gRPC Middleware)
**Acceptance Criteria:** 9.1, 9.2, 9.3, 9.4, 9.5, 9.6

**Files to create:**
- `src/middleware/http.go` - HTTP middleware implementation
- `src/middleware/config.go` - Middleware configuration
- `src/middleware/context.go` - Context utilities for claims
- `tests/middleware/http_test.go` - Unit tests
- `tests/middleware/http_prop_test.go` - Property tests

**Property Tests (Properties 21-22, 24):**
```go
// Property 21: Middleware Skip Patterns
func TestProperty_MiddlewareSkipPatterns(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        pattern := rapid.SampledFrom([]string{"/health", "/metrics", "/ready"}).Draw(t, "pattern")
        path := pattern + rapid.StringMatching(`[a-z]*`).Draw(t, "suffix")
        
        config := &Config{SkipPatterns: []*regexp.Regexp{regexp.MustCompile("^" + pattern)}}
        req := httptest.NewRequest("GET", path, nil)
        skipped := config.shouldSkip(req)
        
        assert.True(t, skipped)
    })
}

// Property 22: Middleware Claims Context
// Property 24: Middleware Audience/Issuer Validation
```

**Deliverables:**
- [ ] Implement `Config` struct with skip patterns, error handler
- [ ] Implement functional options (`WithSkipPatterns`, `WithErrorHandler`)
- [ ] Implement `HTTP()` middleware factory
- [ ] Implement `GetClaimsFromContext()` for claim retrieval
- [ ] Property tests for Properties 21, 22, 24 (100+ iterations each)

---

### Task 4.2: Implement gRPC Interceptors
**Requirement:** REQ-9 (HTTP and gRPC Middleware)
**Acceptance Criteria:** 9.2, 9.3, 9.5

**Files to create:**
- `src/middleware/grpc.go` - gRPC interceptor implementation
- `src/middleware/grpc_errors.go` - gRPC error mapping
- `tests/middleware/grpc_test.go` - Unit tests
- `tests/middleware/grpc_prop_test.go` - Property tests

**Property Tests (Property 23):**
```go
// Property 23: Error to Status Code Mapping
func TestProperty_ErrorToGRPCStatusMapping(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        code := rapid.SampledFrom(allErrorCodes).Draw(t, "code")
        err := NewError(code, "test message")
        
        grpcErr := MapToGRPCError(err)
        status, _ := status.FromError(grpcErr)
        
        expectedCode := errorCodeToGRPCStatus[code]
        assert.Equal(t, expectedCode, status.Code())
    })
}
```

**Deliverables:**
- [ ] Implement `GRPCConfig` struct
- [ ] Implement `UnaryServerInterceptor()` and `StreamServerInterceptor()`
- [ ] Implement `MapToGRPCError()` for error code mapping
- [ ] Property test for Property 23 (100+ iterations)

---

## Phase 5: Observability and Configuration

### Task 5.1: Implement Observability Package
**Requirement:** REQ-10 (Observability Integration)
**Acceptance Criteria:** 10.1, 10.2, 10.3, 10.4, 10.5

**Files to create:**
- `src/internal/observability/tracing.go` - OpenTelemetry tracing
- `src/internal/observability/logging.go` - Structured logging with slog
- `src/internal/observability/filter.go` - Sensitive data filtering
- `tests/internal/observability/filter_prop_test.go` - Property tests

**Property Tests (Property 25):**
```go
// Property 25: Sensitive Data Filtering
func TestProperty_SensitiveDataFiltering(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        sensitivePatterns := []string{"Bearer ", "DPoP ", "secret", "password", "eyJ"}
        pattern := rapid.SampledFrom(sensitivePatterns).Draw(t, "pattern")
        prefix := rapid.StringN(0, 20, 50).Draw(t, "prefix")
        suffix := rapid.StringN(0, 20, 50).Draw(t, "suffix")
        
        input := prefix + pattern + suffix
        filtered := FilterSensitiveData(input)
        
        assert.NotContains(t, filtered, pattern)
        assert.Contains(t, filtered, "[REDACTED]")
    })
}
```

**Deliverables:**
- [ ] Implement `Tracer` with OpenTelemetry integration
- [ ] Implement structured logging with `log/slog`
- [ ] Implement `FilterSensitiveData()` for log/trace sanitization
- [ ] Property test for Property 25 (100+ iterations)

---

### Task 5.2: Implement Configuration Management
**Requirement:** REQ-11 (Configuration Management)
**Acceptance Criteria:** 11.1, 11.2, 11.3, 11.4, 11.5

**Files to create:**
- `src/client/config.go` - Configuration struct and validation
- `src/client/options.go` - Functional options
- `tests/client/config_test.go` - Unit tests
- `tests/client/config_prop_test.go` - Property tests

**Property Tests (Properties 26-29):**
```go
// Property 26: Config Environment Loading
// Property 27: Config Validation
// Property 28: Config Defaults
// Property 29: Config Functional Options
```

**Deliverables:**
- [ ] Implement `Config` struct with env tags
- [ ] Implement `LoadFromEnv()`, `Validate()`, `ApplyDefaults()`
- [ ] Implement functional options pattern
- [ ] Property tests for Properties 26-29 (100+ iterations each)

---

## Phase 6: Client and Integration

### Task 6.1: Implement Main Client
**Requirement:** REQ-1, REQ-13 (Architecture, API Consistency)

**Files to create:**
- `src/client/client.go` - Main client implementation
- `tests/client/client_test.go` - Unit tests

**Deliverables:**
- [ ] Implement `Client` struct with all dependencies
- [ ] Implement `New()`, `ClientCredentials()`, `ValidateToken()`, `GetAccessToken()`
- [ ] Add GoDoc comments for all exported types/functions

---

### Task 6.2: Create Public API Entry Point
**Requirement:** REQ-1 (Architecture Reorganization)

**Files to create:**
- `src/sdk.go` - Public API exports

**Deliverables:**
- [ ] Export all public types from subpackages
- [ ] Provide convenience constructors
- [ ] Add package-level documentation

---

### Task 6.3: Implement Integration Tests
**Requirement:** REQ-12 (Test Coverage and Quality)

**Files to create:**
- `tests/integration/oauth_flows_test.go`
- `tests/integration/middleware_test.go`

**Deliverables:**
- [ ] Integration tests for OAuth flows
- [ ] Integration tests for middleware chains
- [ ] Mock server for OAuth endpoints

---

## Phase 7: Migration and Documentation

### Task 7.1: Migrate Existing Code
**Requirement:** REQ-1 (Architecture Reorganization)

**Deliverables:**
- [ ] Move all source files to `src/` structure
- [ ] Update all import paths
- [ ] Remove old files after verification

---

### Task 7.2: Migrate Existing Tests
**Requirement:** REQ-12 (Test Coverage and Quality)

**Deliverables:**
- [ ] Move tests to mirror `src/` structure
- [ ] Split mixed test files into unit and property tests
- [ ] Verify all tests pass after migration

---

### Task 7.3: Create Examples
**Requirement:** REQ-13 (API Consistency and Documentation)

**Files to create:**
- `examples/client_credentials/main.go`
- `examples/http_middleware/main.go`
- `examples/grpc_interceptor/main.go`
- `examples/dpop/main.go`
- `examples/pkce/main.go`

---

### Task 7.4: Update Documentation
**Requirement:** REQ-13 (API Consistency and Documentation)

**Deliverables:**
- [ ] Update `sdk/go/README.md` with new architecture
- [ ] Add migration guide for breaking changes
- [ ] Create ADR for architecture decisions

---

## Phase 8: Validation and Cleanup

### Task 8.1: Validate Test Coverage
**Requirement:** REQ-12 (Test Coverage and Quality)

**Deliverables:**
- [ ] Run `go test -cover ./...` and verify 80%+ for core modules
- [ ] Generate coverage report with `go tool cover`
- [ ] Identify and fill coverage gaps

---

### Task 8.2: Validate File Size Compliance
**Requirement:** Architecture Rules (Max 400 lines per file)

**Deliverables:**
- [ ] Run line count validation on all `.go` files
- [ ] Split any files exceeding 400 non-blank lines
- [ ] Update imports after splits

---

### Task 8.3: Final Cleanup
**Requirement:** REQ-1 (Architecture Reorganization)

**Deliverables:**
- [ ] Remove deprecated code and files
- [ ] Run `go mod tidy`, `go vet ./...`, `golangci-lint`
- [ ] Verify all property tests run 100+ iterations
- [ ] Final test suite execution
