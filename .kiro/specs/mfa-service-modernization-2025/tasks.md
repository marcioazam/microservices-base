# Implementation Plan: MFA Service Modernization 2025

## Overview

This implementation plan modernizes the MFA Service to state-of-the-art December 2025 standards. Tasks are organized to build incrementally, with property tests validating correctness at each stage.

## Tasks

- [x] 1. Update project configuration and dependencies ✅
  - ✅ Update mix.exs to Elixir 1.17+ requirement
  - ✅ Add auth_platform and auth_platform_clients dependencies
  - ✅ Update existing dependencies to latest stable versions (grpc 0.9, protobuf 0.13, ecto_sql 3.12, postgrex 0.19, stream_data 1.1)
  - ✅ Configure test coverage reporting (ExCoveralls)
  - ✅ Added mox for mocking, benchee for benchmarks, req for HTTP
  - _Requirements: 9.6, 10.5_

- [x] 2. Implement centralized challenge storage module
  - [x] 2.1 Create MfaService.Challenge module using Cache_Service
    - Implement generate/0 with 32-byte cryptographic randomness
    - Implement store/3 with Cache_Service and TTL support
    - Implement retrieve/1 and retrieve_and_delete/1
    - Implement verify/2 using AuthPlatform.Security.constant_time_compare
    - _Requirements: 1.2, 4.1, 4.2_

  - [x] 2.2 Write property test for challenge entropy
    - **Property 5: WebAuthn Challenge Entropy**
    - **Validates: Requirements 4.1**

  - [x] 2.3 Write property test for challenge encode-decode round-trip
    - **Property 6: WebAuthn Challenge Encode-Decode Round-Trip**
    - **Validates: Requirements 4.7**

- [x] 3. Modernize TOTP Generator module
  - [x] 3.1 Refactor TOTP.Generator to use AuthPlatform.Security
    - Use :crypto.strong_rand_bytes for secret generation
    - Implement AES-256-GCM encryption/decryption
    - Generate provisioning URIs per RFC 6238
    - _Requirements: 3.1, 3.4, 3.5_

  - [x] 3.2 Write property test for TOTP secret entropy
    - **Property 1: TOTP Secret Entropy**
    - **Validates: Requirements 3.1**

  - [x] 3.3 Write property test for TOTP encryption round-trip
    - **Property 2: TOTP Encryption Round-Trip**
    - **Validates: Requirements 3.4**

  - [x] 3.4 Write property test for provisioning URI format
    - **Property 3: TOTP Provisioning URI Format**
    - **Validates: Requirements 3.5**

- [x] 4. Modernize TOTP Validator module
  - [x] 4.1 Refactor TOTP.Validator to use AuthPlatform.Security
    - Use constant_time_compare for code comparison
    - Validate codes within ±1 time window
    - _Requirements: 3.2, 3.3_

  - [x] 4.2 Write property test for TOTP generate-validate round-trip
    - **Property 4: TOTP Generate-Validate Round-Trip**
    - **Validates: Requirements 3.2, 3.6**

- [x] 5. Checkpoint - Ensure TOTP tests pass
  - Ensure all TOTP tests pass, ask the user if questions arise.

- [x] 6. Modernize WebAuthn Authentication module
  - [x] 6.1 Refactor WebAuthn.Authentication to use centralized Challenge module
    - Replace direct ETS/Redis with MfaService.Challenge
    - Use Cache_Service for challenge storage
    - Implement sign count monotonicity check
    - _Requirements: 4.2, 4.4, 4.5_

  - [x] 6.2 Write property test for sign count monotonicity
    - **Property 7: WebAuthn Sign Count Monotonicity**
    - **Validates: Requirements 4.5**

  - [x] 6.3 Write property test for authenticator data parsing
    - **Property 8: WebAuthn Authenticator Data Parsing**
    - **Validates: Requirements 4.6**

- [x] 7. Modernize Passkeys Registration module
  - [x] 7.1 Refactor Passkeys.Registration to use centralized Challenge module
    - Replace ETS challenge storage with MfaService.Challenge
    - Use Cache_Service with 5-minute TTL
    - Validate attestation per WebAuthn Level 2
    - _Requirements: 4.2, 4.3_

- [x] 8. Modernize Passkeys Authentication module
  - [x] 8.1 Refactor Passkeys.Authentication to use centralized Challenge module
    - Replace ETS challenge storage with MfaService.Challenge
    - Implement signature verification
    - _Requirements: 4.2, 4.4_

- [x] 9. Modernize Passkeys Management module
  - [x] 9.1 Refactor Passkeys.Management with AuthPlatform.Validation
    - Validate passkey name length (max 255)
    - Enforce recent authentication for deletion (5 minutes)
    - Check for alternative methods before last passkey deletion
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [x] 9.2 Write property test for passkey list structure
    - **Property 13: Passkey List Structure Completeness**
    - **Validates: Requirements 7.1**

  - [x] 9.3 Write property test for passkey rename validation
    - **Property 14: Passkey Rename Validation**
    - **Validates: Requirements 7.2**

  - [x] 9.4 Write property test for passkey delete recent auth
    - **Property 15: Passkey Delete Recent Auth Requirement**
    - **Validates: Requirements 7.3**

- [x] 10. Checkpoint - Ensure Passkeys tests pass
  - Ensure all Passkeys tests pass, ask the user if questions arise.

