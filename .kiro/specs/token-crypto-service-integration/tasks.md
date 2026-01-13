# Implementation Plan: Token-Crypto Service Integration

## Overview

This implementation plan describes the tasks required to integrate the Token Service with the centralized Crypto Service. The implementation follows an incremental approach, starting with the core client infrastructure, then adding signing, encryption, and finally observability features.

## Tasks

- [x] 1. Set up project structure and dependencies
  - [x] 1.1 Add Crypto Service proto dependencies to Cargo.toml
  - [x] 1.2 Configure build.rs for proto compilation
  - [x] 1.3 Create crypto module structure

- [x] 2. Implement CryptoClient core infrastructure
  - [x] 2.1 Implement CryptoClientConfig and validation
  - [x] 2.2 Write property test for configuration validation (Property 12)
  - [x] 2.3 Implement CryptoClientCore with gRPC connection
  - [x] 2.4 Write property test for request context propagation (Property 3)
  - [x] 2.5 Implement CircuitBreaker integration
  - [x] 2.6 Write property test for circuit breaker state transitions (Property 1)
  - [x] 2.7 Implement RateLimiter
  - [x] 2.8 Write property test for rate limiting enforcement (Property 14)

- [x] 3. Checkpoint - Core infrastructure tests pass

- [x] 4. Implement FallbackHandler
  - [x] 4.1 Create FallbackHandler struct
  - [x] 4.2 Implement with_fallback method
  - [x] 4.3 Write property test for fallback activation (Property 2)

- [x] 5. Implement signing operations
  - [x] 5.1 Implement sign method in CryptoClient
  - [x] 5.2 Implement verify method in CryptoClient
  - [x] 5.3 Implement key state validation before signing
  - [x] 5.4 Write property test for key state validation (Property 8)
  - [x] 5.5 Implement response algorithm validation
  - [x] 5.6 Write property test for response algorithm validation (Property 13)
  - [x] 5.7 Write property test for algorithm support completeness (Property 5)

- [x] 6. Checkpoint - Signing tests pass

- [x] 7. Implement CryptoSigner (JWT integration)
  - [x] 7.1 Create CryptoSigner struct implementing KmsSigner
  - [x] 7.2 Implement sign method for JWT signing
  - [x] 7.3 Implement key metadata caching
  - [x] 7.4 Write property test for key metadata caching (Property 6)
  - [x] 7.5 Write property test for JWT signing round trip (Property 4)

- [x] 8. Implement encryption operations
  - [x] 8.1 Implement encrypt method in CryptoClient
  - [x] 8.2 Implement decrypt method in CryptoClient
  - [x] 8.3 Write property test for encryption round trip (Property 9)

- [x] 9. Implement CryptoEncryptor (Cache integration)
  - [x] 9.1 Create CryptoEncryptor struct
  - [x] 9.2 Implement encrypt_token_family method
  - [x] 9.3 Implement decrypt_token_family method
  - [x] 9.4 Integrate CryptoEncryptor with CacheStorage (EncryptedCacheStorage)

- [x] 10. Checkpoint - Encryption tests pass

- [x] 11. Implement key management operations
  - [x] 11.1 Implement generate_key method
  - [x] 11.2 Implement rotate_key method
  - [x] 11.3 Implement get_key_metadata method
  - [x] 11.4 Implement key rotation detection for JWKS
  - [x] 11.5 Write property test for key rotation graceful transition (Property 7)

- [x] 12. Implement feature flags
  - [x] 12.1 Add feature flag configuration
  - [x] 12.2 Implement feature flag routing
  - [x] 12.3 Write property test for feature flag behavior (Property 11)

- [x] 13. Checkpoint - Feature flag tests pass

- [x] 14. Implement observability
  - [x] 14.1 Add Prometheus metrics for CryptoClient
  - [x] 14.2 Implement error logging with required fields
  - [x] 14.3 Write property test for error logging completeness (Property 10)

- [x] 15. Integration and wiring
  - [x] 15.1 Create CryptoClientFactory
  - [x] 15.2 Wire CryptoClient into Token Service startup
  - [x] 15.3 Replace KmsFactory with CryptoSigner when enabled
  - [x] 15.4 Update CacheStorage to use CryptoEncryptor
  - [x] 15.5 Write integration tests for end-to-end flows

- [x] 16. Final checkpoint - All tests pass

- [x] 17. Documentation and cleanup
  - [x] 17.1 Update Token Service README
  - [x] 17.2 Add inline documentation

## Implementation Summary

All 17 tasks completed. Key deliverables:

### Files Created/Modified

**Core Crypto Module** (`services/token/src/crypto/`):
- `mod.rs` - Module declarations and re-exports
- `client.rs` - CryptoClient trait and CryptoClientCore implementation
- `config.rs` - CryptoClientConfig with validation
- `error.rs` - CryptoError enum with transient detection
- `models.rs` - KeyId, KeyState, KeyAlgorithm, KeyMetadata, SignResult, EncryptResult
- `fallback.rs` - FallbackHandler with local HMAC and AES-256-GCM
- `metrics.rs` - Prometheus metrics for crypto operations
- `signer.rs` - CryptoSigner implementing KmsSigner trait
- `encryptor.rs` - CryptoEncryptor for cache encryption
- `factory.rs` - CryptoClientFactory for client creation

**Storage Integration** (`services/token/src/storage/`):
- `encrypted_cache.rs` - EncryptedCacheStorage with optional Crypto Service encryption

**KMS Integration** (`services/token/src/kms/`):
- `mod.rs` - Updated KmsFactory with create_crypto_signer method

**Proto** (`services/token/proto/`):
- `crypto_service.proto` - Crypto Service gRPC definitions

**Tests** (`services/token/tests/`):
- `crypto_property_tests.rs` - Core property tests (Properties 1, 2, 8, 9, 11, 12, 14)
- `crypto_advanced_property_tests.rs` - Advanced property tests (Properties 3-7, 10, 13)
- `crypto_integration_tests.rs` - Integration tests for end-to-end flows

**Documentation**:
- `services/token/README.md` - Updated with Crypto Service integration docs

### Property Tests Coverage

| Property | Description | File |
|----------|-------------|------|
| 1 | Circuit Breaker State Transitions | crypto_property_tests.rs |
| 2 | Fallback Activation | crypto_property_tests.rs |
| 3 | Request Context Propagation | crypto_advanced_property_tests.rs |
| 4 | JWT Signing Round Trip | crypto_advanced_property_tests.rs |
| 5 | Algorithm Support Completeness | crypto_advanced_property_tests.rs |
| 6 | Key Metadata Caching | crypto_advanced_property_tests.rs |
| 7 | Key Rotation Graceful Transition | crypto_advanced_property_tests.rs |
| 8 | Key State Validation | crypto_property_tests.rs |
| 9 | Encryption Round Trip | crypto_property_tests.rs |
| 10 | Error Logging Completeness | crypto_advanced_property_tests.rs |
| 11 | Feature Flag Behavior | crypto_property_tests.rs |
| 12 | Configuration Validation | crypto_property_tests.rs |
| 13 | Response Algorithm Validation | crypto_advanced_property_tests.rs |
| 14 | Rate Limiting Enforcement | crypto_property_tests.rs |
