# Requirements Document

## Introduction

This document specifies the requirements for modernizing the `/platform/resilience-service` to state-of-the-art December 2024 standards. The modernization focuses on upgrading Go version, eliminating code redundancy, centralizing shared logic, applying generics where beneficial, and ensuring security and performance best practices.

## Glossary

- **Resilience Service**: The centralized resilience layer providing circuit breaker, retry, timeout, rate limiting, and bulkhead patterns
- **Circuit Breaker**: A pattern that prevents cascading failures by stopping requests to failing services
- **Bulkhead**: A pattern that isolates failures through concurrency limiting
- **Rate Limiter**: A pattern that controls request throughput using token bucket or sliding window algorithms
- **Event Emitter**: Component responsible for emitting resilience events for observability
- **Policy Engine**: Component managing hot-reloadable resilience policies
- **PBT**: Property-Based Testing using gopter library
- **Generics**: Go type parameters enabling type-safe reusable code
- **Iter Package**: Go 1.23 iterator functions for sequence traversal

## Requirements

### Requirement 1: Go Version and Dependency Modernization

**User Story:** As a platform engineer, I want the resilience service to use the latest stable Go version and dependencies, so that I benefit from performance improvements, security patches, and modern language features.

#### Acceptance Criteria

1. WHEN the service is built THEN the system SHALL use Go 1.23 or later as specified in go.mod
2. WHEN dependencies are resolved THEN the system SHALL use go-redis/v9 version 9.7.0 or later
3. WHEN OpenTelemetry is configured THEN the system SHALL use otel SDK version 1.32.0 or later with semantic conventions v1.34.0
4. WHEN gRPC is used THEN the system SHALL use grpc-go version 1.68.0 or later
5. WHEN protobuf is used THEN the system SHALL use google.golang.org/protobuf version 1.35.0 or later

### Requirement 2: Event ID Generation Centralization

**User Story:** As a developer, I want event ID generation to be centralized in a single location, so that I avoid code duplication and ensure consistent ID format across all components.

#### Acceptance Criteria

1. WHEN an event ID is generated THEN the system SHALL use a single centralized function from the domain package
2. WHEN the centralized function is called THEN the system SHALL return a unique identifier using a consistent format
3. WHEN any component needs an event ID THEN the component SHALL import and use the centralized function
4. THE system SHALL remove all duplicate generateEventID functions from circuitbreaker, ratelimit, retry, timeout, bulkhead, and health packages

### Requirement 3: Correlation Function Centralization

**User Story:** As a developer, I want correlation function handling to be centralized, so that all components use consistent correlation ID patterns.

#### Acceptance Criteria

1. WHEN a component requires a correlation function THEN the system SHALL use a centralized type definition from the domain package
2. WHEN a nil correlation function is provided THEN the system SHALL use a centralized default function that returns an empty string
3. THE system SHALL define a CorrelationFunc type alias in the domain package
4. THE system SHALL provide a DefaultCorrelationFunc function in the domain package

### Requirement 4: Event Emission Pattern Centralization

**User Story:** As a developer, I want event emission logic to follow a consistent pattern, so that observability is uniform across all resilience components.

#### Acceptance Criteria

1. WHEN a resilience event is emitted THEN the system SHALL use a centralized helper function that handles nil emitter checks
2. WHEN the event emitter is nil THEN the centralized helper SHALL return without error
3. THE system SHALL provide an EmitEvent helper function in the domain package
4. THE system SHALL remove duplicate nil-check patterns from all component event emission methods

### Requirement 5: Generic Rate Limiter Interface

**User Story:** As a developer, I want rate limiter implementations to use a generic factory pattern, so that I can easily switch between token bucket and sliding window algorithms.

#### Acceptance Criteria

1. WHEN a rate limiter is created THEN the system SHALL use a factory function that accepts algorithm configuration
2. WHEN the algorithm is TokenBucket THEN the factory SHALL return a TokenBucket implementation
3. WHEN the algorithm is SlidingWindow THEN the factory SHALL return a SlidingWindow implementation
4. THE system SHALL define a RateLimiterFactory function in the ratelimit package
5. WHEN an unknown algorithm is specified THEN the factory SHALL return an error

### Requirement 6: Configuration Validation Centralization

**User Story:** As a developer, I want configuration validation to be centralized in the domain package, so that validation rules are consistent and not duplicated.

#### Acceptance Criteria

1. WHEN a CircuitBreakerConfig is validated THEN the system SHALL use a centralized Validate method on the config struct
2. WHEN a RetryConfig is validated THEN the system SHALL use a centralized Validate method on the config struct
3. WHEN a TimeoutConfig is validated THEN the system SHALL use a centralized Validate method on the config struct
4. WHEN a RateLimitConfig is validated THEN the system SHALL use a centralized Validate method on the config struct
5. WHEN a BulkheadConfig is validated THEN the system SHALL use a centralized Validate method on the config struct
6. THE system SHALL remove duplicate validation logic from the policy engine

### Requirement 7: Serialization Round-Trip Consistency

**User Story:** As a developer, I want all serialization to support round-trip consistency, so that data integrity is preserved when persisting and loading state.

#### Acceptance Criteria

