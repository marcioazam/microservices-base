# Design Document: User Service Modernization 2025

## Overview

This design modernizes the User Service to December 2025 state-of-the-art standards, integrating with platform services (cache-service, logging-service) via gRPC, enabling Java 21 virtual threads, centralizing validation logic, and eliminating code redundancy.

### Key Modernization Goals

1. **Platform Integration**: gRPC clients for cache-service and logging-service with circuit breaker fallbacks
2. **Virtual Threads**: Java 21 virtual threads for HTTP handling, gRPC calls, and outbox processing
3. **Centralized Validation**: Single ValidationService for all input validation
4. **Zero Redundancy**: Single utilities for IP masking, correlation ID, MDC management
5. **Distributed Rate Limiting**: Cache-service backed rate limiting with local fallback
6. **Test Reorganization**: Separate unit/property/integration test packages

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           User Service                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   User      │  │   Email     │  │    Me       │  │   Health    │    │
│  │ Controller  │  │ Verification│  │ Controller  │  │ Controller  │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────────────┘    │
│         │                │                │                             │
│  ┌──────┴────────────────┴────────────────┴──────┐                     │
│  │              Domain Services                   │                     │
│  │  ┌────────────────┐  ┌────────────────┐       │                     │
│  │  │ Registration   │  │ EmailVerify    │       │                     │
│  │  │ Service        │  │ Service        │       │                     │
│  │  └────────────────┘  └────────────────┘       │                     │
│  │  ┌────────────────┐  ┌────────────────┐       │                     │
│  │  │ Profile        │  │ RateLimit      │       │                     │
│  │  │ Service        │  │ Service        │       │                     │
│  │  └────────────────┘  └────────────────┘       │                     │
│  └───────────────────────────────────────────────┘                     │
│         │                │                │                             │
│  ┌──────┴────────────────┴────────────────┴──────┐                     │
│  │           Shared Components                    │                     │
│  │  ┌────────────────┐  ┌────────────────┐       │                     │
│  │  │ Validation     │  │ Security       │       │                     │
│  │  │ Service        │  │ Utils          │       │                     │
│  │  └────────────────┘  └────────────────┘       │                     │
│  │  ┌────────────────┐  ┌────────────────┐       │                     │
│  │  │ Password       │  │ Token          │       │                     │
│  │  │ Service        │  │ Hasher         │       │                     │
│  │  └────────────────┘  └────────────────┘       │                     │
│  └───────────────────────────────────────────────┘                     │
│         │                │                │                             │
│  ┌──────┴────────────────┴────────────────┴──────┐                     │
│  │           Infrastructure Clients               │                     │
│  │  ┌────────────────┐  ┌────────────────┐       │                     │
│  │  │ CacheService   │  │ LoggingService │       │                     │
│  │  │ Client (gRPC)  │  │ Client (gRPC)  │       │                     │
│  │  └────────────────┘  └────────────────┘       │                     │
│  │  ┌────────────────┐  ┌────────────────┐       │                     │
│  │  │ Outbox         │  │ User           │       │                     │
│  │  │ Publisher      │  │ Repository     │       │                     │
│  │  └────────────────┘  └────────────────┘       │                     │
│  └───────────────────────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────────────────┘
         │                        │
         ▼                        ▼
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  Cache Service  │      │ Logging Service │      │    PostgreSQL   │
│     (gRPC)      │      │     (gRPC)      │      │                 │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

## Components and Interfaces

### 1. Platform Service Clients

#### CacheServiceClient

```java
@Component
public class CacheServiceClient {
    private final CacheServiceGrpc.CacheServiceBlockingStub stub;
    private final Cache<String, byte[]> localCache;
    private final CircuitBreaker circuitBreaker;
    
    public Optional<byte[]> get(String namespace, String key);
    public void set(String namespace, String key, byte[] value, Duration ttl);
    public void delete(String namespace, String key);
    
    // Fallback methods use localCache (Caffeine)
}
```

#### LoggingServiceClient

```java
@Component
public class LoggingServiceClient {
    private final LoggingServiceGrpc.LoggingServiceStub asyncStub;
    private final SecurityUtils securityUtils;
    
    public void logAudit(AuditEvent event);
    public void logSecurity(SecurityEvent event);
    
    // Fallback: structured JSON to local logger
}
```

### 2. Centralized Validation Service

```java
@Service
public class ValidationService {
    public ValidationResult validateRegistration(RegistrationRequest request);
    public ValidationResult validateEmail(String email);
    public ValidationResult validatePassword(String password);
    public ValidationResult validateDisplayName(String displayName);
    
    public record ValidationResult(boolean valid, List<FieldError> errors) {}
    public record FieldError(String field, String code, String message) {}
}
```

