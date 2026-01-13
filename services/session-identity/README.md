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
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Shared Utilities                                    │    │
│  │ - Keys (Redis key generation)                       │    │
│  │ - TTL (Time-to-live calculations)                   │    │
│  │ - DateTime (ISO 8601 parsing/formatting)            │    │
│  │ - Errors (Centralized error definitions)            │    │
│  └─────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Telemetry                                           │    │
│  │ - Metrics (Prometheus metrics definitions)          │    │
│  │ - Instrumenter (Telemetry event handlers)           │    │
│  │ - Tracing (OpenTelemetry span management)           │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Shared Utilities

All common operations are centralized in `lib/session_identity_core/shared/`:

| Module | Purpose |
|--------|---------|
| `Keys` | Redis key generation with consistent namespacing |
| `TTL` | TTL calculations and expiry datetime helpers |
| `DateTime` | ISO 8601 datetime parsing and formatting |
| `Errors` | Centralized error types (OAuth, session, PKCE) |

#### Redis Key Prefixes

| Prefix | Usage |
|--------|-------|
| `session:` | Individual session data |
| `user_sessions:` | Set of session IDs per user |
| `oauth_code:` | OAuth authorization codes |
| `refresh_token:` | Refresh token data |
| `events:` | Event store entries |
| `aggregate:` | Event sourcing aggregates |
| `sequence:` | Monotonic sequence counters for event sourcing |
| `snapshot:` | Aggregate snapshots for event replay optimization |

## Tech Stack

- **Language**: Elixir 1.15+
- **Framework**: Phoenix 1.7
- **RPC**: gRPC 0.7+
- **Database**: PostgreSQL (Ecto 3.10+)
- **Cache**: Redis via Platform Cache Service (Redix fallback)
- **Crypto**: Argon2, Joken (JWT), Crypto Service (AES-256-GCM, ECDSA)
- **Resilience**: Fuse 2.5+ (circuit breaker)
- **Observability**: OpenTelemetry, Telemetry
- **Testing**: StreamData (property-based), Mox (mocking)

### Platform Integration

This service integrates with shared platform libraries:

| Dependency | Purpose |
|------------|---------|
| `auth_platform` | Security utilities, validation, common types |
| `auth_platform_clients` | Cache Service client, Logging Service client |
| `fuse` | Circuit breaker for crypto-service resilience |
| `google_protos` | Google Protocol Buffers definitions for gRPC |

The platform clients provide:
- Centralized caching via Cache Service (with circuit breaker fallback to local Redix)
- Structured logging via Logging Service
- Resilience patterns (circuit breaker, retry)

### Crypto Service Integration

The service integrates with the centralized Crypto Service for:
- JWT signing/verification (ECDSA P-256 or RSA-2048)
- Session data encryption (AES-256-GCM)
- Refresh token encryption (AES-256-GCM)
- Automated key rotation support

See `lib/session_identity_core/crypto/` for implementation details.

## Configuration

Configuration is loaded from environment variables with validation at startup. Invalid configuration causes fast failure.

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC service port | `50053` |
| `ISSUER` | OAuth/OIDC issuer identifier | `https://auth.example.com` |
| `SESSION_TTL` | Session lifetime (seconds) | `86400` |
| `CODE_TTL` | OAuth authorization code TTL (seconds) | `600` |
| `ID_TOKEN_TTL` | OIDC ID token TTL (seconds) | `3600` |
| `REFRESH_TOKEN_TTL` | Refresh token TTL (seconds) | `2592000` |
| `CAEP_ENABLED` | Enable CAEP event emission | `false` |
| `CAEP_RECEIVER_URL` | SSF receiver endpoint URL | - |
| `CACHE_ENDPOINT` | Platform Cache Service endpoint | `localhost:50052` |
| `LOGGING_ENDPOINT` | Platform Logging Service endpoint | `localhost:50051` |
| `RISK_STEP_UP_THRESHOLD` | Risk score threshold for step-up auth (0.0-1.0) | `0.7` |
| `RISK_HIGH_THRESHOLD` | Risk score threshold for high-risk factors (0.0-1.0) | `0.9` |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | - |
| `CRYPTO_SERVICE_ENDPOINT` | Crypto Service gRPC endpoint | `localhost:50051` |
| `CRYPTO_SERVICE_TIMEOUT` | Crypto operation timeout (ms) | `5000` |
| `CRYPTO_INTEGRATION_ENABLED` | Enable crypto-service integration | `true` |
| `CRYPTO_FALLBACK_ENABLED` | Enable local fallback when crypto-service unavailable | `true` |
| `CRYPTO_CACHE_TTL` | Key metadata cache TTL (seconds) | `300` |
| `CRYPTO_CB_THRESHOLD` | Circuit breaker failure threshold | `5` |
| `CRYPTO_CB_TIMEOUT` | Circuit breaker recovery timeout (ms) | `30000` |
| `CRYPTO_JWT_KEY_NAMESPACE` | Key namespace for JWT signing | `session_identity:jwt` |
| `CRYPTO_SESSION_KEY_NAMESPACE` | Key namespace for session encryption | `session_identity:session` |
| `CRYPTO_REFRESH_TOKEN_KEY_NAMESPACE` | Key namespace for refresh token encryption | `session_identity:refresh_token` |
| `CRYPTO_JWT_ALGORITHM` | JWT signing algorithm (ECDSA_P256, RSA_2048) | `ECDSA_P256` |

Configuration is validated at startup:
- TTL values must be positive integers
- Risk thresholds must be between 0.0 and 1.0

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

