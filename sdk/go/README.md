# Auth Platform Go SDK

Official Go SDK for the Auth Platform. Provides HTTP middleware, gRPC interceptors, PKCE/DPoP support, and observability integration.

## Features

- OAuth 2.0 client credentials and authorization code flows
- PKCE (Proof Key for Code Exchange) support
- DPoP (Demonstrating Proof of Possession) sender-constrained tokens
- HTTP middleware with skip patterns and custom error handlers
- gRPC interceptors with method filtering
- Automatic retry with exponential backoff
- OpenTelemetry tracing integration
- Structured logging with sensitive data filtering
- JWKS caching with auto-refresh

## Installation

```bash
go get github.com/auth-platform/sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/auth-platform/sdk-go/src/client"
)

func main() {
    c, err := client.New(
        client.WithBaseURL("https://auth.example.com"),
        client.WithClientID("your-client-id"),
        client.WithClientSecret("your-client-secret"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Validate a token
    claims, err := c.ValidateTokenCtx(context.Background(), accessToken)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("User: %s", claims.Subject)
}
```

## Configuration

### Environment Variables

```bash
export AUTH_PLATFORM_BASE_URL=https://auth.example.com
export AUTH_PLATFORM_CLIENT_ID=your-client-id
export AUTH_PLATFORM_CLIENT_SECRET=your-client-secret
export AUTH_PLATFORM_TIMEOUT=30s
export AUTH_PLATFORM_JWKS_CACHE_TTL=1h
export AUTH_PLATFORM_MAX_RETRIES=3
export AUTH_PLATFORM_BASE_DELAY=1s
export AUTH_PLATFORM_MAX_DELAY=30s
export AUTH_PLATFORM_DPOP_ENABLED=true
export AUTH_PLATFORM_DPOP_KEY_PATH=/path/to/dpop-key.pem
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AUTH_PLATFORM_BASE_URL` | Yes | - | Auth platform base URL |
| `AUTH_PLATFORM_CLIENT_ID` | Yes | - | OAuth client ID |
| `AUTH_PLATFORM_CLIENT_SECRET` | No | - | OAuth client secret |
| `AUTH_PLATFORM_TIMEOUT` | No | `30s` | HTTP request timeout |
| `AUTH_PLATFORM_JWKS_CACHE_TTL` | No | `1h` | JWKS cache TTL (1m-24h) |
| `AUTH_PLATFORM_MAX_RETRIES` | No | `3` | Maximum retry attempts |
| `AUTH_PLATFORM_BASE_DELAY` | No | `1s` | Initial retry delay |
| `AUTH_PLATFORM_MAX_DELAY` | No | `30s` | Maximum retry delay |
| `AUTH_PLATFORM_DPOP_ENABLED` | No | `false` | Enable DPoP token binding |
| `AUTH_PLATFORM_DPOP_KEY_PATH` | No | - | Path to DPoP private key |

```go
c, err := client.NewFromEnv()
// Or with additional overrides:
c, err := client.NewFromEnv(
    client.WithTimeout(45*time.Second),
)
```

### Functional Options

```go
import "github.com/auth-platform/sdk-go/src/client"

c, err := client.New(
    client.WithBaseURL("https://auth.example.com"),
    client.WithClientID("your-client-id"),
    client.WithClientSecret("your-client-secret"),
    client.WithTimeout(10*time.Second),
    client.WithJWKSCacheTTL(30*time.Minute),
    client.WithMaxRetries(5),
    client.WithBaseDelay(time.Second),
    client.WithMaxDelay(30*time.Second),
    client.WithDPoP(true),
)
```

## Token Validation

### Basic Validation

```go
claims, err := c.ValidateTokenCtx(ctx, accessToken)
if err != nil {
    if errors.IsValidation(err) {
        // Invalid token
    }
    log.Fatal(err)
}

log.Printf("User: %s", claims.Subject)
log.Printf("Issuer: %s", claims.Issuer)
```

### Validation with Options

For more granular control, use `ValidateTokenWithOpts`:

