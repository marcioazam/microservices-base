# Implementation Plan: Crypto Security Service

## Overview

This implementation plan breaks down the Crypto Security Service into incremental coding tasks. Each task builds on previous work, ensuring no orphaned code. The service is implemented in C++ using OpenSSL for cryptographic operations, gRPC/REST for APIs, and follows security-first principles.

## Tasks

- [x] 1. Project Setup and Core Infrastructure
  - [x] 1.1 Create project structure with CMake build system
    - Create directory structure: `services/crypto-service/{src,include,tests,proto}`
    - Set up CMakeLists.txt with C++20, OpenSSL, gRPC, Protobuf dependencies
    - Configure sanitizers (ASan, MSan) for debug builds
    - _Requirements: 14.2, 14.4_

  - [x] 1.2 Implement Result<T> type and error handling infrastructure
    - Create `include/crypto/common/result.h` with Result<T, E> monad
    - Define error codes enum matching design error categories
    - Implement ErrorResponse struct for API responses
    - _Requirements: Error Handling section_

  - [x] 1.3 Implement secure memory utilities
    - Create `src/crypto/common/secure_memory.cpp`
    - Implement SecureVector with mlock and secure zeroing
    - Implement constant-time comparison function
    - _Requirements: 5.6, 5.7_

- [x] 2. AES Encryption Engine
  - [x] 2.1 Implement AESEngine class with GCM mode
    - Create `src/crypto/engine/aes_engine.cpp`
    - Implement encryptGCM using OpenSSL EVP_aes_256_gcm
    - Implement decryptGCM with tag verification
    - Generate cryptographically secure IVs using RAND_bytes
    - Support AAD (Additional Authenticated Data)
    - _Requirements: 1.1, 1.2, 1.3, 1.5, 1.6_

  - [x] 2.2 Write property test for AES-GCM round-trip
    - **Property 1: AES Encryption Round-Trip**
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.8**

  - [x] 2.3 Write property test for AES IV uniqueness
    - **Property 2: AES IV Uniqueness**
    - **Validates: Requirements 1.5**

  - [x] 2.4 Write property test for AES AAD binding
    - **Property 3: AES AAD Binding**
    - **Validates: Requirements 1.6**

  - [x] 2.5 Write property test for AES tamper detection
    - **Property 4: AES Tamper Detection**
    - **Validates: Requirements 1.7**

  - [x] 2.6 Implement AES-CBC mode for legacy compatibility
    - Add encryptCBC and decryptCBC methods
    - Implement PKCS7 padding
    - _Requirements: 1.4_

- [x] 3. RSA Encryption Engine
  - [x] 3.1 Implement RSAEngine class
    - Create `src/crypto/engine/rsa_engine.cpp`
    - Implement encryptOAEP using RSA_OAEP_SHA256
    - Implement decryptOAEP with proper error handling
    - Implement generateKeyPair for 2048, 3072, 4096 bits
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 3.2 Write property test for RSA round-trip
    - **Property 5: RSA Encryption Round-Trip**
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.7**

  - [x] 3.3 Write property test for RSA size limit enforcement
    - **Property 6: RSA Size Limit Enforcement**
    - **Validates: Requirements 2.5**

  - [x] 3.4 Implement RSA-PSS signatures
    - Add signPSS and verifyPSS methods
    - Support SHA-256, SHA-384, SHA-512 hash algorithms
    - _Requirements: 3.1, 3.2, 3.3_

  - [x] 3.5 Implement hybrid encryption for large payloads
    - Create HybridEncryption class combining RSA key wrapping + AES data encryption
    - _Requirements: 2.6_

  - [x] 3.6 Write property test for hybrid encryption round-trip
    - **Property 7: Hybrid Encryption Round-Trip**
    - **Validates: Requirements 2.6**

- [x] 4. ECDSA Signature Engine
  - [x] 4.1 Implement ECDSAEngine class
    - Create `src/crypto/engine/ecdsa_engine.cpp`
    - Implement sign and verify methods
    - Support P-256, P-384, P-521 curves
    - Implement generateKeyPair for each curve
    - _Requirements: 3.4_

  - [x] 4.2 Write property test for signature consistency
    - **Property 8: Signature Consistency**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.7**

  - [x] 4.3 Write property test for invalid signature rejection
    - **Property 9: Invalid Signature Rejection**
    - **Validates: Requirements 3.6**

