# Design Document: Password Recovery Service

## Overview

The Password Recovery Service is a C# microservice built on .NET 8 that provides secure password recovery functionality for the auth platform. It follows a clean architecture pattern with clear separation between API, domain, and infrastructure layers. The service integrates with the existing email-service for sending recovery emails and uses PostgreSQL for token persistence.

### Key Design Decisions

1. **Token Storage**: Recovery tokens are stored hashed (SHA-256) in PostgreSQL, never in plain text
2. **Password Hashing**: Argon2id with secure parameters (memory: 64MB, iterations: 3, parallelism: 4)
3. **Async Email**: Email sending is decoupled via RabbitMQ to prevent blocking API responses
4. **Rate Limiting**: Redis-based sliding window rate limiting for scalability
5. **Observability**: OpenTelemetry for distributed tracing, Prometheus for metrics

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Password Recovery Service                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Request   │  │  Validate   │  │    Reset    │  │   Health    │        │
│  │  Controller │  │  Controller │  │  Controller │  │  Controller │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────────────┘        │
│         │                │                │                                  │
│  ┌──────┴────────────────┴────────────────┴──────┐                          │
│  │              Application Services              │                          │
│  │  ┌────────────────┐  ┌────────────────────┐   │                          │
│  │  │ RecoveryService│  │ PasswordService    │   │                          │
│  │  └────────────────┘  └────────────────────┘   │                          │
│  └───────────────────────────────────────────────┘                          │
│         │                │                │                                  │
│  ┌──────┴────────────────┴────────────────┴──────┐                          │
│  │                 Domain Layer                   │                          │
│  │  ┌────────────┐  ┌────────────┐  ┌─────────┐  │                          │
│  │  │RecoveryToken│ │PasswordPolicy│ │  User   │  │                          │
│  │  └────────────┘  └────────────┘  └─────────┘  │                          │
│  └───────────────────────────────────────────────┘                          │
│         │                │                │                                  │
│  ┌──────┴────────────────┴────────────────┴──────┐                          │
│  │              Infrastructure Layer              │                          │
│  │  ┌──────────┐ ┌──────────┐ ┌────────────────┐ │                          │
│  │  │PostgreSQL│ │  Redis   │ │  RabbitMQ      │ │                          │
│  │  │TokenStore│ │RateLimiter│ │ EmailPublisher│ │                          │
│  │  └──────────┘ └──────────┘ └────────────────┘ │                          │
│  └───────────────────────────────────────────────┘                          │
└─────────────────────────────────────────────────────────────────────────────┘
                    │                    │                    │
                    ▼                    ▼                    ▼
            ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
            │  PostgreSQL  │    │    Redis     │    │   RabbitMQ   │
            │   Database   │    │    Cache     │    │    Queue     │
            └──────────────┘    └──────────────┘    └──────────────┘
                                                           │
                                                           ▼
                                                   ┌──────────────┐
                                                   │Email Service │
                                                   └──────────────┘
```

## Components and Interfaces

### API Layer

```csharp
// Controllers/PasswordRecoveryController.cs
[ApiController]
[Route("api/v1/password-recovery")]
public class PasswordRecoveryController : ControllerBase
{
    [HttpPost("request")]
    [ProducesResponseType(typeof(RecoveryRequestResponse), 200)]
    [ProducesResponseType(typeof(ErrorResponse), 400)]
    [ProducesResponseType(typeof(ErrorResponse), 429)]
    public Task<IActionResult> RequestRecovery([FromBody] RecoveryRequest request);

    [HttpPost("validate")]
    [ProducesResponseType(typeof(TokenValidationResponse), 200)]
    [ProducesResponseType(typeof(ErrorResponse), 400)]
    public Task<IActionResult> ValidateToken([FromBody] TokenValidationRequest request);

    [HttpPost("reset")]
    [ProducesResponseType(typeof(PasswordResetResponse), 200)]
    [ProducesResponseType(typeof(ErrorResponse), 400)]
    public Task<IActionResult> ResetPassword([FromBody] PasswordResetRequest request);
}
```

### Request/Response DTOs

```csharp
// DTOs/RecoveryRequest.cs
public record RecoveryRequest(string Email);

// DTOs/RecoveryRequestResponse.cs
public record RecoveryRequestResponse(
    string Message,
    string CorrelationId
);

// DTOs/TokenValidationRequest.cs
public record TokenValidationRequest(string Token);

// DTOs/TokenValidationResponse.cs
public record TokenValidationResponse(
    bool IsValid,
    string? ResetToken,  // Short-lived token for password reset
    string CorrelationId
);

