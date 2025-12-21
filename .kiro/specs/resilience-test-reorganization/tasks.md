# Implementation Plan

- [x] 1. Create test directory structure
  - Create `platform/resilience-service/tests/` directory
  - Create subdirectories: `property/`, `benchmark/`, `unit/`, `integration/`, `testutil/`
  - _Requirements: 1.1, 1.2_

- [x] 2. Move and fix test utilities
  - [x] 2.1 Move `internal/testutil/generators.go` to `tests/testutil/generators.go`
    - Update package declaration to `package testutil`
    - Verify imports are accessible
    - _Requirements: 3.1, 3.2_
  - [x] 2.2 Move `internal/testutil/helpers.go` to `tests/testutil/helpers.go`
    - Update package declaration
    - Ensure `t.Helper()` pattern is used
    - _Requirements: 3.1, 3.3_

- [x] 3. Move property tests
  - [x] 3.1 Move bulkhead property tests
    - Move `internal/bulkhead/bulkhead_prop_test.go` to `tests/property/bulkhead_prop_test.go`
    - Update package to `package property`
    - Update imports to reference `internal/bulkhead`
    - _Requirements: 1.1, 1.2, 4.1, 4.2_
  - [x] 3.2 Move circuitbreaker property tests
    - Move `internal/circuitbreaker/breaker_prop_test.go` to `tests/property/circuitbreaker_prop_test.go`
    - Move `internal/circuitbreaker/emitter_prop_test.go` to `tests/property/emitter_prop_test.go`
    - Move `internal/circuitbreaker/serialization_prop_test.go` to `tests/property/serialization_prop_test.go`
    - Update packages and imports
    - _Requirements: 1.1, 1.2, 4.1, 4.2_
  - [x] 3.3 Move grpc property tests
    - Move `internal/grpc/errors_prop_test.go` to `tests/property/grpc_errors_prop_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_
  - [x] 3.4 Move health property tests
    - Move `internal/health/aggregator_prop_test.go` to `tests/property/health_prop_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_
  - [x] 3.5 Move infra property tests
    - Move `internal/infra/audit/logger_prop_test.go` to `tests/property/audit_logger_prop_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_
  - [x] 3.6 Move policy property tests
    - Move `internal/policy/engine_prop_test.go` to `tests/property/policy_prop_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_
  - [x] 3.7 Move ratelimit property tests
    - Move `internal/ratelimit/ratelimit_prop_test.go` to `tests/property/ratelimit_prop_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_
  - [x] 3.8 Move retry property tests
    - Move `internal/retry/handler_prop_test.go` to `tests/property/retry_handler_prop_test.go`
    - Move `internal/retry/policy_prop_test.go` to `tests/property/retry_policy_prop_test.go`
    - Update packages and imports
    - _Requirements: 1.1, 1.2_
  - [x] 3.9 Move server property tests
    - Move `internal/server/shutdown_prop_test.go` to `tests/property/shutdown_prop_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_
  - [x] 3.10 Move timeout property tests
    - Move `internal/timeout/manager_prop_test.go` to `tests/property/timeout_prop_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_

- [x] 4. Move and fix benchmark tests
  - [x] 4.1 Move and fix bulkhead benchmark tests
    - Move `internal/bulkhead/bulkhead_bench_test.go` to `tests/benchmark/bulkhead_bench_test.go`
    - Fix compilation errors: replace `Timeout` with `QueueTimeout`
    - Remove `NewPartitioned` and `PartitionedConfig` references
    - Fix `Acquire()` return type handling (error, not bool)
    - _Requirements: 1.1, 5.1, 5.2, 5.3_
  - [x] 4.2 Move circuitbreaker benchmark tests
    - Move `internal/circuitbreaker/breaker_bench_test.go` to `tests/benchmark/circuitbreaker_bench_test.go`
    - Update package and imports
    - _Requirements: 1.1, 5.3_
  - [x] 4.3 Move ratelimit benchmark tests
    - Move `internal/ratelimit/ratelimit_bench_test.go` to `tests/benchmark/ratelimit_bench_test.go`
    - Update package and imports
    - _Requirements: 1.1, 5.3_
  - [x] 4.4 Move retry benchmark tests
    - Move `internal/retry/handler_bench_test.go` to `tests/benchmark/retry_bench_test.go`
    - Update package and imports
    - _Requirements: 1.1, 5.3_

- [x] 5. Move unit tests
  - [x] 5.1 Move histogram unit tests
    - Move `internal/infra/metrics/histogram_test.go` to `tests/unit/histogram_test.go`
    - Update package and imports
    - _Requirements: 1.1, 1.2_

- [x] 6. Move integration tests
  - [x] 6.1 Move redis integration tests
    - Move `internal/infra/redis/client_integration_test.go` to `tests/integration/redis_client_test.go`
    - Update package and imports
    - Verify `//go:build integration` tag is present
    - Verify `t.Skip` pattern for unavailable Redis
    - _Requirements: 1.1, 6.1, 6.2, 6.3_