- [x] 5. Checkpoint - Crypto Engines Complete
  - Ensure all crypto engine tests pass
  - Run with sanitizers to verify memory safety
  - Ask the user if questions arise

- [x] 6. Key Management System
  - [x] 6.1 Implement KeyId and KeyMetadata data structures
    - Create `src/crypto/keys/key_types.cpp`
    - Implement KeyId with namespace, UUID, version
    - Implement KeyMetadata with all required fields
    - _Requirements: 4.4, 4.5_

  - [x] 6.2 Implement IKeyStore interface and LocalKeyStore
    - Create `src/crypto/keys/key_store.cpp`
    - Implement encrypted key storage using master KEK
    - Store keys with AES-GCM encryption
    - _Requirements: 5.1, 5.4, 5.5_

  - [x] 6.3 Implement KeyService with key generation
    - Create `src/crypto/keys/key_service.cpp`
    - Implement generateKey for AES, RSA, ECDSA
    - Generate unique Key_IDs using UUID v4
    - Store encrypted keys via IKeyStore
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.7_

  - [x] 6.4 Write property test for generated keys functionality
    - **Property 10: Generated Keys Are Functional**
    - **Validates: Requirements 4.1, 4.2, 4.3**

  - [x] 6.5 Write property test for Key ID uniqueness
    - **Property 11: Key ID Uniqueness**
    - **Validates: Requirements 4.4**

  - [x] 6.6 Write property test for key metadata completeness
    - **Property 12: Key Metadata Completeness**
    - **Validates: Requirements 4.5**

  - [x] 6.7 Write property test for private key protection
    - **Property 13: Private Key Protection**
    - **Validates: Requirements 4.7**

  - [x] 6.8 Implement key rotation
    - Add rotateKey method to KeyService
    - Mark old key as DEPRECATED, create new key
    - Maintain key version chain
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [x] 6.9 Write property test for key rotation state machine
    - **Property 14: Key Rotation State Machine**
    - **Validates: Requirements 6.2, 6.4, 6.7**

  - [x] 6.10 Write property test for deprecated key decryption
    - **Property 15: Deprecated Key Decryption**
    - **Validates: Requirements 6.3**

  - [x] 6.11 Implement key cache with secure memory
    - Create KeyCache with configurable TTL
    - Use SecureVector for cached key material
    - _Requirements: 11.6_

- [x] 7. Checkpoint - Key Management Complete
  - Ensure all key management tests pass
  - Verify key rotation lifecycle works correctly
  - Ask the user if questions arise

- [x] 8. Audit Logging System
  - [x] 8.1 Implement AuditEntry and IAuditLogger
    - Create `src/crypto/audit/audit_logger.cpp`
    - Implement AuditEntry struct with all required fields
    - Implement encrypted log storage
    - _Requirements: 10.1, 10.2, 10.4_

  - [x] 8.2 Implement sensitive data filtering
    - Ensure plaintext, ciphertext, key material never logged
    - Implement field sanitization
    - _Requirements: 10.3_

  - [x] 8.3 Write property test for audit entry completeness
    - **Property 20: Audit Entry Completeness**
    - **Validates: Requirements 10.1, 10.2, 10.3**

  - [x] 8.4 Write property test for key rotation audit
    - **Property 16: Key Rotation Audit**
    - **Validates: Requirements 6.6**

- [x] 9. Core Services Layer
  - [x] 9.1 Implement EncryptionService
    - Create `src/crypto/services/encryption_service.cpp`
    - Wire AESEngine with KeyService
    - Add audit logging for all operations
    - _Requirements: 1.1, 1.2, 10.1_

  - [x] 9.2 Implement SignatureService
    - Create `src/crypto/services/signature_service.cpp`
    - Wire RSAEngine and ECDSAEngine with KeyService
    - Add audit logging for sign/verify operations
    - _Requirements: 3.1, 3.2, 10.1_

  - [x] 9.3 Implement FileEncryptionService with streaming
    - Create `src/crypto/services/file_encryption_service.cpp`
    - Implement chunk-based streaming encryption
    - Generate unique DEK per file
    - Create encrypted file header format
    - _Requirements: 7.1, 7.2, 7.4, 7.5, 7.6_

  - [x] 9.4 Write property test for file DEK uniqueness
    - **Property 17: File DEK Uniqueness**
    - **Validates: Requirements 7.4**

  - [x] 9.5 Write property test for file header completeness
    - **Property 18: File Header Completeness**
    - **Validates: Requirements 7.6**

  - [x] 9.6 Write property test for file encryption round-trip
    - **Property 19: File Encryption Round-Trip**
    - **Validates: Requirements 7.7**

