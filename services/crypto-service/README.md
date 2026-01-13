# Crypto Security Service

High-performance C++ microservice providing centralized cryptographic operations for the auth-platform ecosystem.

## Overview

The Crypto Security Service implements:
- **AES-256-GCM** symmetric encryption with AAD support
- **RSA-OAEP** asymmetric encryption (2048, 3072, 4096 bits)
- **Digital signatures** (RSA-PSS, ECDSA P-256/384/521)
- **Key management** with HSM/KMS integration
- **Streaming file encryption** for large files (up to 10GB)
- **Encrypted audit logs** for regulatory compliance

## Architecture (2025 Modernization)

```
┌─────────────────────────────────────────────────────────────┐
│                    crypto-service                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │ LoggingClient   │  │ CacheClient     │                   │
│  │ (gRPC to        │  │ (gRPC to        │                   │
│  │  logging-svc)   │  │  cache-svc)     │                   │
│  └────────┬────────┘  └────────┬────────┘                   │
│           │                    │                             │
│  ┌────────▼────────────────────▼────────────────────────┐   │
│  │              Core Services Layer                      │   │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │   │
│  │  │ Encryption   │ │  Signature   │ │  Key         │  │   │
│  │  │ Service      │ │  Service     │  │  Service     │  │   │
│  │  └──────────────┘ └──────────────┘ └──────────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Crypto Engines (Centralized)             │   │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────────────┐ │   │
│  │  │  AES   │ │  RSA   │ │ ECDSA  │ │ Hybrid         │ │   │
│  │  └────────┘ └────────┘ └────────┘ └────────────────┘ │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Common Layer (Centralized)               │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │   │
│  │  │ Result   │ │ OpenSSL  │ │ Config   │ │ Metrics  │ │   │
│  │  │ <T,E>    │ │ RAII     │ │ Loader   │ │ Exporter │ │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
┌──────────────────────┐    ┌──────────────────────┐
│   logging-service    │    │    cache-service     │
│   (gRPC :5001)       │    │    (gRPC :50051)     │
└──────────────────────┘    └──────────────────────┘
           │                              │
           └──────────────┬───────────────┘
                          ▼
              ┌──────────────────────┐
              │   Linkerd Proxy      │
              │   (Service Mesh)     │
              │   - Circuit Breaker  │
              │   - Retry            │
              │   - mTLS             │
              └──────────────────────┘
```

### Key Changes (2025)

- **Platform Integration**: Uses `LoggingClient` and `CacheClient` for centralized services
- **Service Mesh Resilience**: Circuit breaker, retry, timeout via Linkerd (no code changes)
- **C++23**: Uses `std::expected<T, Error>` for modern error handling
- **OpenSSL 3.3+**: Modern RAII wrappers, FIPS mode support
- **Observability**: W3C Trace Context, correlation_id in all logs/traces

## API

### gRPC (Port 50051)

Protocol Buffer definitions in `proto/crypto_service.proto`:

| RPC | Description |
|-----|-------------|
| `Encrypt` | AES-GCM symmetric encryption |
| `Decrypt` | AES-GCM symmetric decryption |
| `RSAEncrypt` | RSA-OAEP asymmetric encryption |
| `RSADecrypt` | RSA-OAEP asymmetric decryption |
| `Sign` | RSA-PSS or ECDSA signature |
| `Verify` | Signature verification |
| `GenerateKey` | Generate AES/RSA/ECDSA keys |
| `RotateKey` | Rotate existing key |
| `GetKeyMetadata` | Retrieve key metadata |
| `DeleteKey` | Delete key |
| `EncryptFile` | Streaming file encryption |
| `DecryptFile` | Streaming file decryption |
| `HealthCheck` | Service health status |

### REST (Port 8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/encrypt` | AES encryption |
| POST | `/v1/decrypt` | AES decryption |
| POST | `/v1/rsa/encrypt` | RSA encryption |
| POST | `/v1/rsa/decrypt` | RSA decryption |
| POST | `/v1/sign` | Create signature |
| POST | `/v1/verify` | Verify signature |
| POST | `/v1/keys` | Generate key |
| POST | `/v1/keys/{id}/rotate` | Rotate key |
| GET | `/v1/keys/{id}` | Get key metadata |
| DELETE | `/v1/keys/{id}` | Delete key |
| GET | `/health` | Liveness check (Kubernetes/Linkerd) |
| GET | `/ready` | Readiness check (Kubernetes) |
| GET | `/metrics` | Prometheus metrics |

