# Requirements Document

## Introduction

This document specifies the requirements for integrating the Token Service with the centralized Crypto Service. Currently, the Token Service implements cryptographic operations locally (via `rust-common::CacheClient` for AES-256-GCM encryption and AWS KMS for JWT signing). This integration will centralize cryptographic operations through the Crypto Service, providing unified key management, HSM support, and consistent security policies across the platform.

## Glossary

- **Token_Service**: The Rust-based microservice responsible for JWT generation, DPoP validation, refresh token rotation, and JWKS publishing
- **Crypto_Service**: The C++ microservice providing centralized cryptographic operations including encryption, signing, and key management
- **JWT**: JSON Web Token - a compact, URL-safe means of representing claims between parties
- **DPoP**: Demonstrating Proof of Possession - RFC 9449 sender-constrained tokens
- **KMS**: Key Management Service - secure key storage and cryptographic operations
- **HSM**: Hardware Security Module - tamper-resistant hardware for key protection
- **JWKS**: JSON Web Key Set - public keys for token verification
- **CryptoClient**: A Rust gRPC client for communicating with the Crypto_Service
- **Circuit_Breaker**: Resilience pattern that prevents cascading failures

## Requirements

### Requirement 1: CryptoClient Integration

**User Story:** As a platform architect, I want the Token Service to use the centralized Crypto Service for cryptographic operations, so that we have unified key management and consistent security policies.

#### Acceptance Criteria

1. THE Token_Service SHALL include a CryptoClient module for gRPC communication with the Crypto_Service
2. WHEN the CryptoClient is initialized, THE Token_Service SHALL establish a connection to the Crypto_Service using the configured address
3. THE CryptoClient SHALL implement Circuit_Breaker pattern with configurable failure threshold and recovery timeout
4. WHEN the Crypto_Service is unavailable, THE CryptoClient SHALL fallback to local cryptographic operations using existing KMS module
5. THE CryptoClient SHALL propagate W3C Trace Context headers for distributed tracing

### Requirement 2: JWT Signing via Crypto Service

**User Story:** As a security engineer, I want JWT signing to be performed by the centralized Crypto Service, so that signing keys are managed in a single secure location with HSM support.

#### Acceptance Criteria

1. WHEN generating a JWT, THE Token_Service SHALL call the Crypto_Service Sign RPC with the token payload
2. THE Token_Service SHALL support RSA-PSS (PS256, PS384, PS512) and ECDSA (ES256, ES384) algorithms via Crypto_Service
3. WHEN the Crypto_Service returns a signature, THE Token_Service SHALL construct the complete JWT with the signature
4. IF the Crypto_Service Sign RPC fails, THEN THE Token_Service SHALL attempt fallback signing using the local KMS module
5. THE Token_Service SHALL cache the signing key metadata from Crypto_Service to reduce RPC calls
6. WHEN the signing key is rotated in Crypto_Service, THE Token_Service SHALL detect the new key version and update JWKS

### Requirement 3: Key Management Integration

**User Story:** As a DevOps engineer, I want the Token Service to use Crypto Service for key lifecycle management, so that key rotation and revocation are handled centrally.

#### Acceptance Criteria

1. THE Token_Service SHALL request signing keys from Crypto_Service using the GenerateKey RPC during initialization
2. WHEN a key rotation is triggered, THE Token_Service SHALL call the Crypto_Service RotateKey RPC
3. THE Token_Service SHALL maintain the previous key version for a configurable retention period (default 24 hours) for graceful key transitions
4. THE Token_Service SHALL call GetKeyMetadata RPC to verify key state before signing operations
5. IF a key is in DEPRECATED or DESTROYED state, THEN THE Token_Service SHALL refuse to use it for new signatures

### Requirement 4: Cache Encryption via Crypto Service

**User Story:** As a security engineer, I want sensitive cache data to be encrypted using the centralized Crypto Service, so that encryption keys are managed securely with HSM backing.

#### Acceptance Criteria

1. THE Token_Service SHALL use Crypto_Service Encrypt RPC for encrypting token family data before cache storage
2. THE Token_Service SHALL use Crypto_Service Decrypt RPC for decrypting token family data retrieved from cache
3. THE Token_Service SHALL use a dedicated encryption key namespace "token-cache" in Crypto_Service
4. WHEN the Crypto_Service is unavailable, THE Token_Service SHALL fallback to local AES-256-GCM encryption using CacheClient
5. THE Token_Service SHALL include correlation_id in all Crypto_Service requests for audit trail

### Requirement 5: Signature Verification Delegation

**User Story:** As a developer, I want DPoP proof verification to use the centralized Crypto Service, so that signature verification is consistent across all services.

#### Acceptance Criteria

1. WHEN validating a DPoP proof signature, THE Token_Service SHALL call the Crypto_Service Verify RPC
2. THE Token_Service SHALL support verification of RSA and ECDSA signatures via Crypto_Service
3. IF the Crypto_Service Verify RPC fails due to network issues, THEN THE Token_Service SHALL attempt local verification using jsonwebtoken crate
4. THE Token_Service SHALL cache public keys from DPoP proofs to reduce verification overhead

### Requirement 6: Observability and Metrics

**User Story:** As an SRE, I want visibility into Crypto Service integration performance, so that I can monitor and troubleshoot cryptographic operations.

#### Acceptance Criteria

1. THE Token_Service SHALL expose Prometheus metrics for Crypto_Service RPC latency (p50, p95, p99)
2. THE Token_Service SHALL expose metrics for Crypto_Service RPC success/failure rates by operation type
3. THE Token_Service SHALL expose metrics for fallback activation count
4. WHEN a Crypto_Service RPC fails, THE Token_Service SHALL log the error with correlation_id and operation details
5. THE Token_Service SHALL expose Circuit_Breaker state metrics (closed, open, half-open)

### Requirement 7: Configuration and Feature Flags

**User Story:** As a DevOps engineer, I want to control Crypto Service integration via configuration, so that I can enable/disable features and tune performance.

#### Acceptance Criteria

1. THE Token_Service SHALL support configuration for Crypto_Service address via CRYPTO_SERVICE_ADDRESS environment variable
2. THE Token_Service SHALL support CRYPTO_SIGNING_ENABLED flag to enable/disable Crypto_Service signing (default: true)
3. THE Token_Service SHALL support CRYPTO_ENCRYPTION_ENABLED flag to enable/disable Crypto_Service encryption (default: true)
4. THE Token_Service SHALL support CRYPTO_FALLBACK_ENABLED flag to enable/disable local fallback (default: true)
5. THE Token_Service SHALL support CRYPTO_KEY_NAMESPACE configuration for key isolation (default: "token")
6. THE Token_Service SHALL validate configuration at startup and fail fast if required settings are missing

### Requirement 8: Security Requirements

**User Story:** As a security auditor, I want the Crypto Service integration to follow security best practices, so that cryptographic operations are protected.

#### Acceptance Criteria

1. THE Token_Service SHALL use mTLS for all communication with Crypto_Service via Linkerd service mesh
2. THE Token_Service SHALL validate Crypto_Service responses for expected key algorithms and versions
3. THE Token_Service SHALL implement constant-time comparison for all signature verification results
4. THE Token_Service SHALL zeroize sensitive key material in memory after use
5. IF an unexpected key algorithm is returned, THEN THE Token_Service SHALL reject the operation and log a security event
6. THE Token_Service SHALL rate-limit Crypto_Service requests to prevent abuse (configurable, default: 1000 req/s)