// DTOs/PasswordResetRequest.cs
public record PasswordResetRequest(
    string ResetToken,
    string NewPassword,
    string ConfirmPassword
);

// DTOs/PasswordResetResponse.cs
public record PasswordResetResponse(
    bool Success,
    string Message,
    string CorrelationId
);
```

### Application Services

```csharp
// Services/IRecoveryService.cs
public interface IRecoveryService
{
    Task<Result<RecoveryRequestResponse>> RequestRecoveryAsync(
        string email, 
        string ipAddress,
        CancellationToken ct);
    
    Task<Result<TokenValidationResponse>> ValidateTokenAsync(
        string token,
        CancellationToken ct);
    
    Task<Result<PasswordResetResponse>> ResetPasswordAsync(
        string resetToken,
        string newPassword,
        CancellationToken ct);
}

// Services/IPasswordHasher.cs
public interface IPasswordHasher
{
    string Hash(string password);
    bool Verify(string password, string hash);
}

// Services/ITokenGenerator.cs
public interface ITokenGenerator
{
    string GenerateToken(int byteLength = 32);
    string HashToken(string token);
}
```

### Domain Entities

```csharp
// Domain/RecoveryToken.cs
public class RecoveryToken
{
    public Guid Id { get; private set; }
    public Guid UserId { get; private set; }
    public string TokenHash { get; private set; }
    public DateTime CreatedAt { get; private set; }
    public DateTime ExpiresAt { get; private set; }
    public bool IsUsed { get; private set; }
    public DateTime? UsedAt { get; private set; }
    public string? IpAddress { get; private set; }

    public bool IsValid => !IsUsed && DateTime.UtcNow < ExpiresAt;

    public void MarkAsUsed()
    {
        IsUsed = true;
        UsedAt = DateTime.UtcNow;
    }

    public static RecoveryToken Create(
        Guid userId, 
        string tokenHash, 
        TimeSpan validity,
        string? ipAddress)
    {
        return new RecoveryToken
        {
            Id = Guid.NewGuid(),
            UserId = userId,
            TokenHash = tokenHash,
            CreatedAt = DateTime.UtcNow,
            ExpiresAt = DateTime.UtcNow.Add(validity),
            IsUsed = false,
            IpAddress = ipAddress
        };
    }
}

// Domain/PasswordPolicy.cs
public class PasswordPolicy
{
    public int MinLength { get; init; } = 12;
    public bool RequireUppercase { get; init; } = true;
    public bool RequireLowercase { get; init; } = true;
    public bool RequireDigit { get; init; } = true;
    public bool RequireSpecialChar { get; init; } = true;
    public string SpecialCharacters { get; init; } = "!@#$%^&*()_+-=[]{}|;:,.<>?";

    public ValidationResult Validate(string password)
    {
        var errors = new List<string>();
        
        if (password.Length < MinLength)
            errors.Add($"Password must be at least {MinLength} characters");
        if (RequireUppercase && !password.Any(char.IsUpper))
            errors.Add("Password must contain at least one uppercase letter");
        if (RequireLowercase && !password.Any(char.IsLower))
            errors.Add("Password must contain at least one lowercase letter");
        if (RequireDigit && !password.Any(char.IsDigit))
            errors.Add("Password must contain at least one digit");
        if (RequireSpecialChar && !password.Any(c => SpecialCharacters.Contains(c)))
            errors.Add("Password must contain at least one special character");

        return errors.Count == 0 
            ? ValidationResult.Success() 
            : ValidationResult.Failure(errors);
    }
}
```

### Infrastructure Interfaces

```csharp
// Infrastructure/ITokenRepository.cs
public interface ITokenRepository
{
    Task<RecoveryToken?> GetByHashAsync(string tokenHash, CancellationToken ct);
    Task<RecoveryToken?> GetByIdAsync(Guid id, CancellationToken ct);
    Task CreateAsync(RecoveryToken token, CancellationToken ct);
    Task UpdateAsync(RecoveryToken token, CancellationToken ct);
    Task InvalidateUserTokensAsync(Guid userId, CancellationToken ct);
    Task CleanupExpiredAsync(DateTime before, CancellationToken ct);
}

// Infrastructure/IUserRepository.cs
public interface IUserRepository
{
    Task<User?> GetByEmailAsync(string email, CancellationToken ct);
    Task<User?> GetByIdAsync(Guid id, CancellationToken ct);
    Task UpdatePasswordAsync(Guid userId, string passwordHash, CancellationToken ct);
}

// Infrastructure/IRateLimiter.cs
public interface IRateLimiter
{
    Task<RateLimitResult> CheckAsync(string key, int limit, TimeSpan window, CancellationToken ct);
    Task IncrementAsync(string key, TimeSpan window, CancellationToken ct);
}