```go
import (
    "github.com/auth-platform/sdk-go/src/client"
    "github.com/auth-platform/sdk-go/src/token"
)

opts := token.ValidationOptions{
    Audience:       "https://api.example.com",
    Issuer:         "https://auth.example.com",
    RequiredClaims: []string{"email", "roles"},
    SkipExpiry:     false, // Set true to skip expiration check
}

claims, err := c.ValidateTokenWithOpts(ctx, accessToken, opts)
if err != nil {
    log.Fatal(err)
}
```

### ValidationOptions Reference

| Option | Type | Description |
|--------|------|-------------|
| `Audience` | `string` | Expected token audience (aud claim) |
| `Issuer` | `string` | Expected token issuer (iss claim) |
| `RequiredClaims` | `[]string` | Claims that must be present in the token |
| `SkipExpiry` | `bool` | Skip expiration validation (use with caution) |

## HTTP Middleware

### Basic Usage

```go
import (
    "net/http"
    "github.com/auth-platform/sdk-go/src/client"
    "github.com/auth-platform/sdk-go/src/middleware"
)

func main() {
    c, _ := client.New(
        client.WithBaseURL("https://auth.example.com"),
        client.WithClientID("your-client-id"),
    )
    defer c.Close()

    // Wrap handlers with authentication
    mux := http.NewServeMux()
    mux.HandleFunc("/public", publicHandler)
    mux.Handle("/protected", c.HTTPMiddleware()(http.HandlerFunc(protectedHandler)))

    http.ListenAndServe(":8080", mux)
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    claims, ok := middleware.GetClaimsFromContext(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    fmt.Fprintf(w, "Hello, %s!", claims.Subject)
}
```

### Middleware Options

The middleware supports functional options for advanced configuration:

```go
import "github.com/auth-platform/sdk-go/src/middleware"

// Skip authentication for certain paths
mw := c.HTTPMiddleware(
    middleware.WithSkipPatterns("/health", "/metrics", "^/public/.*"),
)

// Custom error handling
mw := c.HTTPMiddleware(
    middleware.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
    }),
)

// Validate audience and issuer
mw := c.HTTPMiddleware(
    middleware.WithAudience("https://api.example.com"),
    middleware.WithIssuer("https://auth.example.com"),
)

// Require specific claims
mw := c.HTTPMiddleware(
    middleware.WithRequiredClaims("email", "roles"),
)

// Combine multiple options
mw := c.HTTPMiddleware(
    middleware.WithSkipPatterns("/health", "/public/.*"),
    middleware.WithAudience("https://api.example.com"),
    middleware.WithRequiredClaims("email"),
    middleware.WithErrorHandler(customErrorHandler),
)
```

### Middleware Options Reference

| Option | Description |
|--------|-------------|
| `WithSkipPatterns(patterns...)` | URL regex patterns to skip authentication |
| `WithErrorHandler(handler)` | Custom error handler function |
| `WithTokenExtractor(extractor)` | Custom token extractor |
| `WithCookieExtraction(name)` | Extract token from named cookie |
| `WithAudience(aud)` | Expected token audience |
| `WithIssuer(iss)` | Expected token issuer |
| `WithRequiredClaims(claims...)` | Claims that must be present |

## gRPC Interceptors

### Server-side

```go
import (
    "google.golang.org/grpc"
    "github.com/auth-platform/sdk-go/src/client"
    "github.com/auth-platform/sdk-go/src/middleware"
)

func main() {
    c, _ := client.New(
        client.WithBaseURL("https://auth.example.com"),
        client.WithClientID("your-client-id"),
    )
    defer c.Close()

    server := grpc.NewServer(
        grpc.UnaryInterceptor(middleware.UnaryServerInterceptor(c)),
        grpc.StreamInterceptor(middleware.StreamServerInterceptor(c)),
    )

    // Register services...
}
```

### Client-side

```go
conn, err := grpc.Dial(
    "localhost:50051",
    grpc.WithUnaryInterceptor(middleware.UnaryClientInterceptor(c)),
    grpc.WithStreamInterceptor(middleware.StreamClientInterceptor(c)),
)
```