## Building

### Prerequisites

- CMake 3.28+
- C++23 compiler (GCC 13+, Clang 17+)
- OpenSSL 3.3+
- gRPC 1.60+
- Protobuf 25+

### Build Commands

```bash
# Configure
cmake -B build -DCMAKE_BUILD_TYPE=Release

# Build
cmake --build build -j$(nproc)

# Run tests
cd build && ctest --output-on-failure

# Build with sanitizers (debug)
cmake -B build-debug -DCMAKE_BUILD_TYPE=Debug -DENABLE_SANITIZERS=ON
cmake --build build-debug

# Build with FIPS mode
cmake -B build -DCMAKE_BUILD_TYPE=Release -DENABLE_FIPS=ON
```

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | 50051 |
| `REST_PORT` | REST server port | 8080 |
| `TLS_CERT_PATH` | TLS certificate path | - |
| `TLS_KEY_PATH` | TLS private key path | - |
| `KMS_PROVIDER` | Key storage backend | local |
| `HSM_SLOT_ID` | HSM slot identifier | - |
| `AWS_KMS_KEY_ARN` | AWS KMS key ARN | - |
| `KEY_CACHE_TTL` | Key cache TTL (seconds) | 300 |
| `LOGGING_SERVICE_ADDRESS` | Logging service gRPC address | localhost:5001 |
| `CACHE_SERVICE_ADDRESS` | Cache service gRPC address | localhost:50051 |
| `CACHE_NAMESPACE` | Cache key namespace prefix | crypto |

## Platform Service Dependencies

The crypto-service integrates with platform services:

### Logging Service
- Centralized structured logging via gRPC
- Batch buffering with configurable flush interval
- Local console fallback when unavailable

### Cache Service  
- Centralized key caching via gRPC
- Namespace-prefixed keys (`crypto:key:...`)
- Local LRU cache fallback when unavailable

### Service Mesh (Linkerd)
- Circuit breaker: 5 consecutive failures
- Retry: 3 attempts for 5xx/429 errors
- Timeout: 30s request timeout
- mTLS: Automatic mutual TLS

## Security

- All keys encrypted at rest with master KEK
- Secure memory allocation (mlock) for key material
- Constant-time comparison for signatures/tags
- TLS 1.3 required for all API communications
- JWT authentication with RBAC authorization
- Input size validation (DoS prevention)
- Safe error messages (no sensitive data leakage)
- FIPS 140-2 mode support (OpenSSL 3.3+)

## Testing

```bash
# Unit tests
./build/crypto_tests --gtest_filter="*Unit*"

# Property-based tests (100+ iterations)
./build/crypto_tests --gtest_filter="*Properties*"

# Run with sanitizers
./build-debug/crypto_tests
```

## Metrics

Prometheus metrics exposed at `/metrics`:

- `crypto_encrypt_operations_total` - Encryption counter
- `crypto_decrypt_operations_total` - Decryption counter
- `crypto_sign_operations_total` - Signing counter
- `crypto_verify_operations_total` - Verification counter
- `crypto_operation_latency_seconds` - Latency histogram (p50/p95/p99)
- `crypto_errors_total{error_code="..."}` - Error counter by error code
- `crypto_logging_service_connected` - Logging service connection status
- `crypto_cache_service_connected` - Cache service connection status

## Kubernetes Deployment

```yaml
# ResiliencePolicy for Service Mesh
apiVersion: resilience.auth-platform.github.com/v1
kind: ResiliencePolicy
metadata:
  name: crypto-service-resilience
spec:
  targetRef:
    name: crypto-service
  circuitBreaker:
    enabled: true
    failureThreshold: 5
  retry:
    enabled: true
    maxAttempts: 3
  timeout:
    enabled: true
    requestTimeout: "30s"
```

## Documentation

- [Design Document](.kiro/specs/crypto-service-modernization-2025/design.md)
- [Requirements](.kiro/specs/crypto-service-modernization-2025/requirements.md)
- [Implementation Tasks](.kiro/specs/crypto-service-modernization-2025/tasks.md)
