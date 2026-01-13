# Design Document: User Service

## Overview

The User Service is a Java/Spring Boot microservice responsible for user registration, profile management, and email verification. It follows the Auth Platform patterns for security, observability, and resilience while integrating with existing platform services (Crypto_Service, Cache_Service, Logging_Service).

The service implements the Outbox pattern for reliable event publishing, ensuring downstream services (like notification service for emails) receive all events even during failures.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           User Service                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────┐  │
│  │ API Layer           │  │ Domain Layer        │  │ Infrastructure  │  │
│  │ - UserController    │  │ - User              │  │ - UserRepo      │  │
│  │ - EmailController   │  │ - EmailToken        │  │ - TokenRepo     │  │
│  │ - MeController      │  │ - OutboxEvent       │  │ - OutboxRepo    │  │
│  │ - DTOs              │  │                     │  │ - CacheClient   │  │
│  └─────────────────────┘  └─────────────────────┘  └─────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │ Service Layer                                                        ││
│  │ - UserRegistrationService  - EmailVerificationService                ││
│  │ - ProfileService           - OutboxDispatcher                        ││
│  │ - PasswordService          - RateLimitService                        ││
│  └─────────────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │ Cross-Cutting                                                        ││
│  │ - SecurityConfig  - CircuitBreaker  - Metrics  - Logging             ││
│  └─────────────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────────────┤
│                        Platform Integration                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │
│  │ Crypto_Service  │  │ Cache_Service   │  │ Logging_Service         │  │
│  │ (gRPC client)   │  │ (gRPC client)   │  │ (gRPC client)           │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
         │                      │                      │
         ▼                      ▼                      ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐
│   PostgreSQL    │  │     Kafka       │  │   Platform Services         │
│   (users, etc)  │  │  (events out)   │  │   (crypto, cache, log)      │
└─────────────────┘  └─────────────────┘  └─────────────────────────────┘
```

## Components and Interfaces

### API Layer

```java
// UserController.java
@RestController
@RequestMapping("/v1/users")
public class UserController {
    
    @PostMapping
    public ResponseEntity<UserRegistrationResponse> register(
        @Valid @RequestBody UserRegistrationRequest request,
        @RequestHeader(value = "X-Correlation-ID", required = false) String correlationId
    );
}

// EmailVerificationController.java
@RestController
@RequestMapping("/v1/users/email")
public class EmailVerificationController {
    
    @PostMapping("/verify")
    public ResponseEntity<Void> verify(
        @Valid @RequestBody EmailVerificationRequest request
    );
    
    @PostMapping("/resend")
    public ResponseEntity<Void> resend(
        @Valid @RequestBody EmailResendRequest request
    );
}

// MeController.java
@RestController
@RequestMapping("/v1/users/me")
public class MeController {
    
    @GetMapping
    public ResponseEntity<UserProfileResponse> getProfile(
        @AuthenticationPrincipal JwtAuthenticationToken token
    );
    
    @PatchMapping
    public ResponseEntity<UserProfileResponse> updateProfile(
        @AuthenticationPrincipal JwtAuthenticationToken token,
        @Valid @RequestBody UserProfileUpdateRequest request
    );
}
```

### Service Layer Interfaces

```java
// UserRegistrationService.java
public interface UserRegistrationService {
    UserRegistrationResult register(UserRegistrationCommand command);
}

// EmailVerificationService.java
public interface EmailVerificationService {
    void verify(String token);
    void resend(String email, String ipAddress);
}

// ProfileService.java
public interface ProfileService {
    UserProfile getProfile(UUID userId);
    UserProfile updateProfile(UUID userId, ProfileUpdateCommand command);
}

// PasswordService.java
public interface PasswordService {
    String hash(String plainPassword);
    boolean verify(String plainPassword, String hash);
}

