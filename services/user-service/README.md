# User Service

Microservice for user registration, email verification, and profile management.

## Stack

- Java 21 (with virtual threads)
- Spring Boot 3.4
- PostgreSQL
- Kafka (outbox events)
- Argon2id (password hashing)
- gRPC (platform service integration)

## Features

- User registration with email verification
- Profile management (GET/PATCH /me)
- Distributed rate limiting via cache-service (with local fallback)
- Outbox pattern for reliable event publishing
- Circuit breaker for platform service integration
- Centralized audit/security logging via logging-service

## Platform Service Integration

The service integrates with platform services via gRPC:

| Service | Proto Location | Purpose |
|---------|----------------|---------|
| cache-service | `src/main/proto/cache/v1/cache.proto` | Distributed caching, rate limiting |
| logging-service | `src/main/proto/logging/v1/logging.proto` | Audit and security event logging |

Both clients include circuit breaker protection with local fallbacks:
- Cache: Falls back to Caffeine local cache
- Logging: Falls back to structured JSON local logging

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /v1/users | Public | Register new user |
| POST | /v1/users/email/verify | Public | Verify email token |
| POST | /v1/users/email/resend | Public | Resend verification |
| GET | /v1/users/me | JWT | Get profile |
| PATCH | /v1/users/me | JWT | Update profile |

## Running Locally

```bash
# Start dependencies
docker-compose -f deploy/docker-compose.yml up -d postgres kafka

# Run service
./gradlew bootRun --args='--spring.profiles.active=dev'
```

## Testing

```bash
# Unit + property tests
./gradlew test

# Integration tests (requires Docker)
./gradlew test --tests '*IntegrationTest'
```

## Configuration

See `application.yml` for all configuration options.

Key environment variables:
- `SPRING_DATASOURCE_URL` - PostgreSQL connection
- `SPRING_KAFKA_BOOTSTRAP_SERVERS` - Kafka brokers
- `SPRING_SECURITY_OAUTH2_RESOURCESERVER_JWT_ISSUER_URI` - JWT issuer
- `GRPC_CACHE_SERVICE_HOST` - Cache service gRPC host
- `GRPC_CACHE_SERVICE_PORT` - Cache service gRPC port
- `GRPC_LOGGING_SERVICE_HOST` - Logging service gRPC host
- `GRPC_LOGGING_SERVICE_PORT` - Logging service gRPC port

## Proto Files

gRPC service definitions are located in `src/main/proto/`:

```
src/main/proto/
├── cache/v1/
│   └── cache.proto      # CacheService: Get, Set, Delete, Health
└── logging/v1/
    └── logging.proto    # LoggingService: Audit, Security events
```

Generated Java classes are placed in `com.authplatform.usersvc.proto.*` packages.

## Events Published

- `UserRegistered` - New user created
- `EmailVerificationRequested` - Verification email needed
- `UserEmailVerified` - Email confirmed