### gRPC Interceptor Options

```go
// Skip authentication for specific methods
server := grpc.NewServer(
    grpc.UnaryInterceptor(middleware.UnaryServerInterceptor(c,
        middleware.WithGRPCSkipMethods("/grpc.health.v1.Health/Check", "/api.Service/PublicMethod"),
        middleware.WithGRPCAudience("https://api.example.com"),
        middleware.WithGRPCIssuer("https://auth.example.com"),
        middleware.WithGRPCRequiredClaims("email", "roles"),
    )),
)
```

### Helper Functions

```go
// ShouldSkipMethod checks if a gRPC method should skip authentication.
// Useful for custom interceptor logic or testing.
skip := middleware.ShouldSkipMethod("/api.Service/Health", []string{"/Health", "/Ping"})
```

## PKCE Support

PKCE (Proof Key for Code Exchange) prevents authorization code interception attacks.

```go
import "github.com/auth-platform/sdk-go/src/auth"

// Generate PKCE pair
pkce, err := auth.GeneratePKCE()
if err != nil {
    log.Fatal(err)
}

// Use pkce.Verifier and pkce.Challenge in your OAuth flow
// pkce.Method is "S256"

// Verify PKCE (server-side)
valid := auth.VerifyPKCE(verifier, challenge)
```

## DPoP Support

DPoP (Demonstrating Proof of Possession) binds tokens to a specific client key pair.

```go
import (
    "github.com/auth-platform/sdk-go/src/client"
    "github.com/auth-platform/sdk-go/src/auth"
)

// Enable DPoP via config option
c, err := client.New(
    client.WithBaseURL("https://auth.example.com"),
    client.WithClientID("your-client-id"),
    client.WithClientSecret("your-client-secret"),
    client.WithDPoP(true), // Auto-generates ES256 key pair
)

// Access the DPoP prover
prover := c.DPoPProver()

// Or create a custom DPoP prover
keyPair, _ := auth.GenerateES256KeyPair()
prover := auth.NewDPoPProver(keyPair)

// Generate a DPoP proof
proof, err := prover.GenerateProof(ctx, "POST", "https://auth.example.com/token", "")
```

## Retry Policy

Automatic retry with exponential backoff for transient failures.

```go
import "github.com/auth-platform/sdk-go/src/retry"

policy := retry.NewPolicy(
    retry.WithMaxRetries(5),
    retry.WithBaseDelay(100*time.Millisecond),
    retry.WithMaxDelay(10*time.Second),
    retry.WithJitter(0.2),
)

// Use with client configuration
c, err := client.New(
    client.WithBaseURL("https://auth.example.com"),
    client.WithClientID("your-client-id"),
    client.WithMaxRetries(5),
    client.WithBaseDelay(100*time.Millisecond),
    client.WithMaxDelay(10*time.Second),
)
```

Retryable conditions:
- HTTP 429 (Too Many Requests)
- HTTP 502, 503, 504 (Server errors)
- Network errors
- Respects `Retry-After` header

## Error Handling

The SDK uses a unified `SDKError` type with error codes for programmatic handling:

```go
import (
    "errors"
    sdkerrors "github.com/auth-platform/sdk-go/src/errors"
)

claims, err := c.ValidateTokenCtx(ctx, token)
if err != nil {
    switch {
    case sdkerrors.IsTokenExpired(err):
        // Re-authenticate
    case sdkerrors.IsRateLimited(err):
        // Wait and retry
    case sdkerrors.IsNetwork(err):
        // Network issue
    default:
        log.Fatal(err)
    }
}

// Extract error code programmatically
if code := sdkerrors.GetCode(err); code != "" {
    // Handle based on error code
}

// Error wrapping and unwrapping works correctly
var sdkErr *sdkerrors.SDKError
if errors.As(err, &sdkErr) {
    log.Printf("Error code: %s, message: %s", sdkErr.Code, sdkErr.Message)
}
```

