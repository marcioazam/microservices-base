# Testing

Test utilities and generators for property-based testing with [rapid](https://github.com/flyingmutant/rapid).

## Features

- Domain-specific generators for all value types
- Property-based test helpers
- Synctest integration for concurrency tests
- Seeded reproducibility support
- Custom assertions

## Domain Generators

Type-safe generators for property-based testing:

```go
import "github.com/authcorp/libs/go/src/testing"

// Email addresses (RFC 5322 compliant format)
email := testing.EmailGen().Draw(t, "email")
// e.g., "abc123@example.com"

// UUID v4 strings
uuid := testing.UUIDGen().Draw(t, "uuid")
// e.g., "550e8400-e29b-41d4-a716-446655440000"

// ULID strings (Crockford's Base32)
ulid := testing.ULIDGen().Draw(t, "ulid")
// e.g., "01ARZ3NDEKTSV4RRFFQ69G5FAV"

// Monetary values
money := testing.MoneyGen().Draw(t, "money")
// e.g., Money{Amount: 1234, Currency: "USD"}

// Phone numbers (E.164 format)
phone := testing.PhoneNumberGen().Draw(t, "phone")
// e.g., "+14155551234"

// URLs (HTTP/HTTPS)
url := testing.URLGen().Draw(t, "url")
// e.g., "https://example.com/path"

// IPv4 addresses
ip := testing.IPAddressGen().Draw(t, "ip")
// e.g., "192.168.1.100"

// Timestamps
recent := testing.RecentTimestampGen().Draw(t, "recent")  // Last 30 days
future := testing.FutureTimestampGen().Draw(t, "future")  // Next 30 days
custom := testing.TimestampGen(start, end).Draw(t, "ts")  // Custom range

// URL-safe slugs
slug := testing.SlugGen().Draw(t, "slug")
// e.g., "my-article-title"

// Usernames
username := testing.UsernameGen().Draw(t, "username")
// e.g., "john_doe123"

// Passwords (meets common requirements)
password := testing.PasswordGen().Draw(t, "password")
// e.g., "ABcdef12!" (upper, lower, digit, special)

// Hex color codes
color := testing.HexColorGen().Draw(t, "color")
// e.g., "#FF5733"

// Observability IDs
correlationID := testing.CorrelationIDGen().Draw(t, "correlationID")  // 32 hex chars
traceID := testing.TraceIDGen().Draw(t, "traceID")                    // W3C trace ID
spanID := testing.SpanIDGen().Draw(t, "spanID")                       // W3C span ID

// JWT-like tokens (not cryptographically valid)
jwt := testing.JWTGen().Draw(t, "jwt")
// e.g., "header.payload.signature"

// Semantic versions
version := testing.SemanticVersionGen().Draw(t, "version")
// e.g., "1.2.3"
```

## Usage in Property Tests

```go
import (
    "testing"
    
    testutil "github.com/authcorp/libs/go/src/testing"
    "pgregory.net/rapid"
)

func TestEmailValidation(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        email := testutil.EmailGen().Draw(t, "email")
        
        // All generated emails should be valid
        result := validation.Field("email", email, validation.Required())
        if !result.IsValid() {
            t.Fatalf("generated email %q should be valid", email)
        }
    })
}

func TestMoneyOperations(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        m1 := testutil.MoneyGen().Draw(t, "m1")
        m2 := testutil.MoneyGen().Draw(t, "m2")
        
        // Property: addition is commutative for same currency
        if m1.Currency == m2.Currency {
            sum1 := m1.Amount + m2.Amount
            sum2 := m2.Amount + m1.Amount
            if sum1 != sum2 {
                t.Fatal("money addition should be commutative")
            }
        }
    })
}
```

## Generator Types

| Generator | Output Type | Description |
|-----------|-------------|-------------|
| `EmailGen()` | `string` | RFC 5322 email addresses |
| `UUIDGen()` | `string` | UUID v4 format |
| `ULIDGen()` | `string` | ULID format (26 chars) |
| `MoneyGen()` | `Money` | Amount + currency |
| `PhoneNumberGen()` | `string` | E.164 phone numbers |
| `URLGen()` | `string` | HTTP/HTTPS URLs |
| `IPAddressGen()` | `string` | IPv4 addresses |
| `TimestampGen(start, end)` | `time.Time` | Custom time range |
| `RecentTimestampGen()` | `time.Time` | Last 30 days |
| `FutureTimestampGen()` | `time.Time` | Next 30 days |
| `SlugGen()` | `string` | URL-safe slugs |
| `UsernameGen()` | `string` | Valid usernames |
| `PasswordGen()` | `string` | Strong passwords |
| `HexColorGen()` | `string` | Hex color codes |
| `CorrelationIDGen()` | `string` | 32-char hex IDs |
| `TraceIDGen()` | `string` | W3C trace IDs |
| `SpanIDGen()` | `string` | W3C span IDs |
| `JWTGen()` | `string` | JWT-like tokens |
| `SemanticVersionGen()` | `string` | Semver strings |

## Best Practices

1. **Minimum 100 iterations** for property tests
2. **Use seeded tests** for reproducibility in CI
3. **Combine generators** for complex domain objects
4. **Name draws descriptively** for debugging failures
