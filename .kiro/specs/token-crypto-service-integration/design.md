# Design Document: Token-Crypto Service Integration

## Overview

This design document describes the integration of the Token Service (Rust) with the centralized Crypto Service (C++) for cryptographic operations. The integration centralizes key management, JWT signing, and cache encryption through a dedicated gRPC client with circuit breaker resilience and local fallback capabilities.

The primary goals are:
1. Centralize cryptographic key management with HSM support
2. Provide consistent security policies across the platform
3. Maintain high availability through fallback mechanisms
4. Enable observability for cryptographic operations

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Token Service (Rust)                          │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      CryptoClient Module                         │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │    │
│  │  │ SignClient  │  │EncryptClient│  │ KeyManagementClient     │  │    │
│  │  │ - sign()    │  │ - encrypt() │  │ - generate_key()        │  │    │
│  │  │ - verify()  │  │ - decrypt() │  │ - rotate_key()          │  │    │
│  │  └──────┬──────┘  └──────┬──────┘  │ - get_key_metadata()    │  │    │
│  │         │                │         └───────────┬─────────────┘  │    │
│  │         └────────────────┼─────────────────────┘                │    │
│  │                          │                                       │    │
│  │  ┌───────────────────────▼───────────────────────────────────┐  │    │
│  │  │                  CryptoClientCore                          │  │    │
│  │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │  │    │
│  │  │  │CircuitBreaker│  │ RateLimiter  │  │ MetadataCache    │ │  │    │
│  │  │  └──────────────┘  └──────────────┘  └──────────────────┘ │  │    │
│  │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │  │    │
│  │  │  │TraceContext  │  │ Metrics      │  │ FallbackHandler  │ │  │    │
│  │  │  └──────────────┘  └──────────────┘  └──────────────────┘ │  │    │
│  │  └───────────────────────┬───────────────────────────────────┘  │    │
│  └──────────────────────────┼──────────────────────────────────────┘    │
│                             │                                           │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │                    Fallback Layer                                │   │
│  │  ┌─────────────────┐  ┌─────────────────┐                       │   │
│  │  │ LocalKmsSigner  │  │ LocalAesEncrypt │                       │   │
│  │  │ (existing KMS)  │  │ (CacheClient)   │                       │   │
│  │  └─────────────────┘  └─────────────────┘                       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                              │ gRPC (mTLS via Linkerd)
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Crypto Service (C++)                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │ Sign/Verify │  │ Encrypt/    │  │ Key         │  │ HSM/KMS     │    │
│  │ Service     │  │ Decrypt     │  │ Management  │  │ Backend     │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. CryptoClient Module

The main entry point for all cryptographic operations via Crypto Service.

```rust
/// Configuration for CryptoClient
#[derive(Debug, Clone)]
pub struct CryptoClientConfig {
    /// Crypto Service gRPC address
    pub address: String,
    /// Key namespace for isolation
    pub namespace: String,
    /// Enable signing via Crypto Service
    pub signing_enabled: bool,
    /// Enable encryption via Crypto Service
    pub encryption_enabled: bool,
    /// Enable fallback to local operations
    pub fallback_enabled: bool,
    /// Circuit breaker configuration
    pub circuit_breaker: CircuitBreakerConfig,
    /// Rate limit (requests per second)
    pub rate_limit: u32,
    /// Connection timeout
    pub connect_timeout: Duration,
    /// Request timeout
    pub request_timeout: Duration,
}

/// CryptoClient trait for cryptographic operations
#[async_trait]
pub trait CryptoClient: Send + Sync {
    /// Sign data using Crypto Service
    async fn sign(&self, data: &[u8], key_id: &KeyId) -> Result<SignResult, CryptoError>;
    
    /// Verify signature using Crypto Service
    async fn verify(&self, data: &[u8], signature: &[u8], key_id: &KeyId) -> Result<bool, CryptoError>;
    
    /// Encrypt data using Crypto Service
    async fn encrypt(&self, plaintext: &[u8], key_id: &KeyId, aad: Option<&[u8]>) -> Result<EncryptResult, CryptoError>;
    
    /// Decrypt data using Crypto Service
    async fn decrypt(&self, ciphertext: &EncryptedData, key_id: &KeyId, aad: Option<&[u8]>) -> Result<Vec<u8>, CryptoError>;
    
    /// Generate a new key
    async fn generate_key(&self, algorithm: KeyAlgorithm, namespace: &str) -> Result<KeyId, CryptoError>;
    
    /// Rotate an existing key
    async fn rotate_key(&self, key_id: &KeyId) -> Result<KeyRotationResult, CryptoError>;
    
    /// Get key metadata
    async fn get_key_metadata(&self, key_id: &KeyId) -> Result<KeyMetadata, CryptoError>;
}
```

