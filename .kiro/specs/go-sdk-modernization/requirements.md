# Requirements Document

## Introduction

This document specifies the requirements for modernizing the Auth Platform Go SDK to December 2025 state-of-the-art standards. The modernization focuses on upgrading dependencies to latest stable versions, eliminating code redundancy, separating tests from source code, implementing modern OAuth 2.1 patterns (PKCE, DPoP), leveraging Go 1.25+ features including generics and `errors.AsType`, and restructuring the architecture for maintainability.

## Glossary

- **SDK**: Software Development Kit - the Go client library for Auth Platform
- **JWKS**: JSON Web Key Set - public keys for JWT validation
- **JWT**: JSON Web Token - authentication token format
- **PKCE**: Proof Key for Code Exchange - OAuth 2.0 security extension
- **DPoP**: Demonstrating Proof-of-Possession - token binding mechanism
- **Interceptor**: gRPC middleware for request/response processing
- **Middleware**: HTTP handler wrapper for authentication
- **Sentinel_Error**: Predefined error value for comparison
- **Functional_Option**: Configuration pattern using function parameters

## Requirements

### Requirement 1: Dependency Modernization

**User Story:** As a developer, I want the SDK to use the latest stable dependencies, so that I benefit from security fixes, performance improvements, and modern features.

#### Acceptance Criteria

1. THE SDK SHALL use Go version 1.25 or higher as the minimum supported version
2. THE SDK SHALL use `github.com/lestrrat-go/jwx/v3` for JWT/JWK operations
3. THE SDK SHALL use `github.com/golang-jwt/jwt/v5` version 5.2.2 or higher
4. THE SDK SHALL use `google.golang.org/grpc` version 1.70.0 or higher
5. THE SDK SHALL use `golang.org/x/crypto` version 0.31.0 or higher
6. THE SDK SHALL use `golang.org/x/net` version 0.33.0 or higher
7. WHEN dependencies are updated THEN the SDK SHALL maintain backward compatibility with existing public APIs

### Requirement 2: Architecture Restructuring

**User Story:** As a maintainer, I want source code separated from tests with clear module boundaries, so that the codebase is easier to navigate and maintain.

#### Acceptance Criteria

1. THE SDK SHALL organize source files in `sdk/go/` directory
2. THE SDK SHALL organize test files in `sdk/go/tests/` directory
3. THE SDK SHALL separate concerns into distinct files: client, auth, jwks, middleware, grpc, errors
4. WHEN a file exceeds 400 lines THEN the SDK SHALL split it into smaller focused modules
5. THE SDK SHALL use internal packages for non-exported implementation details
6. THE SDK SHALL maintain a single public API surface through the main package

### Requirement 3: Token Extraction Centralization

**User Story:** As a developer, I want token extraction logic centralized, so that authentication behavior is consistent across HTTP and gRPC.

#### Acceptance Criteria

1. THE SDK SHALL provide a single `TokenExtractor` interface for extracting tokens from requests
2. THE SDK SHALL implement `HTTPTokenExtractor` for HTTP Authorization headers
3. THE SDK SHALL implement `GRPCTokenExtractor` for gRPC metadata
4. WHEN extracting tokens THEN the SDK SHALL support Bearer token format
5. WHEN extracting tokens THEN the SDK SHALL support DPoP token format
6. THE SDK SHALL NOT duplicate token extraction logic across middleware and interceptors

### Requirement 4: Generic Error Handling

**User Story:** As a developer, I want type-safe error handling using Go 1.26 generics, so that I can handle errors without type assertions.

#### Acceptance Criteria

1. THE SDK SHALL define structured error types with error codes and messages
2. THE SDK SHALL use `errors.AsType[T]()` for type-safe error extraction where available
3. THE SDK SHALL provide `Is*` helper functions for sentinel error checking
4. WHEN wrapping errors THEN the SDK SHALL preserve the error chain for unwrapping
5. THE SDK SHALL define error codes as constants for programmatic handling
6. IF an error occurs THEN the SDK SHALL include contextual information without exposing sensitive data

### Requirement 5: PKCE Support

**User Story:** As a developer, I want PKCE support for authorization code flow, so that I can implement secure OAuth 2.0 for public clients.

#### Acceptance Criteria

1. THE SDK SHALL generate cryptographically secure code verifiers (43-128 characters)
2. THE SDK SHALL compute code challenges using S256 method (SHA-256 hash, base64url encoded)
3. WHEN initiating authorization THEN the SDK SHALL include code_challenge and code_challenge_method parameters
4. WHEN exchanging authorization code THEN the SDK SHALL include code_verifier parameter
5. THE SDK SHALL provide a `PKCEGenerator` interface for custom implementations
6. THE SDK SHALL use `crypto/rand` for secure random generation

### Requirement 6: DPoP Token Support

**User Story:** As a developer, I want DPoP token binding support, so that I can implement proof-of-possession for enhanced security.

#### Acceptance Criteria

1. THE SDK SHALL generate DPoP proofs as signed JWTs
2. THE SDK SHALL include required claims: jti, htm, htu, iat in DPoP proofs
3. WHEN access token is bound THEN the SDK SHALL include ath claim (access token hash)
4. THE SDK SHALL support ES256 and RS256 algorithms for DPoP signing
5. THE SDK SHALL validate DPoP proofs during token validation
6. WHEN DPoP is enabled THEN the SDK SHALL automatically attach DPoP headers to requests

### Requirement 7: JWKS Cache Modernization