// RateLimitService.java
public interface RateLimitService {
    boolean isAllowed(String key, int maxRequests, Duration window);
    void recordRequest(String key, Duration window);
}
```

### Platform Client Interfaces

```java
// CryptoServiceClient.java
public interface CryptoServiceClient {
    byte[] hash(byte[] data, String algorithm);
    byte[] encrypt(byte[] plaintext, String keyNamespace);
    byte[] decrypt(byte[] ciphertext, String keyNamespace);
    boolean isAvailable();
}

// CacheServiceClient.java
public interface CacheServiceClient {
    Optional<String> get(String key);
    void set(String key, String value, Duration ttl);
    long increment(String key, Duration ttl);
    boolean isAvailable();
}

// LoggingServiceClient.java
public interface LoggingServiceClient {
    void log(LogEntry entry);
    boolean isAvailable();
}
```

## Data Models

### Database Schema

```sql
-- V1__create_users_table.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING_EMAIL',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_users_email UNIQUE (email),
    CONSTRAINT chk_users_status CHECK (status IN ('PENDING_EMAIL', 'ACTIVE', 'DISABLED'))
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- V2__create_email_verification_tokens_table.sql
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT uk_tokens_hash UNIQUE (token_hash)
);

CREATE INDEX idx_tokens_user_id ON email_verification_tokens(user_id);
CREATE INDEX idx_tokens_expires_at ON email_verification_tokens(expires_at);

-- V3__create_outbox_events_table.sql
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE INDEX idx_outbox_unprocessed ON outbox_events(created_at) 
    WHERE processed_at IS NULL;
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_type, aggregate_id);
```

### JPA Entities

```java
// User.java
@Entity
@Table(name = "users")
public class User {
    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;
    
    @Column(nullable = false, unique = true)
    private String email;
    
    @Column(name = "email_verified", nullable = false)
    private boolean emailVerified;
    
    @Column(name = "password_hash", nullable = false)
    private String passwordHash;
    
    @Column(name = "display_name", nullable = false, length = 100)
    private String displayName;
    
    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 20)
    private UserStatus status;
    
    @Column(name = "created_at", nullable = false)
    private Instant createdAt;
    
    @Column(name = "updated_at", nullable = false)
    private Instant updatedAt;
}

// EmailVerificationToken.java
@Entity
@Table(name = "email_verification_tokens")
public class EmailVerificationToken {
    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;
    
    @Column(name = "user_id", nullable = false)
    private UUID userId;
    
    @Column(name = "token_hash", nullable = false, length = 64)
    private String tokenHash;
    
    @Column(name = "expires_at", nullable = false)
    private Instant expiresAt;
    
    @Column(name = "used_at")
    private Instant usedAt;
    
    @Column(name = "created_at", nullable = false)
    private Instant createdAt;
    
    @Column(name = "attempt_count", nullable = false)
    private int attemptCount;
}

// OutboxEvent.java
@Entity
@Table(name = "outbox_events")
public class OutboxEvent {
    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;
    
    @Column(name = "aggregate_type", nullable = false, length = 50)
    private String aggregateType;
    
    @Column(name = "aggregate_id", nullable = false)
    private UUID aggregateId;
    
    @Column(name = "event_type", nullable = false, length = 50)
    private String eventType;
    
    @Column(name = "payload_json", nullable = false, columnDefinition = "jsonb")
    private String payloadJson;
    
    @Column(name = "created_at", nullable = false)
    private Instant createdAt;
    
    @Column(name = "processed_at")
    private Instant processedAt;
    
    @Column(name = "retry_count", nullable = false)
    private int retryCount;
    
    @Column(name = "last_error")
    private String lastError;
}
```

### DTOs

```java
// Request DTOs
public record UserRegistrationRequest(
    @NotBlank @Email String email,
    @NotBlank @Size(min = 8, max = 128) String password,
    @NotBlank @Size(min = 1, max = 100) String displayName
) {}

public record EmailVerificationRequest(
    @NotBlank String token
) {}

public record EmailResendRequest(
    @NotBlank @Email String email
) {}

public record UserProfileUpdateRequest(
    @Size(min = 1, max = 100) String displayName
) {}

// Response DTOs
public record UserRegistrationResponse(
    UUID userId,
    String email,
    String status
) {}

