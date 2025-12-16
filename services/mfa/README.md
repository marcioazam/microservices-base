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
│  │ - Generator         │  │ - Challenge generation      │   │
│  │ - Validator         │  │ - Authentication flow       │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ Device Fingerprint  │  │ Storage                     │   │
│  │ - Risk signals      │  │ - PostgreSQL (credentials)  │   │
│  │ - Device binding    │  │ - Redis (challenges)        │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Language**: Elixir 1.15+
- **Framework**: OTP Application
- **RPC**: gRPC
- **Database**: PostgreSQL (Ecto)
- **Cache**: Redis (Redix)
- **Testing**: StreamData (property-based)

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC service port | `50055` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `TOTP_ISSUER` | TOTP issuer name | `AuthPlatform` |

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
mix test test/webauthn_property_test.exs
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

- TOTP with configurable time window tolerance
- WebAuthn attestation verification
- Device binding for trusted devices
- Challenge expiration and replay protection
- Secure credential storage with encryption at rest