### 2. CryptoClientCore

Internal implementation handling gRPC communication, circuit breaker, and fallback.

```rust
pub struct CryptoClientCore {
    /// gRPC client for Crypto Service
    grpc_client: CryptoServiceClient<Channel>,
    /// Circuit breaker for resilience
    circuit_breaker: Arc<CircuitBreaker>,
    /// Rate limiter
    rate_limiter: Arc<RateLimiter>,
    /// Key metadata cache
    metadata_cache: Arc<RwLock<LruCache<String, CachedKeyMetadata>>>,
    /// Fallback handler
    fallback: Arc<FallbackHandler>,
    /// Configuration
    config: CryptoClientConfig,
    /// Metrics collector
    metrics: Arc<CryptoMetrics>,
}
```

### 3. FallbackHandler

Manages fallback to local cryptographic operations when Crypto Service is unavailable.

```rust
pub struct FallbackHandler {
    /// Local KMS signer (existing implementation)
    local_signer: Arc<dyn KmsSigner>,
    /// Local AES encryptor (from CacheClient)
    local_encryptor: Arc<LocalAesEncryptor>,
    /// Fallback enabled flag
    enabled: bool,
    /// Fallback activation counter
    activation_count: AtomicU64,
}

impl FallbackHandler {
    /// Execute operation with fallback
    pub async fn with_fallback<T, F, G>(&self, primary: F, fallback: G) -> Result<T, CryptoError>
    where
        F: Future<Output = Result<T, CryptoError>>,
        G: Future<Output = Result<T, CryptoError>>,
    {
        match primary.await {
            Ok(result) => Ok(result),
            Err(e) if self.enabled && e.is_transient() => {
                self.activation_count.fetch_add(1, Ordering::Relaxed);
                tracing::warn!(error = %e, "Crypto Service unavailable, using fallback");
                fallback.await
            }
            Err(e) => Err(e),
        }
    }
}
```

### 4. CryptoSigner (JWT Integration)

Adapter that implements `KmsSigner` trait using CryptoClient.

```rust
/// Crypto Service based JWT signer
pub struct CryptoSigner {
    /// CryptoClient for signing operations
    client: Arc<dyn CryptoClient>,
    /// Signing key ID
    key_id: KeyId,
    /// Algorithm (PS256, ES256, etc.)
    algorithm: Algorithm,
    /// Cached key metadata
    cached_metadata: RwLock<Option<CachedKeyMetadata>>,
}

#[async_trait]
impl KmsSigner for CryptoSigner {
    async fn sign(&self, data: &[u8]) -> Result<Vec<u8>, TokenError> {
        // Verify key state before signing
        self.verify_key_state().await?;
        
        let result = self.client.sign(data, &self.key_id).await
            .map_err(|e| TokenError::signing(e.to_string()))?;
        
        Ok(result.signature)
    }
    
    fn get_encoding_key(&self) -> Result<EncodingKey, TokenError> {
        // For Crypto Service, we don't have local access to private key
        Err(TokenError::kms("Use sign() method for Crypto Service signing"))
    }
    
    fn key_id(&self) -> &str {
        &self.key_id.id
    }
    
    fn algorithm(&self) -> &str {
        self.algorithm.as_str()
    }
}
```

### 5. CryptoEncryptor (Cache Integration)

Adapter for cache encryption using CryptoClient.

