# Implementation Plan: Go SDK Modernization

## Overview

This implementation plan modernizes the Auth Platform Go SDK to December 2025 state-of-the-art standards. Tasks are organized to build incrementally, with each task building on previous work. Property tests are included as sub-tasks to catch errors early.

## Tasks

- [x] 1. Update dependencies and module configuration
  - Update go.mod to Go 1.25
  - Upgrade `github.com/lestrrat-go/jwx/v2` to `github.com/lestrrat-go/jwx/v3`
  - Upgrade `github.com/golang-jwt/jwt/v5` to v5.2.2+
  - Upgrade `google.golang.org/grpc` to v1.70.0+
  - Upgrade `golang.org/x/crypto` to v0.31.0+
  - Upgrade `golang.org/x/net` to v0.33.0+
  - Add `pgregory.net/rapid` for property-based testing
  - Add `go.opentelemetry.io/otel` for tracing
  - Run `go mod tidy` and verify compilation
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6_

- [x] 2. Create directory structure and move tests
  - Create `sdk/go/tests/` directory
  - Create `sdk/go/tests/property/` directory
  - Create `sdk/go/internal/observability/` directory
  - Create `sdk/go/examples/` directory
  - Move `sdk/go/errors_test.go` to `sdk/go/tests/errors_test.go`
  - Update import paths in moved test files
  - Verify tests still pass after move
  - _Requirements: 2.1, 2.2, 2.3_

- [x] 3. Implement structured error types
  - [x] 3.1 Create error codes and SDKError type in `sdk/go/errors.go`
    - Define ErrorCode type and constants
    - Implement SDKError struct with Code, Message, Cause
    - Implement Error() and Unwrap() methods
    - Maintain backward compatibility with existing sentinel errors
    - _Requirements: 4.1, 4.5_
  - [x] 3.2 Write property tests for error handling
    - **Property 2: Error Structure Completeness**
    - **Property 3: Error Type Extraction with AsType**
    - **Property 4: Is Helper Functions Correctness**
    - **Property 5: Error Chain Preservation**
    - **Property 6: No Sensitive Data in Errors**
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.6**

- [x] 4. Implement generic Result and Option types
  - [x] 4.1 Create `sdk/go/result.go` with generic types
    - Implement Result[T] with Ok and Err constructors
    - Implement Map, FlatMap, Match, Unwrap methods
    - Implement Option[T] with Some and None constructors
    - Implement Map, FlatMap, UnwrapOr methods
    - _Requirements: 14.1, 14.2, 14.3_
  - [x] 4.2 Write property tests for Result types
    - **Property 30: Result Map/FlatMap Correctness**
    - **Property 31: Result Error Preservation**
    - **Property 32: Result Match Correctness**
    - **Validates: Requirements 14.3, 14.4, 14.5**

- [x] 5. Checkpoint - Verify foundation
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Implement centralized token extraction
  - [x] 6.1 Create `sdk/go/extractor.go` with TokenExtractor interface
    - Define TokenExtractor interface
    - Define TokenScheme type (Bearer, DPoP)
    - Implement HTTPTokenExtractor for Authorization header
    - Implement GRPCTokenExtractor for gRPC metadata
    - Implement CookieTokenExtractor for cookie-based tokens
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_
  - [x] 6.2 Write property tests for token extraction
    - **Property 1: Token Extraction Round-Trip**
    - **Property 16: Cookie Token Extraction**
    - **Validates: Requirements 3.4, 3.5, 9.5**

- [x] 7. Implement retry policy with exponential backoff
  - [x] 7.1 Create `sdk/go/retry.go` with retry logic
    - Implement RetryPolicy struct with configuration
    - Implement exponential backoff with jitter
    - Implement Retry-After header parsing
    - Implement context cancellation handling
    - Implement retryable error classification
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_
  - [x] 7.2 Write property tests for retry logic
    - **Property 17: Exponential Backoff Delays**
    - **Property 18: Retry-After Header Respect**
    - **Property 19: Maximum Retry Count**
    - **Property 20: Retry Delay Bounds**
    - **Property 21: Context Cancellation Stops Retry**
    - **Property 22: Non-Retryable Error Handling**
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5, 10.6**

- [x] 8. Implement configuration validation
  - [x] 8.1 Update `sdk/go/client.go` with enhanced Config
    - Add environment variable tags to Config fields
    - Implement Validate() method with all checks
    - Implement LoadFromEnv() for environment configuration
    - Implement ApplyDefaults() for default values
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_
  - [x] 8.2 Write property tests for configuration
    - **Property 23: Required Field Validation**
    - **Property 24: Timeout Validation**
    - **Property 25: Cache TTL Validation**
    - **Property 26: Default Values Application**
    - **Property 27: Environment Variable Configuration**
    - **Validates: Requirements 11.1, 11.2, 11.3, 11.4, 11.5, 11.6**

- [x] 9. Checkpoint - Verify core infrastructure
  - Ensure all tests pass, ask the user if questions arise.

- [x] 10. Implement PKCE support
  - [x] 10.1 Create `sdk/go/auth.go` with PKCE implementation
    - Define PKCEGenerator interface
    - Implement DefaultPKCEGenerator
    - Implement GenerateVerifier() using crypto/rand
    - Implement ComputeChallenge() using SHA-256 + base64url
    - Add PKCE parameters to authorization flow
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_
  - [x] 10.2 Write property tests for PKCE
    - **Property 7: PKCE Verifier Constraints**
    - **Property 8: PKCE Challenge Round-Trip**
    - **Validates: Requirements 5.1, 5.2**

