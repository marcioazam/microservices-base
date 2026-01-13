# Design Document: IAM Policy Crypto Integration

## Overview

Este documento descreve o design para integrar o serviço `iam-policy` com o `crypto-service` centralizado da plataforma. A integração adiciona duas capacidades principais:

1. **Criptografia de Decisões em Cache**: Decisões de autorização armazenadas no cache distribuído serão criptografadas com AES-256-GCM
2. **Assinatura de Decisões**: Decisões de autorização podem ser assinadas digitalmente com ECDSA P-256 para garantir integridade e auditoria

A integração segue o padrão de resiliência da plataforma: fallback gracioso quando o crypto-service está indisponível, métricas detalhadas, e propagação de trace context.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      iam-policy-service                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────┐  │
│  │ Authorization   │  │ Policy Engine   │  │ RBAC Hierarchy │  │
│  │ Service         │  │ (OPA)           │  │                │  │
│  └────────┬────────┘  └────────┬────────┘  └────────────────┘  │
│           │                    │                                 │
│  ┌────────▼────────────────────▼────────────────────────────┐   │
│  │                    Decision Cache                         │   │
│  │  ┌──────────────────────────────────────────────────┐    │   │
│  │  │ EncryptedDecisionCache (decorator)               │    │   │
│  │  │  - Encrypts before Set()                         │    │   │
│  │  │  - Decrypts after Get()                          │    │   │
│  │  │  - Uses AAD for context binding                  │    │   │
│  │  └──────────────────────────────────────────────────┘    │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Crypto Client                          │   │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐  │   │
│  │  │ Encrypt/     │ │ Sign/Verify  │ │ Key Metadata     │  │   │
│  │  │ Decrypt      │ │              │ │ Cache            │  │   │
│  │  └──────────────┘ └──────────────┘ └──────────────────┘  │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ gRPC (mTLS via Linkerd)
              ┌──────────────────────────────────────┐
              │         crypto-service               │
              │  - AES-256-GCM encryption            │
              │  - ECDSA P-256 signatures            │
              │  - Key management                    │
              └──────────────────────────────────────┘
```

## Components and Interfaces

### 1. Crypto Client (`internal/crypto/client.go`)

Cliente gRPC para comunicação com o crypto-service.

```go
// Client provides cryptographic operations via crypto-service.
type Client struct {
    conn          *grpc.ClientConn
    cryptoClient  cryptopb.CryptoServiceClient
    keyCache      *KeyMetadataCache
    logger        *logging.Logger
    metrics       *CryptoMetrics
    config        ClientConfig
}

// ClientConfig holds configuration for the crypto client.
type ClientConfig struct {
    Address           string
    Timeout           time.Duration
    EncryptionKeyID   KeyID
    SigningKeyID      KeyID
    KeyCacheTTL       time.Duration
    Enabled           bool
    CacheEncryption   bool
    DecisionSigning   bool
}

// KeyID identifies a cryptographic key.
type KeyID struct {
    Namespace string
    ID        string
    Version   uint32
}

// NewClient creates a new crypto client.
func NewClient(cfg ClientConfig, logger *logging.Logger) (*Client, error)

// Encrypt encrypts plaintext using AES-256-GCM.
func (c *Client) Encrypt(ctx context.Context, plaintext, aad []byte) (*EncryptResult, error)

// Decrypt decrypts ciphertext using AES-256-GCM.
func (c *Client) Decrypt(ctx context.Context, ciphertext, iv, tag, aad []byte) ([]byte, error)

// Sign creates an ECDSA signature.
func (c *Client) Sign(ctx context.Context, data []byte) (*SignResult, error)

// Verify verifies an ECDSA signature.
func (c *Client) Verify(ctx context.Context, data, signature []byte) (bool, error)

// HealthCheck checks crypto-service connectivity.
func (c *Client) HealthCheck(ctx context.Context) (*HealthStatus, error)

// Close closes the gRPC connection.
func (c *Client) Close() error
```

### 2. Encrypted Decision Cache (`internal/cache/encrypted_cache.go`)

Decorator que adiciona criptografia ao DecisionCache existente.

```go
// EncryptedDecisionCache wraps DecisionCache with encryption.
type EncryptedDecisionCache struct {
    inner        *DecisionCache
    cryptoClient *crypto.Client
    enabled      bool
    logger       *logging.Logger
    metrics      *CacheMetrics
}

// NewEncryptedDecisionCache creates an encrypted cache wrapper.
func NewEncryptedDecisionCache(inner *DecisionCache, client *crypto.Client, enabled bool) *EncryptedDecisionCache

// Get retrieves and decrypts a cached decision.
func (ec *EncryptedDecisionCache) Get(ctx context.Context, input map[string]interface{}) (*Decision, bool)

