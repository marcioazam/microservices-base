# Implementation Plan: Auth Microservices Platform

## Phase 1: Foundation and Shared Infrastructure

- [x] 1. Set up monorepo structure and shared protobuf definitions

  - [x] 1.1 Create monorepo directory structure for all 5 services
    - Create `auth-edge-service/`, `token-service/`, `session-identity-core/`, `iam-policy-service/`, `mfa-service/` directories
    - Set up shared `proto/` directory for gRPC definitions
    - _Requirements: 7.1, 7.5_

  - [x] 1.2 Define shared protobuf contracts for inter-service communication
    - Create `auth_edge.proto`, `token_service.proto`, `session_identity.proto`, `iam_policy.proto`, `mfa_service.proto`
    - Define common message types (TokenPairResponse, ErrorResponse, etc.)
    - _Requirements: 7.1, 7.5_

  - [x] 1.3 Set up shared error types and correlation ID utilities
    - Create shared error code enums matching design document
    - Implement correlation ID generation and propagation
    - _Requirements: 10.1, 8.1, 8.6_

## Phase 2: Token Service (Rust)

- [x] 2. Implement Token Service core functionality

  - [x] 2.1 Set up Rust project with Tokio and Tonic for gRPC
    - Initialize Cargo project with required dependencies (tokio, tonic, jsonwebtoken, etc.)
    - Configure protobuf compilation
    - _Requirements: 2.1_

  - [x] 2.2 Implement JWT builder and claims structure
    - Create JWT claims struct with standard and custom claims
    - Implement token builder with configurable TTL
    - _Requirements: 2.1, 4.6_

  - [x] 2.3 Implement JWT serializer/parser with round-trip support
    - Create serialization to JWT string format
    - Create parsing from JWT string back to claims
    - _Requirements: 2.7_

  - [x] 2.4 Write property test for JWT round-trip consistency
    - **Property 2: JWT Round-Trip Consistency**
    - **Validates: Requirements 2.7**

  - [x] 2.5 Implement KMS signer interface with mock for testing
    - Define KMS signing trait
    - Implement AWS KMS integration stub
    - Create mock signer for testing
    - _Requirements: 2.6_

  - [x] 2.6 Implement refresh token generator with family tracking
    - Generate cryptographically secure refresh tokens
    - Implement token family tracking structure
    - _Requirements: 2.2, 2.3_

  - [x] 2.7 Implement refresh token rotation logic
    - Rotate refresh token on use
    - Invalidate previous token in family
    - _Requirements: 2.3_

  - [x] 2.8 Write property test for refresh token rotation
    - **Property 4: Refresh Token Rotation Invalidates Previous**
    - **Validates: Requirements 2.3**

  - [x] 2.9 Implement replay detection for refresh tokens
    - Detect reused refresh tokens
    - Revoke entire token family on replay
    - Log security event
    - _Requirements: 2.4_

  - [x] 2.10 Write property test for replay detection
    - **Property 5: Refresh Token Replay Detection**
    - **Validates: Requirements 2.4**

  - [x] 2.11 Implement JWKS endpoint with key rotation support
    - Expose current and previous signing keys
    - Support graceful key rotation
    - _Requirements: 2.5_

  - [x] 2.12 Write property test for JWKS rotation keys
    - **Property 6: JWKS Contains Rotation Keys**
    - **Validates: Requirements 2.5**

  - [x] 2.13 Implement Redis storage for revocation list
    - Store revoked tokens with TTL
    - Implement revocation check
    - _Requirements: 9.3_

  - [x] 2.14 Write property test for revocation list consistency
    - **Property 24: Token Revocation List Consistency**
    - **Validates: Requirements 9.3**

  - [x] 2.15 Implement gRPC server with IssueTokenPair, RefreshTokens, RevokeToken endpoints
    - Wire up all token service gRPC handlers
    - _Requirements: 2.1, 2.2, 2.3_

  - [x] 2.16 Write property test for token pair issuance
    - **Property 3: Token Pair Issuance Completeness**
    - **Validates: Requirements 2.1, 2.2**

- [x] 3. Checkpoint - Token Service
  - All tests pass

## Phase 3: Auth Edge Service (Rust)