- [x] 10. Checkpoint - Core Services Complete
  - Ensure all service layer tests pass
  - Verify end-to-end encryption flows work
  - Ask the user if questions arise

- [x] 11. gRPC API Layer
  - [x] 11.1 Define Protocol Buffer messages
    - Create `proto/crypto_service.proto`
    - Define all request/response messages
    - Define streaming messages for file operations
    - _Requirements: 8.1, 8.3_

  - [x] 11.2 Implement gRPC server with TLS
    - Create `src/crypto/api/grpc_server.cpp`
    - Configure TLS 1.3 with certificate validation
    - Implement all service methods
    - _Requirements: 8.1, 8.6_

  - [x] 11.3 Implement JWT authentication interceptor
    - Create `src/crypto/auth/jwt_validator.cpp`
    - Validate JWT tokens from Authorization header
    - Extract caller identity for audit logging
    - _Requirements: 9.1, 9.2_

  - [x] 11.4 Implement RBAC authorization
    - Create `src/crypto/auth/rbac_engine.cpp`
    - Enforce role-based access control for key operations
    - Support namespace isolation
    - _Requirements: 9.3, 9.5, 9.6_

  - [x] 11.5 Implement health check endpoint
    - Add HealthCheck RPC method
    - Check HSM/KMS connectivity status
    - _Requirements: 8.5, 13.5_

- [x] 12. REST API Layer
  - [x] 12.1 Implement REST server with TLS
    - Create `src/crypto/api/rest_server.cpp`
    - Use cpp-httplib or similar library
    - Configure TLS 1.3
    - _Requirements: 8.2, 8.6_

  - [x] 12.2 Implement REST endpoints
    - Map all endpoints from design document
    - Accept/return JSON
    - Reuse core services layer
    - _Requirements: 8.2, 8.4_

  - [x] 12.3 Add input validation middleware
    - Validate all request parameters
    - Return appropriate error responses
    - _Requirements: 8.7_

- [x] 13. Observability
  - [x] 13.1 Implement Prometheus metrics exporter
    - Create `src/crypto/metrics/prometheus_exporter.cpp`
    - Export operation counters, latency histograms
    - Expose /metrics endpoint
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5_

  - [x] 13.2 Implement structured JSON logging
    - Create `src/crypto/logging/json_logger.cpp`
    - Include correlation IDs in all logs
    - _Requirements: 12.6_

  - [x] 13.3 Implement OpenTelemetry tracing
    - Add trace context propagation
    - Create spans for all operations
    - _Requirements: 12.7_

- [x] 14. Resilience
  - [x] 14.1 Implement circuit breaker for KMS/HSM
    - Create `src/crypto/resilience/circuit_breaker.cpp`
    - Configure failure threshold and recovery timeout
    - _Requirements: 13.1, 13.2_

  - [x] 14.2 Implement retry with exponential backoff
    - Add retry logic for transient failures
    - _Requirements: 13.3_

  - [x] 14.3 Implement graceful shutdown
    - Handle SIGTERM signal
    - Drain in-flight requests
    - _Requirements: 13.6_

- [x] 15. Configuration and Deployment
  - [x] 15.1 Implement configuration loading
    - Create `src/crypto/config/config_loader.cpp`
    - Load from environment variables
    - Validate required configuration
    - _Requirements: 14.1, 14.4_

  - [x] 15.2 Create Dockerfile
    - Multi-stage build for minimal image
    - Include only runtime dependencies
    - _Requirements: 14.2_

  - [x] 15.3 Create Kubernetes manifests
    - Deployment, Service, ConfigMap, Secret
    - Health check probes
    - Resource limits
    - _Requirements: 14.3_

- [x] 16. Final Checkpoint - Integration Complete
  - Run full test suite including integration tests
  - Verify all property tests pass with 100+ iterations
  - Run security scans (static analysis)
  - Ask the user if questions arise

## Notes

- Tasks marked with `*` are optional property-based tests that can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests use RapidCheck library for C++
- Unit tests use Google Test framework
