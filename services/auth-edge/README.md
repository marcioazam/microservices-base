# Auth Edge Service

Ultra-low latency JWT validation and edge routing service for the Auth Platform.

[![Rust](https://img.shields.io/badge/rust-1.75%2B-orange.svg)](https://www.rust-lang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Coverage](https://img.shields.io/badge/coverage-90%25-brightgreen.svg)]()

## Overview

The Auth Edge Service is a high-performance authentication gateway built with modern Rust patterns (2025). It provides:

- **JWT Validation**: Type-state pattern ensuring compile-time safety for token validation
- **mTLS/SPIFFE**: Zero Trust workload identity with SPIFFE ID verification
- **Rate Limiting**: Adaptive rate limiting with trust-based adjustments
- **Circuit Breaker**: Generic Tower-based circuit breaker for resilience
- **Observability**: OpenTelemetry integration with W3C trace context

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Auth Edge Service                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    Tower Middleware Stack                         │   │
│  │  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌────────────────────┐  │   │
│  │  │ Tracing │→ │ Timeout │→ │ RateLimit│→ │ Circuit Breaker    │  │   │
│  │  │ Layer   │  │ Layer   │  │ Layer    │  │ Layer              │  │   │
│  │  └─────────┘  └─────────┘  └──────────┘  └────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────▼────────────────────────────────┐   │
│  │                      Core Services                                │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐  │   │
│  │  │   JWT Module    │  │  mTLS/SPIFFE    │  │   JWK Cache      │  │   │
│  │  │                 │  │                 │  │                  │  │   │
│  │  │ Token<State>    │  │ SpiffeId        │  │ Single-Flight    │  │   │
│  │  │ - Unvalidated   │  │ SpiffeValidator │  │ Atomic Updates   │  │   │
│  │  │ - SigValidated  │  │ CertVerifier    │  │ Arc<DecodingKey> │  │   │
│  │  │ - Validated     │  │                 │  │                  │  │   │
│  │  └─────────────────┘  └─────────────────┘  └──────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────▼────────────────────────────────┐   │
│  │                     gRPC Service Layer                            │   │
│  │  ValidateToken │ IntrospectToken │ GetServiceIdentity             │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
            │   Token     │ │   Session   │ │    IAM      │
            │   Service   │ │   Service   │ │   Service   │
            └─────────────┘ └─────────────┘ └─────────────┘
```

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Language | Rust | 2021 Edition |
| Runtime | Tokio | 1.35+ |
| RPC | Tonic (gRPC) | 0.10+ |
| JWT | jsonwebtoken | 9.2+ |
| Crypto | ring | 0.17+ |
| TLS | rustls | 0.21+ |
| Middleware | Tower | 0.4+ |
| Observability | OpenTelemetry | 0.21+ |
| Testing | proptest | 1.4+ |

## Key Features

### Type-State JWT Validation

Compile-time guarantees prevent using unvalidated tokens:

```rust
// Parse returns Unvalidated token
let token = Token::parse(raw_jwt)?;  // Token<Unvalidated>

// Signature validation transitions state
let token = token.validate_signature(&jwk_cache).await?;  // Token<SignatureValidated>

// Claims validation produces final state
let token = token.validate_claims(&["sub", "aud"])?;  // Token<Validated>

// Only Validated tokens expose claims
let subject = token.claims().sub;  // Compile error on other states!
```

### Generic Circuit Breaker

Tower-compatible circuit breaker with const generics:

```rust
let service = ServiceBuilder::new()
    .layer(CircuitBreakerLayer::<5, 3, 30>::new("downstream"))
    .service(inner_service);
```

### Adaptive Rate Limiting

Trust-based rate limiting with load shedding:

```rust
// Trusted clients get 2x limit
// High load (>80%) reduces limits by 50%
let decision = rate_limiter.check("client-id").await;
match decision {
    RateLimitDecision::Allowed => { /* proceed */ }
    RateLimitDecision::Denied { retry_after } => { /* return 429 */ }
}
```

### Error Handling

Non-exhaustive errors with automatic sanitization:

```rust
#[non_exhaustive]
pub enum AuthEdgeError {
    TokenExpired { expired_at: DateTime<Utc> },
    ServiceUnavailable { service: String, retry_after: Duration },
    // Internal errors are automatically sanitized
    Internal(#[from] anyhow::Error),
}
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Service bind address | `0.0.0.0` |
| `PORT` | Service port | `50052` |
| `TOKEN_SERVICE_URL` | Token service endpoint | `http://localhost:50051` |
| `SESSION_SERVICE_URL` | Session service endpoint | `http://localhost:50053` |
| `IAM_SERVICE_URL` | IAM service endpoint | `http://localhost:50054` |
| `JWKS_URL` | JWKS endpoint URL | `http://localhost:50051/.well-known/jwks.json` |
| `JWKS_CACHE_TTL` | JWK cache TTL (seconds) | `3600` |
| `CB_FAILURE_THRESHOLD` | Circuit breaker failure threshold | `5` |
| `CB_TIMEOUT` | Circuit breaker timeout (seconds) | `30` |

## Running

```bash
# Development
cargo run

# Production build
cargo build --release
./target/release/auth-edge-service

# With environment variables
HOST=0.0.0.0 PORT=50052 JWKS_URL=https://auth.example.com/.well-known/jwks.json \
  ./target/release/auth-edge-service
```

## Testing

```bash
# Run all tests
cargo test

# Run with coverage report
cargo tarpaulin --out Html --output-dir coverage

# Unit tests only
cargo test --lib

# Property-based tests (100 iterations each)
cargo test --test property_tests

# Contract tests (Pact)
cargo test --test pact_consumer_tests

# Run specific test
cargo test test_circuit_breaker_state_machine
```

### Test Coverage

The project maintains **90%+ test coverage** with:

- **Unit Tests**: Core logic validation for each module
- **Property-Based Tests**: Invariant verification with proptest
- **Contract Tests**: Pact consumer contracts for downstream services

| Module | Coverage |
|--------|----------|
| `error` | 95% |
| `jwt/claims` | 92% |
| `jwt/validator` | 90% |
| `jwt/jwk_cache` | 88% |
| `mtls/spiffe` | 94% |
| `mtls/verifier` | 85% |
| `circuit_breaker` | 92% |
| `rate_limiter` | 90% |
| `config` | 85% |

## API Reference

See [`proto/auth_edge.proto`](../proto/auth_edge.proto) for the complete gRPC service definition.

### ValidateToken

Validates a JWT and returns extracted claims.

```protobuf
rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);

message ValidateTokenRequest {
  string token = 1;
  repeated string required_claims = 2;
}

message ValidateTokenResponse {
  bool valid = 1;
  string subject = 2;
  map<string, string> claims = 3;
  string error_code = 4;
  string error_message = 5;
}
```

### IntrospectToken

RFC 7662 compliant token introspection.

```protobuf
rpc IntrospectToken(IntrospectRequest) returns (IntrospectResponse);
```

### GetServiceIdentity

Extracts SPIFFE ID from mTLS certificate.

```protobuf
rpc GetServiceIdentity(IdentityRequest) returns (IdentityResponse);
```

## Security Features

### Zero Trust Architecture

- All requests require identity verification regardless of network origin
- SPIFFE/SPIRE integration for workload identity
- mTLS for all service-to-service communication

### Token Security

- JWK rotation with automatic cache refresh
- Single-flight pattern prevents thundering herd on cache miss
- Constant-time signature verification

### Error Sanitization

- Sensitive information (passwords, keys, tokens) automatically removed from error responses
- Correlation IDs for debugging without exposing internals
- Non-exhaustive enums for forward compatibility

### Rate Limiting

- Per-client adaptive limits
- Trust-based multipliers
- Load-based reduction during high traffic

## Observability

### Tracing

```rust
// All requests include correlation ID
// W3C Trace Context propagation
// Structured span attributes
```

### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `auth_edge_requests_total` | Counter | Total requests by status |
| `auth_edge_request_duration_seconds` | Histogram | Request latency |
| `auth_edge_circuit_breaker_state` | Gauge | Circuit breaker state |
| `auth_edge_rate_limit_remaining` | Gauge | Remaining rate limit quota |
| `auth_edge_jwk_cache_hits` | Counter | JWK cache hit rate |

### Health Checks

```bash
# gRPC health check
grpcurl -plaintext localhost:50052 grpc.health.v1.Health/Check

# Readiness (includes downstream checks)
grpcurl -plaintext localhost:50052 auth.edge.AuthEdgeService/CheckHealth
```

## Development

### Prerequisites

- Rust 1.75+
- Protocol Buffers compiler (`protoc`)
- Docker (for integration tests)

### Building

```bash
# Debug build
cargo build

# Release build with optimizations
cargo build --release

# Generate protobuf code
cargo build --build-plan
```

### Code Quality

```bash
# Format code
cargo fmt

# Lint
cargo clippy -- -D warnings

# Security audit
cargo audit
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Ensure tests pass with 90%+ coverage
4. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.