- [x] 4. Implement Auth Edge Service core functionality

  - [x] 4.1 Set up Rust project with Tokio and Tonic
    - Initialize Cargo project
    - Configure for ultra-low latency (<1ms p99)
    - _Requirements: 1.1_

  - [x] 4.2 Implement JWK cache with atomic updates
    - Create thread-safe JWK cache
    - Implement atomic cache refresh
    - _Requirements: 1.5_

  - [x] 4.3 Write property test for JWK cache atomic update
    - **Property 8: JWK Cache Atomic Update**
    - **Validates: Requirements 1.5**

  - [x] 4.4 Implement JWT validator using cached JWK
    - Validate JWT signatures against cached keys
    - Check expiration and required claims
    - _Requirements: 1.1, 1.2, 1.3_

  - [x] 4.5 Write property test for JWT validation rejects invalid tokens
    - **Property 1: JWT Validation Rejects Invalid Tokens**
    - **Validates: Requirements 1.2, 1.3**

  - [x] 4.6 Implement mTLS certificate validation and SPIFFE ID extraction
    - Parse X.509 certificates
    - Extract SPIFFE ID from SAN
    - _Requirements: 1.6, 7.3_

  - [x] 4.7 Write property test for SPIFFE ID extraction
    - **Property 7: SPIFFE ID Extraction Accuracy**
    - **Validates: Requirements 1.6**

  - [x] 4.8 Implement gRPC client pool for downstream services
    - Create connection pool to Token Service, Session Core, IAM Service
    - Configure mTLS for all connections
    - _Requirements: 7.1_

  - [x] 4.9 Implement circuit breaker for downstream calls
    - Configure failure thresholds per service
    - Return 503 with retry guidance when open
    - _Requirements: 9.5, 10.2_

  - [x] 4.10 Write property test for circuit breaker behavior
    - **Property 25: Circuit Breaker Behavior**
    - **Validates: Requirements 9.5**

  - [x] 4.11 Implement gRPC server with ValidateToken, IntrospectToken endpoints
    - Wire up auth edge gRPC handlers
    - Forward claims to IAM for policy evaluation
    - _Requirements: 1.1, 1.4_

  - [x] 4.12 Implement structured error responses
    - Return error code, message, correlation ID
    - Use appropriate HTTP status codes
    - _Requirements: 10.1, 10.3, 10.4, 10.5_

  - [x] 4.13 Write property test for error response structure
    - **Property 26: Error Response Structure**
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5**

- [x] 5. Checkpoint - Auth Edge Service
  - All tests pass

## Phase 4: IAM/Policy Service (Go)

- [x] 6. Implement IAM/Policy Service core functionality

  - [x] 6.1 Set up Go project with gRPC and OPA
    - Initialize Go module
    - Add OPA, gRPC dependencies
    - _Requirements: 5.1_

  - [x] 6.2 Implement OPA policy engine wrapper
    - Load Rego policies
    - Evaluate authorization requests
    - _Requirements: 5.1_

  - [x] 6.3 Implement RBAC role hierarchy resolver
    - Define role structure with parent relationships
    - Resolve effective permissions from hierarchy
    - _Requirements: 5.2_

  - [x] 6.4 Implement ABAC attribute evaluator
    - Evaluate subject, resource, and environment attributes
    - _Requirements: 5.3_

  - [x] 6.5 Write property test for policy evaluation consistency
    - **Property 16: Policy Evaluation Consistency**
    - **Validates: Requirements 5.2, 5.3**

  - [x] 6.6 Implement policy hot-reload without restart
    - Watch policy files for changes
    - Reload policies atomically
    - _Requirements: 5.6_

  - [x] 6.7 Write property test for policy hot reload
    - **Property 17: Policy Hot Reload Effectiveness**
    - **Validates: Requirements 5.6**

  - [x] 6.8 Implement decision caching with configurable TTL
    - Cache authorization decisions
    - Invalidate on policy reload
    - _Requirements: 5.4_

  - [x] 6.9 Implement decision audit logging
    - Log request context, policy matched, result
    - _Requirements: 5.5, 8.2_

  - [x] 6.10 Implement gRPC server with Authorize, BatchAuthorize, GetUserPermissions endpoints
    - Wire up IAM gRPC handlers
    - Ensure <5ms p99 latency
    - _Requirements: 5.1_

- [x] 7. Checkpoint - IAM/Policy Service
  - All tests pass

## Phase 5: Session/Identity Core (Elixir/Phoenix)