# Property-based tests (organized by module)
mix test test/property/oauth/pkce_property_test.exs
mix test test/property/oauth/oauth21_property_test.exs
mix test test/property/oauth/id_token_property_test.exs
mix test test/property/sessions/session_serializer_property_test.exs

# Legacy property tests
mix test test/session_property_test.exs
mix test test/risk_scorer_property_test.exs
```

### Property Test Coverage

| Module | Property Test | Validates |
|--------|---------------|-----------|
| PKCE | `pkce_property_test.exs` | S256 verification, length validation, plain rejection |
| OAuth 2.1 | `oauth21_property_test.exs` | Redirect URI matching, grant type validation |
| ID Token | `id_token_property_test.exs` | Required claims completeness, nonce handling, TTL calculation |
| Session Serializer | `session_serializer_property_test.exs` | Round-trip serialization guarantee |
| Circuit Breaker | `crypto/circuit_breaker_property_test.exs` | Fallback behavior, state transitions |
| Correlation | `crypto/correlation_property_test.exs` | Correlation ID inclusion |
| Errors | `crypto/errors_property_test.exs` | Structured error responses |
| Trace Context | `crypto/trace_context_property_test.exs` | W3C Trace Context propagation |
| JWT Signer | `crypto/jwt_signer_property_test.exs` | JWT signing round-trip |
| Key Manager | `crypto/key_manager_property_test.exs` | Key metadata caching |
| Encrypted Store | `crypto/encrypted_store_property_test.exs` | Encryption round-trip, AAD binding |
| Key Rotation | `crypto/key_rotation_property_test.exs` | Multi-version decryption |
| Feature Toggle | `crypto/feature_toggle_property_test.exs` | Toggle state consistency, branch execution |

All property tests use StreamData with minimum 100 iterations per property.

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

- OAuth 2.1 with mandatory PKCE (S256 only)
- Session binding to device/IP (device_fingerprint + ip_address required)
- Session fixation protection (ID regeneration on privilege escalation)
- MFA verification with automatic session regeneration
- Event sourcing for complete audit trail
- Risk-based adaptive authentication with step-up thresholds
- Argon2 password hashing
- Secure session serialization
- Constant-time comparison for secrets (via AuthPlatform.Security)
- 256-bit entropy session tokens (32 bytes via crypto.strong_rand_bytes)
- Centralized cryptographic operations via Crypto Service
- AES-256-GCM encryption for session data and refresh tokens
- HSM/KMS-backed key management through Crypto Service
- Automated key rotation with multi-version decryption support
- Circuit breaker protection for crypto-service failures

## Observability

### Prometheus Metrics

The service exposes Prometheus metrics via Telemetry. Metrics are defined in `lib/session_identity_core/telemetry/metrics.ex`.

| Metric | Type | Description |
|--------|------|-------------|
| `session_identity.session.created.total` | Counter | Total sessions created (tags: status) |
| `session_identity.session.deleted.total` | Counter | Total sessions deleted (tags: reason) |
| `session_identity.session.refreshed.total` | Counter | Total sessions refreshed |
| `session_identity.session.duration.seconds` | Distribution | Session duration in seconds |
| `session_identity.oauth.authorize.total` | Counter | OAuth authorization requests (tags: status) |
| `session_identity.oauth.token.total` | Counter | OAuth token requests (tags: grant_type, status) |
| `session_identity.oauth.refresh.total` | Counter | OAuth refresh token requests (tags: status) |
| `session_identity.oauth.token.duration.milliseconds` | Distribution | Token generation duration |
| `session_identity.pkce.verification.total` | Counter | PKCE verifications (tags: status) |
| `session_identity.risk.score` | Distribution | Risk score distribution |
| `session_identity.risk.step_up_required.total` | Counter | Step-up authentications required |
| `session_identity.caep.event.emitted.total` | Counter | CAEP events emitted (tags: event_type, status) |
| `session_identity.cache.hit.total` | Counter | Cache hits |
| `session_identity.cache.miss.total` | Counter | Cache misses |
| `session_identity.cache.operation.duration.milliseconds` | Distribution | Cache operation duration |
| `session_identity.events.appended.total` | Counter | Events appended (tags: event_type) |
| `session_identity.events.sequence_number` | LastValue | Current event sequence number |
| `session_identity.health.status` | LastValue | Health check status (1=healthy, 0=unhealthy) |
| `session_identity.crypto.operation.duration` | Distribution | Crypto operation duration (tags: operation, status) |
| `session_identity.crypto.circuit_breaker.count` | Counter | Circuit breaker state changes (tags: state) |
| `session_identity.crypto.fallback.count` | Counter | Fallback operations (tags: operation, reason) |
| `session_identity.crypto.reencryption.count` | Counter | Re-encryption operations (tags: namespace) |

### OpenTelemetry Tracing

W3C Trace Context propagation is supported via OpenTelemetry. The application tracer is registered at startup as `:session_identity_core`. Configure the exporter endpoint via `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Startup Sequence

The application performs the following steps at startup:

1. **Configuration Validation**: All environment variables are validated. Invalid configuration causes immediate failure (fail-fast).
2. **OpenTelemetry Registration**: The application tracer is registered for distributed tracing.
3. **Telemetry Metrics**: Prometheus metrics reporter is started.
4. **Core Infrastructure**: Database (Repo), PubSub, and Redis connections are established.
5. **Application Services**: Session Manager and other domain services are started.
6. **Web Endpoints**: Phoenix and gRPC endpoints begin accepting traffic.

If configuration validation fails, the application will not start and will log the specific validation errors.
