# Requirements Document

## Introduction

This document specifies the requirements for the State of the Art modernization of the `platform/resilience-service`. The goal is to eliminate all redundancy, ensure zero duplication, and modernize the codebase to December 2024 standards.

## Glossary

- **Resilience Service**: The microservice providing circuit breaker, rate limiting, retry, timeout, and bulkhead patterns
- **libs/go/resilience**: Shared library containing reusable resilience primitives
- **libs/go/error**: Shared library containing error types and gRPC mapping
- **libs/go/validation**: Shared library containing generic validators
- **Re-export**: Pattern of exposing library functions through domain package for backward compatibility

## Requirements

### Requirement 1

**User Story:** As a developer, I want all re-exported functions to be callable, so that the service compiles without errors.

#### Acceptance Criteria

1. WHEN a function is re-exported from libs/go/resilience THEN the domain package SHALL use function wrappers instead of variable assignments
2. WHEN domain.GenerateEventID is called THEN the system SHALL delegate to resilience.GenerateEventID
3. WHEN domain.EnsureCorrelationFunc is called THEN the system SHALL delegate to resilience.EnsureCorrelationFunc
4. WHEN domain.NowUTC is called THEN the system SHALL delegate to resilience.NowUTC

### Requirement 2

**User Story:** As a developer, I want zero redundant code, so that maintenance is simplified.

#### Acceptance Criteria

1. WHEN generating event IDs THEN the system SHALL use only domain.GenerateEventID (no local implementations)
2. WHEN handling nil correlation functions THEN the system SHALL use only domain.EnsureCorrelationFunc
3. WHEN emitting events THEN the system SHALL use only domain.EmitEvent helper
4. WHEN getting current time THEN the system SHALL use only domain.NowUTC

### Requirement 3

**User Story:** As a developer, I want the service to use state-of-the-art dependencies, so that security and performance are optimal.

#### Acceptance Criteria

1. THE system SHALL use Go 1.23 with iter package support
2. THE system SHALL use go-redis/v9 version 9.7.0 or later
3. THE system SHALL use OpenTelemetry Go SDK version 1.32.0 or later
4. THE system SHALL use grpc-go version 1.68.0 or later

### Requirement 4

**User Story:** As a developer, I want centralized validation, so that validation logic is not duplicated.

#### Acceptance Criteria

1. WHEN validating configuration THEN the system SHALL delegate to config.Validate() methods
2. WHEN validating policies THEN the system SHALL use ResiliencePolicy.Validate()
3. THE system SHALL NOT have duplicate validation logic in multiple locations

### Requirement 5

**User Story:** As a developer, I want all files under 400 lines, so that the codebase is maintainable.

#### Acceptance Criteria

1. THE system SHALL have no source files exceeding 400 non-blank lines
2. WHEN a file exceeds 400 lines THEN the system SHALL split it into smaller files
