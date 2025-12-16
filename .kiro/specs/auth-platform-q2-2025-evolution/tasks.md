# Implementation Plan

## Phase 1: Passkeys (WebAuthn Discoverable Credentials)

- [x] 1. Set up Passkeys infrastructure
  - [x] 1.1 Create database schema for passkey credentials
    - Create passkey_credentials table with all required fields
    - Add indexes for user_id and credential_id
    - Create migration scripts
    - _Requirements: 1.2, 4.1_
  - [x] 1.2 Add WebAuthn dependencies to MFA Service


    - Add CBOR library to mix.exs for attestation parsing
    - Configure RP ID and origin settings in runtime config
    - _Requirements: 1.1_

- [x] 2. Implement Passkey Registration
  - [x] 2.1 Create registration options endpoint
    - Implement POST /passkeys/register/begin
    - Generate challenge with residentKey="required"
    - Support platform and roaming authenticators
    - _Requirements: 1.1, 1.3_
  - [x] 2.2 Write property test for registration options
    - **Property 1: Passkey Registration Options Correctness**
    - **Validates: Requirements 1.1, 2.1**
  - [x] 2.3 Create registration verification endpoint
    - Implement POST /passkeys/register/finish
    - Verify attestation statement
    - Store credential with discoverable flag
    - _Requirements: 1.2, 1.4_
  - [x] 2.4 Write property test for credential storage
    - **Property 2: Passkey Credential Storage Integrity**
    - **Validates: Requirements 1.2, 1.3**

- [x] 3. Implement Passkey Authentication
  - [x] 3.1 Create authentication options endpoint
    - Implement POST /passkeys/authenticate/begin
    - Support Conditional UI (mediation: "conditional")
    - Generate challenge with userVerification="required"
    - _Requirements: 2.1, 2.2_
  - [x] 3.2 Create authentication verification endpoint
    - Implement POST /passkeys/authenticate/finish
    - Verify assertion signature
    - Update sign count
    - Create session with passkey metadata
    - _Requirements: 2.3, 2.4_
  - [x] 3.3 Write property test for session metadata
    - **Property 3: Passkey Authentication Session Metadata**
    - **Validates: Requirements 2.4, 2.5**

- [x] 4. Implement Cross-Device Authentication
  - [x] 4.1 Create QR code generation for hybrid transport
    - Generate CTAP hybrid transport data
    - Create QR code endpoint
    - _Requirements: 3.1_
  - [x] 4.2 Implement hybrid transport ceremony
    - Handle hybrid transport WebAuthn flow
    - Offer local passkey registration on success
    - _Requirements: 3.3, 3.4_
  - [x] 4.3 Write property test for fallback
    - **Property 4: Cross-Device Authentication Fallback**
    - **Validates: Requirements 3.5**

- [x] 5. Implement Passkey Management



  - [x] 5.1 Create list passkeys endpoint

    - Implement GET /passkeys
    - Return device name, creation date, last used

    - _Requirements: 4.1_

  - [x] 5.2 Create rename passkey endpoint
    - Implement PATCH /passkeys/:id
    - Update friendly name only
    - _Requirements: 4.4_
  - [x] 5.3 Create delete passkey endpoint
    - Implement DELETE /passkeys/:id
    - Require re-authentication
    - Prevent deletion of last passkey without alternative
    - _Requirements: 4.2, 4.3_
  - [x] 5.4 Write property test for re-authentication

    - **Property 5: Passkey Management Re-authentication**
    - **Validates: Requirements 4.2, 4.3**

- [x] 6. Checkpoint - Passkeys Integration
  - Ensure all tests pass, ask the user if questions arise.

## Phase 2: CAEP (Continuous Access Evaluation Protocol)

- [x] 7. Set up CAEP infrastructure
  - [x] 7.1 Create CAEP shared library

    - Create auth/shared/caep Cargo project
    - Define event types and subject identifiers


    - Implement SET structure
    - _Requirements: 5.1_


  - [x] 7.2 Create database schema for CAEP streams

    - Create caep_streams table
    - Create caep_events table for audit
    - _Requirements: 7.1_


