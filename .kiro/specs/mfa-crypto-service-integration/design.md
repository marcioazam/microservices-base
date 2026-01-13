# Design Document: MFA Crypto Service Integration

## Overview

This design document describes the integration of the MFA Service (Elixir/OTP) with the centralized Crypto Security Service (C++/gRPC). The integration delegates cryptographic operations for TOTP secret encryption/decryption to the crypto-service, enabling centralized key management, HSM/KMS support, and consistent cryptographic practices.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           MFA Service (Elixir)                          │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐  ┌─────────────────────────────────────────┐   │
│  │ TOTP Module         │  │ Passkeys Module                         │   │
│  │ ┌─────────────────┐ │  │ (unchanged - uses local :crypto for    │   │
│  │ │ Generator       │ │  │  WebAuthn signature verification)       │   │
│  │ │ - encrypt_secret│─┼──┼──────────────────────────────────────┐  │   │
│  │ │ - decrypt_secret│ │  │                                      │  │   │
│  │ └─────────────────┘ │  └──────────────────────────────────────┼──┘   │
│  └─────────────────────┘                                         │      │
│                                                                  │      │
│  ┌───────────────────────────────────────────────────────────────▼──┐   │
│  │                    Crypto Client Layer                           │   │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐  │   │
│  │  │ CryptoClient     │  │ KeyManager       │  │ Config         │  │   │
│  │  │ - encrypt/1      │  │ - ensure_key/0   │  │ - host/port    │  │   │
│  │  │ - decrypt/1      │  │ - rotate_key/0   │  │ - timeouts     │  │   │
│  │  │ - health_check/0 │  │ - get_metadata/1 │  │ - namespace    │  │   │
│  │  └────────┬─────────┘  └────────┬─────────┘  └────────────────┘  │   │
│  │           │                     │                                 │   │
│  │  ┌────────▼─────────────────────▼─────────────────────────────┐  │   │
│  │  │              Resilience Layer (AuthPlatform.Resilience)    │  │   │
│  │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │  │   │
│  │  │  │ CircuitBreaker│  │ Retry        │  │ Timeout          │  │  │   │
│  │  │  │ (5 failures)  │  │ (3 attempts) │  │ (30s request)    │  │  │   │
│  │  │  └──────────────┘  └──────────────┘  └──────────────────┘  │  │   │
│  │  └────────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│                                    │ gRPC (mTLS)                         │
└────────────────────────────────────┼─────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Crypto Service (C++)                               │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐   │
│  │ Encrypt/Decrypt  │  │ Key Management   │  │ Health Check         │   │
│  │ (AES-256-GCM)    │  │ (Generate/Rotate)│  │                      │   │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘   │
│                                    │                                     │
│                                    ▼                                     │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    Key Storage (HSM/KMS/Local)                   │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. CryptoClient Module

The main gRPC client for communicating with crypto-service.

```elixir
defmodule MfaService.Crypto.Client do
  @moduledoc """
  gRPC client for Crypto Service operations.
  Wraps all calls with circuit breaker and retry patterns.
  """

  alias MfaService.Crypto.{Config, Telemetry}
  alias AuthPlatform.Resilience.{CircuitBreaker, Retry}

  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}
  @type encrypt_result :: {:ok, %{ciphertext: binary(), iv: binary(), tag: binary(), key_id: key_id()}}
  @type decrypt_result :: {:ok, binary()}

  @spec encrypt(binary(), key_id(), binary(), String.t()) :: encrypt_result() | {:error, term()}
  def encrypt(plaintext, key_id, aad, correlation_id)

  @spec decrypt(binary(), binary(), binary(), key_id(), binary(), String.t()) :: decrypt_result() | {:error, term()}
  def decrypt(ciphertext, iv, tag, key_id, aad, correlation_id)

  @spec health_check() :: {:ok, :serving} | {:error, term()}
  def health_check()
end
```

### 2. KeyManager Module

Manages encryption keys lifecycle with caching.

```elixir
defmodule MfaService.Crypto.KeyManager do
  @moduledoc """
  Manages MFA encryption keys in Crypto Service.
  Handles key creation, rotation, and metadata caching.
  """

  @type key_metadata :: %{
    id: key_id(),
    algorithm: atom(),
    state: atom(),
    created_at: DateTime.t(),
    expires_at: DateTime.t() | nil
  }

  @spec ensure_key_exists() :: {:ok, key_id()} | {:error, term()}
  def ensure_key_exists()

  @spec rotate_key() :: {:ok, key_id()} | {:error, term()}
  def rotate_key()

  @spec get_active_key_id() :: {:ok, key_id()} | {:error, term()}
  def get_active_key_id()

  @spec get_key_metadata(key_id()) :: {:ok, key_metadata()} | {:error, term()}
  def get_key_metadata(key_id)
end
```