// Set encrypts and stores a decision.
func (ec *EncryptedDecisionCache) Set(ctx context.Context, input map[string]interface{}, decision *Decision) error

// generateAAD creates AAD from authorization input for context binding.
func (ec *EncryptedDecisionCache) generateAAD(input map[string]interface{}) []byte
```

### 3. Decision Signer (`internal/crypto/signer.go`)

Componente para assinar e verificar decisões de autorização.

```go
// SignedDecision represents a signed authorization decision.
type SignedDecision struct {
    DecisionID   string    `json:"decision_id"`
    Timestamp    int64     `json:"timestamp"`
    SubjectID    string    `json:"subject_id"`
    ResourceID   string    `json:"resource_id"`
    Action       string    `json:"action"`
    Allowed      bool      `json:"allowed"`
    PolicyName   string    `json:"policy_name"`
    Signature    []byte    `json:"signature,omitempty"`
    KeyID        KeyID     `json:"key_id,omitempty"`
}

// DecisionSigner signs authorization decisions.
type DecisionSigner struct {
    cryptoClient *Client
    enabled      bool
    logger       *logging.Logger
}

// NewDecisionSigner creates a new decision signer.
func NewDecisionSigner(client *Client, enabled bool, logger *logging.Logger) *DecisionSigner

// Sign signs an authorization decision.
func (s *DecisionSigner) Sign(ctx context.Context, decision *SignedDecision) error

// Verify verifies a signed decision.
func (s *DecisionSigner) Verify(ctx context.Context, decision *SignedDecision) (bool, error)

// buildSignaturePayload creates the canonical payload for signing.
func (s *DecisionSigner) buildSignaturePayload(decision *SignedDecision) []byte
```

### 4. Key Metadata Cache (`internal/crypto/key_cache.go`)

Cache local para metadados de chaves.

```go
// KeyMetadataCache caches key metadata locally.
type KeyMetadataCache struct {
    cache map[string]*CachedKeyMetadata
    mu    sync.RWMutex
    ttl   time.Duration
}

// CachedKeyMetadata holds cached key information.
type CachedKeyMetadata struct {
    Metadata  *cryptopb.KeyMetadata
    CachedAt  time.Time
    ExpiresAt time.Time
}

// NewKeyMetadataCache creates a new key metadata cache.
func NewKeyMetadataCache(ttl time.Duration) *KeyMetadataCache

// Get retrieves cached key metadata.
func (c *KeyMetadataCache) Get(keyID KeyID) (*cryptopb.KeyMetadata, bool)

// Set stores key metadata in cache.
func (c *KeyMetadataCache) Set(keyID KeyID, metadata *cryptopb.KeyMetadata)

// Invalidate removes a key from cache.
func (c *KeyMetadataCache) Invalidate(keyID KeyID)
```

### 5. Crypto Metrics (`internal/crypto/metrics.go`)

Métricas Prometheus para operações criptográficas.

```go
// CryptoMetrics holds Prometheus metrics for crypto operations.
type CryptoMetrics struct {
    encryptTotal   *prometheus.CounterVec
    decryptTotal   *prometheus.CounterVec
    signTotal      *prometheus.CounterVec
    verifyTotal    *prometheus.CounterVec
    latency        *prometheus.HistogramVec
    errorsTotal    *prometheus.CounterVec
    fallbackTotal  prometheus.Counter
}

// NewCryptoMetrics creates and registers crypto metrics.
func NewCryptoMetrics(registry prometheus.Registerer) *CryptoMetrics

// RecordEncrypt records an encryption operation.
func (m *CryptoMetrics) RecordEncrypt(status string, duration time.Duration)

// RecordDecrypt records a decryption operation.
func (m *CryptoMetrics) RecordDecrypt(status string, duration time.Duration)

// RecordSign records a signing operation.
func (m *CryptoMetrics) RecordSign(status string, duration time.Duration)

// RecordVerify records a verification operation.
func (m *CryptoMetrics) RecordVerify(status string, duration time.Duration)

