# Requirements Document

## Introduction

This specification defines the requirements for modernizing the `libs/go` library collection to state-of-the-art December 2025 standards. The modernization focuses on eliminating redundancy, centralizing logic, adopting Go 1.25+ features, and restructuring the architecture for maximum reusability and maintainability.

## Glossary

- **Go_Libs**: The collection of Go libraries in `libs/go/` providing reusable components for microservices
- **Module**: A Go module with its own `go.mod` file
- **Package**: A Go package within a module
- **Redundancy**: Duplicated logic, types, or functionality across multiple locations
- **Centralization**: Single authoritative location for shared behavior
- **State_of_Art**: Latest stable, officially recommended patterns and practices as of December 2025
- **Property_Based_Testing**: Testing approach using generated inputs to verify universal properties
- **Functional_Options**: Configuration pattern using variadic function parameters
- **Generics**: Go type parameters for type-safe reusable code

## Requirements

### Requirement 1: Eliminate LRU Cache Redundancy

**User Story:** As a developer, I want a single authoritative LRU cache implementation, so that I don't have to choose between duplicate implementations or maintain multiple versions.

#### Acceptance Criteria

1. THE Go_Libs SHALL provide exactly one LRU cache implementation in `src/collections/`
2. WHEN the cache module exists separately, THE Go_Libs SHALL remove `src/cache/` and redirect imports to `src/collections/`
3. THE unified LRU cache SHALL support TTL, eviction callbacks, and statistics
4. THE unified LRU cache SHALL return `functional.Option[V]` for Get operations
5. THE unified LRU cache SHALL provide `GetOrCompute` for lazy initialization

### Requirement 2: Eliminate Codec Redundancy

**User Story:** As a developer, I want a single codec package, so that encoding/decoding logic is centralized and consistent.

#### Acceptance Criteria

1. THE Go_Libs SHALL provide exactly one codec implementation in `src/codec/`
2. WHEN codec utilities exist in `src/utils/codec.go`, THE Go_Libs SHALL remove them and redirect to `src/codec/`
3. THE unified codec SHALL support JSON, YAML, and Base64 encoding
4. THE unified codec SHALL provide generic type-safe decode functions
5. THE unified codec SHALL integrate with `functional.Result[T]` for error handling

### Requirement 3: Eliminate Validation Redundancy

**User Story:** As a developer, I want a single validation package, so that validation rules are defined once and reused consistently.

#### Acceptance Criteria

1. THE Go_Libs SHALL provide exactly one validation implementation in `src/validation/`
2. WHEN validation utilities exist in `src/utils/validation.go`, THE Go_Libs SHALL remove them and redirect to `src/validation/`
3. THE unified validation SHALL support composable validators with `And()`, `Or()`, `Not()`
4. THE unified validation SHALL provide error accumulation via `Result`
5. THE unified validation SHALL support nested field paths for struct validation

### Requirement 4: Consolidate Micro-Modules in go.work

**User Story:** As a developer, I want a clean workspace structure, so that I can easily navigate and understand the codebase.

#### Acceptance Criteria

1. THE Go_Libs SHALL consolidate the 60+ micro-modules in `go.work` into domain-aligned modules
2. WHEN micro-modules exist (e.g., `functional/option`, `functional/result`), THE Go_Libs SHALL merge them into parent modules
3. THE consolidated structure SHALL have maximum 24 top-level modules in `src/`
4. THE Go_Libs SHALL update `go.work` to reference only consolidated modules
5. THE Go_Libs SHALL provide a migration guide for import path changes

### Requirement 5: Adopt Go 1.25+ Features

**User Story:** As a developer, I want the libraries to use modern Go features, so that I benefit from improved performance and type safety.

#### Acceptance Criteria

1. THE Go_Libs SHALL require Go 1.25 as minimum version
2. THE Go_Libs SHALL use `errors.AsType[T]()` for generic error type assertions
3. THE Go_Libs SHALL use Go 1.23+ iterator patterns (`iter.Seq`, `iter.Seq2`)
4. THE Go_Libs SHALL use `slog` for structured logging in observability module
5. THE Go_Libs SHALL use `testing/synctest` for deterministic concurrency tests

### Requirement 6: Centralize Error Handling

**User Story:** As a developer, I want consistent error handling across all modules, so that error propagation and handling is predictable.

#### Acceptance Criteria

1. THE Go_Libs SHALL define base error types in `src/errors/`
2. WHEN modules define custom errors, THE Go_Libs SHALL extend the base error types
3. THE error types SHALL support `errors.Is()` and `errors.As()` for error checking
4. THE error types SHALL provide HTTP status and gRPC code mapping
5. THE error types SHALL support JSON serialization for API responses

### Requirement 7: Centralize Resilience Patterns

