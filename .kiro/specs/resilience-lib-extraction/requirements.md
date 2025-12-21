# Requirements Document

## Introduction

This document specifies the requirements for extracting reusable code from the `platform/resilience-service` microservice into shared libraries under `libs/go/`. The goal is to maximize code reuse across the monorepo while maintaining service autonomy and ensuring zero breaking changes to the existing resilience-service.

## Glossary

- **Resilience Service**: The microservice at `platform/resilience-service` that provides centralized resilience patterns (circuit breaker, retry, rate limiting, bulkhead, timeout) for the auth platform.
- **libs/go/resilience**: Existing shared library containing resilience primitives (config types, events, correlation, time helpers).
- **libs/go/error**: Existing shared library containing error types and gRPC error mapping.
- **Extraction Candidate**: Code within the resilience-service that could be moved to a shared library for cross-service reuse.
- **Domain Primitives**: Core types like IDs, timestamps, enums, and value objects.
- **Resilience Pattern**: A design pattern for building fault-tolerant systems (circuit breaker, retry, rate limiting, bulkhead, timeout).
- **Property-Based Testing (PBT)**: Testing methodology that verifies properties hold across many randomly generated inputs.
- **Round-Trip Property**: A correctness property where serializing then deserializing (or vice versa) produces an equivalent value.

## Requirements

### Requirement 1

**User Story:** As a platform developer, I want resilience pattern implementations extracted to shared libraries, so that other services can use battle-tested circuit breaker, retry, rate limiting, bulkhead, and timeout implementations without duplicating code.

#### Acceptance Criteria

1. WHEN the circuit breaker implementation is extracted THEN the system SHALL provide a reusable `CircuitBreaker` interface and default implementation in `libs/go/resilience/circuitbreaker/`.
2. WHEN the retry handler implementation is extracted THEN the system SHALL provide a reusable `RetryHandler` interface and default implementation in `libs/go/resilience/retry/`.
3. WHEN the rate limiter implementations are extracted THEN the system SHALL provide reusable `TokenBucket` and `SlidingWindow` implementations in `libs/go/resilience/ratelimit/`.
4. WHEN the bulkhead implementation is extracted THEN the system SHALL provide a reusable `Bulkhead` interface and default implementation in `libs/go/resilience/bulkhead/`.
5. WHEN the timeout manager implementation is extracted THEN the system SHALL provide a reusable `TimeoutManager` interface and default implementation in `libs/go/resilience/timeout/`.

### Requirement 2

**User Story:** As a platform developer, I want health aggregation utilities extracted to a shared library, so that other services can implement consistent health checking and aggregation patterns.

#### Acceptance Criteria

1. WHEN the health aggregator is extracted THEN the system SHALL provide a reusable `HealthAggregator` interface and implementation in `libs/go/resilience/health/`.
2. WHEN health types are extracted THEN the system SHALL provide `HealthStatus`, `ServiceHealth`, and `AggregatedHealth` types in the shared library.
3. WHEN the health checker interface is extracted THEN the system SHALL provide a `HealthChecker` interface that services can implement.

### Requirement 3

**User Story:** As a platform developer, I want graceful shutdown utilities extracted to a shared library, so that all services can implement consistent request draining during shutdown.

#### Acceptance Criteria

1. WHEN the graceful shutdown manager is extracted THEN the system SHALL provide a reusable `GracefulShutdown` implementation in `libs/go/resilience/shutdown/`.
2. WHEN the shutdown manager is used THEN the system SHALL track in-flight requests and drain them before completing shutdown.
3. WHEN shutdown is initiated THEN the system SHALL respect the configured drain timeout.

### Requirement 4

**User Story:** As a platform developer, I want test utilities and generators extracted to a shared library, so that other services can use consistent property-based testing patterns for resilience configurations.

#### Acceptance Criteria

1. WHEN test generators are extracted THEN the system SHALL provide gopter generators for all resilience configuration types in `libs/go/resilience/testutil/`.
2. WHEN generators are used THEN the system SHALL produce valid configurations that pass validation.
3. WHEN the mock event emitter is extracted THEN the system SHALL provide a reusable `MockEventEmitter` for testing event emission.

### Requirement 5

**User Story:** As a platform developer, I want serialization utilities extracted to a shared library, so that circuit breaker state and retry policies can be consistently serialized across services.

#### Acceptance Criteria

1. WHEN circuit breaker state serialization is extracted THEN the system SHALL provide `MarshalState` and `UnmarshalState` functions in the shared library.
2. WHEN retry policy serialization is extracted THEN the system SHALL provide `ParsePolicy`, `MarshalPolicy`, and `ValidatePolicy` functions in the shared library.
3. WHEN serialization round-trips are performed THEN the system SHALL preserve all data without loss.

### Requirement 6

**User Story:** As a platform developer, I want the extraction to maintain backward compatibility, so that the existing resilience-service continues to work without modification after extraction.

#### Acceptance Criteria

1. WHEN code is extracted to shared libraries THEN the resilience-service SHALL continue to compile and pass all existing tests.
2. WHEN domain types are already re-exported from libs THEN the system SHALL NOT duplicate those types in new extractions.
3. WHEN new library packages are created THEN the system SHALL update the resilience-service to import from the new locations.
4. WHEN extraction is complete THEN the system SHALL remove duplicated code from the resilience-service internal packages.

### Requirement 7

**User Story:** As a platform developer, I want the random source abstraction extracted to a shared library, so that other services can use deterministic random sources for testing retry jitter and other randomized behaviors.

#### Acceptance Criteria

1. WHEN the random source interface is extracted THEN the system SHALL provide `RandSource`, `CryptoRandSource`, `DeterministicRandSource`, and `FixedRandSource` in `libs/go/resilience/rand/`.
2. WHEN the crypto random source is used THEN the system SHALL provide cryptographically seeded random numbers.
3. WHEN the deterministic random source is used THEN the system SHALL provide reproducible random sequences for testing.

### Requirement 8

**User Story:** As a platform developer, I want policy event types extracted to a shared library, so that services can emit and consume policy change events consistently.

#### Acceptance Criteria

1. WHEN policy event types are extracted THEN the system SHALL provide `PolicyEventType` and `PolicyEvent` types in `libs/go/resilience/`.
2. WHEN policy events are emitted THEN the system SHALL support `PolicyCreated`, `PolicyUpdated`, and `PolicyDeleted` event types.