1. WHEN a CircuitBreakerState is serialized and deserialized THEN the system SHALL produce an equivalent state
2. WHEN a ResiliencePolicy is serialized and deserialized THEN the system SHALL produce an equivalent policy
3. WHEN a RetryConfig is serialized and deserialized THEN the system SHALL produce an equivalent config
4. THE system SHALL provide Marshal and Unmarshal functions for all persistable types
5. THE system SHALL use consistent time format (RFC3339Nano) across all serialization

### Requirement 8: Go 1.23 Iterator Support

**User Story:** As a developer, I want the service to leverage Go 1.23 iterators where beneficial, so that collection traversal is more idiomatic and efficient.

#### Acceptance Criteria

1. WHEN iterating over policies THEN the system SHALL provide an iter.Seq function for policy iteration
2. WHEN iterating over health services THEN the system SHALL provide an iter.Seq function for service iteration
3. WHEN iterating over bulkhead partitions THEN the system SHALL provide an iter.Seq2 function for partition iteration
4. THE system SHALL import the iter package from the standard library

### Requirement 9: Structured Logging with slog

**User Story:** As a platform engineer, I want the service to use Go's standard slog package consistently, so that logging is structured, performant, and follows modern Go practices.

#### Acceptance Criteria

1. WHEN logging is performed THEN the system SHALL use log/slog package exclusively
2. WHEN a log entry is created THEN the system SHALL include structured attributes using slog.Attr
3. WHEN errors are logged THEN the system SHALL include the error using slog.Any("error", err)
4. THE system SHALL configure slog with JSON handler for production use
5. THE system SHALL support configurable log levels via environment variable

### Requirement 10: Security Hardening

**User Story:** As a security engineer, I want the resilience service to follow security best practices, so that the service is protected against common vulnerabilities.

#### Acceptance Criteria

1. WHEN Redis connection is established THEN the system SHALL support TLS encryption
2. WHEN gRPC server is started THEN the system SHALL support mTLS authentication
3. WHEN configuration is loaded THEN the system SHALL validate all inputs against allowlists
4. WHEN secrets are used THEN the system SHALL read them from environment variables only
5. THE system SHALL NOT log sensitive information including passwords, tokens, or keys

### Requirement 11: Test Generator Modernization

**User Story:** As a developer, I want test generators to use Go 1.23 features and be centralized, so that property-based tests are consistent and maintainable.

#### Acceptance Criteria

1. WHEN generating test data THEN the system SHALL use centralized generators from testutil package
2. WHEN a new domain type is added THEN the testutil package SHALL provide a corresponding generator
3. THE system SHALL provide generators for all domain configuration types
4. THE system SHALL provide generators for all domain state types
5. WHEN generators are used THEN the system SHALL produce valid configurations that pass validation

### Requirement 12: Error Handling Consistency

**User Story:** As a developer, I want error handling to follow consistent patterns, so that errors are properly typed, wrapped, and traceable.

#### Acceptance Criteria

1. WHEN a domain error is created THEN the system SHALL use the ResilienceError type
2. WHEN an error is wrapped THEN the system SHALL preserve the error chain using fmt.Errorf with %w
3. WHEN a gRPC error is returned THEN the system SHALL map domain errors to appropriate gRPC status codes
4. THE system SHALL provide error constructors for all error codes in the domain package
5. WHEN errors are compared THEN the system SHALL use errors.Is or errors.As

### Requirement 13: Metrics and Observability Centralization

**User Story:** As a platform engineer, I want metrics collection to be centralized, so that observability is consistent and metrics are not duplicated.

#### Acceptance Criteria

1. WHEN metrics are recorded THEN the system SHALL use a centralized metrics registry
2. WHEN a resilience operation completes THEN the system SHALL record latency, success, and failure metrics
3. WHEN circuit breaker state changes THEN the system SHALL emit a metric with the new state
4. THE system SHALL use OpenTelemetry metrics API for all metric recording
5. THE system SHALL provide a centralized MetricsRecorder interface in the infra package

### Requirement 14: Graceful Shutdown Enhancement

**User Story:** As a platform engineer, I want graceful shutdown to properly drain all in-flight requests, so that no requests are lost during deployment.

#### Acceptance Criteria

1. WHEN shutdown is initiated THEN the system SHALL stop accepting new requests immediately
2. WHEN shutdown is initiated THEN the system SHALL wait for in-flight requests to complete up to the configured timeout
3. WHEN the drain timeout is exceeded THEN the system SHALL force shutdown and log remaining in-flight count
4. THE system SHALL emit a health event when shutdown is initiated
5. THE system SHALL close all infrastructure connections (Redis, OTEL) during shutdown

### Requirement 15: Policy Engine Hot-Reload Improvement

**User Story:** As a platform engineer, I want policy hot-reload to be more robust, so that configuration changes are applied without service restart.

#### Acceptance Criteria

1. WHEN a policy file changes THEN the system SHALL detect the change within the configured reload interval
2. WHEN a policy is reloaded THEN the system SHALL validate the policy before applying
3. IF policy validation fails THEN the system SHALL keep the existing policy and log the error
4. WHEN a policy is successfully reloaded THEN the system SHALL emit a PolicyUpdated event
5. THE system SHALL support both YAML and JSON policy file formats