```rust
/// Crypto Service based encryptor for cache data
pub struct CryptoEncryptor {
    /// CryptoClient for encryption operations
    client: Arc<dyn CryptoClient>,
    /// Encryption key ID
    key_id: KeyId,
    /// Namespace for cache keys
    namespace: String,
}

impl CryptoEncryptor {
    /// Encrypt token family data for cache storage
    pub async fn encrypt_token_family(&self, family: &TokenFamily) -> Result<Vec<u8>, TokenError> {
        let plaintext = serde_json::to_vec(family)
            .map_err(|e| TokenError::internal(format!("Serialization failed: {}", e)))?;
        
        let aad = family.family_id.as_bytes();
        let result = self.client.encrypt(&plaintext, &self.key_id, Some(aad)).await
            .map_err(|e| TokenError::encryption(e.to_string()))?;
        
        // Serialize encrypted result for storage
        result.to_bytes()
    }
    
    /// Decrypt token family data from cache
    pub async fn decrypt_token_family(&self, encrypted: &[u8], family_id: &str) -> Result<TokenFamily, TokenError> {
        let encrypted_data = EncryptedData::from_bytes(encrypted)?;
        let aad = family_id.as_bytes();
        
        let plaintext = self.client.decrypt(&encrypted_data, &self.key_id, Some(aad)).await
            .map_err(|e| TokenError::decryption(e.to_string()))?;
        
        serde_json::from_slice(&plaintext)
            .map_err(|e| TokenError::internal(format!("Deserialization failed: {}", e)))
    }
}
```

## Data Models

### KeyId

```rust
/// Key identifier matching Crypto Service proto
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct KeyId {
    /// Namespace for key isolation
    pub namespace: String,
    /// Unique key identifier
    pub id: String,
    /// Key version
    pub version: u32,
}
```

### SignResult

```rust
/// Result of a signing operation
#[derive(Debug, Clone)]
pub struct SignResult {
    /// Signature bytes
    pub signature: Vec<u8>,
    /// Key ID used for signing
    pub key_id: KeyId,
    /// Algorithm used
    pub algorithm: String,
}
```

### EncryptResult / EncryptedData

```rust
/// Result of an encryption operation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptResult {
    /// Ciphertext
    pub ciphertext: Vec<u8>,
    /// Initialization vector
    pub iv: Vec<u8>,
    /// Authentication tag
    pub tag: Vec<u8>,
    /// Key ID used
    pub key_id: KeyId,
    /// Algorithm used
    pub algorithm: String,
}

/// Encrypted data for storage/transmission
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptedData {
    pub ciphertext: Vec<u8>,
    pub iv: Vec<u8>,
    pub tag: Vec<u8>,
}
```

### KeyMetadata

```rust
/// Key metadata from Crypto Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyMetadata {
    pub id: KeyId,
    pub algorithm: KeyAlgorithm,
    pub state: KeyState,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
    pub rotated_at: Option<DateTime<Utc>>,
    pub previous_version: Option<KeyId>,
    pub owner_service: String,
    pub allowed_operations: Vec<String>,
    pub usage_count: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum KeyState {
    PendingActivation,
    Active,
    Deprecated,
    PendingDestruction,
    Destroyed,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum KeyAlgorithm {
    Aes256Gcm,
    RsaPss256,
    RsaPss384,
    RsaPss512,
    EcdsaP256,
    EcdsaP384,
}
```

### CryptoError

