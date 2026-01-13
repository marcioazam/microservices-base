# Requirements Document

## Introduction

Este documento define os requisitos para integrar o serviço `auth-edge` com o `crypto-service` centralizado da plataforma. A integração visa eliminar a criptografia local no `auth-edge`, delegando todas as operações criptográficas para o `crypto-service` via gRPC, garantindo gerenciamento centralizado de chaves, rotação automática e conformidade com padrões de segurança.

## Glossary

- **Auth_Edge**: Serviço de borda responsável por validação JWT, mTLS/SPIFFE e roteamento de autenticação
- **Crypto_Service**: Microserviço C++ centralizado que fornece operações criptográficas via gRPC
- **Cache_Client**: Cliente gRPC do `rust-common` para cache distribuído com criptografia local
- **Crypto_Client**: Novo cliente gRPC Rust para comunicação com o Crypto_Service
- **KEK**: Key Encryption Key - chave mestra para criptografar outras chaves
- **DEK**: Data Encryption Key - chave usada para criptografar dados
- **JWK**: JSON Web Key - representação de chave criptográfica para JWT
- **Circuit_Breaker**: Padrão de resiliência que previne chamadas a serviços indisponíveis
- **Correlation_ID**: Identificador único para rastreamento de requisições entre serviços

## Requirements

### Requirement 1: Crypto Client para Auth-Edge

**User Story:** As a developer, I want a gRPC client for crypto-service in Rust, so that auth-edge can delegate cryptographic operations to the centralized service.

#### Acceptance Criteria

1. THE Crypto_Client SHALL connect to Crypto_Service via gRPC with mTLS
2. THE Crypto_Client SHALL support all symmetric encryption operations (Encrypt, Decrypt)
3. THE Crypto_Client SHALL support key management operations (GenerateKey, GetKeyMetadata, RotateKey)
4. THE Crypto_Client SHALL propagate correlation_id in all requests for distributed tracing
5. WHEN Crypto_Service is unavailable, THE Crypto_Client SHALL return a PlatformError::Unavailable
6. THE Crypto_Client SHALL integrate with rust-common CircuitBreaker for resilience
7. THE Crypto_Client SHALL expose Prometheus metrics for operation latency and error rates

### Requirement 2: Cache Encryption via Crypto-Service

**User Story:** As a security engineer, I want cache data encrypted by the centralized crypto-service, so that key management is centralized and keys can be rotated without service restarts.

#### Acceptance Criteria

1. WHEN storing data in cache, THE Cache_Client SHALL call Crypto_Service.Encrypt before storage
2. WHEN retrieving data from cache, THE Cache_Client SHALL call Crypto_Service.Decrypt after retrieval
3. THE Cache_Client SHALL use a dedicated namespace key (auth-edge:cache) managed by Crypto_Service
4. WHEN Crypto_Service is unavailable, THE Cache_Client SHALL fall back to local encryption with cached DEK
5. THE Cache_Client SHALL support key rotation without data loss or service interruption
6. THE Cache_Client SHALL include AAD (Additional Authenticated Data) with namespace and key name

### Requirement 3: Key Management Integration

**User Story:** As a platform operator, I want auth-edge to use centrally managed encryption keys, so that I can rotate keys, audit usage, and enforce security policies.

#### Acceptance Criteria

1. WHEN Auth_Edge starts, THE system SHALL request or create a dedicated KEK from Crypto_Service
2. THE system SHALL store the KEK reference (KeyId) not the actual key material
3. WHEN a key rotation is triggered, THE system SHALL seamlessly transition to the new key version
4. THE system SHALL support decryption with previous key versions during rotation window
5. THE system SHALL log all key operations with correlation_id for audit trail
6. IF key creation fails, THEN THE system SHALL fail startup with clear error message

### Requirement 4: Graceful Degradation

**User Story:** As a site reliability engineer, I want auth-edge to continue operating when crypto-service is temporarily unavailable, so that authentication is not a single point of failure.

#### Acceptance Criteria

1. WHEN Crypto_Service becomes unavailable, THE Auth_Edge SHALL use locally cached DEK for encryption
2. WHEN Crypto_Service becomes unavailable, THE Auth_Edge SHALL queue key rotation requests for retry
3. THE Circuit_Breaker SHALL open after 5 consecutive failures to Crypto_Service
4. WHEN Circuit_Breaker is open, THE system SHALL use fallback mode without calling Crypto_Service
5. WHEN Crypto_Service recovers, THE system SHALL sync any pending operations
6. THE system SHALL emit metrics and alerts when operating in degraded mode

### Requirement 5: Configuration and Security

**User Story:** As a DevOps engineer, I want to configure the crypto-service integration via environment variables, so that I can deploy auth-edge in different environments.

#### Acceptance Criteria

1. THE system SHALL read CRYPTO_SERVICE_URL from environment for Crypto_Service endpoint
2. THE system SHALL read CRYPTO_KEY_NAMESPACE from environment for key isolation (default: auth-edge)
3. THE system SHALL read CRYPTO_FALLBACK_ENABLED from environment to enable/disable local fallback
4. THE system SHALL validate all configuration at startup and fail fast on invalid values
5. THE system SHALL NOT log or expose key material in any error messages or traces
6. THE system SHALL use TLS 1.3 for all communication with Crypto_Service

### Requirement 6: Observability

**User Story:** As a platform engineer, I want comprehensive observability for crypto operations, so that I can monitor performance, detect issues, and troubleshoot problems.

#### Acceptance Criteria

1. THE system SHALL emit crypto_client_requests_total metric with labels (operation, status)
2. THE system SHALL emit crypto_client_latency_seconds histogram with labels (operation)
3. THE system SHALL emit crypto_client_fallback_active gauge indicating degraded mode
4. THE system SHALL propagate W3C Trace Context headers to Crypto_Service
5. WHEN errors occur, THE system SHALL log structured JSON with correlation_id and error_code
6. THE system SHALL emit crypto_key_rotation_total metric when keys are rotated

### Requirement 7: Testing and Validation

**User Story:** As a developer, I want comprehensive tests for the crypto integration, so that I can ensure correctness and catch regressions.

#### Acceptance Criteria

1. THE system SHALL include unit tests for Crypto_Client with mocked gRPC responses
2. THE system SHALL include property-based tests for encryption round-trip correctness
3. THE system SHALL include integration tests with a real Crypto_Service instance
4. THE system SHALL include tests for fallback behavior when Crypto_Service is unavailable
5. THE system SHALL include tests for key rotation scenarios
6. FOR ALL valid data, encrypting then decrypting SHALL produce the original data (round-trip property)