### 3. Security Utilities (Centralized)

```java
@Component
public class SecurityUtils {
    public String maskIp(String ip);           // Single IP masking implementation
    public String maskEmail(String email);     // Single email masking implementation
    public String getOrCreateCorrelationId(String provided);  // Correlation ID management
    public void setMdcContext(String correlationId, String userId);  // MDC management
    public void clearMdcContext();
}
```

### 4. Rate Limit Service (Distributed)

```java
@Service
public class RateLimitService {
    private static final String NAMESPACE = "user-service:ratelimit";
    private final CacheServiceClient cacheClient;
    private final RateLimitConfig config;
    
    public void checkRegistrationLimit(String ipAddress);
    public void checkResendLimit(String email, String ipAddress);
    public void checkVerifyLimit(String ipAddress);
}
```


## Data Models

### User Entity

```java
@Entity
@Table(name = "users")
public class User {
    @Id @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;
    
    @Column(unique = true, nullable = false)
    private String email;
    
    @Column(nullable = false)
    private boolean emailVerified;
    
    @Column(nullable = false)
    private String passwordHash;
    
    @Column(nullable = false)
    private String displayName;
    
    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private UserStatus status;
    
    @Column(nullable = false)
    private Instant createdAt;
    
    private Instant updatedAt;
}
```

### EmailVerificationToken Entity

```java
@Entity
@Table(name = "email_verification_tokens")
public class EmailVerificationToken {
    @Id @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;
    
    @Column(nullable = false)
    private UUID userId;
    
    @Column(nullable = false, length = 64)
    private String tokenHash;
    
    @Column(nullable = false)
    private Instant expiresAt;
    
    private Instant usedAt;
    
    @Column(nullable = false)
    private Instant createdAt;
}
```

### Outbox Event Entity

```java
@Entity
@Table(name = "outbox_events")
public class OutboxEvent {
    @Id @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;
    
    @Column(nullable = false)
    private String aggregateType;
    
    @Column(nullable = false)
    private UUID aggregateId;
    
    @Column(nullable = false)
    private String eventType;
    
    @Column(columnDefinition = "jsonb", nullable = false)
    private String payload;
    
    @Column(nullable = false)
    private Instant createdAt;
    
    private Instant processedAt;
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do.*

### Property 1: Platform Service Fallback Resilience

*For any* cache or logging operation, when the remote platform service is unavailable, the operation SHALL complete successfully using the local fallback (Caffeine cache for cache-service, structured JSON logging for logging-service).

**Validates: Requirements 1.2, 1.3, 2.4**

### Property 2: Sensitive Data Masking Consistency

*For any* IP address or email address that appears in logs or security events, the data SHALL be masked according to the masking rules (IP: last octet replaced with `***`, email: characters after first 2 before @ replaced with `***`).

**Validates: Requirements 1.5, 9.2, 9.3**

### Property 3: Rate Limit Enforcement

*For any* rate-limited operation (registration, resend, verify), when the limit is exceeded, the service SHALL return HTTP 429 with a valid Retry-After header containing the remaining window time in seconds.

**Validates: Requirements 2.5, 7.4**

### Property 4: Rate Limit Namespace Consistency

*For any* rate limit key stored in the cache, the key SHALL be prefixed with the namespace "user-service:ratelimit:" followed by the operation type and identifier.

**Validates: Requirements 2.3**

### Property 5: Password Hash Format Compliance

*For any* password that is hashed, the resulting hash SHALL be a valid Argon2id hash string starting with `$argon2id$` and containing the algorithm parameters.

**Validates: Requirements 5.2**

### Property 6: Password Hash Round-Trip Verification

*For any* password, hashing it and then verifying the original password against the hash SHALL return true, while verifying any different password SHALL return false.

**Validates: Requirements 5.2**

### Property 7: Token Hash Determinism

*For any* verification token, hashing it multiple times SHALL produce the same 64-character hexadecimal SHA-256 hash, and different tokens SHALL produce different hashes.

**Validates: Requirements 5.3**

### Property 8: Token Verification Round-Trip

*For any* generated verification token, hashing it and then verifying the original token against the hash SHALL return true.

**Validates: Requirements 5.3, 6.1**

### Property 9: User Registration State Transition

*For any* valid registration request, the created user SHALL have status PENDING_EMAIL and emailVerified=false, and both UserRegistered and EmailVerificationRequested events SHALL be published to the outbox.

**Validates: Requirements 5.1, 5.4, 5.6**

### Property 10: Duplicate Email Rejection

*For any* registration request with an email that already exists in the database, the service SHALL return HTTP 409 with error code EMAIL_EXISTS.

**Validates: Requirements 5.5**

### Property 11: Validation Service Completeness

*For any* input validation request (email, password, display name), the ValidationService SHALL return a ValidationResult containing either valid=true or valid=false with a non-empty list of FieldError objects.

**Validates: Requirements 4.2, 4.3, 4.4, 4.5**

### Property 12: Email Verification State Transition

*For any* valid verification token, verifying it SHALL transition the user to ACTIVE status with emailVerified=true, mark the token as used with timestamp, and publish UserEmailVerified event.

**Validates: Requirements 6.1, 6.2, 6.6**

### Property 13: Resend Verification Idempotency

*For any* resend verification request, the service SHALL return HTTP 202 regardless of whether the email exists, and if the email exists with unverified status, previous tokens SHALL be invalidated and EmailVerificationRequested event SHALL be published.

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 14: RFC 7807 Error Response Compliance

*For any* error response, the body SHALL conform to RFC 7807 Problem Detail format containing type, title, status, detail, instance, timestamp, correlationId, and errorCode fields.

**Validates: Requirements 10.1, 10.2, 10.3, 10.4**

### Property 15: Correlation ID Propagation

*For any* API request, the response SHALL include an X-Correlation-ID header, and all log entries for that request SHALL include the same correlation ID in the MDC context.

**Validates: Requirements 9.1**

### Property 16: Profile Cache Consistency

*For any* profile retrieval, the result SHALL be cached in Cache_Service with namespace "user-service:profile" and 5-minute TTL, and profile updates SHALL invalidate the cache entry.

**Validates: Requirements 8.3, 8.5**


## Error Handling

### Error Response Format (RFC 7807)

```java
public record ProblemDetail(
    String type,
    String title,
    int status,
    String detail,
    String instance,
    Instant timestamp,
    String correlationId,
    String errorCode,
    Map<String, Object> extensions
) {}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| EMAIL_EXISTS | 409 | Email already registered |
| INVALID_TOKEN | 400 | Verification token not found |
| EXPIRED_TOKEN | 400 | Verification token expired |
| ALREADY_USED | 400 | Verification token already used |
| USER_NOT_FOUND | 404 | User does not exist |
| RATE_LIMITED | 429 | Rate limit exceeded |
| VALIDATION_ERROR | 400 | Input validation failed |
| INTERNAL_ERROR | 500 | Unexpected server error |