```rust
/// Errors from CryptoClient operations
#[derive(Debug, thiserror::Error)]
pub enum CryptoError {
    #[error("Connection failed: {0}")]
    Connection(String),
    
    #[error("Signing failed: {0}")]
    Signing(String),
    
    #[error("Verification failed: {0}")]
    Verification(String),
    
    #[error("Encryption failed: {0}")]
    Encryption(String),
    
    #[error("Decryption failed: {0}")]
    Decryption(String),
    
    #[error("Key not found: {0}")]
    KeyNotFound(String),
    
    #[error("Invalid key state: {state:?} for operation {operation}")]
    InvalidKeyState { state: KeyState, operation: String },
    
    #[error("Invalid algorithm: expected {expected}, got {actual}")]
    InvalidAlgorithm { expected: String, actual: String },
    
    #[error("Rate limited")]
    RateLimited,
    
    #[error("Circuit breaker open")]
    CircuitBreakerOpen,
    
    #[error("Timeout")]
    Timeout,
    
    #[error("Internal error: {0}")]
    Internal(String),
}

impl CryptoError {
    /// Check if error is transient (suitable for fallback)
    pub fn is_transient(&self) -> bool {
        matches!(
            self,
            CryptoError::Connection(_)
                | CryptoError::CircuitBreakerOpen
                | CryptoError::Timeout
                | CryptoError::RateLimited
        )
    }
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Circuit Breaker State Transitions

*For any* sequence of N consecutive failures where N >= failure_threshold, the circuit breaker SHALL transition to Open state and reject subsequent requests until the recovery timeout expires.

**Validates: Requirements 1.3**

### Property 2: Fallback Activation on Service Unavailability

*For any* cryptographic operation (sign, verify, encrypt, decrypt) when the Crypto Service is unavailable and fallback is enabled, the operation SHALL complete successfully using local fallback mechanisms.

**Validates: Requirements 1.4, 2.4, 4.4, 5.3**

### Property 3: Request Context Propagation

*For any* request to Crypto Service, the request SHALL contain a valid correlation_id and W3C Trace Context headers (traceparent, tracestate).

**Validates: Requirements 1.5, 4.5**

### Property 4: JWT Signing Round Trip

*For any* valid JWT claims and supported algorithm, signing via Crypto Service followed by verification SHALL produce a valid JWT that can be decoded with the corresponding public key.

**Validates: Requirements 2.1, 2.3**

### Property 5: Algorithm Support Completeness

*For any* algorithm in {PS256, PS384, PS512, ES256, ES384}, both signing and verification operations SHALL succeed when using a key of the corresponding type.

**Validates: Requirements 2.2, 5.2**

### Property 6: Key Metadata Caching Consistency

*For any* key ID, after the first successful GetKeyMetadata call, subsequent calls within the cache TTL SHALL return cached data without making an RPC call, and the cached data SHALL match the original response.

**Validates: Requirements 2.5, 5.4**

### Property 7: Key Rotation Graceful Transition

*For any* key rotation event, tokens signed with the previous key version SHALL remain verifiable for the configured retention period (default 24 hours), and the JWKS endpoint SHALL include both key versions during this period.

**Validates: Requirements 2.6, 3.2, 3.3**

### Property 8: Key State Validation

*For any* signing operation, if the key is in DEPRECATED or DESTROYED state, the operation SHALL fail with InvalidKeyState error and no signature SHALL be produced.

**Validates: Requirements 3.4, 3.5, 8.5**

### Property 9: Encryption Round Trip

*For any* valid TokenFamily data, encrypting via Crypto Service and then decrypting SHALL produce data equivalent to the original input.

**Validates: Requirements 4.1, 4.2**

### Property 10: Error Logging Completeness

*For any* failed Crypto Service RPC, the error log entry SHALL contain: correlation_id, operation type, error code, error message, and latency.

**Validates: Requirements 6.4**

### Property 11: Feature Flag Behavior

*For any* configuration where CRYPTO_SIGNING_ENABLED=false, all signing operations SHALL use local KMS without attempting Crypto Service RPC. Similarly for CRYPTO_ENCRYPTION_ENABLED and encryption operations.

**Validates: Requirements 7.2, 7.3, 7.4**

### Property 12: Configuration Validation

*For any* startup with missing required configuration (CRYPTO_SERVICE_ADDRESS when signing or encryption is enabled), the service SHALL fail to start with a descriptive error message.

**Validates: Requirements 7.6**

### Property 13: Response Algorithm Validation

*For any* Crypto Service response, if the returned algorithm does not match the expected algorithm for the key, the operation SHALL fail with InvalidAlgorithm error and log a security event.

**Validates: Requirements 8.2**

### Property 14: Rate Limiting Enforcement

*For any* request rate exceeding the configured limit (default 1000 req/s), excess requests SHALL be rejected with RateLimited error without being sent to Crypto Service.

**Validates: Requirements 8.6**

## Error Handling

### Error Categories

| Category | Behavior | Fallback |
|----------|----------|----------|
| Connection errors | Circuit breaker tracks, fallback if enabled | Local KMS/AES |
| Timeout | Circuit breaker tracks, fallback if enabled | Local KMS/AES |
| Rate limited | Immediate rejection | None |
| Invalid key state | Immediate rejection | None |
| Invalid algorithm | Immediate rejection, security log | None |
| Decryption failure | Return error | None |
| Key not found | Return error | None |

### Circuit Breaker Configuration

```rust
pub struct CircuitBreakerConfig {
    /// Number of failures before opening circuit
    pub failure_threshold: u32,  // default: 5
    /// Time to wait before attempting recovery
    pub recovery_timeout: Duration,  // default: 30s
    /// Number of successful calls to close circuit
    pub success_threshold: u32,  // default: 3
}
```

### Fallback Decision Flow

```
┌─────────────────┐
│ Crypto Service  │
│    Request      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     No      ┌─────────────────┐
│ Circuit Breaker ├────────────►│ Return Error    │
│    Allows?      │             │ (CircuitOpen)   │
└────────┬────────┘             └─────────────────┘
         │ Yes
         ▼
