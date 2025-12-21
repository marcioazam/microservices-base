# Requirements Document

## Introduction

This specification defines the state-of-the-art modernization of the `platform/resilience-service` for December 2025. The modernization focuses on eliminating redundancies, centralizing logic, adopting modern Go 1.25+ patterns, and achieving a clean architecture with strict separation between source code and tests.

The resilience service provides circuit breaker, retry, timeout, rate limiting, and bulkhead patterns for protecting distributed services. This modernization will consolidate duplicated implementations, leverage the centralized `libs/go` packages, and ensure the architecture follows current best practices.

## Glossary

- **Resilience_Service**: The platform service providing resilience patterns (circuit breaker, retry, timeout, rate limit, bulkhead) for protecting distributed operations
- **Policy_Engine**: The centralized component managing resilience policy lifecycle (CRUD, validation, hot-reload)
- **Failsafe_Executor**: The infrastructure component implementing resilience patterns using failsafe-go library
- **Circuit_Breaker**: A pattern that prevents cascading failures by opening when failure threshold is exceeded
- **Rate_Limiter**: A pattern that controls request throughput using token bucket or sliding window algorithms
- **Bulkhead**: A pattern that isolates failures by limiting concurrent executions per partition
- **Event_Emitter**: The component responsible for publishing domain events and audit logs
- **Policy_Repository**: The persistence layer for resilience policies using Redis
- **Health_Aggregator**: The component aggregating health status from multiple checkers
- **Libs_Go**: The centralized shared library at `libs/go/src` containing reusable resilience components

## Requirements

### Requirement 1: Eliminate Redundant Domain Types

**User Story:** As a maintainer, I want domain types to exist in exactly one location, so that changes propagate consistently and there is no divergence between implementations.

#### Acceptance Criteria

1. THE Resilience_Service SHALL use domain types exclusively from `libs/go/src/resilience` without local re-definitions
2. WHEN a domain type is needed, THE Resilience_Service SHALL import it from the centralized library rather than defining a local wrapper
3. THE Resilience_Service SHALL remove all backward-compatibility re-export files that duplicate library types
4. WHEN error types are needed, THE Resilience_Service SHALL use `libs/go/src/errors` directly without local wrappers
5. THE Resilience_Service SHALL consolidate `internal/domain/circuit_breaker.go` types into library imports
6. THE Resilience_Service SHALL consolidate `internal/domain/errors.go` types into library imports

### Requirement 2: Eliminate Duplicate Configuration Structures

**User Story:** As a developer, I want configuration types defined once, so that validation rules and defaults are consistent across the codebase.

#### Acceptance Criteria

1. THE Resilience_Service SHALL have exactly one configuration package at `internal/infrastructure/config`
2. WHEN configuration is loaded, THE Config_Loader SHALL use viper with struct validation tags
3. THE Resilience_Service SHALL remove the duplicate `internal/config/config.go` file
4. THE Configuration SHALL use `libs/go/src/resilience` types for resilience-specific configs (CircuitBreakerConfig, RetryConfig, etc.)
5. WHEN environment-specific validation is needed, THE Config_Validator SHALL enforce TLS in production environments
6. THE Configuration SHALL support hot-reload for policy files without service restart

### Requirement 3: Centralize Event Emission

**User Story:** As an operator, I want all events emitted through a single mechanism, so that observability is consistent and audit trails are complete.

#### Acceptance Criteria

1. THE Resilience_Service SHALL have exactly one EventEmitter implementation in `internal/infrastructure/observability`
2. WHEN events are emitted, THE Event_Emitter SHALL use OpenTelemetry for tracing and metrics
3. THE Resilience_Service SHALL remove duplicate mockEmitter implementations from test files
4. WHEN audit events are needed, THE Event_Emitter SHALL emit structured JSON logs with correlation IDs
5. THE Event_Emitter SHALL support both resilience events and audit events through a unified interface
6. WHEN the emitter is nil, THE Event_Emitter helper functions SHALL handle gracefully without panics

### Requirement 4: Consolidate Resilience Pattern Implementations

**User Story:** As a developer, I want resilience patterns implemented once using failsafe-go, so that behavior is consistent and maintenance is simplified.

#### Acceptance Criteria