public record UserProfileResponse(
    UUID userId,
    String email,
    boolean emailVerified,
    String displayName,
    Instant createdAt
) {}

// Event Payloads
public record UserRegisteredEvent(
    UUID userId,
    String email,
    String displayName,
    Instant registeredAt
) {}

public record EmailVerificationRequestedEvent(
    UUID userId,
    String email,
    String verificationLink,
    String templateId,
    String locale
) {}

public record UserEmailVerifiedEvent(
    UUID userId,
    String email,
    Instant verifiedAt
) {}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Email Normalization Idempotence

*For any* email string, normalizing it should produce a lowercase, trimmed result, and normalizing the result again should produce the same value (idempotent).

**Validates: Requirements 1.2, 3.1**

### Property 2: Password Hashing Round-Trip

*For any* valid password string, hashing it with Argon2id and then verifying the original password against the hash should return true.

**Validates: Requirements 1.5, 6.1**

### Property 3: Token Hash Consistency

*For any* verification token, computing its SHA-256 hash should always produce the same 64-character hexadecimal result.

**Validates: Requirements 1.8, 2.1**

### Property 4: User Registration Initial State

*For any* valid registration request, the created User entity should have status=PENDING_EMAIL, emailVerified=false, and a valid Argon2id password hash.

**Validates: Requirements 1.6, 1.7**

### Property 5: Email Verification State Transition

*For any* valid email verification, the User should transition to emailVerified=true and status=ACTIVE, and the token should be marked as used.

**Validates: Requirements 2.6, 2.7**

### Property 6: Outbox Event Completeness

*For any* domain event (UserRegistered, EmailVerificationRequested, UserEmailVerified), the corresponding OutboxEvent should contain non-null aggregateType, aggregateId, eventType, payloadJson, and createdAt.

**Validates: Requirements 1.9, 2.8, 7.1, 7.2**

### Property 7: Rate Limiting Enforcement

*For any* sequence of requests exceeding the configured rate limit, subsequent requests should be rejected with 429 status and Retry-After header.

**Validates: Requirements 3.2, 9.1, 9.2, 9.3, 9.4**

### Property 8: Sensitive Data Non-Exposure

*For any* API response or log entry, the content should never contain password values, raw tokens, or password hashes.

**Validates: Requirements 4.5, 6.3, 10.5**

### Property 9: Input Validation Rejection

*For any* invalid input (malformed email, weak password, oversized displayName, injection attempts), the service should reject with 400 status and field-specific errors.

**Validates: Requirements 1.1, 5.1, 8.1, 8.2, 8.3, 8.4**

### Property 10: Profile Update Field Restriction

*For any* profile update request, only the allowed fields (displayName) should be modified, and updatedAt should be set to a timestamp >= the previous value.

**Validates: Requirements 5.2, 5.3**

### Property 11: Resend Anti-Enumeration

*For any* email resend request (whether user exists or not), the response should always be 202 Accepted.

**Validates: Requirements 3.5**

### Property 12: Circuit Breaker State Transitions

*For any* sequence of external service failures exceeding the threshold, the circuit breaker should open and subsequent calls should fail fast; after recovery timeout, it should transition to half-open.

**Validates: Requirements 12.4, 12.5**



## Error Handling

### Error Response Format

All errors follow RFC 7807 Problem Details format:

```java
public record ProblemDetail(
    String type,
    String title,
    int status,
    String detail,
    String instance,
    Map<String, Object> extensions
) {}
```

### Error Categories

| Category | HTTP Status | Error Code | Description |
|----------|-------------|------------|-------------|
| Validation | 400 | `VALIDATION_ERROR` | Invalid input fields |
| Conflict | 409 | `EMAIL_ALREADY_EXISTS` | Email already registered |
| Not Found | 404 | `USER_NOT_FOUND` | User does not exist |
| Unauthorized | 401 | `INVALID_TOKEN` | JWT validation failed |
| Rate Limited | 429 | `RATE_LIMIT_EXCEEDED` | Too many requests |
| Token Expired | 400 | `TOKEN_EXPIRED` | Verification token expired |
| Token Invalid | 400 | `TOKEN_INVALID` | Verification token invalid/used |
| Internal | 500 | `INTERNAL_ERROR` | Unexpected server error |

