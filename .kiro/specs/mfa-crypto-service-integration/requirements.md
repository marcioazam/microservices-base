# Requirements Document

## Introduction

This document specifies the requirements for integrating the MFA (Multi-Factor Authentication) Service with the centralized Crypto Security Service. The integration aims to delegate cryptographic operations (encryption, decryption, key management) from the MFA service to the crypto-service, enabling centralized key management, HSM/KMS integration, and consistent cryptographic practices across the platform.

## Glossary

- **MFA_Service**: The Multi-Factor Authentication microservice (Elixir/OTP) providing TOTP, WebAuthn/Passkeys capabilities
- **Crypto_Service**: The centralized C++ microservice providing cryptographic operations via gRPC
- **TOTP_Secret**: The shared secret used for Time-based One-Time Password generation (160 bits)
- **Encryption_Key**: AES-256 key used to encrypt TOTP secrets at rest
- **Key_Namespace**: Logical grouping of keys in crypto-service (e.g., "mfa:totp")
- **Key_Rotation**: Process of replacing an encryption key while maintaining access to data encrypted with previous versions
- **AAD**: Additional Authenticated Data used in AES-GCM for integrity verification
- **Crypto_Client**: gRPC client module in MFA service for communicating with crypto-service
- **KEK**: Key Encryption Key used to wrap data encryption keys

## Requirements

### Requirement 1: Crypto Service Client Integration

**User Story:** As a platform architect, I want the MFA service to communicate with the crypto-service via gRPC, so that cryptographic operations are centralized and consistent.

#### Acceptance Criteria

1. WHEN the MFA_Service starts, THE Application SHALL initialize a gRPC client connection to Crypto_Service
2. THE Crypto_Client SHALL use the proto definitions from crypto-service for type-safe communication
3. WHEN Crypto_Service is unavailable, THE Crypto_Client SHALL apply circuit breaker pattern with 5 consecutive failure threshold
4. WHEN Crypto_Service calls fail transiently, THE Crypto_Client SHALL retry up to 3 times with exponential backoff
5. THE Crypto_Client SHALL propagate correlation_id in all requests for distributed tracing
6. WHEN connection is established, THE Crypto_Client SHALL verify Crypto_Service health via HealthCheck RPC

### Requirement 2: TOTP Secret Encryption Migration

**User Story:** As a security engineer, I want TOTP secrets to be encrypted using the centralized crypto-service, so that key management is centralized and HSM-backed.

#### Acceptance Criteria

1. WHEN encrypting a TOTP secret, THE Generator SHALL call Crypto_Service.Encrypt with AES_256_GCM algorithm
2. WHEN decrypting a TOTP secret, THE Generator SHALL call Crypto_Service.Decrypt with the stored key_id
3. THE Generator SHALL use "mfa:totp" as the key namespace for TOTP encryption keys
4. WHEN encrypting, THE Generator SHALL include user_id as AAD (Additional Authenticated Data) for binding
5. THE Generator SHALL store the key_id, iv, and tag alongside the ciphertext for decryption
6. FOR ALL valid TOTP secrets, encrypting via Crypto_Service then decrypting SHALL return the original secret (round-trip property)

### Requirement 3: Key Management for MFA

**User Story:** As a security administrator, I want MFA encryption keys to be managed centrally with rotation support, so that key lifecycle is controlled and auditable.

#### Acceptance Criteria

1. WHEN the MFA_Service initializes, THE Key_Manager SHALL ensure a TOTP encryption key exists in namespace "mfa:totp"
2. IF no key exists, THEN THE Key_Manager SHALL call Crypto_Service.GenerateKey with AES_256_GCM algorithm
3. WHEN key rotation is triggered, THE Key_Manager SHALL call Crypto_Service.RotateKey and update active key reference
4. WHEN decrypting with a rotated key, THE Generator SHALL use the key_id stored with the ciphertext (not current active key)
5. THE Key_Manager SHALL cache key metadata locally with 5-minute TTL to reduce Crypto_Service calls
6. WHEN retrieving key metadata, THE Key_Manager SHALL call Crypto_Service.GetKeyMetadata

### Requirement 4: Fallback Behavior

**User Story:** As a platform operator, I want the MFA service to handle crypto-service unavailability gracefully, so that authentication remains functional during outages.

#### Acceptance Criteria

1. IF Crypto_Service is unavailable during encryption, THEN THE Generator SHALL return an error indicating encryption failure
2. IF Crypto_Service is unavailable during decryption, THEN THE Generator SHALL return an error indicating decryption failure
3. WHEN circuit breaker is open, THE Crypto_Client SHALL fail fast without attempting connection
4. THE MFA_Service SHALL emit telemetry events for all Crypto_Service failures with error codes
5. WHEN Crypto_Service recovers, THE circuit breaker SHALL close after successful health check

### Requirement 5: Migration Strategy

**User Story:** As a DevOps engineer, I want a safe migration path from local encryption to crypto-service, so that existing TOTP secrets remain accessible.

#### Acceptance Criteria

1. THE Generator SHALL support reading secrets encrypted with both local and crypto-service methods
2. WHEN reading a locally-encrypted secret, THE Generator SHALL detect format and use local decryption
3. WHEN reading a crypto-service-encrypted secret, THE Generator SHALL detect format and use Crypto_Service.Decrypt
4. WHEN a locally-encrypted secret is accessed, THE Generator SHALL re-encrypt using Crypto_Service on next write (lazy migration)
5. THE migration format SHALL include a version byte to distinguish encryption methods
6. FOR ALL existing TOTP secrets, migration to crypto-service encryption SHALL preserve the original secret value

### Requirement 6: Security Requirements

**User Story:** As a security officer, I want the crypto-service integration to follow security best practices, so that cryptographic operations are protected.

#### Acceptance Criteria

1. THE Crypto_Client SHALL use mTLS for all gRPC communication with Crypto_Service
2. THE Crypto_Client SHALL validate Crypto_Service TLS certificate against trusted CA
3. THE MFA_Service SHALL never log plaintext TOTP secrets or encryption keys
4. THE MFA_Service SHALL use secure memory handling for key material (clear after use)
5. WHEN errors occur, THE Crypto_Client SHALL not expose internal crypto-service details in error messages
6. THE Crypto_Client SHALL validate all responses from Crypto_Service before processing

### Requirement 7: Observability

**User Story:** As a platform operator, I want visibility into crypto-service integration, so that I can monitor performance and troubleshoot issues.

#### Acceptance Criteria

1. THE Crypto_Client SHALL emit telemetry events for all RPC calls with latency measurements
2. THE Crypto_Client SHALL emit metrics for success/failure rates per operation type
3. THE Crypto_Client SHALL include correlation_id in all log entries for distributed tracing
4. WHEN circuit breaker state changes, THE Crypto_Client SHALL emit telemetry event with new state
5. THE MFA_Service SHALL expose Prometheus metrics for crypto-service integration health

### Requirement 8: Configuration

**User Story:** As a DevOps engineer, I want the crypto-service integration to be configurable, so that I can adjust settings per environment.

#### Acceptance Criteria

1. THE Crypto_Client SHALL read Crypto_Service address from CRYPTO_SERVICE_HOST and CRYPTO_SERVICE_PORT environment variables
2. THE Crypto_Client SHALL support configurable connection timeout (default: 5 seconds)
3. THE Crypto_Client SHALL support configurable request timeout (default: 30 seconds)
4. THE Key_Manager SHALL support configurable key namespace prefix (default: "mfa")
5. THE circuit breaker SHALL support configurable failure threshold (default: 5)
6. THE retry policy SHALL support configurable max attempts (default: 3)