- [x] 7. Checkpoint - Verify compilation
  - Ensure all tests pass, ask the user if questions arise.

- [x] 8. Delete original test files from internal/
  - [x] 8.1 Delete moved property test files
    - Remove `internal/bulkhead/bulkhead_prop_test.go`
    - Remove `internal/circuitbreaker/breaker_prop_test.go`
    - Remove `internal/circuitbreaker/emitter_prop_test.go`
    - Remove `internal/circuitbreaker/serialization_prop_test.go`
    - Remove `internal/grpc/errors_prop_test.go`
    - Remove `internal/health/aggregator_prop_test.go`
    - Remove `internal/infra/audit/logger_prop_test.go`
    - Remove `internal/policy/engine_prop_test.go`
    - Remove `internal/ratelimit/ratelimit_prop_test.go`
    - Remove `internal/retry/handler_prop_test.go`
    - Remove `internal/retry/policy_prop_test.go`
    - Remove `internal/server/shutdown_prop_test.go`
    - Remove `internal/timeout/manager_prop_test.go`
    - _Requirements: 1.1_
  - [x] 8.2 Delete moved benchmark test files
    - Remove `internal/bulkhead/bulkhead_bench_test.go`
    - Remove `internal/circuitbreaker/breaker_bench_test.go`
    - Remove `internal/ratelimit/ratelimit_bench_test.go`
    - Remove `internal/retry/handler_bench_test.go`
    - _Requirements: 1.1_
  - [x] 8.3 Delete moved unit and integration test files
    - Remove `internal/infra/metrics/histogram_test.go`
    - Remove `internal/infra/redis/client_integration_test.go`
    - _Requirements: 1.1_
  - [x] 8.4 Delete internal/testutil/ directory
    - Remove `internal/testutil/` directory entirely
    - _Requirements: 1.1_

- [x] 9. Verify line count constraints
  - [x] 9.1 Check file line counts
    - Verify no test file exceeds 400 lines
    - Split files if necessary
    - _Requirements: 2.1_
  - [x] 9.2 Check function line counts
    - Verify no test function exceeds 50 lines
    - Refactor if necessary
    - _Requirements: 2.2_

- [x] 10. Write property tests for reorganization validation
  - [x] 10.1 Write property test for file location correctness
    - **Property 1: Test File Location Correctness**
    - **Validates: Requirements 1.1**
    - Created tests/property/structure_prop_test.go
  - [x] 10.2 Write property test for test type directory mapping
    - **Property 2: Test Type Directory Mapping**
    - **Validates: Requirements 1.2, 2.3**
    - Verifies all test directories exist and no mixed test types

- [x] 11. Final Checkpoint - Verify all tests pass

  - Ensure all tests pass, ask the user if questions arise.

## Summary

**Reorganization Complete** - December 17, 2025

### Test Directory Structure:
```
platform/resilience-service/tests/
├── property/       # Property-based tests (*_prop_test.go)
├── benchmark/      # Benchmark tests (*_bench_test.go)
├── unit/           # Unit tests
├── integration/    # Integration tests
└── testutil/       # Test utilities and generators
```

### Files Moved:
- 13 property test files → tests/property/
- 4 benchmark test files → tests/benchmark/
- 1 unit test file → tests/unit/
- 1 integration test file → tests/integration/
- 2 testutil files → tests/testutil/

### Property Tests Added:
- structure_prop_test.go - Validates test file organization