1. THE Resilience_Service SHALL use failsafe-go as the single resilience implementation library
2. WHEN circuit breaker is needed, THE Failsafe_Executor SHALL create failsafe-go circuit breaker policies
3. WHEN retry is needed, THE Failsafe_Executor SHALL create failsafe-go retry policies with exponential backoff
4. WHEN timeout is needed, THE Failsafe_Executor SHALL create failsafe-go timeout policies
5. WHEN rate limiting is needed, THE Failsafe_Executor SHALL create failsafe-go rate limiter policies
6. WHEN bulkhead is needed, THE Failsafe_Executor SHALL create failsafe-go bulkhead policies
7. THE Resilience_Service SHALL remove local implementations in `internal/bulkhead`, `internal/retry`, `internal/ratelimit`, `internal/timeout` that duplicate failsafe-go functionality
8. THE Failsafe_Executor SHALL compose multiple policies into a single executor per resilience policy

### Requirement 5: Modernize Dependency Injection

**User Story:** As a developer, I want dependency injection using uber-go/fx, so that component wiring is explicit and testable.

#### Acceptance Criteria

1. THE Resilience_Service SHALL use uber-go/fx for dependency injection
2. WHEN components are created, THE FX_Module SHALL provide them through fx.Provide
3. THE Resilience_Service SHALL remove mock implementations from main.go (mockMetricsRecorder, mockPolicyValidator)
4. WHEN mocks are needed for testing, THE Test_Suite SHALL define them in test files only
5. THE FX_Module SHALL wire all infrastructure components (Redis, OpenTelemetry, gRPC server)
6. WHEN lifecycle hooks are needed, THE FX_Module SHALL use fx.Lifecycle for startup and shutdown

### Requirement 6: Separate Source Code from Tests

**User Story:** As a developer, I want tests in a dedicated directory structure, so that source code is clean and test organization is clear.

#### Acceptance Criteria

1. THE Resilience_Service SHALL have all tests under `tests/` directory
2. WHEN property tests exist in source directories, THE Migration SHALL move them to `tests/property/`
3. THE Resilience_Service SHALL remove `*_prop_test.go` files from `internal/` subdirectories
4. THE Test_Structure SHALL mirror source structure: `tests/unit/`, `tests/integration/`, `tests/property/`, `tests/benchmark/`
5. WHEN test utilities are needed, THE Test_Suite SHALL use `tests/testutil/` for shared helpers
6. THE Resilience_Service SHALL maintain test coverage above 80% after migration

### Requirement 7: Modernize Go Version and Dependencies

**User Story:** As a maintainer, I want the service using Go 1.25+ features and latest stable dependencies, so that we benefit from performance improvements and security fixes.

#### Acceptance Criteria

1. THE Resilience_Service SHALL use Go 1.25.5 or later
2. WHEN iterators are needed, THE Resilience_Service SHALL use Go 1.23+ iter.Seq patterns
3. THE Resilience_Service SHALL use log/slog for structured logging (Go 1.21+)
4. WHEN OpenTelemetry is used, THE Resilience_Service SHALL use v1.39.0 or later
5. WHEN Redis client is used, THE Resilience_Service SHALL use go-redis/v9 v9.17.0 or later
6. WHEN gRPC is used, THE Resilience_Service SHALL use grpc v1.77.0 or later
7. THE Resilience_Service SHALL use failsafe-go v0.7.0 or later for resilience patterns

### Requirement 8: Clean Architecture Layers

**User Story:** As an architect, I want clear separation between domain, application, infrastructure, and presentation layers, so that the codebase is maintainable and testable.

#### Acceptance Criteria

1. THE Domain_Layer SHALL contain only pure business logic with no external dependencies
2. THE Application_Layer SHALL orchestrate domain operations and define use cases
3. THE Infrastructure_Layer SHALL implement interfaces defined in domain layer
4. THE Presentation_Layer SHALL handle gRPC request/response transformation
5. WHEN dependencies flow, THE Architecture SHALL ensure dependencies point inward (presentation → application → domain ← infrastructure)
6. THE Domain_Layer SHALL define interfaces that infrastructure implements
7. THE Resilience_Service SHALL remove the duplicate `internal/infra` directory (consolidate with `internal/infrastructure`)

### Requirement 9: Policy Management Centralization

**User Story:** As an operator, I want policy management through a single engine, so that policy lifecycle is consistent and auditable.

