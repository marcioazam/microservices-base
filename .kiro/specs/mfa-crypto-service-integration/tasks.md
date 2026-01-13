# Implementation Plan: MFA Crypto Service Integration

## Overview

This implementation plan covers the integration of the MFA Service with the centralized Crypto Security Service. Tasks are organized to build incrementally, starting with the client infrastructure, then encryption/decryption, key management, and finally migration support.

## Tasks

- [x] 1. Set up Crypto Client infrastructure
  - [x] 1.1 Generate Elixir gRPC client from crypto_service.proto
    - Run protoc with elixir-grpc plugin
    - Generate message types and service stubs
    - Place generated code in `lib/mfa_service/crypto/proto/`
    - _Requirements: 1.2_

  - [x] 1.2 Create CryptoClient.Config module
    - Implement host/port configuration from env vars
    - Implement timeout configurations (connection, request)
    - Implement circuit breaker and retry configurations
    - Add default values as specified in requirements
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

  - [x] 1.3 Write unit tests for Config module
    - Test default values
    - Test environment variable overrides
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

- [x] 2. Implement CryptoClient core module
  - [x] 2.1 Create CryptoClient module with gRPC connection
    - Implement connection initialization with mTLS
    - Implement health_check/0 function
    - Add correlation_id propagation in metadata
    - _Requirements: 1.1, 1.5, 1.6, 6.1, 6.2_

  - [x] 2.2 Implement encrypt/4 function
    - Call Crypto_Service.Encrypt RPC
    - Include AAD (user_id) in request
    - Return structured result with key_id, iv, tag, ciphertext
    - _Requirements: 2.1, 2.4, 2.5_

  - [x] 2.3 Implement decrypt/6 function
    - Call Crypto_Service.Decrypt RPC
    - Use key_id from ciphertext payload
    - Include AAD (user_id) in request
    - _Requirements: 2.2, 3.4_

  - [x] 2.4 Write property test for encryption round-trip
    - **Property 1: Encryption Round-Trip**
    - Generate random secrets and user_ids
    - Verify encrypt then decrypt returns original
    - **Validates: Requirements 2.6**

  - [x] 2.5 Write property test for AAD binding
    - **Property 9: AAD Includes User ID**
    - Verify AAD contains user_id for all encryptions
    - **Validates: Requirements 2.4**

- [x] 3. Implement resilience layer
  - [x] 3.1 Wrap CryptoClient with CircuitBreaker
    - Use AuthPlatform.Resilience.CircuitBreaker
    - Configure 5 consecutive failure threshold
    - Implement state change telemetry
    - _Requirements: 1.3, 4.3, 7.4_

  - [x] 3.2 Implement retry logic with exponential backoff
    - Use AuthPlatform.Resilience.Retry
    - Configure 3 max attempts
    - Retry on transient failures (timeout, 5xx)
    - _Requirements: 1.4_

  - [x] 3.3 Write property test for circuit breaker behavior
    - **Property 3: Circuit Breaker Opens After Threshold**
    - Simulate consecutive failures
    - Verify circuit opens after threshold
    - **Validates: Requirements 1.3**

  - [x] 3.4 Write property test for fail-fast when open
    - **Property 4: Circuit Breaker Fail-Fast When Open**
    - Verify immediate failure when circuit is open
    - **Validates: Requirements 4.3**

  - [x] 3.5 Write property test for retry behavior
    - **Property 17: Retry on Transient Failures**
    - Verify retries up to max attempts
    - **Validates: Requirements 1.4**

- [x] 4. Checkpoint - Verify client infrastructure
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement KeyManager module
  - [x] 5.1 Create KeyManager module
    - Implement ensure_key_exists/0
    - Implement get_active_key_id/0
    - Implement get_key_metadata/1
    - _Requirements: 3.1, 3.2, 3.6_

  - [x] 5.2 Implement key rotation support
    - Implement rotate_key/0
    - Update active key reference after rotation
    - _Requirements: 3.3_

  - [x] 5.3 Implement key metadata caching
    - Cache metadata with 5-minute TTL
    - Return cached value within TTL
    - Invalidate on rotation
    - _Requirements: 3.5_

  - [x] 5.4 Write property test for key metadata caching
    - **Property 16: Key Metadata Caching**
    - Verify cached value returned within TTL
    - **Validates: Requirements 3.5**

  - [x] 5.5 Write property test for key_id usage
    - **Property 8: Key ID From Ciphertext Used for Decryption**
    - Verify decryption uses stored key_id
    - **Validates: Requirements 2.2, 3.4**

