# MFA Service

Multi-Factor Authentication service supporting TOTP, WebAuthn/FIDO2, and device fingerprinting.

## Overview

The MFA Service provides:

- **TOTP**: Time-based One-Time Password (RFC 6238)
- **WebAuthn/FIDO2**: Passwordless authentication with hardware keys
- **Device Fingerprinting**: Risk-based authentication signals
- **Challenge Management**: Secure challenge generation and validation

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       MFA Service                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ TOTP Module         │  │ WebAuthn Module             │   │
│  │ - Generator         │  │ - Challenge (centralized)   │   │
│  │ - Validator         │  │ - Authentication flow       │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ Device Fingerprint  │  │ Passkeys Module             │   │
│  │ - Risk signals      │  │ - Registration              │   │
│  │ - Device binding    │  │ - Authentication            │   │
│  └─────────────────────┘  │ - Management                │   │
│  ┌─────────────────────┐  │ - Cross-device              │   │
│  │ CAEP Emitter        │  └─────────────────────────────┘   │
│  │ - Credential events │                                    │
│  └─────────────────────┘                                    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Crypto Client (lib/mfa_service/crypto/)             │    │
│  │ - TOTP secret encryption via Crypto Service (gRPC)  │    │
│  │ - Key management with caching                       │    │
│  │ - Circuit breaker + retry resilience                │    │
│  └─────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                    Platform Integration                     │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ Cache_Service       │  │ Logging_Service             │   │
│  │ (challenges, TTL)   │  │ (structured, correlation)   │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ PostgreSQL          │  │ Crypto_Service              │   │
│  │ (credentials)       │  │ (TOTP encryption, HSM/KMS)  │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Language**: Elixir 1.17+ / OTP 27+
- **Framework**: OTP Application
- **RPC**: gRPC 0.9+
- **Database**: PostgreSQL (Ecto 3.12+)
- **Cache**: Cache_Service (via auth_platform_clients)
- **Logging**: Logging_Service (via auth_platform_clients)
- **Testing**: StreamData 1.1+ (property-based), ExCoveralls, Mox, Benchee

## Dependencies

### Platform Libraries

- `auth_platform` - Security, validation, resilience patterns, observability
- `auth_platform_clients` - gRPC clients for Cache_Service and Logging_Service

### External Dependencies

- `grpc` ~> 0.9 - gRPC server
- `protobuf` ~> 0.13 - Protocol Buffers
- `ecto_sql` ~> 3.12 - Database ORM
- `postgrex` ~> 0.19 - PostgreSQL driver
- `cbor` ~> 1.0 - WebAuthn CBOR encoding
- `req` ~> 0.5 - HTTP client for CAEP
- `jason` ~> 1.4 - JSON encoding

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC service port | `50055` |
| `CACHE_SERVICE_HOST` | Cache_Service gRPC host | `localhost` |
| `CACHE_SERVICE_PORT` | Cache_Service gRPC port | `50060` |
| `LOGGING_SERVICE_HOST` | Logging_Service gRPC host | `localhost` |
| `LOGGING_SERVICE_PORT` | Logging_Service gRPC port | `50061` |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `TOTP_ISSUER` | TOTP issuer name | `AuthPlatform` |
| `CRYPTO_SERVICE_HOST` | Crypto Service gRPC host | `localhost` |
| `CRYPTO_SERVICE_PORT` | Crypto Service gRPC port | `50051` |
| `CRYPTO_CONNECTION_TIMEOUT` | Connection timeout (ms) | `5000` |
| `CRYPTO_REQUEST_TIMEOUT` | Request timeout (ms) | `30000` |
| `CRYPTO_KEY_NAMESPACE` | Key namespace prefix | `mfa` |
| `CRYPTO_CB_THRESHOLD` | Circuit breaker failure threshold | `5` |
| `CRYPTO_RETRY_ATTEMPTS` | Max retry attempts | `3` |
| `CRYPTO_CACHE_TTL` | Key metadata cache TTL (seconds) | `300` |
| `CRYPTO_MTLS_ENABLED` | Enable mTLS for crypto-service | `true` (prod) |

## Running

```bash
# Development
mix deps.get
mix ecto.setup
mix run --no-halt

# Production
MIX_ENV=prod mix release
_build/prod/rel/mfa_service/bin/mfa_service start
```

## Testing

```bash
# All tests
mix test

# Property-based tests
mix test test/property/

# With coverage
mix coveralls

# HTML coverage report
mix coveralls.html

# Benchmarks
mix run -e "Benchee.run(...)"
```

## API

See `proto/mfa_service.proto` for the complete gRPC service definition.

### Key Endpoints

- `GenerateTOTPSecret`: Creates new TOTP secret for user
- `ValidateTOTP`: Validates TOTP code
- `StartWebAuthnRegistration`: Initiates WebAuthn registration
- `CompleteWebAuthnRegistration`: Completes credential registration
- `StartWebAuthnAuthentication`: Initiates WebAuthn authentication
- `CompleteWebAuthnAuthentication`: Validates WebAuthn assertion
- `GetDeviceFingerprint`: Returns device risk signals

## Security Features

- TOTP secrets encrypted via centralized Crypto Service (AES-256-GCM)
- HSM/KMS-backed key management through Crypto Service
- User-bound encryption with AAD (Additional Authenticated Data)
- Automatic key rotation support with backward compatibility
- WebAuthn attestation verification
- Device binding for trusted devices
- Challenge expiration and replay protection
- Secure credential storage with encryption at rest

## Crypto Service Integration

TOTP secrets are encrypted using the centralized Crypto Service via gRPC:

- **Encryption**: AES-256-GCM with user_id as AAD
- **Key Management**: Keys stored in `mfa:totp` namespace
- **Resilience**: Circuit breaker (5 failures) + retry (3 attempts)
- **Migration**: Supports lazy migration from local encryption (v1) to crypto-service (v2)
- **Observability**: Structured logging with correlation_id propagation and sensitive data sanitization

### Crypto Client Modules

| Module | Purpose |
|--------|---------|
| `Crypto.Client` | gRPC client for encrypt/decrypt operations |
| `Crypto.KeyManager` | Key lifecycle management with caching |
| `Crypto.Config` | Environment-based configuration |
| `Crypto.CircuitBreaker` | Fail-fast on service unavailability |
| `Crypto.Retry` | Exponential backoff for transient failures |
| `Crypto.Telemetry` | Prometheus metrics and telemetry events |
| `Crypto.Logger` | Structured logging with correlation_id and data sanitization |
| `Crypto.Error` | User-safe error handling |
| `Crypto.SecretFormat` | Versioned encryption format (v1: local, v2: crypto-service) |
| `Crypto.TOTPEncryptor` | High-level TOTP secret encryption/decryption |

See `.kiro/specs/mfa-crypto-service-integration/` for detailed design documentation.
