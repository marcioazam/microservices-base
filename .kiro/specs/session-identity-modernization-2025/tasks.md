# Implementation Plan: Session Identity Modernization 2025

## Overview

This implementation plan modernizes the Session Identity Core service to December 2025 state-of-the-art standards. The approach follows incremental development with property-based testing validation at each step.

## Tasks

- [x] 1. Set up project structure and shared utilities
  - [x] 1.1 Create centralized Shared.Keys module for Redis key generation
    - Implement session_key/1, user_sessions_key/1, oauth_code_key/1, event_key/1
    - Remove duplicate key generation from SessionStore, Authorization, Store
    - _Requirements: 2.4_
  - [x] 1.2 Create centralized Shared.TTL module for TTL calculations
    - Implement calculate/1, default_session_ttl/0, default_code_ttl/0, default_expiry/0
    - Remove duplicate TTL logic from SessionStore, Authorization
    - _Requirements: 2.6_
  - [x] 1.3 Create centralized Shared.DateTime module
    - Implement to_iso8601/1, from_iso8601/1
    - Remove duplicate datetime parsing from SessionSerializer, Event, Aggregate
    - _Requirements: 2.2_
  - [x] 1.4 Create centralized Shared.Errors module
    - Define all error types: session errors, OAuth errors, PKCE errors, event store errors
    - Implement RFC 6749 compliant OAuth error responses
    - _Requirements: 2.7, 11.1, 11.2, 11.3_

- [x] 2. Checkpoint - Verify shared utilities
  - Shared utilities created and ready for use.

- [x] 3. Integrate platform services
  - [x] 3.1 Add auth_platform_clients dependency to mix.exs
    - Add {:auth_platform, path: "../../libs/elixir/apps/auth_platform"}
    - Add {:auth_platform_clients, path: "../../libs/elixir/apps/auth_platform_clients"}
    - _Requirements: 1.1, 1.2_
  - [x] 3.2 Refactor SessionStore to use Cache Service client
    - Replace Redix.command calls with AuthPlatform.Clients.Cache calls
    - Use Shared.Keys for key generation
    - Use Shared.TTL for TTL calculations
    - _Requirements: 1.1, 1.3_
  - [x] 3.3 Add structured logging using Logging Service client
    - Replace Logger calls with AuthPlatform.Clients.Logging calls
    - Add correlation_id to all log entries
    - Implement fallback to local Logger when service unavailable
    - _Requirements: 1.2, 1.4_
  - [x] 3.4 Write property test for session key namespacing
    - **Property 8: Session Key Namespacing**
    - **Validates: Requirements 4.7**

- [x] 4. Modernize Session Serializer
  - [x] 4.1 Refactor SessionSerializer to use Shared.DateTime
    - Use centralized datetime functions
    - Ensure round-trip guarantee
    - _Requirements: 2.1, 2.5_
  - [x] 4.2 Write property test for session serialization round-trip
    - **Property 1: Session Serialization Round-Trip**
    - **Validates: Requirements 2.5**

- [x] 5. Checkpoint - Verify platform integration
  - Platform integration complete.

- [x] 6. Modernize OAuth 2.1 module
  - [x] 6.1 Refactor OAuth21 module to use Shared.Errors
    - Use centralized error definitions
    - Ensure RFC 9700 compliance
    - _Requirements: 3.1, 3.2, 3.3_
  - [x] 6.2 Enforce PKCE for all clients (including confidential)
    - Remove client_type check for PKCE requirement
    - Reject plain method
    - _Requirements: 3.3, 3.4_
  - [x] 6.3 Implement exact redirect_uri matching
    - Remove any pattern matching logic
    - Enforce string equality only
    - _Requirements: 3.5_
  - [x] 6.4 Implement refresh token rotation
    - Generate new token on every refresh
    - Invalidate old token
    - _Requirements: 3.6_
  - [x] 6.5 Write property test for redirect URI exact matching
    - **Property 3: Redirect URI Exact Matching**
    - **Validates: Requirements 3.5**
  - [x] 6.6 Write property test for refresh token rotation
    - **Property 4: Refresh Token Rotation**
    - **Validates: Requirements 3.6**

