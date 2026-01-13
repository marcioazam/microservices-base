# Token Service

JWT signing and token management service with DPoP support and secure key management.

## Overview

The Token Service is responsible for:

- **JWT Generation**: Creating signed access and refresh tokens
- **DPoP Support**: Demonstrating Proof of Possession tokens (RFC 9449)
- **Key Management**: Integration with AWS KMS for secure signing
- **Token Rotation**: Secure refresh token rotation with family tracking
- **JWKS Publishing**: Exposing public keys for token verification

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Token Service                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ JWT Module  │  │ DPoP Module │  │ Refresh Module      │  │
│  │ - Builder   │  │ - Proof     │  │ - Family tracking   │  │
│  │ - Claims    │  │ - Thumbprint│  │ - Generator         │  │
│  │ - Signer    │  │ - Validator │  │ - Rotator           │  │
│  │ - Serializer│  │             │  │                     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ KMS Module  │  │ JWKS Module │  │ Storage             │  │
│  │ - AWS KMS   │  │ - Publisher │  │ - CacheClient       │  │
│  │ - Crypto Svc│  │             │  │ - Encrypted         │  │
│  │ - Mock      │  │             │  │                     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    Crypto Module                         ││
│  │  - CryptoClient (gRPC)  - CryptoSigner  - CryptoEncryptor││
│  │  - Circuit Breaker      - Fallback      - Metrics        ││
│  └─────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────┤
│                   Platform Integration                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ CacheClient │  │LoggingClient│  │ CircuitBreaker      │  │
│  │ → Cache Svc │  │ → Log Svc   │  │ → KMS resilience    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Language**: Rust (Edition 2024, rust-version 1.85+)
- **Runtime**: Tokio async runtime
- **RPC**: gRPC via Tonic 0.12 / Prost 0.13
- **Crypto**: jsonwebtoken 9.3, sha2, subtle (constant-time)
- **Platform**: rust-common (CacheClient, LoggingClient, CircuitBreaker)
- **Observability**: tracing, OpenTelemetry, Prometheus

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Service bind address | `0.0.0.0` |
| `PORT` | Service port | `50051` |
| `JWT_ISSUER` | JWT issuer claim | `auth-platform` |
| `JWT_ALGORITHM` | Signing algorithm (RS256, PS256, ES256) | `RS256` |
| `ACCESS_TOKEN_TTL` | Access token lifetime (seconds) | `900` |
| `REFRESH_TOKEN_TTL` | Refresh token lifetime (seconds) | `604800` |
| `KMS_PROVIDER` | KMS provider (`aws` or `mock`) | `mock` |
| `KMS_KEY_ID` | AWS KMS key ID for signing | `default-key` |
| `CRYPTO_SERVICE_ADDRESS` | Crypto Service gRPC address | `http://localhost:50051` |
| `CRYPTO_SIGNING_ENABLED` | Enable signing via Crypto Service | `true` |
| `CRYPTO_ENCRYPTION_ENABLED` | Enable encryption via Crypto Service | `true` |
| `CRYPTO_FALLBACK_ENABLED` | Enable fallback to local operations | `true` |
| `CRYPTO_KEY_NAMESPACE` | Key namespace for isolation | `token` |
| `CRYPTO_RATE_LIMIT` | Rate limit for Crypto Service (req/s) | `1000` |
| `CACHE_SERVICE_ADDRESS` | Cache service gRPC address | `http://localhost:50051` |
| `LOGGING_SERVICE_ADDRESS` | Logging service gRPC address | `http://localhost:5001` |
| `ENCRYPTION_KEY` | Base64-encoded 32-byte AES key for cache encryption | (auto-generated) |
| `DPOP_CLOCK_SKEW` | DPoP clock skew tolerance (seconds) | `60` |
| `DPOP_JTI_TTL` | DPoP JTI cache TTL (seconds) | `300` |
| `JWKS_KEY_RETENTION` | Previous key retention period after rotation (seconds) | `86400` |

## Building

The service uses `tonic-build` to compile protobuf definitions at build time.

### Prerequisites