- [x] 8. Implement Session/Identity Core foundation

  - [x] 8.1 Set up Elixir/Phoenix project with OTP
    - Initialize Mix project
    - Configure Phoenix, Ecto, gRPC dependencies
    - _Requirements: 3.1_

  - [x] 8.2 Implement PostgreSQL schemas for users, roles, oauth_clients
    - Create Ecto schemas matching data model
    - Set up migrations
    - _Requirements: 9.1_

  - [x] 8.3 Implement Redis session store with TTL
    - Create session storage module
    - Configure cluster mode
    - _Requirements: 9.2_

  - [x] 8.4 Implement session creation with device fingerprint
    - Create session with all required fields
    - Store in Redis with TTL
    - _Requirements: 3.1_

  - [x] 8.5 Write property test for session record completeness
    - **Property 9: Session Record Completeness**
    - **Validates: Requirements 3.1**

  - [x] 8.6 Implement session list retrieval
    - Return all active sessions for user
    - Include device info and last activity
    - _Requirements: 3.3_

  - [x] 8.7 Write property test for session list accuracy
    - **Property 10: Session List Accuracy**
    - **Validates: Requirements 3.3**

  - [x] 8.8 Implement session termination with token revocation
    - Invalidate session immediately
    - Notify Token Service to revoke tokens
    - _Requirements: 3.4_

  - [x] 8.9 Write property test for session termination effectiveness
    - **Property 11: Session Termination Effectiveness**
    - **Validates: Requirements 3.4**

  - [x] 8.10 Implement risk scoring and step-up authentication trigger
    - Calculate risk score from device/behavior
    - Trigger step-up when threshold exceeded
    - _Requirements: 3.5, 3.6_

  - [x] 8.11 Write property test for risk scoring triggers step-up
    - **Property 12: Risk Scoring Triggers Step-Up**
    - **Validates: Requirements 3.5**

  - [x] 8.12 Implement WebSocket channel for session events
    - Broadcast session create/terminate events
    - _Requirements: 3.2_

- [x] 9. Checkpoint - Session Core Foundation
  - All tests pass

- [x] 10. Implement OAuth 2.0/OIDC flows

  - [x] 10.1 Implement OAuth client validation
    - Validate client_id and redirect_uri
    - _Requirements: 4.1_

  - [x] 10.2 Implement PKCE code_challenge validation
    - Require PKCE for public clients
    - Store code_challenge with auth code
    - _Requirements: 4.1, 4.2_

  - [x] 10.3 Write property test for OAuth request validation
    - **Property 13: OAuth Request Validation**
    - **Validates: Requirements 4.1, 4.2**

  - [x] 10.4 Implement authorization code generation
    - Generate code bound to code_challenge
    - _Requirements: 4.3_

  - [x] 10.5 Implement PKCE code_verifier verification (S256)
    - Verify S256(code_verifier) == code_challenge
    - Log attack attempts on failure
    - _Requirements: 4.4, 4.5_

  - [x] 10.6 Write property test for PKCE verification correctness
    - **Property 14: PKCE Verification Correctness**
    - **Validates: Requirements 4.4**

  - [x] 10.7 Implement OIDC id_token generation with standard claims
    - Include sub, iss, aud, exp, iat, nonce
    - _Requirements: 4.6_

  - [x] 10.8 Write property test for OIDC token claims completeness
    - **Property 15: OIDC Token Claims Completeness**
    - **Validates: Requirements 4.6**

  - [x] 10.9 Implement gRPC server with OAuth endpoints
    - AuthorizeOAuth, ExchangeCode, ValidatePKCE
    - _Requirements: 4.1, 4.4_

- [x] 11. Checkpoint - OAuth Flows
  - All tests pass

## Phase 6: MFA Service (Elixir)

