# Design Document: Session-Identity Crypto Service Integration

## Overview

This design document describes the integration between the `session-identity` Elixir service and the centralized `crypto-service` C++ microservice. The integration enables centralized cryptographic operations for JWT signing, session data encryption, and refresh token protection with automated key management.

The integration follows the platform's service mesh architecture, using gRPC for communication with circuit breaker resilience patterns via Linkerd.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Session Identity Core                               │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    Crypto Integration Layer                      │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │    │
│  │  │ CryptoClient │  │ JWTSigner    │  │ EncryptedStore       │   │    │
│  │  │ (gRPC)       │  │ (Sign/Verify)│  │ (Session/Token)      │   │    │
│  │  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘   │    │
│  │         │                 │                      │               │    │
│  │  ┌──────▼─────────────────▼──────────────────────▼───────────┐  │    │
│  │  │                  KeyManager                                │  │    │
│  │  │  - Key metadata caching                                    │  │    │
│  │  │  - Version tracking                                        │  │    │
│  │  │  - Rotation handling                                       │  │    │
│  │  └───────────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │
│  │ SessionManager  │  │ OAuth Module    │  │ Existing Modules        │  │
│  │ (uses Encrypted │  │ (uses JWTSigner)│  │ (unchanged)             │  │
│  │  Store)         │  │                 │  │                         │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ gRPC (mTLS via Linkerd)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Crypto Service                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────────┐   │
│  │ Sign/Verify  │  │ Encrypt/     │  │ Key Management               │   │
│  │ (ECDSA/RSA)  │  │ Decrypt      │  │ (Generate/Rotate/Metadata)   │   │
│  │              │  │ (AES-256-GCM)│  │                              │   │
│  └──────────────┘  └──────────────┘  └──────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. CryptoClient Module

The gRPC client for communicating with crypto-service.

```elixir
defmodule SessionIdentityCore.Crypto.Client do
  @moduledoc """
  gRPC client for crypto-service with circuit breaker support.
  """
  
  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}
  @type encrypt_result :: {:ok, %{ciphertext: binary(), iv: binary(), tag: binary(), key_id: key_id()}} | {:error, term()}
  @type decrypt_result :: {:ok, binary()} | {:error, term()}
  @type sign_result :: {:ok, %{signature: binary(), key_id: key_id()}} | {:error, term()}
  @type verify_result :: {:ok, boolean()} | {:error, term()}
  
  @callback encrypt(plaintext :: binary(), key_id :: key_id(), aad :: binary() | nil, opts :: keyword()) :: encrypt_result()
  @callback decrypt(ciphertext :: binary(), iv :: binary(), tag :: binary(), key_id :: key_id(), aad :: binary() | nil, opts :: keyword()) :: decrypt_result()
  @callback sign(data :: binary(), key_id :: key_id(), hash_algorithm :: atom(), opts :: keyword()) :: sign_result()
  @callback verify(data :: binary(), signature :: binary(), key_id :: key_id(), hash_algorithm :: atom(), opts :: keyword()) :: verify_result()
  @callback get_key_metadata(key_id :: key_id(), opts :: keyword()) :: {:ok, map()} | {:error, term()}
  @callback health_check() :: {:ok, :serving} | {:error, term()}
end
```

### 2. JWTSigner Module

Handles JWT signing and verification via crypto-service.

```elixir
defmodule SessionIdentityCore.Crypto.JWTSigner do
  @moduledoc """
  JWT signing and verification using crypto-service.
  Falls back to local Joken when crypto-service is unavailable.
  """
  
  @type claims :: map()
  @type jwt :: String.t()
  
  @callback sign_jwt(claims :: claims(), opts :: keyword()) :: {:ok, jwt()} | {:error, term()}
  @callback verify_jwt(token :: jwt(), opts :: keyword()) :: {:ok, claims()} | {:error, term()}
end
```

### 3. EncryptedStore Module

Provides encrypted storage for session data and refresh tokens.