// RecordFallback records a fallback to unencrypted mode.
func (m *CryptoMetrics) RecordFallback()
```

## Data Models

### EncryptedCacheEntry

Estrutura armazenada no cache quando criptografia está habilitada:

```go
// EncryptedCacheEntry represents an encrypted decision in cache.
type EncryptedCacheEntry struct {
    Ciphertext []byte `json:"ciphertext"`
    IV         []byte `json:"iv"`
    Tag        []byte `json:"tag"`
    KeyID      KeyID  `json:"key_id"`
    Algorithm  string `json:"algorithm"`
    CachedAt   int64  `json:"cached_at"`
    ExpiresAt  int64  `json:"expires_at"`
}
```

### Extended Configuration

Extensão da configuração existente:

```go
// CryptoConfig holds crypto client configuration.
type CryptoConfig struct {
    Enabled           bool
    Address           string
    Timeout           time.Duration
    CacheEncryption   bool
    DecisionSigning   bool
    EncryptionKeyID   string  // formato: namespace/id/version
    SigningKeyID      string  // formato: namespace/id/version
    KeyCacheTTL       time.Duration
}
```

## Correctness Properties

*Uma propriedade é uma característica ou comportamento que deve ser verdadeiro em todas as execuções válidas do sistema. Propriedades servem como ponte entre especificações legíveis por humanos e garantias de correção verificáveis por máquina.*

### Property 1: Encryption Round-Trip Consistency

*For any* valid authorization decision, encrypting then decrypting SHALL produce a decision equivalent to the original.

**Validates: Requirements 2.1, 2.2, 2.6**

### Property 2: AAD Context Binding

*For any* encrypted decision, attempting to decrypt with different AAD (subject_id or resource_id) SHALL fail and return cache miss.

**Validates: Requirements 2.3, 2.4**

### Property 3: Sign-Then-Verify Consistency

*For any* authorization decision that is signed, verifying the signature with the same data SHALL return true.

**Validates: Requirements 3.1, 3.3**

### Property 4: Signature Payload Completeness

*For any* signed decision, the signature payload SHALL contain all required fields: timestamp, decision_id, subject_id, resource_id, action, allowed, policy_name.

**Validates: Requirements 3.2**

### Property 5: Invalid Signature Detection

*For any* signed decision with tampered data or invalid signature, verification SHALL return false and error SIGNATURE_INVALID.

**Validates: Requirements 3.4**

### Property 6: Graceful Degradation

*For any* authorization request when crypto-service is unavailable, the service SHALL continue operating and return valid decisions (without encryption/signing).

**Validates: Requirements 1.4, 2.5**

### Property 7: Key Version Backward Compatibility

*For any* decision signed with a previous key version, verification with the current key configuration SHALL still succeed.

**Validates: Requirements 4.3**

### Property 8: Configuration Validation

*For any* invalid configuration (missing required fields, invalid key ID format), service initialization SHALL fail with descriptive error.

**Validates: Requirements 5.7**

### Property 9: Trace Context Propagation

*For any* request with W3C Trace Context, the crypto client SHALL propagate trace_id and span_id to crypto-service.

**Validates: Requirements 1.3**

### Property 10: Error Correlation

*For any* error returned by crypto client, the error SHALL contain correlation_id matching the request context.

**Validates: Requirements 1.2**

### Property 11: Health Check Degraded Status

*For any* health check when crypto-service is unavailable, the status SHALL be DEGRADED (not UNHEALTHY).

**Validates: Requirements 7.2**

## Error Handling

### Error Types

```go
// CryptoError represents a crypto operation error.
type CryptoError struct {
    Code          string
    Message       string
    CorrelationID string
    Cause         error
}

const (
    ErrCodeEncryptionFailed   = "ENCRYPTION_FAILED"
    ErrCodeDecryptionFailed   = "DECRYPTION_FAILED"
    ErrCodeSignatureFailed    = "SIGNATURE_FAILED"
    ErrCodeSignatureInvalid   = "SIGNATURE_INVALID"
    ErrCodeKeyNotFound        = "KEY_NOT_FOUND"
    ErrCodeServiceUnavailable = "CRYPTO_SERVICE_UNAVAILABLE"
    ErrCodeAADMismatch        = "AAD_MISMATCH"
)
```

### Fallback Behavior

Quando o crypto-service está indisponível:

1. **Cache Encryption**: Armazena decisões em texto plano (JSON)
2. **Decision Signing**: Omite campo signature na resposta
3. **Metrics**: Incrementa `iam_crypto_fallback_total`
4. **Logs**: Registra warning com correlation_id
5. **Health**: Retorna status DEGRADED

## Testing Strategy

### Unit Tests

- Crypto client connection handling
- Encryption/decryption with mock crypto-service
- AAD generation from authorization input
- Signature payload serialization
- Key metadata cache TTL behavior
- Configuration parsing and validation
- Error handling and fallback logic

### Property-Based Tests

Usando `pgregory.net/rapid` com mínimo 100 iterações:

1. **Encryption Round-Trip**: Gerar decisões aleatórias, criptografar, descriptografar, verificar equivalência
2. **AAD Binding**: Gerar decisões e AADs aleatórios, verificar que AAD diferente causa falha
3. **Sign-Verify**: Gerar decisões aleatórias, assinar, verificar
4. **Signature Tampering**: Gerar decisões assinadas, modificar campos, verificar detecção
5. **Configuration Validation**: Gerar configurações válidas e inválidas, verificar comportamento

### Integration Tests

- Comunicação real com crypto-service em ambiente de teste
- Rotação de chaves e verificação de backward compatibility
- Health check com crypto-service up/down
- Métricas Prometheus expostas corretamente