#### Acceptance Criteria

1. THE Policy_Engine SHALL be the single source of truth for policy CRUD operations
2. WHEN policies are loaded, THE Policy_Engine SHALL support YAML and JSON formats
3. WHEN policies change, THE Policy_Engine SHALL emit PolicyCreated, PolicyUpdated, PolicyDeleted events
4. THE Policy_Engine SHALL validate policies before persisting using centralized validation rules
5. WHEN hot-reload is enabled, THE Policy_Engine SHALL watch configuration files for changes
6. THE Policy_Engine SHALL prevent path traversal attacks when loading policy files
7. WHEN policies are retrieved, THE Policy_Engine SHALL support iteration using Go 1.23+ iter.Seq

### Requirement 10: Health Check Aggregation

**User Story:** As an operator, I want aggregated health status from all components, so that I can monitor service health comprehensively.

#### Acceptance Criteria

1. THE Health_Aggregator SHALL collect health status from all registered checkers
2. WHEN any checker reports unhealthy, THE Health_Aggregator SHALL report overall status as unhealthy
3. THE Health_Aggregator SHALL support Redis, gRPC server, and policy engine health checks
4. WHEN health is queried, THE Health_Aggregator SHALL return detailed status per component
5. THE Health_Aggregator SHALL use `libs/go/src/server/health` for health check interfaces
6. WHEN timeouts occur during health checks, THE Health_Aggregator SHALL report degraded status

### Requirement 11: Metrics and Observability

**User Story:** As an operator, I want comprehensive metrics for all resilience operations, so that I can monitor and alert on service behavior.

#### Acceptance Criteria

1. THE Metrics_Recorder SHALL record execution duration, success/failure counts per policy
2. WHEN circuit breaker state changes, THE Metrics_Recorder SHALL record state transitions
3. WHEN retry attempts occur, THE Metrics_Recorder SHALL record attempt counts
4. WHEN rate limits are hit, THE Metrics_Recorder SHALL record limit events
5. WHEN bulkhead queues, THE Metrics_Recorder SHALL record queue depth
6. THE Metrics_Recorder SHALL use OpenTelemetry metrics API for all recordings
7. THE Resilience_Service SHALL expose metrics endpoint for Prometheus scraping

### Requirement 12: Security Hardening

**User Story:** As a security engineer, I want the service hardened against common vulnerabilities, so that it meets production security requirements.

#### Acceptance Criteria

1. THE Resilience_Service SHALL enforce TLS for Redis connections in production
2. WHEN loading policy files, THE Policy_Engine SHALL validate paths to prevent traversal attacks
3. THE Resilience_Service SHALL not log sensitive data (passwords, tokens)
4. WHEN gRPC is used, THE Presentation_Layer SHALL support mTLS authentication
5. THE Configuration SHALL load secrets from environment variables, not config files
6. WHEN audit events are emitted, THE Event_Emitter SHALL include SPIFFE IDs for identity
7. THE Resilience_Service SHALL validate all inputs before processing

### Requirement 13: File Size Compliance

**User Story:** As a maintainer, I want all files under 400 lines, so that code is readable and maintainable.

#### Acceptance Criteria

1. THE Resilience_Service SHALL have no source files exceeding 400 non-blank lines
2. WHEN a file exceeds 400 lines, THE Refactoring SHALL split it into smaller focused modules
3. THE Test_Files SHALL also comply with the 400-line limit
4. WHEN splitting files, THE Refactoring SHALL maintain single responsibility principle
5. THE CI_Pipeline SHALL fail builds if any file exceeds 400 lines

### Requirement 14: Remove Legacy and Transitional Code

**User Story:** As a maintainer, I want all legacy patterns removed, so that the codebase uses only modern approaches.

#### Acceptance Criteria

1. THE Resilience_Service SHALL remove all "backward compatibility" re-export patterns
2. WHEN git merge conflicts exist (<<<<<<< markers), THE Cleanup SHALL resolve them
3. THE Resilience_Service SHALL remove `.gitkeep` files from directories with content
4. WHEN deprecated APIs are used, THE Modernization SHALL replace them with current alternatives
5. THE Resilience_Service SHALL remove unused imports and dead code
6. THE Resilience_Service SHALL resolve the merge conflict in `internal/timeout/manager.go`