┌─────────────────┐     No      ┌─────────────────┐
│  Rate Limiter   ├────────────►│ Return Error    │
│    Allows?      │             │ (RateLimited)   │
└────────┬────────┘             └─────────────────┘
         │ Yes
         ▼
┌─────────────────┐
│  Execute RPC    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     Yes     ┌─────────────────┐
│    Success?     ├────────────►│ Return Result   │
└────────┬────────┘             └─────────────────┘
         │ No
         ▼
┌─────────────────┐     No      ┌─────────────────┐
│ Is Transient    ├────────────►│ Return Error    │
│    Error?       │             │                 │
└────────┬────────┘             └─────────────────┘
         │ Yes
         ▼
┌─────────────────┐     No      ┌─────────────────┐
│   Fallback      ├────────────►│ Return Error    │
│   Enabled?      │             │                 │
└────────┬────────┘             └─────────────────┘
         │ Yes
         ▼
┌─────────────────┐
│ Execute Local   │
│   Fallback      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Return Result   │
│ (with fallback  │
│    metric)      │
└─────────────────┘
```

## Testing Strategy

### Unit Tests

Unit tests verify specific examples and edge cases:

- CryptoClient initialization with valid/invalid config
- Circuit breaker state transitions
- Rate limiter behavior at boundary conditions
- Key metadata cache hit/miss scenarios
- Error mapping from gRPC status codes
- Configuration validation at startup

### Property-Based Tests

Property-based tests verify universal properties across all inputs using `proptest`:

```rust
// Example: Property 9 - Encryption Round Trip
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]
    
    #[test]
    fn prop_encryption_round_trip(
        family_id in "[a-z0-9-]{36}",
        user_id in "[a-z0-9-]{36}",
        session_id in "[a-z0-9-]{36}",
        token_hash in "[a-f0-9]{64}",
    ) {
        // Feature: token-crypto-service-integration, Property 9: Encryption Round Trip
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let encryptor = create_test_encryptor().await;
            let family = TokenFamily::new(family_id, user_id, session_id, token_hash);
            
            let encrypted = encryptor.encrypt_token_family(&family).await.unwrap();
            let decrypted = encryptor.decrypt_token_family(&encrypted, &family.family_id).await.unwrap();
            
            prop_assert_eq!(family, decrypted);
        });
    }
}
```

### Integration Tests

Integration tests verify end-to-end flows with mock Crypto Service:

- JWT generation and verification flow
- Key rotation with JWKS update
- Fallback activation and recovery
- Metrics emission verification

### Test Configuration

- Property tests: minimum 100 iterations per property
- Integration tests: use `wiremock` for Crypto Service mocking
- All tests tagged with property reference for traceability