**User Story:** As a developer, I want an efficient JWKS cache with automatic refresh, so that token validation is fast and reliable.

#### Acceptance Criteria

1. THE SDK SHALL use `jwx/v3` auto-refresh cache for JWKS
2. THE SDK SHALL support configurable refresh intervals (minimum, maximum)
3. WHEN JWKS fetch fails THEN the SDK SHALL use cached keys if available
4. THE SDK SHALL support multiple JWKS endpoints for key rotation
5. THE SDK SHALL provide cache invalidation method for forced refresh
6. THE SDK SHALL emit metrics for cache hits, misses, and refresh operations

### Requirement 8: gRPC Interceptor Modernization

**User Story:** As a developer, I want modern gRPC interceptors with proper context propagation, so that authentication integrates seamlessly with gRPC services.

#### Acceptance Criteria

1. THE SDK SHALL provide unary and stream server interceptors for token validation
2. THE SDK SHALL provide unary and stream client interceptors for token injection
3. WHEN validation succeeds THEN the SDK SHALL store claims in context using typed keys
4. THE SDK SHALL support interceptor chaining with other middleware
5. THE SDK SHALL propagate trace context through interceptors
6. IF token validation fails THEN the SDK SHALL return appropriate gRPC status codes

### Requirement 9: HTTP Middleware Modernization

**User Story:** As a developer, I want flexible HTTP middleware with configurable options, so that I can customize authentication behavior per route.

#### Acceptance Criteria

1. THE SDK SHALL provide middleware compatible with `http.Handler` interface
2. THE SDK SHALL support skip patterns for excluding paths from authentication
3. THE SDK SHALL support custom error handlers for authentication failures
4. WHEN validation succeeds THEN the SDK SHALL store claims in request context
5. THE SDK SHALL support extracting tokens from cookies as alternative to headers
6. THE SDK SHALL provide middleware options using functional options pattern

### Requirement 10: Retry and Resilience

**User Story:** As a developer, I want built-in retry logic with exponential backoff, so that transient failures are handled gracefully.

#### Acceptance Criteria

1. THE SDK SHALL implement exponential backoff for retryable errors
2. THE SDK SHALL respect Retry-After headers from rate limit responses
3. THE SDK SHALL support configurable maximum retry attempts (default: 3)
4. THE SDK SHALL support configurable base delay and maximum delay
5. WHEN context is cancelled THEN the SDK SHALL stop retrying immediately
6. THE SDK SHALL NOT retry on non-retryable errors (4xx except 429, 503)

### Requirement 11: Configuration Validation

**User Story:** As a developer, I want configuration validated at client creation, so that I catch misconfigurations early.

#### Acceptance Criteria

1. WHEN creating a client THEN the SDK SHALL validate required fields (BaseURL, ClientID)
2. WHEN BaseURL is invalid THEN the SDK SHALL return a descriptive error
3. THE SDK SHALL validate timeout values are positive durations
4. THE SDK SHALL validate cache TTL values are within reasonable bounds
5. THE SDK SHALL provide default values for optional configuration
6. THE SDK SHALL support configuration from environment variables

### Requirement 12: Observability Integration

**User Story:** As an operator, I want observability hooks in the SDK, so that I can monitor authentication operations.

#### Acceptance Criteria

1. THE SDK SHALL support OpenTelemetry tracing for all operations
2. THE SDK SHALL create spans for token validation, refresh, and JWKS fetch
3. THE SDK SHALL record span attributes for operation metadata (not sensitive data)
4. THE SDK SHALL support custom logger injection
5. WHEN errors occur THEN the SDK SHALL log with appropriate severity levels
6. THE SDK SHALL NOT log sensitive data (tokens, secrets, credentials)

### Requirement 13: Property-Based Testing

**User Story:** As a maintainer, I want property-based tests for core logic, so that edge cases are discovered automatically.

#### Acceptance Criteria

1. THE SDK SHALL include property tests for PKCE code verifier/challenge generation
2. THE SDK SHALL include property tests for token extraction round-trips
3. THE SDK SHALL include property tests for error wrapping/unwrapping
4. THE SDK SHALL include property tests for DPoP proof generation/validation
5. WHEN running property tests THEN the SDK SHALL execute minimum 100 iterations
6. THE SDK SHALL use `pgregory.net/rapid` or equivalent for property-based testing

### Requirement 14: Generic Result Types

**User Story:** As a developer, I want generic result types for operations, so that I can handle success and failure uniformly.

#### Acceptance Criteria

1. THE SDK SHALL provide `Result[T]` type for operations that may fail
2. THE SDK SHALL provide `Option[T]` type for optional values
3. THE SDK SHALL provide methods for mapping, flat-mapping, and unwrapping results
4. WHEN using Result types THEN the SDK SHALL preserve error context
5. THE SDK SHALL support pattern matching style handling with `Match` method
6. THE SDK SHALL integrate Result types with existing error handling

### Requirement 15: Documentation and Examples

**User Story:** As a developer, I want comprehensive documentation with examples, so that I can integrate the SDK quickly.

#### Acceptance Criteria

1. THE SDK SHALL include GoDoc comments for all exported types and functions
2. THE SDK SHALL include example functions for common use cases
3. THE SDK SHALL include a README with quick start guide
4. THE SDK SHALL include migration guide from previous version
5. WHEN APIs change THEN the SDK SHALL document breaking changes in CHANGELOG
6. THE SDK SHALL include runnable examples in `examples/` directory
