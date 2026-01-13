# Auth Edge Service

Ultra-low latency JWT validation and edge routing service for the Auth Platform.

## Features

- **JWT Validation**: Type-state pattern ensuring compile-time validation guarantees
- **SPIFFE/mTLS**: Zero Trust workload identity with certificate-based authentication
- **Distributed Caching**: JWK cache with Cache_Service integration and local fallback
- **Crypto-Service Integration**: Centralized cryptographic operations via gRPC with local fallback
- **Structured Logging**: Logging_Service integration with correlation ID propagation
- **Circuit Breaker**: rust-common CircuitBreaker for downstream service protection
- **Graceful Shutdown**: Proper cleanup of connections and in-flight requests

## Tech Stack (December 2025)

| Component | Version |
|-----------|---------|
| Rust Edition | 2024 |
| Rust Version | 1.85+ |
| Tokio | 1.42 |
| Tonic | 0.12 |
| OpenTelemetry | 0.27 |
| rustls | 0.23 |
| thiserror | 2.0 |
| proptest | 1.5 |

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | Server bind address |
| `PORT` | `50052` | Server port |
| `TOKEN_SERVICE_URL` | `http://localhost:50051` | Token service endpoint |
| `SESSION_SERVICE_URL` | `http://localhost:50053` | Session service endpoint |
| `IAM_SERVICE_URL` | `http://localhost:50054` | IAM service endpoint |
| `JWKS_URL` | `http://localhost:50051/.well-known/jwks.json` | JWKS endpoint |
| `CACHE_SERVICE_URL` | `http://localhost:50060` | Cache service endpoint |
| `LOGGING_SERVICE_URL` | `http://localhost:50061` | Logging service endpoint |
| `OTLP_ENDPOINT` | `http://localhost:4317` | OpenTelemetry collector |
| `JWKS_CACHE_TTL` | `3600` | JWK cache TTL in seconds |
| `CB_FAILURE_THRESHOLD` | `5` | Circuit breaker failure threshold |
| `CB_TIMEOUT` | `30` | Circuit breaker timeout seconds |
| `REQUEST_TIMEOUT` | `30` | Request timeout seconds |
| `SHUTDOWN_TIMEOUT` | `30` | Graceful shutdown timeout |
| `ALLOWED_SPIFFE_DOMAINS` | `` | Comma-separated SPIFFE domains |
| `CACHE_ENCRYPTION_KEY` | `` | 32-byte hex-encoded AES key (deprecated, use CRYPTO_SERVICE) |
| `CRYPTO_SERVICE_URL` | `http://localhost:50051` | Crypto service gRPC endpoint |
| `CRYPTO_KEY_NAMESPACE` | `auth-edge` | Key namespace for isolation |
| `CRYPTO_FALLBACK_ENABLED` | `true` | Enable local fallback when crypto-service unavailable |
| `CRYPTO_TIMEOUT_SECS` | `5` | Crypto service request timeout |

## Building

```bash
cargo build --release
```

## Testing

```bash
# Unit and integration tests
cargo test

# Integration tests only
cargo test --test crypto_integration_test

# Property-based tests (100 iterations per property)
cargo test property_tests

# Coverage
cargo tarpaulin --out Html
```

### Integration Tests

The `tests/integration/` directory contains integration tests for crypto module functionality:

| Test | Description |
|------|-------------|
| `test_fallback_encryption_round_trip` | Verifies local fallback encrypt/decrypt cycle |
| `test_aad_mismatch_fails` | Ensures AAD binding is enforced |
| `test_key_rotation_continuity` | Validates keys remain valid during rotation window |
| `test_dek_caching` | Tests DEK caching for fallback mode |
| `test_config_validation` | Verifies invalid configs are rejected |
| `test_error_sanitization` | Confirms key material is redacted from errors |
| `test_pending_operations_queue` | Tests operation queuing for service recovery |
| `test_metrics_recording` | Validates Prometheus metrics recording |