## Error Codes

| Error Code | Helper Function | Description |
|------------|-----------------|-------------|
| `INVALID_CONFIG` | `IsInvalidConfig(err)` | Invalid client configuration |
| `TOKEN_EXPIRED` | `IsTokenExpired(err)` | Access token has expired |
| `TOKEN_INVALID` | `IsTokenInvalid(err)` | Token is malformed or invalid |
| `TOKEN_MISSING` | `IsTokenMissing(err)` | No token provided in request |
| `TOKEN_REFRESH_FAILED` | - | Token refresh failed |
| `NETWORK_ERROR` | `IsNetwork(err)` | Network error occurred |
| `RATE_LIMITED` | `IsRateLimited(err)` | Rate limit exceeded |
| `VALIDATION_FAILED` | `IsValidation(err)` | Token validation failed |
| `UNAUTHORIZED` | `IsUnauthorized(err)` | Request unauthorized |
| `DPOP_REQUIRED` | `IsDPoPRequired(err)` | DPoP proof required |
| `DPOP_INVALID` | `IsDPoPInvalid(err)` | DPoP proof invalid |
| `PKCE_INVALID` | `IsPKCEInvalid(err)` | PKCE parameters invalid |

### gRPC Error Mapping

SDK errors are automatically mapped to appropriate gRPC status codes in interceptors:

```go
import "github.com/auth-platform/sdk-go/src/middleware"

// Convert SDK error to gRPC status error
grpcErr := middleware.MapToGRPCError(err)
```

| SDK Error Code | gRPC Status Code |
|----------------|------------------|
| `INVALID_CONFIG` | `InvalidArgument` |
| `TOKEN_EXPIRED` | `Unauthenticated` |
| `TOKEN_INVALID` | `Unauthenticated` |
| `TOKEN_MISSING` | `Unauthenticated` |
| `TOKEN_REFRESH_FAILED` | `Unauthenticated` |
| `NETWORK_ERROR` | `Unavailable` |
| `RATE_LIMITED` | `ResourceExhausted` |
| `VALIDATION_FAILED` | `InvalidArgument` |
| `UNAUTHORIZED` | `PermissionDenied` |
| `DPOP_REQUIRED` | `Unauthenticated` |
| `DPOP_INVALID` | `Unauthenticated` |
| `PKCE_INVALID` | `InvalidArgument` |

### Error Sanitization

The SDK automatically sanitizes error messages to prevent sensitive data leakage:

```go
import sdkerrors "github.com/auth-platform/sdk-go/src/errors"

// Sensitive patterns (tokens, secrets, JWTs) are automatically redacted
sanitizedErr := sdkerrors.SanitizeError(err)
```

## Unified Package Exports

The SDK is organized into focused sub-packages:

```go
import (
    "github.com/auth-platform/sdk-go/src/client"     // Client creation and configuration
    "github.com/auth-platform/sdk-go/src/errors"     // Error types and helpers
    "github.com/auth-platform/sdk-go/src/auth"       // PKCE and DPoP
    "github.com/auth-platform/sdk-go/src/token"      // Token extraction and validation
    "github.com/auth-platform/sdk-go/src/middleware" // HTTP/gRPC middleware
    "github.com/auth-platform/sdk-go/src/retry"      // Retry policies
    "github.com/auth-platform/sdk-go/src/types"      // Result, Option, Claims
)

// Client creation
c, err := client.New(
    client.WithBaseURL("https://auth.example.com"),
    client.WithClientID("your-client-id"),
)

// Error handling
if errors.IsTokenExpired(err) { /* ... */ }

// Result/Option types
result := types.Ok(42)
opt := types.Some("value")

// PKCE
pkce, _ := auth.GeneratePKCE()
auth.VerifyPKCE(pkce.Verifier, pkce.Challenge)

// DPoP
keyPair, _ := auth.GenerateES256KeyPair()
prover := auth.NewDPoPProver(keyPair)

// Token extraction
extractor := token.NewHTTPExtractor(req)
tok, scheme, _ := extractor.Extract(ctx)

// Middleware context helpers
claims, ok := middleware.GetClaimsFromContext(ctx)
subject := middleware.GetSubject(ctx)
```

