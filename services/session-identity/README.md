# Session Identity Core

Session management and OAuth 2.1 identity provider with event sourcing.

## Overview

The Session Identity Core provides:

- **Session Management**: Secure session lifecycle with Redis-backed storage
- **OAuth 2.1**: Full OAuth 2.1 implementation with PKCE
- **Event Sourcing**: Audit trail via event store
- **Risk Scoring**: Adaptive authentication based on risk signals
- **ID Tokens**: OpenID Connect ID token generation

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Session Identity Core                      │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ Sessions Module     │  │ OAuth Module                │   │
│  │ - Session Manager   │  │ - OAuth 2.1 flows           │   │
│  │ - Session Store     │  │ - PKCE support              │   │
│  │ - Serializer        │  │ - Authorization             │   │
│  └─────────────────────┘  │ - ID Token                  │   │
│                           └─────────────────────────────┘   │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ Event Store         │  │ Identity Module             │   │
│  │ - Aggregate         │  │ - Risk Scorer               │   │
│  │ - Events            │  │                             │   │
│  │ - Store             │  │                             │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Language**: Elixir 1.15+
- **Framework**: Phoenix 1.7
- **RPC**: gRPC
- **Database**: PostgreSQL (Ecto)
- **Cache**: Redis (Redix)
- **Crypto**: Argon2, Joken (JWT)
- **Testing**: StreamData (property-based)

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC service port | `50053` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `SESSION_TTL` | Session lifetime (seconds) | `86400` |

## Running

```bash
# Development
mix deps.get
mix ecto.setup
mix phx.server

# Production
MIX_ENV=prod mix release
_build/prod/rel/session_identity_core/bin/session_identity_core start
```

## Testing

```bash
# All tests
mix test

# Property-based tests
mix test test/session_property_test.exs
mix test test/risk_scorer_property_test.exs
```

## API

See `proto/session_identity.proto` for the complete gRPC service definition.

### Key Endpoints

- `CreateSession`: Creates new authenticated session
- `GetSession`: Retrieves session by ID
- `RefreshSession`: Extends session lifetime
- `RevokeSession`: Terminates session
- `Authorize`: OAuth 2.1 authorization endpoint
- `Token`: OAuth 2.1 token endpoint
- `GetRiskScore`: Returns risk assessment for request

## Security Features

- OAuth 2.1 with mandatory PKCE
- Session binding to device/IP
- Event sourcing for complete audit trail
- Risk-based adaptive authentication
- Argon2 password hashing
- Secure session serialization