### 3. Updated TOTP Generator

Modified to use crypto-service for encryption.

```elixir
defmodule MfaService.TOTP.Generator do
  @moduledoc """
  TOTP secret generation with crypto-service encryption.
  Supports both legacy local encryption and crypto-service encryption.
  """

  @version_local 0x01
  @version_crypto_service 0x02

  @spec encrypt_secret(String.t(), String.t()) :: {:ok, String.t()} | {:error, term()}
  def encrypt_secret(secret, user_id)

  @spec decrypt_secret(String.t(), String.t()) :: {:ok, String.t()} | {:error, term()}
  def decrypt_secret(encrypted, user_id)

  @spec detect_encryption_version(binary()) :: :local | :crypto_service | :unknown
  def detect_encryption_version(encrypted)
end
```

### 4. Config Module

Configuration management for crypto-service integration.

```elixir
defmodule MfaService.Crypto.Config do
  @moduledoc """
  Configuration for Crypto Service client.
  """

  @spec host() :: String.t()
  def host(), do: System.get_env("CRYPTO_SERVICE_HOST", "localhost")

  @spec port() :: non_neg_integer()
  def port(), do: String.to_integer(System.get_env("CRYPTO_SERVICE_PORT", "50051"))

  @spec connection_timeout() :: non_neg_integer()
  def connection_timeout(), do: String.to_integer(System.get_env("CRYPTO_CONNECTION_TIMEOUT", "5000"))

  @spec request_timeout() :: non_neg_integer()
  def request_timeout(), do: String.to_integer(System.get_env("CRYPTO_REQUEST_TIMEOUT", "30000"))

  @spec key_namespace() :: String.t()
  def key_namespace(), do: System.get_env("CRYPTO_KEY_NAMESPACE", "mfa")

  @spec circuit_breaker_threshold() :: non_neg_integer()
  def circuit_breaker_threshold(), do: String.to_integer(System.get_env("CRYPTO_CB_THRESHOLD", "5"))

  @spec retry_max_attempts() :: non_neg_integer()
  def retry_max_attempts(), do: String.to_integer(System.get_env("CRYPTO_RETRY_ATTEMPTS", "3"))
end
```

## Data Models

### Encrypted Secret Format

The encrypted secret format includes a version byte to distinguish encryption methods:

```
┌─────────┬──────────────────────────────────────────────────────────────┐
│ Version │ Payload                                                      │
│ (1 byte)│                                                              │
├─────────┼──────────────────────────────────────────────────────────────┤
│  0x01   │ Local: IV (12) + Tag (16) + Ciphertext (variable)           │
├─────────┼──────────────────────────────────────────────────────────────┤
│  0x02   │ Crypto-Service: KeyID (JSON) + IV (12) + Tag (16) + Cipher  │
└─────────┴──────────────────────────────────────────────────────────────┘
```

### Crypto-Service Encrypted Payload (v2)

```elixir
%{
  version: 0x02,
  key_id: %{
    namespace: "mfa:totp",
    id: "uuid-here",
    version: 1
  },
  iv: <<12 bytes>>,
  tag: <<16 bytes>>,
  ciphertext: <<variable>>
}
```

### Key Metadata Cache Entry

