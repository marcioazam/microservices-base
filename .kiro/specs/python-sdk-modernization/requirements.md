# Requirements Document

## Introduction

This document specifies the requirements for modernizing the Auth Platform Python SDK to December 2025 state-of-art standards. The modernization focuses on eliminating redundancies, centralizing logic, improving architecture, and ensuring the SDK follows current best practices for Python development.

## Glossary

- **SDK**: Software Development Kit - the auth-platform-sdk Python package
- **JWKS**: JSON Web Key Set - public keys for JWT validation
- **DPoP**: Demonstrating Proof of Possession - RFC 9449 sender-constrained tokens
- **PKCE**: Proof Key for Code Exchange - RFC 7636 for public clients
- **Pydantic**: Data validation library using Python type hints
- **httpx**: Modern async-capable HTTP client for Python
- **Hypothesis**: Property-based testing library for Python
- **Ruff**: Fast Python linter and formatter written in Rust
- **uv**: Fast Python package manager written in Rust

## Requirements

### Requirement 1: Eliminate Redundant Type Definitions

**User Story:** As a developer, I want a single source of truth for data models, so that I can avoid confusion and maintenance burden from duplicate definitions.

#### Acceptance Criteria

1. THE SDK SHALL have exactly one definition for TokenResponse model
2. THE SDK SHALL have exactly one definition for TokenData model
3. THE SDK SHALL have exactly one definition for TokenClaims model
4. WHEN models are needed, THE SDK SHALL import from the centralized models.py module
5. THE SDK SHALL NOT contain duplicate dataclass definitions in types.py that mirror Pydantic models

### Requirement 2: Centralize HTTP Client Logic

**User Story:** As a developer, I want HTTP client creation and retry logic in one place, so that changes propagate consistently.

#### Acceptance Criteria

1. THE SDK SHALL have a single HTTP client factory function for sync clients
2. THE SDK SHALL have a single HTTP client factory function for async clients
3. THE SDK SHALL have centralized retry logic with exponential backoff
4. THE SDK SHALL have a single circuit breaker implementation
5. WHEN HTTP requests are made, THE SDK SHALL use the centralized retry mechanism

### Requirement 3: Modernize Project Configuration

**User Story:** As a developer, I want the SDK to use modern Python tooling, so that I benefit from faster builds and better developer experience.

#### Acceptance Criteria

1. THE SDK SHALL use pyproject.toml as the single source of project configuration
2. THE SDK SHALL support Python 3.11, 3.12, and 3.13
3. THE SDK SHALL use Ruff for linting and formatting
4. THE SDK SHALL use mypy with strict mode for type checking
5. THE SDK SHALL use pytest with pytest-asyncio for testing
6. THE SDK SHALL use Hypothesis for property-based testing

### Requirement 4: Improve Test Organization

**User Story:** As a developer, I want tests organized by type and module, so that I can easily find and run relevant tests.

#### Acceptance Criteria

1. THE SDK SHALL have tests separated into unit/, integration/, and property/ directories
2. THE SDK SHALL have test files that mirror source file structure
3. WHEN a source module exists, THE SDK SHALL have corresponding test modules
4. THE SDK SHALL NOT have test files at the root of the tests/ directory
5. THE SDK SHALL have property-based tests for all cryptographic operations

### Requirement 5: Centralize Error Handling

**User Story:** As a developer, I want consistent error handling across the SDK, so that I can handle errors predictably.

#### Acceptance Criteria

1. THE SDK SHALL have a single error hierarchy rooted at AuthPlatformError
2. THE SDK SHALL use ErrorCode enum for all error codes
3. WHEN errors occur, THE SDK SHALL include correlation IDs when available
4. WHEN errors occur, THE SDK SHALL include structured details for debugging
5. THE SDK SHALL serialize errors consistently via to_dict() method

### Requirement 6: Improve JWKS Cache Implementation

**User Story:** As a developer, I want efficient JWKS caching, so that token validation is fast and reduces network calls.

#### Acceptance Criteria

1. THE SDK SHALL cache JWKS with configurable TTL
2. THE SDK SHALL support refresh-ahead to prevent cache misses
3. THE SDK SHALL be thread-safe for sync cache operations
4. THE SDK SHALL be async-safe for async cache operations
5. WHEN cache is invalidated, THE SDK SHALL reset all internal state
6. THE SDK SHALL NOT make redundant JWKS fetches within TTL period

### Requirement 7: Ensure DPoP RFC 9449 Compliance

**User Story:** As a developer, I want DPoP implementation that follows RFC 9449, so that I can use sender-constrained tokens securely.

#### Acceptance Criteria

1. THE SDK SHALL generate DPoP proofs with correct JWT structure
2. THE SDK SHALL support ES256, ES384, and ES512 algorithms
3. THE SDK SHALL compute JWK thumbprints per RFC 7638
4. THE SDK SHALL include access token hash (ath) when binding to tokens
5. THE SDK SHALL handle server-provided nonces correctly
6. THE SDK SHALL verify DPoP proofs with timing-safe comparisons

### Requirement 8: Ensure PKCE RFC 7636 Compliance

**User Story:** As a developer, I want PKCE implementation that follows RFC 7636, so that I can secure authorization code flows.

#### Acceptance Criteria

1. THE SDK SHALL generate code verifiers between 43-128 characters
2. THE SDK SHALL use only S256 code challenge method (plain is insecure)
3. THE SDK SHALL generate URL-safe base64 encoded challenges
4. THE SDK SHALL verify code challenges with timing-safe comparisons
5. THE SDK SHALL generate cryptographically random state and nonce values

### Requirement 9: Provide Framework Middleware

**User Story:** As a developer, I want ready-to-use middleware for popular frameworks, so that I can quickly integrate authentication.

#### Acceptance Criteria

1. THE SDK SHALL provide FastAPI dependency injection middleware
2. THE SDK SHALL provide Flask decorator middleware
3. THE SDK SHALL provide Django middleware class
4. WHEN middleware validates tokens, THE SDK SHALL use the centralized validation logic
5. THE SDK SHALL support optional authentication (allow unauthenticated requests)

### Requirement 10: Integrate OpenTelemetry Observability

**User Story:** As a developer, I want built-in observability, so that I can monitor SDK operations in production.

#### Acceptance Criteria

1. THE SDK SHALL integrate with OpenTelemetry for tracing
2. THE SDK SHALL use structlog for structured logging
3. THE SDK SHALL trace HTTP requests with method, URL, and attempt count
4. THE SDK SHALL trace token operations (validation, refresh)
5. WHEN errors occur, THE SDK SHALL record exceptions in spans

### Requirement 11: Ensure Configuration Validation

**User Story:** As a developer, I want configuration validated at startup, so that I catch errors early.

#### Acceptance Criteria

1. THE SDK SHALL validate configuration using Pydantic v2
2. THE SDK SHALL use frozen models for immutable configuration
3. THE SDK SHALL derive default endpoints from base_url
4. THE SDK SHALL support loading configuration from environment variables
5. WHEN configuration is invalid, THE SDK SHALL raise InvalidConfigError with field details

### Requirement 12: Support Both Sync and Async Clients

**User Story:** As a developer, I want both sync and async clients, so that I can use the SDK in any application.

#### Acceptance Criteria

1. THE SDK SHALL provide AuthPlatformClient for synchronous operations
2. THE SDK SHALL provide AsyncAuthPlatformClient for asynchronous operations
3. THE SDK SHALL support context managers for both client types
4. THE SDK SHALL share configuration and validation logic between clients
5. WHEN clients are closed, THE SDK SHALL release HTTP connections properly