### Security Considerations

- Generic error messages for authentication failures (no account enumeration)
- No stack traces in production responses
- Correlation IDs for internal debugging
- Rate limiting on error-prone endpoints

## Testing Strategy

### Dual Testing Approach

The service uses both unit tests and property-based tests for comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs

### Property-Based Testing Configuration

- **Framework**: jqwik (Java property-based testing library)
- **Minimum iterations**: 100 per property test
- **Tag format**: `@Tag("Feature: user-service, Property N: {property_text}")`

### Test Categories

| Category | Framework | Purpose |
|----------|-----------|---------|
| Unit | JUnit 5 | Service logic, validation, edge cases |
| Property | jqwik | Universal properties (100+ iterations) |
| Integration | Testcontainers | PostgreSQL, Kafka, gRPC clients |
| Contract | Spring Cloud Contract | API contract verification |

### Test Structure

```
src/test/java/com/authplatform/usersvc/
├── unit/
│   ├── service/
│   │   ├── UserRegistrationServiceTest.java
│   │   ├── EmailVerificationServiceTest.java
│   │   ├── ProfileServiceTest.java
│   │   └── PasswordServiceTest.java
│   ├── validation/
│   │   ├── EmailValidatorTest.java
│   │   └── PasswordValidatorTest.java
│   └── util/
│       └── TokenHasherTest.java
├── property/
│   ├── EmailNormalizationPropertyTest.java
│   ├── PasswordHashingPropertyTest.java
│   ├── TokenHashPropertyTest.java
│   ├── UserStateTransitionPropertyTest.java
│   ├── OutboxEventPropertyTest.java
│   ├── RateLimitPropertyTest.java
│   ├── SensitiveDataPropertyTest.java
│   ├── InputValidationPropertyTest.java
│   ├── ProfileUpdatePropertyTest.java
│   ├── ResendAntiEnumerationPropertyTest.java
│   └── CircuitBreakerPropertyTest.java
├── integration/
│   ├── UserControllerIntegrationTest.java
│   ├── EmailVerificationIntegrationTest.java
│   ├── OutboxDispatcherIntegrationTest.java
│   └── PlatformClientIntegrationTest.java
└── contract/
    └── UserServiceContractTest.java
```

### Property Test Example

```java
@Property(tries = 100)
@Tag("Feature: user-service, Property 1: Email Normalization Idempotence")
void emailNormalizationIsIdempotent(@ForAll @Email String email) {
    String normalized = emailNormalizer.normalize(email);
    String normalizedAgain = emailNormalizer.normalize(normalized);
    
    assertThat(normalized).isEqualTo(normalizedAgain);
    assertThat(normalized).isEqualTo(email.toLowerCase().trim());
}
```

## Project Structure

