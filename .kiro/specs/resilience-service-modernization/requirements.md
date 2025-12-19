# Requirements Document

## Introduction

This specification defines the modernization requirements for the `platform/resilience-service` Go microservice. The modernization effort focuses on upgrading to state-of-the-art December 2025 standards, eliminating code redundancy, centralizing shared logic, and addressing security vulnerabilities. The resilience service provides circuit breaker, retry, rate limiting, bulkhead, and timeout patterns for the Auth Platform.

## Glossary

- **Resilience Service**: The Go microservice providing resilience patterns (circuit breaker, retry, rate limiting, bulkhead, timeout) for the Auth Platform
- **Circuit Breaker**: A pattern that prevents cascading failures by stopping requests to failing services
- **Rate Limiter**: A component that controls request throughput using token bucket or sliding window algorithms
- **Bulkhead**: A pattern providing isolation through concurrency limits
- **Event Emitter**: An interface for emitting resilience events for observability
- **UUID v7**: Time-ordered universally unique identifier defined in RFC 9562
- **OpenTelemetry**: An observability framework for distributed tracing, metrics, and logs
- **go-redis**: The official Redis client library for Go
- **grpc-go**: The Go implementation of gRPC
- **gopter**: A property-based testing library for Go
- **Swiss Tables**: A high-performance hash table implementation introduced in Go 1.24

## Requirements

### Requirement 1: Dependency Modernization

**User Story:** As a platform engineer, I want all dependencies upgraded to their latest stable versions, so that the service benefits from performance improvements, security fixes, and new features.

#### Acceptance Criteria

1. WHEN the service is built THEN the Go version SHALL be 1.25 or higher
2. WHEN the service connects to Redis THEN go-redis version SHALL be v9.17.0 or higher to address CVE-2025-29923
3. WHEN the service exports telemetry THEN OpenTelemetry SDK version SHALL be v1.39.0 or higher
4. WHEN the service handles gRPC requests THEN grpc-go version SHALL be v1.77.0 or higher
5. WHEN dependencies are updated THEN the go.mod file SHALL specify exact versions with checksums in go.sum

### Requirement 2: Code Redundancy Elimination

**User Story:** As a developer, I want duplicated code eliminated and centralized, so that maintenance is simplified and behavior is consistent across all resilience patterns.

#### Acceptance Criteria

1. WHEN generating event IDs THEN the system SHALL use a single centralized `GenerateEventID` function in the domain package
2. WHEN emitting resilience events THEN all components SHALL use a centralized event builder pattern
3. WHEN creating resilience components THEN correlation function defaults SHALL be handled by a single factory function
4. IF duplicate functions exist across packages THEN the system SHALL consolidate them into a single authoritative location
5. WHEN the codebase is analyzed THEN zero duplicate function implementations SHALL exist

### Requirement 3: Event ID Generation Modernization

**User Story:** As a security engineer, I want event IDs to be unpredictable and time-ordered, so that they cannot be guessed and can be sorted chronologically.

#### Acceptance Criteria

1. WHEN generating event IDs THEN the system SHALL use UUID v7 format (RFC 9562)
2. WHEN UUID v7 is generated THEN the ID SHALL contain a 48-bit Unix timestamp in milliseconds
3. WHEN UUID v7 is generated THEN the remaining bits SHALL be cryptographically random
4. WHEN event IDs are compared THEN lexicographic sorting SHALL produce chronological order
5. WHEN serializing event IDs THEN the format SHALL be the standard UUID string representation (36 characters with hyphens)

### Requirement 4: Centralized Event Emission

**User Story:** As a developer, I want a unified event emission pattern, so that all resilience components emit events consistently with proper metadata.

#### Acceptance Criteria

1. WHEN emitting any resilience event THEN the system SHALL use a centralized EventBuilder in the domain package
2. WHEN building an event THEN the EventBuilder SHALL automatically set ID, Timestamp, and Type fields
3. WHEN the event emitter is nil THEN the EventBuilder SHALL handle the nil check internally
4. WHEN emitting events THEN all components SHALL include correlation ID and service name metadata
5. WHEN a new event type is added THEN only the domain package SHALL require modification