**User Story:** As a developer, I want unified resilience patterns, so that circuit breakers, retries, and rate limiters work consistently.

#### Acceptance Criteria

1. THE Go_Libs SHALL provide all resilience patterns in `src/resilience/`
2. THE resilience module SHALL use functional options for configuration
3. THE resilience module SHALL integrate with `functional.Result[T]` for operation results
4. THE resilience module SHALL provide unified error types extending `ResilienceError`
5. THE resilience module SHALL support context cancellation and timeouts

### Requirement 8: Centralize Functional Primitives

**User Story:** As a developer, I want unified functional types, so that Option, Result, Either work together seamlessly.

#### Acceptance Criteria

1. THE Go_Libs SHALL provide all functional types in `src/functional/`
2. THE functional module SHALL provide `Option[T]`, `Result[T]`, `Either[L,R]`
3. THE functional module SHALL provide type conversion functions between types
4. THE functional module SHALL implement `Functor` interface for all types
5. THE functional module SHALL support Go 1.23+ iterator patterns

### Requirement 9: Separate Source and Test Code

**User Story:** As a developer, I want clear separation between source and test code, so that the codebase is organized and navigable.

#### Acceptance Criteria

1. THE Go_Libs SHALL place all source code in `src/` directory
2. THE Go_Libs SHALL place all test code in `tests/` directory
3. THE test directory structure SHALL mirror the source directory structure
4. WHEN tests exist alongside source files, THE Go_Libs SHALL move them to `tests/`
5. THE Go_Libs SHALL maintain separate `go.work` files for `src/` and `tests/`

### Requirement 10: Adopt OpenTelemetry for Observability

**User Story:** As a developer, I want OpenTelemetry integration, so that I can collect traces, metrics, and logs consistently.

#### Acceptance Criteria

1. THE observability module SHALL integrate with OpenTelemetry SDK
2. THE observability module SHALL use `slog` with OpenTelemetry bridge
3. THE observability module SHALL support trace context propagation (W3C format)
4. THE observability module SHALL provide correlation ID middleware
5. THE observability module SHALL support PII redaction in logs

### Requirement 11: Provide Property-Based Testing Infrastructure

**User Story:** As a developer, I want property-based testing support, so that I can verify universal properties across generated inputs.

#### Acceptance Criteria

1. THE testing module SHALL provide generators for all domain types
2. THE testing module SHALL integrate with `rapid` library for property testing
3. THE testing module SHALL support seeded reproducibility
4. THE testing module SHALL provide `synctest` helpers for concurrency tests
5. WHEN property tests exist, THE Go_Libs SHALL run minimum 100 iterations

### Requirement 12: Ensure Zero File Redundancy

**User Story:** As a developer, I want no duplicate files or logic, so that maintenance is simplified and behavior is consistent.

#### Acceptance Criteria

1. THE Go_Libs SHALL have exactly one implementation for each concept
2. WHEN duplicate implementations exist, THE Go_Libs SHALL consolidate to single location
3. THE Go_Libs SHALL remove all deprecated or transitional code
4. THE Go_Libs SHALL update all import paths to consolidated locations
5. THE Go_Libs SHALL provide deprecation warnings for old import paths

### Requirement 13: Enforce Maximum File Size

**User Story:** As a developer, I want manageable file sizes, so that code is readable and maintainable.

#### Acceptance Criteria

1. THE Go_Libs SHALL enforce maximum 400 non-blank lines per file
2. WHEN files exceed 400 lines, THE Go_Libs SHALL split into logical units
3. THE split files SHALL maintain single responsibility principle
4. THE Go_Libs SHALL provide barrel exports for split modules
5. THE Go_Libs SHALL update imports after file splits

### Requirement 14: Provide Comprehensive Documentation

**User Story:** As a developer, I want clear documentation, so that I can understand and use the libraries effectively.

#### Acceptance Criteria

1. THE Go_Libs SHALL provide README.md for each module
2. THE Go_Libs SHALL provide migration guide for breaking changes
3. THE Go_Libs SHALL provide changelog with version history
4. THE Go_Libs SHALL provide code examples in documentation
5. THE Go_Libs SHALL document all public APIs with GoDoc comments

### Requirement 15: Ensure Type Safety with Generics

**User Story:** As a developer, I want type-safe APIs, so that I catch errors at compile time rather than runtime.

#### Acceptance Criteria

1. THE Go_Libs SHALL use generics for all collection types
2. THE Go_Libs SHALL use generics for functional types (Option, Result, Either)
3. THE Go_Libs SHALL use generics for validation rules
4. THE Go_Libs SHALL avoid `interface{}` or `any` in public APIs where generics apply
5. THE Go_Libs SHALL provide type constraints for generic parameters
