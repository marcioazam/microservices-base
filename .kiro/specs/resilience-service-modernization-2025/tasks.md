# Implementation Plan

## Phase 1: Dependency Updates

- [x] 1. Update grpc-go to latest stable version



  - [x] 1.1 Update go.mod to use grpc-go v1.77.0

    - Run `go get google.golang.org/grpc@v1.77.0`
    - Update transitive dependencies
    - _Requirements: 1.1, 1.5_

  - [x] 1.2 Run go mod tidy and verify build

    - Execute `go mod tidy`
    - Execute `go build ./...`
    - _Requirements: 1.5_


- [ ] 2. Checkpoint - Verify compilation and tests
  - Ensure all tests pass, ask the user if questions arise.

## Phase 2: Timestamp Consistency

- [x] 3. Replace time.Now() with domain.NowUTC() in circuitbreaker



  - [x] 3.1 Update circuitbreaker/breaker.go timestamp calls

    - Replace `time.Now()` in `New()`, `transitionTo()`, and `emitStateChangeEvent()`
    - Use `domain.NowUTC()` for consistent UTC timestamps
    - _Requirements: 2.2_
  - [x] 3.2 Write property test for timestamp consistency

    - **Property 3: Serialization Round-Trip**
    - **Validates: Requirements 7.2**



- [x] 4. Replace time.Now() with domain.NowUTC() in retry handler

  - [x] 4.1 Update retry/handler.go timestamp calls

    - Replace `time.Now()` in `emitRetryEvent()`
    - _Requirements: 2.3_

- [x] 5. Replace time.Now() with domain.NowUTC() in bulkhead



  - [x] 5.1 Update bulkhead/bulkhead.go timestamp calls

    - Replace `time.Now()` in `emitRejectionEvent()`
    - _Requirements: 2.4_

- [x] 6. Replace time.Now() with domain.NowUTC() in health aggregator



  - [x] 6.1 Update health/aggregator.go timestamp calls

    - Replace `time.Now()` in `GetAggregatedHealth()`, `RegisterService()`, `UpdateHealth()`
    - _Requirements: 2.5_


- [ ] 7. Checkpoint - Verify timestamp consistency
  - Ensure all tests pass, ask the user if questions arise.

## Phase 3: Secure Random Number Generation

- [x] 8. Implement secure random source for retry handler



  - [x] 8.1 Create RandSource interface for testability

    - Add `RandSource` interface to retry package
    - Implement `CryptoRandSource` using crypto/rand
    - Implement `DeterministicRandSource` for testing
    - _Requirements: 8.1, 8.2_

  - [x] 8.2 Update Handler to use RandSource interface

    - Replace `*rand.Rand` with `RandSource` interface
    - Default to `CryptoRandSource` in production
    - _Requirements: 8.1, 8.3_
  - [x] 8.3 Write property test for retry delay bounds


    - **Property 9: Retry Delay Bounds**
    - **Validates: Requirements 8.1**

  - [x] 8.4 Write property test for deterministic retry

    - **Property 10: Deterministic Retry with Fixed Seed**

    - **Validates: Requirements 8.2**

- [ ] 9. Checkpoint - Verify secure randomness
  - Ensure all tests pass, ask the user if questions arise.

## Phase 4: Property-Based Test Coverage

- [x] 10. Verify existing property tests cover correctness properties



  - [x] 10.1 Review and update validation property tests

    - Verify tests cover all config types
    - Ensure minimum 100 iterations
    - **Property 1: Configuration Validation Consistency**
    - **Validates: Requirements 3.2, 7.1**


  - [ ] 10.2 Review and update error property tests
    - Verify error constructor/checker round-trip
    - **Property 2: Error Constructor Type Preservation**
    - **Validates: Requirements 4.1, 4.3**

- [x] 11. Verify gRPC error mapping property tests


  - [x] 11.1 Review grpc_errors_prop_test.go coverage


    - Verify all error codes are tested
    - **Property 5: gRPC Error Mapping Completeness**
    - **Property 6: gRPC Error Code Correctness**
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5**

- [x] 12. Verify iterator and event property tests



  - [x] 12.1 Review structure_prop_test.go for iterator coverage

    - Verify iterator completeness tests
    - **Property 4: Iterator Completeness**
    - **Validates: Requirements 5.1, 5.3**


  - [x] 12.2 Review centralization_prop_test.go for event coverage

    - Verify correlation ID propagation tests
    - Verify JSON serialization tests
    - **Property 7: Correlation ID Propagation**
    - **Property 8: Event JSON Serialization**

    - **Validates: Requirements 9.1, 9.2**

- [ ] 13. Final Checkpoint - Verify all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Phase 5: Documentation and Cleanup

- [x] 14. Update documentation


  - [x] 14.1 Update README.md with new dependency versions


    - Update grpc-go version in dependencies table
    - Document secure randomness changes
    - _Requirements: 1.1_

  - [x] 14.2 Verify all files are under 400 lines

    - Run line count check on all source files
    - _Requirements: 6.1_



- [x] 15. Final validation

  - [x] 15.1 Run full test suite

    - Execute `go test ./...`
    - Execute `go test ./tests/property/... -v`
    - _Requirements: 7.4_

  - [x] 15.2 Run go vet and staticcheck

    - Execute `go vet ./...`
    - Verify no warnings or errors
    - _Requirements: 1.5_