### Requirement 5: Go 1.25+ Feature Adoption

**User Story:** As a developer, I want the codebase to leverage Go 1.25+ features, so that the service benefits from improved performance and modern idioms.

#### Acceptance Criteria

1. WHEN the service runs THEN it SHALL benefit from Swiss Tables map implementation (Go 1.24+)
2. WHEN using generic type aliases THEN the code SHALL use Go 1.24+ syntax where applicable
3. WHEN logging THEN the service SHALL use log/slog with structured JSON output
4. WHEN handling JSON THEN the service MAY use the new JSON v2 encoder if performance benefits are measured
5. WHEN building the service THEN no deprecated Go features SHALL be used

### Requirement 6: Security Hardening

**User Story:** As a security engineer, I want all known vulnerabilities addressed and security best practices applied, so that the service is protected against known attack vectors.

#### Acceptance Criteria

1. WHEN connecting to Redis THEN the client SHALL use go-redis v9.17.0+ to fix CVE-2025-29923 (out-of-order response vulnerability)
2. WHEN generating random values THEN the system SHALL use crypto/rand instead of math/rand
3. WHEN running vulnerability scans THEN govulncheck SHALL report zero known vulnerabilities
4. WHEN handling secrets THEN no credentials SHALL be hardcoded in source code
5. WHEN logging THEN no sensitive data (passwords, tokens, PII) SHALL be included in log output

### Requirement 7: OpenTelemetry Integration Enhancement

**User Story:** As an SRE, I want comprehensive observability with the latest OpenTelemetry features, so that I can monitor and troubleshoot the service effectively.

#### Acceptance Criteria

1. WHEN exporting traces THEN the service SHALL use OpenTelemetry SDK v1.39.0+ with improved metric recording performance
2. WHEN creating spans THEN the service SHALL use WithInstrumentationAttributeSet for concurrent-safe attribute handling
3. WHEN emitting metrics THEN the service SHALL benefit from hashing-based map keys optimization
4. WHEN correlating events THEN trace context SHALL be propagated through all resilience operations
5. WHEN the service starts THEN OpenTelemetry providers SHALL be properly initialized and shut down gracefully

### Requirement 8: Test Infrastructure Modernization

**User Story:** As a developer, I want modernized test infrastructure, so that tests are reliable, fast, and provide comprehensive coverage.

#### Acceptance Criteria

1. WHEN running property-based tests THEN gopter SHALL execute a minimum of 100 iterations per property
2. WHEN testing concurrent code THEN the -race flag SHALL be used to detect data races
3. WHEN measuring coverage THEN critical paths SHALL have at least 80% coverage
4. WHEN tests complete THEN all property-based tests SHALL pass consistently
5. WHEN adding new resilience patterns THEN corresponding property-based tests SHALL be required

### Requirement 9: Architecture Consistency

**User Story:** As an architect, I want consistent patterns across all resilience components, so that the codebase is maintainable and predictable.

#### Acceptance Criteria

1. WHEN creating any resilience component THEN the constructor SHALL follow the same Config struct pattern
2. WHEN components need event emission THEN they SHALL receive an EventEmitter through dependency injection
3. WHEN components need correlation IDs THEN they SHALL receive a correlation function through the Config
4. WHEN implementing interfaces THEN all implementations SHALL be in separate packages from their interfaces
5. WHEN adding cross-cutting concerns THEN they SHALL be centralized in the domain or shared packages

### Requirement 10: Backward Compatibility

**User Story:** As a platform engineer, I want the modernization to be backward compatible, so that existing clients continue to work without modification.

#### Acceptance Criteria

1. WHEN the service API is called THEN all existing gRPC endpoints SHALL maintain their signatures
2. WHEN events are emitted THEN the event structure SHALL remain compatible with existing consumers
3. WHEN configuration is loaded THEN existing configuration files SHALL continue to work
4. WHEN the service is deployed THEN existing health checks SHALL continue to function
5. IF breaking changes are unavoidable THEN they SHALL be documented with migration guides