```elixir
defmodule SessionIdentityCore.Crypto.EncryptedStore do
  @moduledoc """
  Encrypted storage wrapper for sensitive data.
  """
  
  @type store_result :: {:ok, String.t()} | {:error, term()}
  @type retrieve_result :: {:ok, binary()} | {:error, term()}
  
  @callback store(namespace :: atom(), key :: String.t(), data :: binary(), aad :: binary()) :: store_result()
  @callback retrieve(namespace :: atom(), key :: String.t(), aad :: binary()) :: retrieve_result()
  @callback delete(namespace :: atom(), key :: String.t()) :: :ok | {:error, term()}
end
```

### 4. KeyManager Module

Manages key metadata caching and rotation handling.

```elixir
defmodule SessionIdentityCore.Crypto.KeyManager do
  @moduledoc """
  Key metadata caching and rotation management.
  """
  
  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}
  
  @callback get_active_key(namespace :: String.t()) :: {:ok, key_id()} | {:error, term()}
  @callback get_key_metadata(key_id :: key_id()) :: {:ok, map()} | {:error, term()}
  @callback invalidate_cache(namespace :: String.t()) :: :ok
end
```

## Data Models

### Encrypted Session Data Format

```elixir
%{
  # Stored in Redis as JSON
  "v" => 1,                          # Format version
  "key_id" => %{                     # Key used for encryption
    "namespace" => "session",
    "id" => "session-dek",
    "version" => 3
  },
  "iv" => "base64_encoded_iv",       # Initialization vector
  "tag" => "base64_encoded_tag",     # GCM authentication tag
  "ciphertext" => "base64_encoded",  # Encrypted session JSON
  "encrypted_at" => 1703520000       # Unix timestamp
}
```

### Encrypted Refresh Token Format

```elixir
%{
  "v" => 1,
  "key_id" => %{
    "namespace" => "refresh_token",
    "id" => "refresh-dek",
    "version" => 2
  },
  "iv" => "base64_encoded_iv",
  "tag" => "base64_encoded_tag",
  "ciphertext" => "base64_encoded",
  "encrypted_at" => 1703520000
}
```

### Key Namespaces

| Namespace | Purpose | Key Type |
|-----------|---------|----------|
| `session_identity:jwt` | JWT signing | ECDSA P-256 or RSA-2048 |
| `session_identity:session` | Session data encryption | AES-256-GCM |
| `session_identity:refresh_token` | Refresh token encryption | AES-256-GCM |

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Circuit Breaker Fallback Behavior

*For any* sequence of N consecutive failures to crypto-service where N exceeds the failure threshold, the circuit breaker SHALL open and subsequent operations SHALL use local fallback (when enabled) without attempting remote calls until the recovery period elapses.

**Validates: Requirements 1.2, 2.5**

### Property 2: Trace Context Propagation

*For any* crypto operation initiated with a W3C Trace Context, the outgoing gRPC request to crypto-service SHALL contain the same traceparent and tracestate headers.

**Validates: Requirements 1.3**

### Property 3: Correlation ID Inclusion

*For any* crypto operation, the gRPC request to crypto-service SHALL include a non-empty correlation_id field.

**Validates: Requirements 1.4**

### Property 4: Structured Error Responses

*For any* failed crypto operation, the returned error SHALL contain an error_code (atom) and message (string) that can be used for debugging and monitoring.

**Validates: Requirements 1.5**

### Property 5: JWT Signing Round-Trip

*For any* valid claims map, signing the claims via crypto-service and then verifying the resulting JWT SHALL return the original claims (modulo standard JWT transformations like atom-to-string keys).

**Validates: Requirements 2.1, 2.3**

### Property 6: Key Metadata Caching

*For any* key metadata request within the cache TTL period, the KeyManager SHALL return cached metadata without making a remote call to crypto-service.

**Validates: Requirements 2.4**

### Property 7: Session Data Encryption Round-Trip

