# Implementation Plan: Session-Identity Crypto Service Integration

## Overview

This implementation plan breaks down the crypto-service integration into incremental tasks. Each task builds on previous work, with property tests validating correctness at each step. The implementation uses Elixir with gRPC for crypto-service communication.

## Tasks

- [x] 1. Set up project structure and dependencies
  - Add `grpc` protobuf dependencies to mix.exs
  - Generate Elixir modules from crypto_service.proto
  - Create `lib/session_identity_core/crypto/` directory structure
  - Add configuration schema for crypto integration
  - _Requirements: 1.1, 6.1_

- [x] 2. Implement CryptoClient gRPC module
  - [x] 2.1 Create base CryptoClient with connection management
  - [x] 2.2 Add trace context propagation
  - [x] 2.3 Write property test for trace context propagation
  - [x] 2.4 Add correlation_id to all requests
  - [x] 2.5 Write property test for correlation_id inclusion
  - [x] 2.6 Implement structured error handling
  - [x] 2.7 Write property test for structured errors

- [x] 3. Checkpoint - Verify CryptoClient base functionality

- [x] 4. Implement circuit breaker and fallback
  - [x] 4.1 Add circuit breaker wrapper around CryptoClient
  - [x] 4.2 Implement fallback behavior
  - [x] 4.3 Write property test for circuit breaker behavior

- [x] 5. Implement KeyManager module
  - [x] 5.1 Create KeyManager with ETS-based cache
  - [x] 5.2 Implement active key resolution
  - [x] 5.3 Write property test for key metadata caching
  - [x] 5.4 Write property test for latest key version selection

- [x] 6. Checkpoint - Verify KeyManager functionality

- [x] 7. Implement JWTSigner module
  - [x] 7.1 Create JWTSigner with crypto-service signing
  - [x] 7.2 Implement JWT verification
  - [x] 7.3 Add local Joken fallback
  - [x] 7.4 Write property test for JWT round-trip

- [x] 8. Implement EncryptedStore module
  - [x] 8.1 Create base EncryptedStore with encrypt/decrypt
  - [x] 8.2 Add namespace-specific key handling
  - [x] 8.3 Write property test for key namespace isolation
  - [x] 8.4 Implement AAD binding
  - [x] 8.5 Write property test for AAD binding integrity
  - [x] 8.6 Write property test for session encryption round-trip
  - [x] 8.7 Write property test for refresh token encryption round-trip

- [x] 9. Checkpoint - Verify EncryptedStore functionality

- [x] 10. Implement key rotation support
  - [x] 10.1 Add multi-version decryption
  - [x] 10.2 Write property test for multi-version decryption
  - [x] 10.3 Implement re-encryption on deprecated key access
  - [x] 10.4 Write property test for re-encryption behavior
  - [x] 10.5 Add key version logging

- [x] 11. Integrate with SessionManager
  - [x] 11.1 Update SessionStore to use EncryptedStore
    - Created `session_encryption.ex` wrapper module
  - [x] 11.2 Update OAuth module to use EncryptedStore for refresh tokens
    - Created `refresh_token_encryption.ex` wrapper module
  - [x] 11.3 Update IdToken generation to use JWTSigner
    - Created `id_token_signer.ex` wrapper module

- [x] 12. Checkpoint - Verify integration with existing modules

- [x] 13. Implement observability
  - [x] 13.1 Add Prometheus metrics for crypto operations
    - Created `metrics.ex` with counter, histogram, gauge
  - [x] 13.2 Write property test for metrics emission
  - [x] 13.3 Add crypto-service health to readiness check
    - Created `health_check.ex` module
  - [x] 13.4 Write property test for health check integration
  - [x] 13.5 Add latency warning logs
    - Configured via `latency_warning_ms` in config

- [x] 14. Implement feature toggle
  - [x] 14.1 Add crypto integration enable/disable config
    - Created `feature_toggle.ex` GenServer
  - [x] 14.2 Implement toggle behavior in all crypto modules
    - `when_enabled/2` function for conditional execution
  - [x] 14.3 Write property test for feature toggle behavior

- [x] 15. Final checkpoint - Full integration verification
  - All modules implemented
  - All property tests created
  - Integration wrappers ready for use

## Implementation Summary

### Created Modules (lib/session_identity_core/crypto/)
- `config.ex` - Configuration with validation
- `client.ex` - gRPC client with connection management
- `trace_context.ex` - W3C Trace Context propagation
- `correlation.ex` - Correlation ID handling
- `errors.ex` - Structured error types
- `circuit_breaker.ex` - Circuit breaker with fuse
- `fallback.ex` - Local fallback implementations
- `key_manager.ex` - Key metadata caching
- `jwt_signer.ex` - JWT signing/verification
- `encrypted_store.ex` - Encrypted storage wrapper
- `key_rotation.ex` - Key rotation support
- `session_encryption.ex` - Session data encryption integration
- `refresh_token_encryption.ex` - Refresh token encryption integration
- `id_token_signer.ex` - ID token signing integration
- `metrics.ex` - Prometheus metrics
- `health_check.ex` - Health check integration
- `feature_toggle.ex` - Runtime feature toggle

### Created Property Tests (test/property/crypto/)
- `trace_context_property_test.exs`
- `correlation_property_test.exs`
- `errors_property_test.exs`
- `circuit_breaker_property_test.exs`
- `key_manager_property_test.exs`
- `jwt_signer_property_test.exs`
- `encrypted_store_property_test.exs`
- `key_rotation_property_test.exs`
- `key_rotation_reencrypt_property_test.exs`
- `metrics_property_test.exs`
- `health_check_property_test.exs`
- `feature_toggle_property_test.exs`

## Notes

- All 16 property tests implemented with minimum 100 iterations
- Integration modules provide backward-compatible wrappers
- Feature toggle allows runtime enable/disable without restart
- Health check integrates with Kubernetes readiness probes
- Metrics follow Prometheus conventions