## API Reference

### Client Creation

| Function | Description |
|----------|-------------|
| `client.New(opts...)` | Create new client with functional options |
| `client.NewFromEnv(opts...)` | Create client from environment variables |
| `client.NewFromConfig(config)` | Create client from config struct |

### Client Methods

| Method | Description |
|--------|-------------|
| `ValidateTokenCtx(ctx, token)` | Validate JWT with context and return claims |
| `ValidateTokenWithOpts(ctx, token, opts)` | Validate JWT with custom options |
| `HTTPMiddleware(opts...)` | HTTP middleware with options |
| `Config()` | Get client configuration |
| `DPoPProver()` | Get DPoP prover if enabled |
| `Close()` | Release resources |

### Configuration Options

| Option | Description |
|--------|-------------|
| `WithBaseURL(url)` | Set auth platform base URL |
| `WithClientID(id)` | Set OAuth client ID |
| `WithClientSecret(secret)` | Set OAuth client secret |
| `WithTimeout(duration)` | Set HTTP request timeout |
| `WithJWKSCacheTTL(duration)` | Set JWKS cache TTL |
| `WithMaxRetries(n)` | Set maximum retry attempts |
| `WithBaseDelay(duration)` | Set initial retry delay |
| `WithMaxDelay(duration)` | Set maximum retry delay |
| `WithDPoP(enabled)` | Enable/disable DPoP |

### PKCE Functions

| Function | Description |
|----------|-------------|
| `GeneratePKCE()` | Generate verifier/challenge pair |
| `VerifyPKCE(verifier, challenge)` | Verify PKCE pair |
| `GenerateState()` | Generate secure state parameter |
| `GenerateNonce()` | Generate secure nonce |

### DPoP Functions

| Function | Description |
|----------|-------------|
| `GenerateES256KeyPair()` | Generate ES256 key pair |
| `GenerateRS256KeyPair()` | Generate RS256 key pair |
| `NewDPoPProver(keyPair)` | Create DPoP prover |
| `ComputeATH(token)` | Compute access token hash |
| `VerifyATH(token, ath)` | Verify access token hash |
| `ComputeJWKThumbprint(key)` | Compute JWK thumbprint |

### Context Helpers

| Function | Description |
|----------|-------------|
| `GetClaimsFromContext(ctx)` | Extract claims from context |
| `GetTokenFromContext(ctx)` | Extract raw token from context |
| `GetSubject(ctx)` | Get subject claim from context |
| `GetClientID(ctx)` | Get client ID from context |
| `HasScope(ctx, scope)` | Check if context has scope |
| `RequireScope(ctx, scope)` | Require specific scope |
| `RequireAnyScope(ctx, scopes...)` | Require any of the scopes |
| `RequireAllScopes(ctx, scopes...)` | Require all scopes |

### Token Extraction

| Function | Description |
|----------|-------------|
| `NewHTTPExtractor(req)` | Create HTTP header extractor |
| `NewGRPCExtractor()` | Create gRPC metadata extractor |
| `NewCookieExtractor(req, name, scheme)` | Create cookie extractor |
| `NewChainedExtractor(extractors...)` | Chain multiple extractors |
| `ExtractBearerToken(req)` | Extract Bearer token from request |
| `ExtractDPoPToken(req)` | Extract DPoP token from request |

## Migration from v1

Key changes in v2:
- **Unified imports**: All common types and functions available from `github.com/auth-platform/sdk-go/src`
- `Option` renamed to `ClientOption` to avoid conflict with generic `Option[T]`
- DPoP support added via `WithDPoPProver` option
- Retry policy configurable via `WithRetryPolicy` option
- Observability via `WithTracer` and `WithLogger` options
- PKCE helpers added: `GeneratePKCE()`, `VerifyPKCE()`
- Result/Option generic types for functional error handling
- Context helpers exported from main package

## License

MIT