- [x] 7. Modernize PKCE module
  - [x] 7.1 Refactor PKCE module to use AuthPlatform.Security
    - Use Security.constant_time_compare for verification
    - Use Shared.Errors for error responses
    - _Requirements: 1.5, 3.7_
  - [x] 7.2 Enforce S256 only, reject plain method
    - Return pkce_plain_not_allowed error for plain method
    - _Requirements: 3.4_
  - [x] 7.3 Validate code_verifier length and charset
    - Enforce 43-128 character length
    - Enforce valid charset [A-Za-z0-9\-._~]
    - _Requirements: 3.8_
  - [x] 7.4 Write property test for PKCE verification correctness
    - **Property 2: PKCE Verification Correctness**
    - **Validates: Requirements 3.3, 3.7, 3.8**

- [x] 8. Checkpoint - Verify OAuth/PKCE modernization
  - OAuth 2.1 and PKCE modules modernized.

- [x] 9. Modernize Session Manager
  - [x] 9.1 Implement 256-bit entropy token generation
    - Use :crypto.strong_rand_bytes(32) for session tokens
    - Use AuthPlatform.Security.generate_token for consistency
    - _Requirements: 4.1_
  - [x] 9.2 Ensure device binding on session creation
    - Validate device_fingerprint and ip_address are present
    - Store in session record
    - _Requirements: 4.3_
  - [x] 9.3 Implement session fixation protection
    - Regenerate session ID on privilege escalation (MFA verification)
    - _Requirements: 4.6_
  - [x] 9.4 Write property test for session token entropy
    - **Property 5: Session Token Entropy**
    - **Validates: Requirements 4.1**
  - [x] 9.5 Write property test for session device binding
    - **Property 6: Session Device Binding**
    - **Validates: Requirements 4.3**

- [x] 10. Modernize Risk Scorer
  - [x] 10.1 Ensure risk score bounds [0.0, 1.0]
    - Add clamp function to ensure bounds
    - _Requirements: 5.1_
  - [x] 10.2 Implement step-up threshold at 0.7
    - requires_step_up? returns true for score >= 0.7
    - _Requirements: 5.2_
  - [x] 10.3 Implement high-risk factors at 0.9
    - get_required_factors returns [:webauthn, :totp] for score >= 0.9
    - _Requirements: 5.3_
  - [x] 10.4 Implement known device risk reduction
    - device_risk = 0.0 when device_fingerprint in known_devices
    - _Requirements: 5.5_
  - [x] 10.5 Implement failed attempts behavior risk
    - behavior_risk = 0.9 when failed_attempts >= 5
    - _Requirements: 5.6_
  - [x] 10.6 Write property test for risk score bounds and thresholds
    - **Property 9: Risk Score Bounds and Thresholds**
    - **Validates: Requirements 5.1, 5.2, 5.3**
  - [x] 10.7 Write property test for risk factors affect score
    - **Property 10: Risk Factors Affect Score**
    - **Validates: Requirements 5.4, 5.5, 5.6**

- [x] 11. Checkpoint - Verify session and risk modernization
  - Session and risk modules modernized.

- [x] 12. Modernize ID Token module
  - [x] 12.1 Ensure required claims completeness
    - Validate sub, iss, aud, exp, iat are present
    - _Requirements: 6.1_
  - [x] 12.2 Implement nonce inclusion
    - Include nonce in token when provided in request
    - _Requirements: 6.2_
  - [x] 12.3 Implement configurable TTL
    - exp = iat + ttl (default 3600)
    - _Requirements: 6.3_
  - [x] 12.4 Support optional claims
    - Include auth_time, acr, amr, azp when provided
    - _Requirements: 6.4_
  - [x] 12.5 Implement claims validation
    - Validate all required claims before signing
    - _Requirements: 6.5_
  - [x] 12.6 Write property test for ID token claims completeness
    - **Property 11: ID Token Claims Completeness**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**