// Infrastructure/IEmailPublisher.cs
public interface IEmailPublisher
{
    Task PublishRecoveryEmailAsync(RecoveryEmailMessage message, CancellationToken ct);
}
```

## Data Models

### Database Schema (PostgreSQL)

```sql
-- Recovery Tokens Table
CREATE TABLE recovery_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    ip_address INET,
    CONSTRAINT uk_token_hash UNIQUE (token_hash)
);

CREATE INDEX idx_recovery_tokens_user_id ON recovery_tokens(user_id);
CREATE INDEX idx_recovery_tokens_expires_at ON recovery_tokens(expires_at) WHERE NOT is_used;

-- Audit Log Table
CREATE TABLE password_recovery_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    user_id UUID,
    email VARCHAR(255),
    ip_address INET,
    correlation_id UUID NOT NULL,
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_user_id ON password_recovery_audit(user_id);
CREATE INDEX idx_audit_created_at ON password_recovery_audit(created_at);
```

### Entity Framework Configuration

```csharp
// Infrastructure/Data/RecoveryTokenConfiguration.cs
public class RecoveryTokenConfiguration : IEntityTypeConfiguration<RecoveryToken>
{
    public void Configure(EntityTypeBuilder<RecoveryToken> builder)
    {
        builder.ToTable("recovery_tokens");
        
        builder.HasKey(t => t.Id);
        builder.Property(t => t.Id).HasColumnName("id");
        builder.Property(t => t.UserId).HasColumnName("user_id").IsRequired();
        builder.Property(t => t.TokenHash).HasColumnName("token_hash")
            .HasMaxLength(64).IsRequired();
        builder.Property(t => t.CreatedAt).HasColumnName("created_at").IsRequired();
        builder.Property(t => t.ExpiresAt).HasColumnName("expires_at").IsRequired();
        builder.Property(t => t.IsUsed).HasColumnName("is_used").IsRequired();
        builder.Property(t => t.UsedAt).HasColumnName("used_at");
        builder.Property(t => t.IpAddress).HasColumnName("ip_address");

        builder.HasIndex(t => t.TokenHash).IsUnique();
        builder.HasIndex(t => t.UserId);
    }
}
```

### Message Contracts

```csharp
// Messages/RecoveryEmailMessage.cs
public record RecoveryEmailMessage(
    Guid CorrelationId,
    string RecipientEmail,
    string RecoveryLink,
    DateTime ExpiresAt,
    string UserName
);
```



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Email Format Validation

*For any* string input to the recovery request endpoint, the service SHALL accept only strings that match valid email format (RFC 5322) and reject all others with a validation error.

**Validates: Requirements 1.1**

### Property 2: Token Generation Security

*For any* generated recovery token, the token SHALL have at least 32 bytes of entropy (256 bits), be generated using a cryptographically secure random number generator, and have an expiration time set within the configured bounds.

**Validates: Requirements 2.1, 2.2, 2.3**

### Property 3: Token Storage Security (Hashing)

*For any* recovery token stored in the Token_Store, the stored value SHALL be a cryptographic hash of the original token, such that the original token cannot be recovered from the stored hash.

**Validates: Requirements 2.4**

### Property 4: Token Invalidation on New Request

*For any* user with existing recovery tokens, when a new recovery token is generated, all previous tokens for that user SHALL be invalidated and no longer usable for password reset.

**Validates: Requirements 2.5**

### Property 5: Response Uniformity (Email Enumeration Prevention)

*For any* recovery request with a valid email format, the response message and timing SHALL be identical regardless of whether the email exists in the User_Store, preventing email enumeration attacks.

**Validates: Requirements 1.5, 1.6**

### Property 6: Email Message Completeness

*For any* recovery email message published to the Email_Service, the message SHALL contain a valid recovery link with the token as a URL parameter AND the token expiration time.

**Validates: Requirements 3.2, 3.3**

### Property 7: Token Validation Correctness

*For any* token submitted for validation:
- If the token exists, is not expired, and has not been used, validation SHALL succeed and return the associated user identifier
- If the token does not exist, is expired, or has been used, validation SHALL fail with a generic error message (same message for all failure cases)

**Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5**

### Property 8: Password Policy Validation

*For any* password submitted during password reset, the password SHALL be accepted only if it meets ALL of: minimum 12 characters, at least one uppercase letter, at least one lowercase letter, at least one digit, and at least one special character. Passwords not meeting all criteria SHALL be rejected.

**Validates: Requirements 5.1, 5.2**

### Property 9: Password Hashing with Argon2id

*For any* password that passes validation and is stored, the stored hash SHALL be in Argon2id format with secure parameters, and verifying the original password against the hash SHALL succeed.

**Validates: Requirements 5.3**

### Property 10: Token Single-Use Enforcement

*For any* successful password reset operation, the recovery token used SHALL be marked as used immediately, and subsequent attempts to use the same token SHALL fail.

**Validates: Requirements 5.5**

### Property 11: Rate Limiting Enforcement

*For any* sequence of requests:
- The (N+1)th recovery request from the same email within an hour SHALL be rejected with HTTP 429 when N >= 5
- The (N+1)th recovery request from the same IP within an hour SHALL be rejected with HTTP 429 when N >= 10
- The (N+1)th validation attempt for the same token SHALL be rejected with HTTP 429 when N >= 5
- All 429 responses SHALL include a Retry-After header

**Validates: Requirements 6.1, 6.2, 6.3, 6.4**

### Property 12: Audit Logging Completeness

*For any* recovery request, token validation, or password change operation, the audit log SHALL contain: timestamp, correlation ID, event type, and relevant identifiers (user ID for authenticated operations, IP address for all operations).

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 13: Sensitive Data Exclusion from Logs

*For any* log entry produced by the service, the log content SHALL NOT contain: plain text tokens, passwords (plain or hashed), or email body content.

**Validates: Requirements 7.4**

### Property 14: Correlation ID in All Responses

*For any* API response from the service (success or error), the response body SHALL contain a non-empty correlation ID that matches the OpenTelemetry trace context.

**Validates: Requirements 9.5**

### Property 15: OpenTelemetry Tracing Coverage

*For any* API request processed by the service, at least one OpenTelemetry span SHALL be created with the operation name, and the span SHALL be properly closed with success/failure status.

**Validates: Requirements 10.2**

## Error Handling

### Error Categories

| Category | HTTP Status | Error Code | Description |
|----------|-------------|------------|-------------|
| Validation | 400 | INVALID_EMAIL | Email format is invalid |
| Validation | 400 | INVALID_TOKEN | Token format is invalid |
| Validation | 400 | WEAK_PASSWORD | Password doesn't meet policy |
| Validation | 400 | PASSWORD_MISMATCH | Passwords don't match |
| Rate Limit | 429 | RATE_LIMIT_EXCEEDED | Too many requests |
| Auth | 400 | TOKEN_EXPIRED | Recovery token has expired |
| Auth | 400 | TOKEN_USED | Recovery token already used |
| Auth | 400 | TOKEN_INVALID | Recovery token not found |
| Server | 500 | INTERNAL_ERROR | Unexpected server error |

### Error Response Format

```csharp
public record ErrorResponse(
    string Code,
    string Message,
    string CorrelationId,
    Dictionary<string, string[]>? ValidationErrors = null
);
```

### Error Handling Strategy

1. **Validation Errors**: Return immediately with 400 and specific validation messages
2. **Security Errors**: Return generic messages to prevent information leakage
3. **Rate Limit Errors**: Return 429 with Retry-After header
4. **Infrastructure Errors**: Log details, return generic 500 to user
5. **All Errors**: Include correlation ID for troubleshooting

## Testing Strategy

### Testing Framework

- **Unit Tests**: xUnit with FluentAssertions
- **Property-Based Tests**: FsCheck for .NET
- **Integration Tests**: TestContainers for PostgreSQL and Redis
- **API Tests**: WebApplicationFactory with in-memory test server

### Test Categories

#### Unit Tests
- Password policy validation (edge cases)
- Token generator output format
- Domain entity state transitions
- Validator rules

#### Property-Based Tests (FsCheck)
Each correctness property will be implemented as a property-based test with minimum 100 iterations.

Configuration:
```csharp
public class PropertyTestConfig
{
    public static Arbitrary<string> EmailArbitrary => 
        Arb.From(Gen.Elements(ValidEmails).Concat(Gen.Elements(InvalidEmails)));
    
    public static Arbitrary<string> PasswordArbitrary =>
        Arb.From(Gen.Frequency(
            (7, Gen.Elements(StrongPasswords)),
            (3, Gen.Elements(WeakPasswords))));
}
```

#### Integration Tests
- Token repository CRUD operations
- Rate limiter behavior with Redis
- Email publisher message format
- End-to-end recovery flow

### Test Annotations

Each property test must be annotated with:
```csharp
[Property(MaxTest = 100)]
[Trait("Feature", "password-recovery-service")]
[Trait("Property", "N")]
// **Validates: Requirements X.Y**
```

### Coverage Requirements

- Unit tests: 80% line coverage minimum
- Property tests: All 15 correctness properties
- Integration tests: All repository and external service interactions
- API tests: All endpoints with success and error scenarios