- [x] 11. Modernize Cross-Device Authentication module
  - [x] 11.1 Refactor Passkeys.CrossDevice to use Cache_Service
    - Store sessions in Cache_Service with 5-minute TTL
    - Generate FIDO:// QR codes with CBOR-encoded data
    - Implement session status checking
    - _Requirements: 8.1, 8.2, 8.3, 8.5_

  - [x] 11.2 Write property test for QR code format
    - **Property 16: Cross-Device QR Code Format**
    - **Validates: Requirements 8.1**

  - [x] 11.3 Write property test for session lifecycle
    - **Property 17: Cross-Device Session Lifecycle**
    - **Validates: Requirements 8.3, 8.5**

- [x] 12. Modernize Device Fingerprint module
  - [x] 12.1 Refactor Device.Fingerprint with AuthPlatform integration
    - Use SHA-256 for fingerprint hashing
    - Calculate similarity as matching attributes ratio
    - Flag significant changes (>30% difference)
    - Handle missing headers gracefully
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

  - [x] 12.2 Write property test for fingerprint determinism
    - **Property 9: Device Fingerprint Determinism**
    - **Validates: Requirements 5.1, 5.5**

  - [x] 12.3 Write property test for fingerprint reflexivity
    - **Property 10: Device Fingerprint Reflexivity**
    - **Validates: Requirements 5.6**

  - [x] 12.4 Write property test for similarity calculation
    - **Property 11: Device Fingerprint Similarity Calculation**
    - **Validates: Requirements 5.2**

  - [x] 12.5 Write property test for significant change threshold
    - **Property 12: Device Fingerprint Significant Change Threshold**
    - **Validates: Requirements 5.3**

- [x] 13. Checkpoint - Ensure Device Fingerprint tests pass
  - Ensure all Device Fingerprint tests pass, ask the user if questions arise.

- [x] 14. Modernize CAEP Emitter module
  - [x] 14.1 Refactor Caep.Emitter to use Logging_Service
    - Use AuthPlatform.Clients.Logging for audit logging
    - Emit credential-change events for passkey/TOTP changes
    - Handle CAEP service unavailability gracefully
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 14.2 Write unit tests for CAEP event emission
    - Test passkey added/removed events
    - Test TOTP enabled/disabled events
    - Test fallback on CAEP unavailability
    - _Requirements: 6.1, 6.2, 6.3, 6.5_

- [x] 15. Modernize Application startup
  - [x] 15.1 Update MfaService.Application for platform client initialization
    - Initialize Cache_Service circuit breaker
    - Initialize Logging_Service circuit breaker
    - Configure telemetry handlers
    - _Requirements: 1.1, 1.4, 1.5, 2.3, 2.4, 2.6_

  - [x] 15.2 Write integration tests for platform clients
    - Test Cache_Service connection and operations
    - Test Logging_Service connection and fallback
    - Test circuit breaker behavior
    - _Requirements: 1.4, 1.5_

- [x] 16. Add error handling and observability
  - [x] 16.1 Implement consistent error handling with AppError
    - Use AuthPlatform.Errors.AppError for all errors
    - Sanitize error messages to prevent internal detail leakage
    - Add correlation IDs to all log entries
    - _Requirements: 9.4, 11.6_

  - [x] 16.2 Write property test for error message sanitization
    - **Property 18: Error Message Sanitization**
    - **Validates: Requirements 11.6**

  - [x] 16.3 Add telemetry events for all MFA operations
    - Emit latency metrics for registration/authentication
    - Emit success/failure counts
    - _Requirements: 9.5, 12.5_

- [x] 17. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 18. Reorganize test directory structure
  - [x] 18.1 Move existing tests to proper directories
    - Move unit tests to test/mfa_service/{module}/
    - Move property tests to test/property/
    - Create test/integration/ for platform client tests
    - _Requirements: 9.1_

  - [x] 18.2 Update test helpers and fixtures
    - Create shared test generators for StreamData
    - Create mock fixtures for platform services
    - _Requirements: 10.1, 10.2, 10.3_

- [x] 19. Add benchmark tests for latency SLOs
  - [x] 19.1 Write benchmark tests for registration options
    - Test p99 latency < 200ms
    - _Requirements: 12.1_

  - [x] 19.2 Write benchmark tests for authentication options
    - Test p99 latency < 100ms
    - _Requirements: 12.2_

  - [x] 19.3 Write benchmark tests for TOTP validation
    - Test p99 latency < 50ms
    - _Requirements: 12.3_

  - [x] 19.4 Write benchmark tests for WebAuthn assertion
    - Test p99 latency < 150ms
    - _Requirements: 12.4_

- [x] 20. Final cleanup and validation
  - [x] 20.1 Remove redundant code and dead imports
    - Remove direct Redis/ETS challenge storage code
    - Remove unused dependencies
    - Ensure no files exceed 400 lines
    - _Requirements: 9.2, 9.3_

  - [x] 20.2 Run full test suite and coverage report
    - Verify 80%+ code coverage
    - Verify all tests pass
    - _Requirements: 10.5, 10.6_

- [x] 21. Final checkpoint - Production ready
  - Ensure all tests pass and coverage meets requirements, ask the user if questions arise.

## Notes

- All tasks are required for production readiness
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (18 properties total)
- Unit tests validate specific examples and edge cases
- Benchmark tests validate latency SLOs
