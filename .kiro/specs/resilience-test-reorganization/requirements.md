# Requirements Document

## Introduction

Este documento especifica os requisitos para reorganização dos testes do `platform/resilience-service`. O objetivo é mover todos os arquivos de teste para uma pasta `tests/` centralizada, seguindo boas práticas de organização de projetos Go e garantindo conformidade com limites de complexidade e linhas de código.

## Glossary

- **Resilience Service**: Microserviço Go que implementa padrões de resiliência (circuit breaker, retry, rate limit, bulkhead, timeout)
- **Property Test**: Teste baseado em propriedades usando gopter para validar invariantes
- **Benchmark Test**: Teste de performance usando o framework de benchmark do Go
- **Unit Test**: Teste unitário tradicional que valida comportamento específico
- **Integration Test**: Teste que valida integração com dependências externas (Redis)
- **Test Utilities**: Código auxiliar compartilhado entre testes (generators, helpers)

## Requirements

### Requirement 1

**User Story:** As a developer, I want all tests organized in a centralized `tests/` directory, so that I can easily find and maintain test code separately from production code.

#### Acceptance Criteria

1. WHEN the test reorganization is complete THEN the system SHALL have all test files located under `platform/resilience-service/tests/` directory
2. WHEN organizing tests THEN the system SHALL maintain separate subdirectories for each test type: `unit/`, `property/`, `benchmark/`, `integration/`
3. WHEN moving test files THEN the system SHALL preserve the package structure mirroring the source code organization
4. WHEN tests are reorganized THEN the system SHALL ensure all tests remain executable via `go test ./...`

### Requirement 2

**User Story:** As a developer, I want test files to follow Go best practices and coding standards, so that the codebase remains maintainable and consistent.

#### Acceptance Criteria

1. WHEN a test file is created or modified THEN the system SHALL ensure the file has no more than 400 lines (500 max with justification)
2. WHEN a test function is created THEN the system SHALL ensure the function has no more than 50 lines (75 max with justification)
3. WHEN test files are organized THEN the system SHALL use consistent naming conventions: `*_test.go` for unit, `*_prop_test.go` for property, `*_bench_test.go` for benchmark
4. WHEN tests are written THEN the system SHALL follow the AAA pattern (Arrange, Act, Assert)
5. WHEN test files have compilation errors THEN the system SHALL fix them to ensure all tests compile successfully

### Requirement 3

**User Story:** As a developer, I want test utilities centralized and reusable, so that I can avoid code duplication across test files.

#### Acceptance Criteria

1. WHEN test utilities are organized THEN the system SHALL place them in `tests/testutil/` directory
2. WHEN generators are used THEN the system SHALL ensure they are accessible from all test packages
3. WHEN helper functions are created THEN the system SHALL ensure they follow the `t.Helper()` pattern for proper error reporting

### Requirement 4

**User Story:** As a developer, I want property-based tests to be properly annotated, so that I can trace them back to requirements.

#### Acceptance Criteria

1. WHEN a property test is written THEN the system SHALL include a comment linking to the feature and property number
2. WHEN a property test is written THEN the system SHALL include a comment linking to the requirements it validates
3. WHEN property tests are configured THEN the system SHALL run a minimum of 100 iterations

### Requirement 5

**User Story:** As a developer, I want benchmark tests to be correct and runnable, so that I can measure performance accurately.

#### Acceptance Criteria

1. WHEN benchmark tests are moved THEN the system SHALL fix any compilation errors in the benchmark files
2. WHEN benchmark tests reference types THEN the system SHALL ensure they use the correct field names and method signatures
3. WHEN benchmark tests are organized THEN the system SHALL place them in `tests/benchmark/` directory

### Requirement 6

**User Story:** As a developer, I want integration tests properly isolated, so that they only run when external dependencies are available.

#### Acceptance Criteria

1. WHEN integration tests are organized THEN the system SHALL place them in `tests/integration/` directory
2. WHEN integration tests are written THEN the system SHALL use build tags (`//go:build integration`) to isolate them
3. WHEN integration tests require external services THEN the system SHALL skip gracefully if the service is unavailable