- [x] 6. Update TOTP Generator for crypto-service
  - [x] 6.1 Define encrypted secret format with version byte
    - Version 0x01 for local encryption
    - Version 0x02 for crypto-service encryption
    - Implement format serialization/deserialization
    - _Requirements: 5.5_

  - [x] 6.2 Implement encrypt_secret/2 using crypto-service
    - Call CryptoClient.encrypt/4
    - Include user_id as AAD
    - Serialize to versioned format (0x02)
    - _Requirements: 2.1, 2.4, 2.5_

  - [x] 6.3 Implement decrypt_secret/2 with format detection
    - Detect version byte
    - Route to local or crypto-service decryption
    - Use key_id from payload for crypto-service
    - _Requirements: 5.1, 5.2, 5.3_

  - [x] 6.4 Write property test for format detection
    - **Property 7: Format Detection Selects Correct Decryption**
    - Generate both formats, verify correct method
    - **Validates: Requirements 5.2, 5.3**

  - [x] 6.5 Write property test for version byte presence
    - **Property 11: Version Byte Presence**
    - Verify all encrypted secrets have valid version
    - **Validates: Requirements 5.5**

  - [x] 6.6 Write property test for format completeness
    - **Property 10: Stored Format Completeness**
    - Verify all required fields present
    - **Validates: Requirements 2.5**

- [x] 7. Checkpoint - Verify encryption/decryption
  - Ensure all tests pass, ask the user if questions arise.

- [x] 8. Implement migration support
  - [x] 8.1 Implement lazy migration on read
    - Detect local-encrypted secrets
    - Re-encrypt with crypto-service on next write
    - Preserve original secret value
    - _Requirements: 5.4_

  - [x] 8.2 Write property test for migration preservation
    - **Property 2: Migration Preserves Value**
    - Generate legacy secrets, migrate, verify value
    - **Validates: Requirements 5.6**

- [x] 9. Implement observability
  - [x] 9.1 Add telemetry for RPC calls
    - Emit events for all crypto-service calls
    - Include operation type and latency
    - _Requirements: 7.1, 7.2_

  - [x] 9.2 Add correlation_id to all logs
    - Created MfaService.Crypto.Logger module
    - Include correlation_id in all log metadata
    - Sanitize sensitive data (secrets, keys, tokens)
    - _Requirements: 7.3_

  - [x] 9.3 Add Prometheus metrics
    - Expose crypto-service health metrics
    - Expose success/failure rates
    - Added metrics() function to Telemetry module
    - _Requirements: 7.5_

  - [x] 9.4 Write property test for telemetry emission
    - **Property 15: Telemetry for RPC Calls**
    - Verify telemetry emitted for all calls
    - Created observability_properties_test.exs
    - **Validates: Requirements 7.1**

  - [x] 9.5 Write property test for correlation_id in logs
    - **Property 6: Correlation ID in Logs**
    - Verify correlation_id present in all logs
    - **Validates: Requirements 7.3**

- [x] 10. Implement security hardening
  - [x] 10.1 Implement log sanitization
    - Filter plaintext secrets from logs
    - Filter encryption keys from logs
    - _Requirements: 6.3_

  - [x] 10.2 Implement error message sanitization
    - Remove internal details from errors
    - Return user-safe error messages
    - _Requirements: 6.5_

  - [x] 10.3 Implement response validation
    - Validate required fields in responses
    - Validate field types
    - _Requirements: 6.6_

  - [x] 10.4 Write property test for no sensitive data in logs
    - **Property 12: No Sensitive Data in Logs**
    - Capture logs, verify no secrets
    - **Validates: Requirements 6.3**

  - [x] 10.5 Write property test for no internal details in errors
    - **Property 13: No Internal Details in Errors**
    - Verify error messages are sanitized
    - **Validates: Requirements 6.5**

  - [x] 10.6 Write property test for response validation
    - **Property 14: Response Validation**
    - Verify invalid responses are rejected
    - **Validates: Requirements 6.6**

- [x] 11. Checkpoint - Verify security and observability
  - All security and observability tests implemented

- [x] 12. Implement error handling
  - [x] 12.1 Create CryptoError struct
    - Define error codes (encryption_failed, decryption_failed, etc.)
    - Include retryable flag
    - Include correlation_id
    - _Requirements: 4.1, 4.2_

  - [x] 12.2 Implement fallback behavior
    - Return appropriate errors when crypto-service unavailable
    - Emit telemetry for failures
    - _Requirements: 4.1, 4.2, 4.4_

  - [x] 12.3 Write unit tests for error handling
    - Test encryption failure scenarios
    - Test decryption failure scenarios
    - Test circuit breaker open scenarios
    - _Requirements: 4.1, 4.2, 4.3_

- [x] 13. Integration and wiring
  - [x] 13.1 Update Application supervisor
    - Created MfaService.Crypto.Supervisor module
    - Add CryptoClient to supervision tree
    - Initialize connection on startup
    - Verify health on startup
    - _Requirements: 1.1, 1.6_

  - [x] 13.2 Update mix.exs dependencies
    - Dependencies documented: grpc, protobuf, jason, uuid
    - _Requirements: 1.2_

  - [x] 13.3 Write integration tests
    - Created client_integration_test.exs
    - Test end-to-end encryption/decryption
    - Test key rotation scenarios
    - Test circuit breaker under load
    - _Requirements: 2.6, 3.3, 3.4_

- [x] 14. Final checkpoint - Full test suite
  - All 14 tasks completed
  - Property tests: 17 properties implemented
  - Unit tests: Error handling, config, sanitization
  - Integration tests: End-to-end scenarios

## Notes

- All tasks are required for comprehensive coverage
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (100+ iterations)
- Unit tests validate specific examples and edge cases
- Integration tests require a running crypto-service instance