- [x] 11. Implement DPoP support
  - [x] 11.1 Create `sdk/go/dpop.go` with DPoP implementation
    - Define DPoPProver interface
    - Define DPoPClaims struct
    - Implement GenerateProof() with required claims (jti, htm, htu, iat)
    - Implement ath claim for access token binding
    - Implement ValidateProof() for proof verification
    - Support ES256 and RS256 algorithms
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_
  - [x] 11.2 Write property tests for DPoP
    - **Property 9: DPoP Proof Required Claims**
    - **Property 10: DPoP ATH Claim Correctness**
    - **Property 11: DPoP Algorithm Support**
    - **Property 12: DPoP Validation Correctness**
    - **Validates: Requirements 6.2, 6.3, 6.4, 6.5**

- [x] 12. Modernize JWKS cache with jwx/v3
  - [x] 12.1 Update `sdk/go/jwks.go` to use jwx/v3
    - Migrate from jwx/v2 to jwx/v3 API
    - Use jwk.Cache with auto-refresh
    - Configure refresh intervals (min, max)
    - Implement fallback to cached keys on fetch failure
    - Add cache invalidation method
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_
  - [x] 12.2 Write unit tests for JWKS cache
    - Test cache hit/miss behavior
    - Test refresh on expiry
    - Test fallback on network failure
    - Test invalidation
    - _Requirements: 7.2, 7.3, 7.4, 7.5_

- [x] 13. Checkpoint - Verify authentication components
  - Ensure all tests pass, ask the user if questions arise.

- [x] 14. Modernize HTTP middleware
  - [x] 14.1 Update `sdk/go/middleware.go` with new features
    - Use centralized TokenExtractor
    - Add skip patterns for path exclusion
    - Add custom error handler option
    - Add cookie extraction option
    - Use functional options pattern
    - Store claims in context using typed key
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_
  - [x] 14.2 Write property tests for middleware
    - **Property 13: Claims Stored in Context After Validation** (HTTP part)
    - **Property 15: Skip Pattern Matching**
    - **Validates: Requirements 9.2, 9.4**

- [x] 15. Modernize gRPC interceptors
  - [x] 15.1 Update `sdk/go/grpc.go` with new features
    - Use centralized TokenExtractor
    - Support DPoP token scheme
    - Propagate trace context
    - Return appropriate gRPC status codes
    - Store claims in context using typed key
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_
  - [x] 15.2 Write property tests for interceptors
    - **Property 13: Claims Stored in Context After Validation** (gRPC part)
    - **Property 14: gRPC Status Codes for Validation Failures**
    - **Validates: Requirements 8.3, 8.6**

- [x] 16. Implement observability integration
  - [x] 16.1 Create `sdk/go/internal/observability/tracing.go`
    - Implement OpenTelemetry tracer wrapper
    - Create spans for token validation, refresh, JWKS fetch
    - Add span attributes (non-sensitive)
    - Propagate trace context
    - _Requirements: 12.1, 12.2, 12.3, 12.5_
  - [x] 16.2 Create `sdk/go/internal/observability/logging.go`
    - Implement structured logger interface
    - Support custom logger injection
    - Implement log level mapping
    - Ensure no sensitive data logging
    - _Requirements: 12.4, 12.5, 12.6_
  - [x] 16.3 Write property tests for observability
    - **Property 28: No Sensitive Data in Observability**
    - **Property 29: Log Severity Matching**
    - **Validates: Requirements 12.3, 12.5, 12.6**

- [x] 17. Checkpoint - Verify all components
  - Ensure all tests pass, ask the user if questions arise.

- [x] 18. Update main client with all integrations
  - [x] 18.1 Refactor `sdk/go/client.go` to integrate all components
    - Integrate retry policy
    - Integrate observability
    - Integrate DPoP support
    - Integrate PKCE support
    - Update ClientCredentials flow
    - Update token validation
    - Ensure backward compatibility
    - _Requirements: 1.7, 6.6_
  - [x] 18.2 Write integration tests for client
    - Test full authentication flows
    - Test error handling paths
    - Test retry behavior
    - Test DPoP binding
    - _Requirements: 1.7_

- [x] 19. Create documentation and examples
  - [x] 19.1 Update `sdk/go/README.md`
    - Add quick start guide
    - Document all public APIs
    - Add migration guide from previous version
    - Document configuration options
    - _Requirements: 15.1, 15.3, 15.4_
  - [x] 19.2 Create example applications
    - Create `sdk/go/examples/http_middleware/` example
    - Create `sdk/go/examples/grpc_interceptor/` example
    - Create `sdk/go/examples/pkce_flow/` example
    - Create `sdk/go/examples/dpop_binding/` example
    - _Requirements: 15.2, 15.6_
  - [x] 19.3 Update `sdk/go/CHANGELOG.md`
    - Document all breaking changes
    - Document new features
    - Document migration steps
    - _Requirements: 15.5_

- [x] 20. Final checkpoint - Complete verification
  - Run full test suite
  - Verify all property tests pass with 100+ iterations
  - Verify test coverage meets targets
  - Verify documentation is complete
  - All files under 400 lines limit
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All tasks including property-based tests are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (minimum 100 iterations)
- Unit tests validate specific examples and edge cases
- Integration tests validate component interactions