```elixir
%{
  key_id: %{namespace: "mfa:totp", id: "uuid", version: 1},
  algorithm: :aes_256_gcm,
  state: :active,
  created_at: ~U[2025-01-01 00:00:00Z],
  cached_at: ~U[2025-01-01 00:00:00Z],
  ttl_seconds: 300
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Encryption Round-Trip

*For any* valid TOTP secret and user_id, encrypting via Crypto_Service then decrypting with the same user_id SHALL return the original secret.

**Validates: Requirements 2.6**

### Property 2: Migration Preserves Value

*For any* existing TOTP secret encrypted with local method, migrating to crypto-service encryption SHALL preserve the original secret value when decrypted.

**Validates: Requirements 5.6**

### Property 3: Circuit Breaker Opens After Threshold

*For any* sequence of consecutive failures equal to or exceeding the configured threshold (default 5), the circuit breaker SHALL transition to open state.

**Validates: Requirements 1.3**

### Property 4: Circuit Breaker Fail-Fast When Open

*For any* request made while the circuit breaker is in open state, the client SHALL fail immediately without attempting connection to crypto-service.

**Validates: Requirements 4.3**

### Property 5: Correlation ID Propagation

*For any* request with a correlation_id, the outgoing gRPC call to crypto-service SHALL include that correlation_id in the request metadata.

**Validates: Requirements 1.5**

### Property 6: Correlation ID in Logs

*For any* log entry emitted during crypto-service operations, the correlation_id SHALL be present in the log metadata.

**Validates: Requirements 7.3**

### Property 7: Format Detection Selects Correct Decryption

*For any* encrypted secret, the decryption method selected SHALL match the encryption method indicated by the version byte (0x01 → local, 0x02 → crypto-service).

**Validates: Requirements 5.2, 5.3**

### Property 8: Key ID From Ciphertext Used for Decryption

*For any* ciphertext encrypted with crypto-service, decryption SHALL use the key_id stored within the ciphertext payload, not the current active key.

**Validates: Requirements 2.2, 3.4**

### Property 9: AAD Includes User ID

*For any* encryption operation, the Additional Authenticated Data (AAD) SHALL include the user_id to bind the ciphertext to the user.

**Validates: Requirements 2.4**

### Property 10: Stored Format Completeness

*For any* crypto-service encrypted secret, the stored format SHALL include: version byte, key_id (namespace, id, version), iv (12 bytes), tag (16 bytes), and ciphertext.

**Validates: Requirements 2.5**

### Property 11: Version Byte Presence

*For any* encrypted secret (local or crypto-service), the first byte SHALL be a valid version identifier (0x01 or 0x02).

**Validates: Requirements 5.5**

### Property 12: No Sensitive Data in Logs

*For any* log entry emitted by the MFA service during crypto operations, the log content SHALL NOT contain plaintext TOTP secrets or encryption keys.

**Validates: Requirements 6.3**

### Property 13: No Internal Details in Errors

*For any* error returned to callers, the error message SHALL NOT expose internal crypto-service implementation details, stack traces, or internal error codes.

**Validates: Requirements 6.5**

### Property 14: Response Validation

*For any* response received from crypto-service, the client SHALL validate required fields are present and correctly typed before processing.

**Validates: Requirements 6.6**

### Property 15: Telemetry for RPC Calls

*For any* RPC call to crypto-service (success or failure), a telemetry event SHALL be emitted with operation type and latency measurement.

**Validates: Requirements 7.1**

### Property 16: Key Metadata Caching

*For any* key metadata request within the cache TTL (5 minutes), the cached value SHALL be returned without calling crypto-service.

**Validates: Requirements 3.5**

### Property 17: Retry on Transient Failures

*For any* transient failure (network timeout, 5xx errors), the client SHALL retry up to the configured maximum attempts (default 3) with exponential backoff.

**Validates: Requirements 1.4**

## Error Handling

### Error Categories

| Category | Examples | Handling |
|----------|----------|----------|
| Transient | Network timeout, 503 | Retry with backoff |
| Permanent | Invalid key, auth failure | Return error immediately |
| Circuit Open | Too many failures | Fail fast, return error |

### Error Response Format

```elixir
{:error, %MfaService.Crypto.Error{
  code: :encryption_failed | :decryption_failed | :key_not_found | :service_unavailable,
  message: "User-safe error message",
  correlation_id: "uuid",
  retryable: boolean()
}}
```

### Circuit Breaker States

```
┌─────────┐     5 failures     ┌────────┐
│ CLOSED  │ ─────────────────► │  OPEN  │
└─────────┘                    └────────┘
     ▲                              │
     │                              │ 30s timeout
     │                              ▼
     │                        ┌───────────┐
     │    success             │ HALF-OPEN │
     └────────────────────────┴───────────┘
              failure → OPEN
```

## Testing Strategy

### Unit Tests

- CryptoClient: Mock gRPC calls, test request/response handling
- KeyManager: Test key lifecycle, caching behavior
- Generator: Test encryption format, version detection
- Config: Test environment variable parsing, defaults

### Property-Based Tests

Using StreamData for Elixir property-based testing:

1. **Round-trip encryption** (Property 1): Generate random secrets, encrypt/decrypt, verify equality
2. **Migration preservation** (Property 2): Generate legacy encrypted secrets, migrate, verify value
3. **Format detection** (Property 7): Generate both formats, verify correct method selection
4. **Key ID usage** (Property 8): Generate ciphertexts with various key versions, verify correct key used
5. **AAD binding** (Property 9): Generate user_ids, verify AAD contains user_id
6. **Format completeness** (Property 10): Generate encrypted secrets, verify all fields present
7. **Version byte** (Property 11): Generate encrypted secrets, verify valid version byte
8. **No sensitive data** (Property 12): Generate operations, capture logs, verify no secrets
9. **Correlation ID propagation** (Property 5, 6): Generate requests, verify correlation_id in calls and logs

### Integration Tests

- End-to-end encryption/decryption with real crypto-service
- Key rotation scenarios
- Circuit breaker behavior under load
- Migration from local to crypto-service encryption

### Test Configuration

- Property tests: Minimum 100 iterations
- Integration tests: Use test crypto-service instance
- Mocking: Use Mox for gRPC client mocking in unit tests