- [x] 8. Implement CAEP Transmitter
  - [x] 8.1 Create SET signing module
    - Implement ES256 signing
    - Use platform signing key
    - Include all required claims
    - _Requirements: 5.5_
  - [x] 8.2 Write property test for SET signature
    - **Property 6: CAEP SET Signature Validity**
    - **Validates: Requirements 5.5, 6.1**
  - [x] 8.3 Implement event emission

    - Create emit() method for each event type




    - Deliver to all active streams
    - Track delivery status
    - _Requirements: 5.2, 5.3, 5.4_

  - [x] 8.4 Write property test for event emission

    - **Property 7: CAEP Event Emission Completeness**
    - **Validates: Requirements 5.2, 5.3, 5.4**



- [x] 9. Implement CAEP Receiver
  - [x] 9.1 Create SET validation module
    - Validate signature against transmitter JWKS
    - Verify claims (iss, aud, iat, jti)
    - _Requirements: 6.1_
  - [x] 9.2 Implement event handlers
    - Handle session-revoked: terminate session
    - Handle credential-change: invalidate cache
    - Handle assurance-level-change: re-evaluate auth
    - _Requirements: 6.2, 6.3, 6.4_
  - [x] 9.3 Write property test for session revocation
    - **Property 8: CAEP Session Revocation Effect**
    - **Validates: Requirements 6.2**
  - [x] 9.4 Implement retry with exponential backoff
    - Retry failed event processing
    - Alert on persistent failures
    - _Requirements: 6.5_

- [x] 10. Implement Stream Management

  - [x] 10.1 Create stream configuration endpoints
    - POST /caep/streams - create stream
    - GET /caep/streams - list streams
    - DELETE /caep/streams/:id - remove stream
    - _Requirements: 7.1, 7.5_
  - [x] 10.2 Implement stream health monitoring
    - Track delivery success rate
    - Track latency percentiles
    - Track last successful delivery
    - _Requirements: 7.3_
  - [x] 10.3 Write property test for stream health
    - **Property 9: CAEP Stream Health Tracking**
    - **Validates: Requirements 7.3, 7.5**
  - [x] 10.4 Implement SSF discovery endpoint

    - Create /.well-known/ssf-configuration
    - Register streams with discovery
    - _Requirements: 7.2_


- [x] 11. Integrate CAEP with existing services

  - [x] 11.1 Add CAEP emission to Session Identity Core


    - Emit session-revoked on logout

    - Emit session-revoked on admin termination

    - _Requirements: 5.2_
  - [x] 11.2 Add CAEP emission to MFA Service
    - Emit credential-change on passkey add/remove
    - Emit credential-change on TOTP change
    - _Requirements: 5.3_
  - [x] 11.3 Add CAEP emission to IAM Policy Service
    - Emit assurance-level-change on risk change
    - Emit token-claims-change on role update
    - _Requirements: 5.4_

- [x] 12. Checkpoint - CAEP Integration
  - Ensure all tests pass, ask the user if questions arise.


## Phase 3: SDK Client Libraries


- [x] 13. Create TypeScript SDK
  - [x] 13.1 Set up TypeScript SDK project
    - Create @auth-platform/sdk package
    - Configure TypeScript, ESLint, Jest
    - Set up npm publishing
    - _Requirements: 8.1_
  - [x] 13.2 Implement OAuth 2.1 with PKCE
    - Create authorize() method
    - Implement PKCE challenge generation
    - Handle callback and token exchange
    - _Requirements: 8.2_
  - [x] 13.3 Write property test for PKCE
    - **Property 10: SDK PKCE Enforcement**
    - **Validates: Requirements 8.2, 9.4, 10.1**
  - [x] 13.4 Implement Passkeys wrapper
    - Create registerPasskey() method
    - Create authenticateWithPasskey() method
    - Handle platform-specific quirks
    - _Requirements: 8.3_
  - [x] 13.5 Implement token management
    - Create TokenManager class
    - Implement automatic refresh
    - Support multiple storage backends
    - _Requirements: 8.4_
  - [x] 13.6 Write property test for token refresh
    - **Property 11: SDK Token Refresh Automation**
    - **Validates: Requirements 8.4**
  - [x] 13.7 Implement typed errors
    - Create error class hierarchy
    - Include error codes and messages
    - _Requirements: 8.5_

