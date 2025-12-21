# Requirements Document

## Introduction

This document specifies the requirements for the comprehensive State of the Art modernization of `platform/resilience-service` for December 2024/2025 standards. Building on the previous modernization phase that fixed re-export patterns, this phase focuses on deeper redundancy elimination, dependency upgrades, security hardening, performance optimization, and architectural improvements.

## Glossary

- **Resilience Service**: The microservice providing circuit breaker, rate limiting, retry, timeout, and bulkhead patterns
- **libs/go/resilience**: Shared library containing reusable resilience primitives
- **libs/go/error**: Shared library containing error types and gRPC mapping
- **gopter**: Property-based testing library for Go (v0.2.11)
- **go-redis/v9**: Redis client library for Go
- **OpenTelemetry**: Observability framework for distributed tracing and metrics
- **grpc-go**: gRPC framework for Go
- **iter.Seq/iter.Seq2**: Go 1.23 iterator types for custom sequences
- **State of the Art**: Latest stable, officially recommended patterns and versions as of December 2024

## Requirements

### Requirement 1

**User Story:** As a developer, I want all dependencies to use the latest stable versions, so that security vulnerabilities are minimized and performance is optimal.

#### Acceptance Criteria

1. THE system SHALL use grpc-go version 1.77.0 or later (latest stable as of December 2024)
2. THE system SHALL use go-redis/v9 version 9.7.0 or later with maintenance notification support
3. THE system SHALL use OpenTelemetry Go SDK version 1.32.0 or later
4. THE system SHALL use protobuf version 1.35.2 or later
5. WHEN updating dependencies THEN the system SHALL verify all transitive dependencies are compatible

### Requirement 2

**User Story:** As a developer, I want consistent timestamp usage across all components, so that event ordering and debugging are reliable.

#### Acceptance Criteria

1. WHEN creating timestamps for events THEN the system SHALL use domain.NowUTC() exclusively
2. WHEN creating timestamps in circuitbreaker.breaker.go THEN the system SHALL use domain.NowUTC() instead of time.Now()
3. WHEN creating timestamps in retry.handler.go THEN the system SHALL use domain.NowUTC() instead of time.Now()
4. WHEN creating timestamps in bulkhead.bulkhead.go THEN the system SHALL use domain.NowUTC() instead of time.Now()
5. WHEN creating timestamps in health.aggregator.go THEN the system SHALL use domain.NowUTC() instead of time.Now()

### Requirement 3

**User Story:** As a developer, I want all configuration validation to be centralized, so that validation logic is not duplicated.

#### Acceptance Criteria

1. THE system SHALL have exactly one Validate() method per configuration type in libs/go/resilience
2. WHEN validating configurations THEN the system SHALL delegate to the centralized Validate() methods
3. THE system SHALL NOT have duplicate validation logic in internal/domain or other packages

### Requirement 4

**User Story:** As a developer, I want all error creation to use centralized constructors, so that error handling is consistent.

#### Acceptance Criteria

1. WHEN creating resilience errors THEN the system SHALL use domain.New*Error() constructors exclusively
2. THE system SHALL NOT create ResilienceError structs directly outside of libs/go/error
3. WHEN checking error types THEN the system SHALL use domain.Is*() functions exclusively

### Requirement 5

**User Story:** As a developer, I want Go 1.23 iter patterns used consistently, so that the codebase follows modern Go idioms.

#### Acceptance Criteria

1. WHEN iterating over collections THEN the system SHALL use iter.Seq or iter.Seq2 where appropriate
2. THE system SHALL use range-over-function syntax for custom iterators
3. WHEN exposing collection iteration THEN the system SHALL prefer iterator functions over slice returns

### Requirement 6

**User Story:** As a developer, I want all files under 400 lines, so that the codebase is maintainable.

#### Acceptance Criteria

1. THE system SHALL have no source files exceeding 400 non-blank lines
2. WHEN a file exceeds 400 lines THEN the system SHALL split it into smaller files with clear responsibilities

### Requirement 7

**User Story:** As a developer, I want property-based tests to cover all correctness properties, so that the system behavior is verified across many inputs.

#### Acceptance Criteria

1. THE system SHALL have property-based tests for all configuration validation
2. THE system SHALL have property-based tests for serialization round-trips
3. THE system SHALL have property-based tests for state machine transitions
4. THE system SHALL run property tests with minimum 100 iterations

### Requirement 8

**User Story:** As a developer, I want the random number generation to be secure and testable, so that retry jitter is unpredictable in production but deterministic in tests.

#### Acceptance Criteria

1. WHEN generating random values for jitter THEN the system SHALL use crypto/rand for seeding in production
2. WHEN testing retry behavior THEN the system SHALL support injecting a deterministic random source
3. THE system SHALL NOT use math/rand with time-based seeds in production code

### Requirement 9

**User Story:** As a developer, I want structured logging with correlation IDs, so that distributed tracing is effective.

#### Acceptance Criteria

1. WHEN logging events THEN the system SHALL include correlation IDs from context
2. WHEN emitting events THEN the system SHALL use structured JSON format
3. THE system SHALL propagate trace context through all resilience operations

### Requirement 10

**User Story:** As a developer, I want the gRPC error mapping to be complete and consistent, so that clients receive appropriate error codes.

#### Acceptance Criteria

1. WHEN mapping resilience errors to gRPC THEN the system SHALL use appropriate status codes
2. THE system SHALL map ErrCircuitOpen to codes.Unavailable
3. THE system SHALL map ErrRateLimitExceeded to codes.ResourceExhausted
4. THE system SHALL map ErrTimeout to codes.DeadlineExceeded
5. THE system SHALL map ErrBulkheadFull to codes.ResourceExhausted