- **protoc**: Protocol Buffers compiler must be installed on your system
  - macOS: `brew install protobuf`
  - Ubuntu/Debian: `apt install protobuf-compiler`
  - Windows: Download from [protobuf releases](https://github.com/protocolbuffers/protobuf/releases)

```bash
# Build (compiles protos automatically via build.rs)
cargo build

# Production build
cargo build --release
./target/release/token-service
```

Proto files are located at `api/proto/auth/token_service.proto` and compiled during the build process.

## Running

```bash
# Development
cargo run

# Production
./target/release/token-service
```

## Testing

```bash
# Unit tests
cargo test

# Property-based tests (JWT)
cargo test --test jwt_property_tests

# Property-based tests (DPoP)
cargo test --test dpop_property_tests

# Property-based tests (Cache)
cargo test --test cache_property_tests

# Property-based tests (Error handling)
cargo test --test error_property_tests

# Property-based tests (Observability)
cargo test --test observability_property_tests

# Property-based tests (Crypto Service integration)
cargo test --test crypto_property_tests

# Advanced property tests (Crypto Service)
cargo test --test crypto_advanced_property_tests

# Integration tests (Crypto Service)
cargo test --test crypto_integration_tests

# All property tests
cargo test --test property_tests
```

## API

See `api/proto/auth/token_service.proto` for the complete gRPC service definition.

### Key Endpoints

- `GenerateTokens`: Creates access and refresh token pair
- `RefreshTokens`: Rotates refresh token and issues new access token
- `RevokeToken`: Revokes a token family
- `GetJWKS`: Returns public keys for verification
- `ValidateDPoP`: Validates DPoP proof

## Metrics

The service exposes Prometheus metrics for observability:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `token_service_tokens_issued_total` | Counter | `token_type`, `algorithm` | Total tokens issued |
| `token_service_tokens_refreshed_total` | Counter | `status` | Total token refresh operations |
| `token_service_tokens_revoked_total` | Counter | `reason` | Total token revocations |
| `token_service_dpop_validations_total` | Counter | `status`, `error_type` | DPoP validation attempts |
| `token_service_grpc_latency_seconds` | Histogram | `method` | gRPC method latency |
| `token_service_kms_operations_total` | Counter | `operation`, `status` | KMS signing operations |
| `token_service_crypto_operations_total` | Counter | `operation`, `status` | Crypto Service operations |
| `token_service_crypto_latency_seconds` | Histogram | `operation` | Crypto Service latency (p50/p95/p99) |
| `token_service_crypto_fallback_total` | Counter | `operation` | Fallback activations |
| `token_service_crypto_circuit_breaker_open` | Gauge | - | Circuit breaker state (1=open) |
| `token_service_cache_operations_total` | Counter | `operation`, `status` | Cache read/write operations |
| `token_service_security_events_total` | Counter | `event_type` | Security events (replay attacks, revocations) |

## Security Features

- DPoP (RFC 9449) for sender-constrained tokens
- AWS KMS integration for HSM-backed signing
- **Crypto Service integration** for centralized key management with HSM support
- Refresh token rotation with family tracking
- Token revocation with immediate propagation
- Secure token serialization
- Correlation ID tracking for audit trails
- AES-256-GCM encrypted cache storage via rust-common::CacheClient
- Circuit breaker resilience for KMS and Crypto Service operations
- Automatic fallback to local operations when Crypto Service is unavailable
- Rate limiting for Crypto Service requests
- JWKS key rotation with configurable retention period (RFC 7517) for graceful key transitions

## Crypto Service Integration

The Token Service can optionally use the centralized Crypto Service for cryptographic operations:

### Signing Providers

| Provider | Use Case | Configuration |
|----------|----------|---------------|
| AWS KMS | Production with AWS | `KMS_PROVIDER=aws` |
| Crypto Service | Centralized HSM-backed signing | `CRYPTO_SIGNING_ENABLED=true` |
| Mock | Development/testing | `KMS_PROVIDER=mock` |

### Creating a CryptoSigner

```rust
use token_service::kms::KmsFactory;
use token_service::crypto::{CryptoClientFactory, CryptoClientConfig};

// Create CryptoClient
let config = CryptoClientConfig::from_env();
let client = CryptoClientFactory::create(config, signing_key, encryption_key).await?;

// Create CryptoSigner via KmsFactory
let signer = KmsFactory::create_crypto_signer(
    client,
    "token",      // namespace
    "signing-key", // key name
    1,            // version
    "PS256",      // algorithm
);
```

### Fallback Behavior

When `CRYPTO_FALLBACK_ENABLED=true`, the service automatically falls back to local operations if the Crypto Service is unavailable:

- **Signing**: Falls back to local HMAC-SHA256
- **Encryption**: Falls back to local AES-256-GCM
- **Circuit Breaker**: Prevents cascading failures with configurable thresholds