- [x] 14. Create Python SDK
  - [x] 14.1 Set up Python SDK project
    - Create auth-platform-sdk package
    - Configure Poetry, pytest, mypy
    - Set up PyPI publishing
    - _Requirements: 9.1_
  - [x] 14.2 Implement sync and async clients
    - Create AuthPlatformClient (sync)
    - Create AsyncAuthPlatformClient (async)
    - Share common logic
    - _Requirements: 9.1_
  - [x] 14.3 Implement JWKS caching
    - Create JWKSCache class
    - Configurable TTL
    - Thread-safe implementation
    - _Requirements: 9.2_
  - [x] 14.4 Write property test for JWKS caching
    - **Property 12: SDK JWKS Caching**
    - **Validates: Requirements 9.2**
  - [x] 14.5 Implement rate limiting handling
    - Detect 429 responses
    - Automatic retry with backoff
    - _Requirements: 9.3_
  - [x] 14.6 Implement client credentials flow
    - Create client_credentials() method
    - Support service accounts
    - _Requirements: 9.4_
  - [x] 14.7 Create framework middleware
    - FastAPI middleware
    - Flask middleware
    - Django middleware
    - _Requirements: 9.5_


- [x] 15. Create Go SDK
  - [x] 15.1 Set up Go SDK project
    - Create auth-platform-sdk-go module
    - Configure go.mod, golangci-lint
    - Set up Go module publishing
    - _Requirements: 10.1_
  - [x] 15.2 Implement functional options
    - Create Option type
    - Implement WithTimeout, WithHTTPClient, etc.
    - _Requirements: 10.1_
  - [x] 15.3 Implement HTTP middleware
    - Create Middleware() method
    - Compatible with http.Handler
    - Extract and validate tokens
    - _Requirements: 10.2_
  - [x] 15.4 Implement gRPC interceptors
    - Create UnaryInterceptor
    - Create StreamInterceptor
    - _Requirements: 10.3_
  - [x] 15.5 Implement error handling
    - Create sentinel errors
    - Use error wrapping
    - _Requirements: 10.4_
  - [x] 15.6 Write property test for error types
    - **Property 13: SDK Error Type Safety**
    - **Validates: Requirements 8.5, 10.4**
  - [x] 15.7 Implement connection management
    - Connection pooling
    - Graceful shutdown
    - _Requirements: 10.5_

- [x] 16. Checkpoint - SDK Libraries
  - Ensure all tests pass, ask the user if questions arise.

## Phase 4: Performance and Documentation

- [x] 17. Performance optimization
  - [x] 17.1 Implement passkey operation benchmarks
    - Benchmark registration latency
    - Benchmark authentication latency
    - Ensure p99 < 200ms/100ms
    - _Requirements: 12.1, 12.2_
  - [x] 17.2 Write property test for passkey latency
    - **Property 14: Passkey Latency SLO**
    - **Validates: Requirements 12.1, 12.2**
  - [x] 17.3 Implement CAEP delivery benchmarks
    - Benchmark event emission latency
    - Ensure p99 < 100ms
    - _Requirements: 12.3_
  - [x] 17.4 Write property test for CAEP latency
    - **Property 15: CAEP Event Delivery Latency**
    - **Validates: Requirements 12.3**

- [x] 18. Create SDK documentation
  - [x] 18.1 Write TypeScript SDK documentation
    - Quickstart guide
    - API reference
    - Code examples
    - _Requirements: 11.1, 11.2_
  - [x] 18.2 Write Python SDK documentation
    - Quickstart guide
    - Framework integration guides
    - Code examples
    - _Requirements: 11.1, 11.2_
  - [x] 18.3 Write Go SDK documentation
    - Quickstart guide
    - Middleware examples
    - gRPC integration guide
    - _Requirements: 11.1, 11.2_
  - [x] 18.4 Create troubleshooting guides
    - Common errors and solutions
    - Migration guides
    - _Requirements: 11.3, 11.4_

- [x] 19. Create Passkeys and CAEP documentation
  - [x] 19.1 Write Passkeys documentation
    - Architecture overview
    - Registration flow
    - Authentication flow
    - Management API
    - _Requirements: 11.1_
  - [x] 19.2 Write CAEP documentation
    - Event types reference
    - Stream configuration
    - Integration guide
    - _Requirements: 11.1_
  - [x] 19.3 Create runbooks
    - Passkey troubleshooting
    - CAEP stream recovery
    - SDK debugging
    - _Requirements: 11.3_

- [x] 20. Final Checkpoint
  - Ensure all tests pass, ask the user if questions arise.