```
services/user-service/
├── src/
│   ├── main/
│   │   ├── java/com/authplatform/usersvc/
│   │   │   ├── api/
│   │   │   │   ├── controller/
│   │   │   │   │   ├── UserController.java
│   │   │   │   │   ├── EmailVerificationController.java
│   │   │   │   │   └── MeController.java
│   │   │   │   ├── dto/
│   │   │   │   │   ├── request/
│   │   │   │   │   └── response/
│   │   │   │   └── advice/
│   │   │   │       └── GlobalExceptionHandler.java
│   │   │   ├── domain/
│   │   │   │   ├── model/
│   │   │   │   │   ├── User.java
│   │   │   │   │   ├── EmailVerificationToken.java
│   │   │   │   │   ├── OutboxEvent.java
│   │   │   │   │   └── UserStatus.java
│   │   │   │   └── service/
│   │   │   │       ├── UserRegistrationService.java
│   │   │   │       ├── EmailVerificationService.java
│   │   │   │       ├── ProfileService.java
│   │   │   │       └── PasswordService.java
│   │   │   ├── infra/
│   │   │   │   ├── persistence/
│   │   │   │   │   ├── UserRepository.java
│   │   │   │   │   ├── EmailVerificationTokenRepository.java
│   │   │   │   │   └── OutboxEventRepository.java
│   │   │   │   ├── outbox/
│   │   │   │   │   ├── OutboxPublisher.java
│   │   │   │   │   └── OutboxDispatcher.java
│   │   │   │   ├── messaging/
│   │   │   │   │   └── KafkaEventPublisher.java
│   │   │   │   ├── platform/
│   │   │   │   │   ├── CryptoServiceClient.java
│   │   │   │   │   ├── CacheServiceClient.java
│   │   │   │   │   └── LoggingServiceClient.java
│   │   │   │   └── security/
│   │   │   │       ├── SecurityConfig.java
│   │   │   │       └── JwtAuthConverter.java
│   │   │   ├── config/
│   │   │   │   ├── AppConfig.java
│   │   │   │   ├── OpenApiConfig.java
│   │   │   │   └── ClockConfig.java
│   │   │   └── common/
│   │   │       ├── errors/
│   │   │       │   ├── ProblemDetail.java
│   │   │       │   └── ErrorCodes.java
│   │   │       └── util/
│   │   │           ├── EmailNormalizer.java
│   │   │           ├── TokenHasher.java
│   │   │           └── PasswordValidator.java
│   │   └── resources/
│   │       ├── application.yml
│   │       ├── application-dev.yml
│   │       ├── application-prod.yml
│   │       └── db/migration/
│   │           ├── V1__create_users_table.sql
│   │           ├── V2__create_email_verification_tokens_table.sql
│   │           └── V3__create_outbox_events_table.sql
│   └── test/
│       └── java/com/authplatform/usersvc/
│           ├── unit/
│           ├── property/
│           ├── integration/
│           └── contract/
├── proto/
│   └── user_service.proto
├── build.gradle
├── Dockerfile
└── README.md
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `GRPC_PORT` | gRPC server port | `50056` |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `KAFKA_BOOTSTRAP_SERVERS` | Kafka broker addresses | `localhost:9092` |
| `CRYPTO_SERVICE_URL` | Crypto Service gRPC endpoint | `localhost:50051` |
| `CACHE_SERVICE_URL` | Cache Service gRPC endpoint | `localhost:50060` |
| `LOGGING_SERVICE_URL` | Logging Service gRPC endpoint | `localhost:50061` |
| `JWT_ISSUER` | Expected JWT issuer | `auth-platform` |
| `JWT_JWKS_URI` | JWKS endpoint for JWT validation | - |
| `EMAIL_TOKEN_TTL_MINUTES` | Verification token TTL | `60` |
| `ARGON2_MEMORY_KB` | Argon2id memory cost | `65536` |
| `ARGON2_ITERATIONS` | Argon2id time cost | `3` |
| `ARGON2_PARALLELISM` | Argon2id parallelism | `1` |
| `RATE_LIMIT_REGISTRATION_PER_MINUTE` | Registration rate limit per IP | `5` |
| `RATE_LIMIT_RESEND_PER_HOUR` | Resend rate limit per email | `3` |
| `SHUTDOWN_TIMEOUT_SECONDS` | Graceful shutdown timeout | `30` |

## Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `user_registrations_total` | Counter | `status` | Total registration attempts |
| `user_verifications_total` | Counter | `status` | Total verification attempts |
| `user_profile_updates_total` | Counter | `status` | Total profile updates |
| `outbox_events_published_total` | Counter | `event_type` | Outbox events published |
| `outbox_events_pending` | Gauge | - | Pending outbox events |
| `rate_limit_exceeded_total` | Counter | `endpoint` | Rate limit violations |
| `platform_client_requests_total` | Counter | `service`, `status` | Platform service calls |
| `platform_client_latency_seconds` | Histogram | `service` | Platform service latency |
| `circuit_breaker_state` | Gauge | `service` | Circuit breaker state (0=closed, 1=open, 2=half-open) |