- [x] 13. Modernize Event Store
  - [x] 13.1 Refactor Event module to use Shared.DateTime
    - Use centralized datetime functions for serialization
    - _Requirements: 7.5_
  - [x] 13.2 Ensure monotonic sequence numbers
    - Use atomic increment for sequence generation
    - _Requirements: 7.1_
  - [x] 13.3 Ensure correlation_id on all events
    - Generate correlation_id if not provided
    - _Requirements: 7.3_
  - [x] 13.4 Implement schema versioning
    - Add migrate_schema function for version upgrades
    - _Requirements: 7.4_
  - [x] 13.5 Implement snapshot loading
    - Support loading from snapshot + subsequent events
    - _Requirements: 7.6_
  - [x] 13.6 Write property test for event structure correctness
    - **Property 12: Event Structure Correctness**
    - **Validates: Requirements 7.1, 7.3, 7.5**
  - [x] 13.7 Write property test for event replay consistency
    - **Property 13: Event Replay Consistency**
    - **Validates: Requirements 7.2, 7.6**

- [x] 14. Modernize Session Aggregate
  - [x] 14.1 Ensure SessionCreated events have correlation_id
    - Generate correlation_id on session creation
    - _Requirements: 4.4_
  - [x] 14.2 Ensure SessionInvalidated events have reason
    - Include reason field in termination events
    - _Requirements: 4.5_
  - [x] 14.3 Write property test for session events correlation
    - **Property 7: Session Events Correlation**
    - **Validates: Requirements 4.4, 4.5**

- [x] 15. Checkpoint - Verify event store modernization
  - Event store modules modernized.

- [x] 16. Modernize CAEP Emitter
  - [x] 16.1 Implement SSF-compliant event format
    - Include event_type, subject, event_timestamp, reason_admin
    - _Requirements: 8.4_
  - [x] 16.2 Implement logout event emission
    - Emit session-revoked on user logout
    - _Requirements: 8.1_
  - [x] 16.3 Implement admin termination event emission
    - Emit session-revoked with admin context
    - _Requirements: 8.2_
  - [x] 16.4 Implement security violation event emission
    - Emit session-revoked with reason
    - _Requirements: 8.3_
  - [x] 16.5 Implement failure handling
    - Log failure and continue operation
    - _Requirements: 8.5_
  - [x] 16.6 Implement configurable enable/disable
    - Support CAEP_ENABLED environment variable
    - _Requirements: 8.6_
  - [x] 16.7 Write property test for CAEP event format
    - **Property 14: CAEP Event Format**
    - **Validates: Requirements 8.4**

- [x] 17. Configuration and health checks
  - [x] 17.1 Implement environment-based configuration
    - Read all config from environment variables
    - Provide sensible defaults
    - _Requirements: 12.1, 12.2_
  - [x] 17.2 Implement configuration validation
    - Validate config at startup, fail fast on invalid
    - _Requirements: 12.3_
  - [x] 17.3 Implement health checks
    - Add readiness and liveness endpoints
    - _Requirements: 9.5_
  - [x] 17.4 Implement Prometheus metrics
    - Expose metrics via telemetry
    - _Requirements: 9.6_
  - [x] 17.5 Implement OpenTelemetry tracing
    - Add W3C Trace Context support
    - _Requirements: 9.7_

- [x] 18. Final cleanup and test organization
  - [x] 18.1 Reorganize test directory structure
    - Separate source from test code
    - Mirror source structure in tests
    - _Requirements: 9.1_
  - [x] 18.2 Create shared test generators module
    - Centralize StreamData generators
    - _Requirements: 10.6_
  - [x] 18.3 Remove dead code and legacy patterns
    - Remove any unused modules
    - Remove deprecated patterns
    - _Requirements: Architecture modernization_
  - [x] 18.4 Run full test suite and verify coverage
    - Ensure 80% minimum coverage
    - Ensure all property tests pass with 100+ iterations
    - _Requirements: 10.7_

- [x] 19. Final checkpoint - Production readiness
  - All 14 correctness properties implemented
  - Platform service integration complete
  - Configuration and health checks implemented

## Notes

- All tasks are required for comprehensive correctness validation
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (14 total)
- Unit tests validate specific examples and edge cases
- All property tests must run with minimum 100 iterations
- 80% minimum code coverage required on core modules