- [x] 12. Implement MFA Service core functionality

  - [x] 12.1 Set up Elixir project for MFA service
    - Initialize Mix project
    - Add NimbleTOTP, WebAuthn dependencies
    - _Requirements: 6.1_

  - [x] 12.2 Implement TOTP enrollment with encrypted secret storage
    - Generate secret key
    - Store encrypted in PostgreSQL
    - Return provisioning URI
    - _Requirements: 6.1_

  - [x] 12.3 Implement TOTP validation with time window tolerance
    - Validate current and ±1 time steps
    - _Requirements: 6.2_

  - [x] 12.4 Write property test for TOTP validation window
    - **Property 18: TOTP Validation Window**
    - **Validates: Requirements 6.2**

  - [x] 12.5 Implement WebAuthn credential registration
    - Generate challenge
    - Verify attestation
    - Store credential public key
    - _Requirements: 6.3_

  - [x] 12.6 Implement WebAuthn authentication with sign count validation
    - Generate challenge
    - Verify assertion signature
    - Update and validate sign count
    - _Requirements: 6.4_

  - [x] 12.7 Write property test for WebAuthn sign count monotonicity
    - **Property 19: WebAuthn Sign Count Monotonicity**
    - **Validates: Requirements 6.4**

  - [x] 12.8 Implement push authentication with timeout
    - Send push notification
    - Await approval within timeout (default 60s)
    - _Requirements: 6.5_

  - [x] 12.9 Implement device fingerprint change detection
    - Compare fingerprints
    - Require re-auth on significant change
    - _Requirements: 6.6_

  - [x] 12.10 Write property test for device fingerprint change detection
    - **Property 20: Device Fingerprint Change Detection**
    - **Validates: Requirements 6.6**

  - [x] 12.11 Implement backup codes generation and verification
    - Generate one-time backup codes
    - Mark as used after verification
    - _Requirements: 6.1_

  - [x] 12.12 Implement gRPC server with MFA endpoints
    - EnrollTOTP, VerifyTOTP, WebAuthn registration/auth, Push, BackupCodes
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 13. Checkpoint - MFA Service
  - All tests pass

## Phase 7: Inter-Service Security and mTLS

- [x] 14. Implement mTLS and SPIFFE/SPIRE integration

  - [x] 14.1 Configure SPIFFE/SPIRE workload identity for all services
    - Set up SPIRE agent configuration
    - Define SPIFFE IDs for each service
    - _Requirements: 7.1_

  - [x] 14.2 Implement mTLS certificate validation in all services
    - Reject requests without valid certificates
    - Validate against SPIFFE trust bundle
    - _Requirements: 7.3, 7.4_

  - [x] 14.3 Write property test for mTLS certificate validation
    - **Property 21: mTLS Certificate Validation**
    - **Validates: Requirements 7.3, 7.4**

  - [x] 14.4 Configure automatic certificate rotation (24h before expiry)
    - Set up SPIRE agent rotation
    - _Requirements: 7.2_

- [x] 15. Checkpoint - mTLS Integration
  - All tests pass

## Phase 8: Audit Logging and Observability

- [x] 16. Implement audit logging infrastructure

  - [x] 16.1 Set up ClickHouse schema for audit events
    - Create auth_audit_events table
    - Configure partitioning and TTL
    - _Requirements: 8.3_

  - [x] 16.2 Implement structured audit event logging in all services
    - Log timestamp, user ID, action, result, correlation ID
    - _Requirements: 8.1_

  - [x] 16.3 Write property test for audit log completeness
    - **Property 22: Audit Log Completeness**
    - **Validates: Requirements 8.1, 8.2**

  - [x] 16.4 Implement Kafka/NATS event publishing
    - Publish auth events for downstream consumers
    - _Requirements: 9.4_

  - [x] 16.5 Implement audit log query with filtering
    - Filter by user, action, time range, correlation ID
    - _Requirements: 8.4_

  - [x] 16.6 Write property test for audit log query filtering
    - **Property 23: Audit Log Query Filtering**
    - **Validates: Requirements 8.4**

  - [x] 16.7 Implement Prometheus metrics endpoints in all services
    - Expose latency percentiles (p50, p95, p99)
    - Expose error rates and request counts
    - _Requirements: 8.5_

  - [x] 16.8 Implement W3C Trace Context propagation
    - Propagate traceparent and tracestate headers
    - _Requirements: 8.6_

- [x] 17. Checkpoint - Observability
  - All tests pass

## Phase 9: Integration and End-to-End Testing

- [x] 18. Integration testing across services

  - [x] 18.1 Create integration test suite for full authentication flow
    - Test OAuth flow with PKCE end-to-end
    - Test MFA enrollment and verification
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 6.1_

  - [x] 18.2 Create integration test suite for authorization flow
    - Test policy evaluation with RBAC/ABAC
    - Test token validation and claims forwarding
    - _Requirements: 1.4, 5.1, 5.2, 5.3_

  - [x] 18.3 Create integration test suite for session management
    - Test session creation, listing, termination
    - Test WebSocket event broadcasting
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 19. Final Checkpoint - All Tests Passing
  - All implementation complete