*For any* valid session struct, encrypting and storing the session, then retrieving and decrypting it SHALL return a session equivalent to the original.

**Validates: Requirements 3.1, 3.2**

### Property 8: Key Namespace Isolation

*For any* encryption operation, the key_id used SHALL have a namespace matching the data type being encrypted (session data uses "session_identity:session", refresh tokens use "session_identity:refresh_token").

**Validates: Requirements 3.3, 4.3**

### Property 9: AAD Binding Integrity

*For any* encrypted data with AAD, attempting to decrypt with different AAD values SHALL fail with an authentication error.

**Validates: Requirements 3.4, 4.4**

### Property 10: Multi-Version Key Decryption

*For any* data encrypted with key version N, decryption SHALL succeed when key versions N through N+K are available (where K is the rotation overlap period).

**Validates: Requirements 3.5, 5.1, 5.3**

### Property 11: Refresh Token Encryption Round-Trip

*For any* valid refresh token payload, encrypting and storing the token, then retrieving and decrypting it SHALL return the original payload.

**Validates: Requirements 4.1, 4.2**

### Property 12: Latest Key Version for New Encryptions

*For any* new encryption operation, the key version used SHALL be the highest active version for that namespace.

**Validates: Requirements 5.2**

### Property 13: Re-encryption on Deprecated Key Access

*For any* data encrypted with a deprecated key version, after successful retrieval, the data SHALL be re-encrypted with the latest active key version.

**Validates: Requirements 5.5**

### Property 14: Metrics Emission

*For any* crypto operation (success or failure), the corresponding Prometheus counter SHALL be incremented and latency histogram SHALL be updated.

**Validates: Requirements 6.2**

### Property 15: Health Check Integration

*For any* readiness check when crypto integration is enabled, the result SHALL reflect the crypto-service health status.

**Validates: Requirements 6.3**

### Property 16: Feature Toggle Behavior

*For any* crypto operation when crypto integration is disabled, the operation SHALL use local implementation without attempting remote calls.

**Validates: Requirements 6.5**

## Error Handling

### Error Categories

| Category | Error Code | Recovery Strategy |
|----------|------------|-------------------|
| Connection | `:crypto_service_unavailable` | Circuit breaker + fallback |
| Timeout | `:crypto_operation_timeout` | Retry with backoff |
| Authentication | `:crypto_auth_failed` | Log + fail (no retry) |
| Key Not Found | `:key_not_found` | Refresh key cache + retry once |
| Decryption Failed | `:decryption_failed` | Try previous key version |
| Invalid AAD | `:aad_mismatch` | Log security event + fail |

### Fallback Behavior

When crypto-service is unavailable and fallback is enabled:

1. **JWT Signing**: Use local Joken with configured local keys
2. **Session Encryption**: Store unencrypted (with warning log)
3. **Refresh Token Encryption**: Store unencrypted (with warning log)

Fallback operations emit `crypto.fallback.used` metric with `operation` tag.

## Testing Strategy

### Unit Tests

- CryptoClient request/response serialization
- KeyManager cache behavior
- Error mapping and handling
- Configuration validation

### Property-Based Tests

Using StreamData with minimum 100 iterations per property:

| Property | Generator | Assertion |
|----------|-----------|-----------|
| JWT Round-Trip | Random claims maps | `verify(sign(claims)) == claims` |
| Session Encryption Round-Trip | Random Session structs | `decrypt(encrypt(session)) == session` |
| AAD Binding | Random data + AAD pairs | `decrypt(ct, wrong_aad) == error` |
| Key Version Selection | Random key states | `encrypt uses latest active` |

### Integration Tests

- End-to-end crypto-service communication
- Circuit breaker state transitions
- Key rotation scenarios
- Fallback activation

### Test Configuration

```elixir
# Property test configuration
config :stream_data,
  max_runs: 100,
  max_shrinks: 50

# Test tags for property tests
# @tag property: true
# @tag validates: "Requirements 3.1, 3.2"
```