### Exception Hierarchy

```java
public sealed abstract class UserServiceException extends RuntimeException {
    public abstract String getErrorCode();
    public abstract int getHttpStatus();
}

public final class EmailExistsException extends UserServiceException { }
public final class InvalidTokenException extends UserServiceException { }
public final class ExpiredTokenException extends UserServiceException { }
public final class AlreadyUsedException extends UserServiceException { }
public final class UserNotFoundException extends UserServiceException { }
public final class RateLimitedException extends UserServiceException {
    private final Duration retryAfter;
}
public final class ValidationException extends UserServiceException {
    private final List<FieldError> errors;
}
```

## Testing Strategy

### Test Organization

```
src/
├── main/java/com/auth/userservice/
│   ├── api/
│   ├── domain/
│   ├── infrastructure/
│   └── shared/
└── test/java/com/auth/userservice/
    ├── unit/           # Fast, isolated unit tests
    │   ├── domain/
    │   ├── shared/
    │   └── infrastructure/
    ├── property/       # jqwik property-based tests
    │   ├── domain/
    │   └── shared/
    └── integration/    # Testcontainers-based tests
        ├── api/
        └── infrastructure/
```

### Property-Based Testing Configuration

- **Framework**: jqwik 1.9+ (latest stable)
- **Minimum iterations**: 100 per property
- **Shrinking**: Enabled for counterexample minimization
- **Tag format**: `@Tag("Feature: user-service-modernization-2025, Property N: description")`

### Unit Testing

- **Framework**: JUnit 5.11+
- **Mocking**: Mockito 5.14+
- **Assertions**: AssertJ 3.26+
- **Focus**: Individual component behavior, edge cases, error conditions

### Integration Testing

- **Framework**: Spring Boot Test 3.4+
- **Containers**: Testcontainers 1.20+ for PostgreSQL, Kafka
- **gRPC Testing**: grpc-testing for mock platform services
- **Focus**: End-to-end flows, database interactions, event publishing

### Test Commands

```bash
# Run all tests
./gradlew test

# Run only unit tests
./gradlew test --tests "com.auth.userservice.unit.*"

# Run only property tests
./gradlew test --tests "com.auth.userservice.property.*"

# Run only integration tests
./gradlew test --tests "com.auth.userservice.integration.*"
```

### Coverage Requirements

- **Unit tests**: 80% line coverage for domain and shared packages
- **Property tests**: All 16 correctness properties implemented
- **Integration tests**: All API endpoints and event flows covered
