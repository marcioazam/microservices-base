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
│  │ KMS Module  │  │ JWKS Module │  │ Storage (Redis)     │  │
│  │ - AWS KMS   │  │ - Publisher │  │                     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Language**: Rust (Edition 2021)
- **Runtime**: Tokio async runtime
- **RPC**: gRPC via Tonic
- **Crypto**: ring, jsonwebtoken
- **Storage**: Redis (cluster support)
- **Observability**: tracing, prometheus

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Service bind address | `0.0.0.0` |
| `PORT` | Service port | `50052` |
| `REDIS_URL` | Redis connection URL | - |
| `KMS_KEY_ID` | AWS KMS key ID for signing | - |
| `ACCESS_TOKEN_TTL` | Access token lifetime (seconds) | `900` |
| `REFRESH_TOKEN_TTL` | Refresh token lifetime (seconds) | `604800` |

## Running

```bash
# Development
cargo run

# Production
cargo build --release
./target/release/token-service
```

## Testing

```bash
# Unit tests
cargo test

# Property-based tests (DPoP)
cargo test --test dpop_property_tests

# All property tests
cargo test --test property_tests
```

## API

See `proto/token_service.proto` for the complete gRPC service definition.

### Key Endpoints

- `GenerateTokens`: Creates access and refresh token pair
- `RefreshTokens`: Rotates refresh token and issues new access token
- `RevokeToken`: Revokes a token family
- `GetJWKS`: Returns public keys for verification
- `ValidateDPoP`: Validates DPoP proof

## Security Features

- DPoP (RFC 9449) for sender-constrained tokens
- AWS KMS integration for HSM-backed signing
- Refresh token rotation with family tracking
- Token revocation with immediate propagation
- Secure token serialization