### Property-Based Tests

The crypto module includes property-based tests (using `proptest`) that validate correctness properties:

| Property | Description |
|----------|-------------|
| Encryption Round-Trip | `decrypt(encrypt(P, A), A) == P` for any plaintext and AAD |
| Fallback Consistency | Local fallback produces valid AES-256-GCM ciphertext |
| AAD Binding | Decryption fails if AAD doesn't match |
| No Key Material Exposure | Error messages never contain key material |
| Configuration Validation | Invalid configs are rejected before use |
| Key Rotation Continuity | Old keys remain valid during rotation window |

## Architecture

```
src/
├── config.rs          # Type-safe configuration
├── error.rs           # PlatformError integration
├── crypto/            # Crypto-service integration
│   ├── cache_integration.rs # EncryptedCacheClient wrapper
│   ├── client.rs      # CryptoClient gRPC client
│   ├── config.rs      # CryptoClientConfig
│   ├── error.rs       # CryptoError types
│   ├── fallback.rs    # FallbackHandler for degraded mode
│   ├── key_manager.rs # KeyManager for KEK/DEK lifecycle
│   ├── metrics.rs     # CryptoMetrics (Prometheus)
│   └── tests.rs       # Property-based tests
├── grpc/              # gRPC service implementation
├── jwt/               # Type-state JWT validation
│   ├── claims.rs      # Claims with has_claim
│   ├── jwk_cache.rs   # Distributed JWK cache
│   ├── token.rs       # Type-state Token<S>
│   └── validator.rs   # JwtValidator
├── middleware/        # Tower middleware stack
├── mtls/              # SPIFFE/mTLS support
├── observability/     # Telemetry and logging
│   ├── logging.rs     # AuthEdgeLogger
│   ├── metrics.rs     # Prometheus metrics
│   └── telemetry.rs   # OpenTelemetry setup
├── rate_limiter/      # Rate limiting
└── shutdown.rs        # Graceful shutdown

tests/
├── integration/
│   ├── crypto_integration_test.rs  # Crypto module integration tests
│   ├── mod.rs
│   └── validation_flow.rs
├── contract/          # Contract tests
├── property/          # Property-based tests
└── unit/              # Unit tests

proto/
└── crypto_service.proto  # Crypto-service gRPC contract
```

## Crypto-Service Integration

The auth-edge service delegates cryptographic operations to the centralized `crypto-service` via gRPC:

- **Symmetric Encryption**: AES-GCM encrypt/decrypt for cache data
- **Key Management**: Centralized KEK/DEK lifecycle with automatic rotation
- **Fallback Mode**: Local AES-256-GCM encryption when crypto-service is unavailable
- **Observability**: Prometheus metrics for latency, errors, and fallback status
- **EncryptedCacheClient**: Wrapper that transparently encrypts/decrypts cache data using CryptoClient

### EncryptedCacheClient

The `EncryptedCacheClient` wraps the standard `CacheClient` to provide transparent encryption:

```rust
let crypto = Arc::new(CryptoClient::new(crypto_config).await?);
let cache = EncryptedCacheClient::new(cache_config, crypto).await?;

// Data is automatically encrypted before storage
cache.set("key", b"secret data", Some(Duration::from_secs(3600)), correlation_id).await?;

// Data is automatically decrypted after retrieval
let data = cache.get("key", correlation_id).await?;
```

Features:
- AAD binding with `namespace:key` format for integrity
- Automatic serialization/deserialization of encrypted data
- Delegates to CryptoClient (with fallback support)

### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `crypto_client_requests_total` | Counter | Total requests by operation and status |
| `crypto_client_latency_seconds` | Histogram | Request latency by operation |
| `crypto_client_fallback_active` | Gauge | 1 if operating in fallback mode |
| `crypto_key_rotation_total` | Counter | Key rotation events |
| `crypto_client_errors_total` | Counter | Errors by operation and type |

## License

Proprietary - Auth Platform
