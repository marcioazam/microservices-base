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

    authplatform "github.com/auth-platform/sdk-go"
)

func main() {
    client, err := authplatform.New(authplatform.Config{
        BaseURL:      "https://auth.example.com",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Client credentials flow
    tokens, err := client.ClientCredentials(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Access token: %s", tokens.AccessToken)
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
config := authplatform.LoadFromEnv()
client, err := authplatform.New(*config)
```

### Functional Options

```go
client, err := authplatform.New(
    authplatform.Config{
        BaseURL:  "https://auth.example.com",
        ClientID: "your-client-id",
    },
    authplatform.WithTimeout(10*time.Second),
    authplatform.WithJWKSCacheTTL(30*time.Minute),
    authplatform.WithHTTPClient(customHTTPClient),
    authplatform.WithRetryPolicy(authplatform.NewRetryPolicy(
        authplatform.WithMaxRetries(5),
        authplatform.WithBaseDelay(time.Second),
    )),
)
```

## Token Validation

```go
claims, err := client.ValidateToken(ctx, accessToken)
if err != nil {
    if authplatform.IsValidation(err) {
        // Invalid token
    }
    log.Fatal(err)
}

log.Printf("User: %s", claims.Subject)
log.Printf("Issuer: %s", claims.Issuer)
```

## HTTP Middleware

### Basic Usage

```go
import (
    "net/http"
    authplatform "github.com/auth-platform/sdk-go"
)

func main() {
    client, _ := authplatform.New(config)

    // Wrap handlers with authentication
    mux := http.NewServeMux()
    mux.HandleFunc("/public", publicHandler)
    mux.Handle("/protected", client.Middleware()(http.HandlerFunc(protectedHandler)))

    http.ListenAndServe(":8080", mux)
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    claims, ok := authplatform.GetClaimsFromContext(r.Context())
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
// Skip authentication for certain paths
middleware := client.Middleware(
    authplatform.WithSkipPatterns("/health", "/metrics", "^/public/.*"),
)

// Custom error handling
middleware := client.Middleware(
    authplatform.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
    }),
)

// Token extraction from cookies
middleware := client.Middleware(
    authplatform.WithCookieExtraction("access_token"),
)

// Custom token extractor
middleware := client.Middleware(
    authplatform.WithTokenExtractor(customExtractor),
)

// Validate audience and issuer
middleware := client.Middleware(
    authplatform.WithAudience("https://api.example.com"),
    authplatform.WithIssuer("https://auth.example.com"),
)

// Require specific claims
middleware := client.Middleware(
    authplatform.WithRequiredClaims("email", "roles"),
)

// Combine multiple options
middleware := client.Middleware(
    authplatform.WithSkipPatterns("/health", "/public/.*"),
    authplatform.WithAudience("https://api.example.com"),
    authplatform.WithRequiredClaims("email"),
    authplatform.WithErrorHandler(customErrorHandler),
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
    authplatform "github.com/auth-platform/sdk-go"
)

func main() {
    client, _ := authplatform.New(config)

    server := grpc.NewServer(
        grpc.UnaryInterceptor(client.UnaryServerInterceptor()),
        grpc.StreamInterceptor(client.StreamServerInterceptor()),
    )

    // Register services...
}
```

### Client-side

```go
conn, err := grpc.Dial(
    "localhost:50051",
    grpc.WithUnaryInterceptor(client.UnaryClientInterceptor()),
    grpc.WithStreamInterceptor(client.StreamClientInterceptor()),
)
```

### gRPC Interceptor Options

```go
// Skip authentication for specific methods
server := grpc.NewServer(
    grpc.UnaryInterceptor(client.UnaryServerInterceptor(
        authplatform.WithGRPCSkipMethods("/grpc.health.v1.Health/Check", "/api.Service/PublicMethod"),
        authplatform.WithGRPCAudience("https://api.example.com"),
        authplatform.WithGRPCIssuer("https://auth.example.com"),
        authplatform.WithGRPCRequiredClaims("email", "roles"),
    )),
)
```

### Helper Functions

```go
// ShouldSkipMethod checks if a gRPC method should skip authentication.
// Useful for custom interceptor logic or testing.
skip := authplatform.ShouldSkipMethod("/api.Service/Health", []string{"/Health", "/Ping"})
```

## PKCE Support

PKCE (Proof Key for Code Exchange) prevents authorization code interception attacks.

```go
// Generate PKCE pair
pkce, err := authplatform.GeneratePKCE()
if err != nil {
    log.Fatal(err)
}

// Build authorization URL with PKCE
authURL, err := client.BuildAuthorizationURL(authplatform.AuthorizationRequest{
    RedirectURI:         "https://app.example.com/callback",
    Scope:               "openid profile email",
    State:               state,
    CodeChallenge:       pkce.Challenge,
    CodeChallengeMethod: pkce.Method, // "S256"
})

// Exchange code with verifier
tokens, err := client.ExchangeCode(ctx, authplatform.TokenExchangeRequest{
    Code:         authorizationCode,
    RedirectURI:  "https://app.example.com/callback",
    CodeVerifier: pkce.Verifier,
})
```

## DPoP Support

DPoP (Demonstrating Proof of Possession) binds tokens to a specific client key pair.

```go
// Enable DPoP via config
client, err := authplatform.New(authplatform.Config{
    BaseURL:      "https://auth.example.com",
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    DPoPEnabled:  true, // Auto-generates ES256 key pair
})

// Or provide custom DPoP prover
keyPair, _ := authplatform.GenerateES256KeyPair()
prover := authplatform.NewDPoPProver(keyPair)

client, err := authplatform.New(config,
    authplatform.WithDPoPProver(prover),
)

// DPoP proofs are automatically added to token requests
tokens, err := client.ClientCredentials(ctx)
// tokens.TokenType will be "DPoP"
```

## Retry Policy

Automatic retry with exponential backoff for transient failures.

```go
policy := authplatform.NewRetryPolicy(
    authplatform.WithMaxRetries(5),
    authplatform.WithBaseDelay(100*time.Millisecond),
    authplatform.WithMaxDelay(10*time.Second),
    authplatform.WithJitter(0.2),
)

client, err := authplatform.New(config,
    authplatform.WithRetryPolicy(policy),
)
```

Retryable conditions:
- HTTP 429 (Too Many Requests)
- HTTP 502, 503, 504 (Server errors)
- Network errors
- Respects `Retry-After` header

## Error Handling

```go
import (
    "errors"
    authplatform "github.com/auth-platform/sdk-go"
)

token, err := client.GetAccessToken(ctx)
if err != nil {
    switch {
    case authplatform.IsTokenExpired(err):
        // Re-authenticate
    case authplatform.IsRateLimited(err):
        // Wait and retry
    case authplatform.IsNetwork(err):
        // Network issue
    default:
        log.Fatal(err)
    }
}

// Error wrapping works correctly
if errors.Is(err, authplatform.ErrTokenExpired) {
    // Handle expired token
}
```

## Sentinel Errors

| Error | Description |
|-------|-------------|
| `ErrInvalidConfig` | Invalid client configuration |
| `ErrTokenExpired` | Access token has expired |
| `ErrTokenRefresh` | Token refresh failed |
| `ErrNetwork` | Network error occurred |
| `ErrRateLimited` | Rate limit exceeded |
| `ErrValidation` | Token validation failed |
| `ErrUnauthorized` | Request unauthorized |
| `ErrDPoPRequired` | DPoP proof required |
| `ErrDPoPInvalid` | DPoP proof invalid |
| `ErrPKCEInvalid` | PKCE parameters invalid |

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `New(config, opts...)` | Create new client |
| `ClientCredentials(ctx)` | Obtain token via client credentials |
| `ValidateToken(ctx, token)` | Validate JWT and return claims |
| `GetAccessToken(ctx)` | Get valid access token |
| `BuildAuthorizationURL(req)` | Build OAuth authorization URL |
| `ExchangeCode(ctx, req)` | Exchange authorization code for tokens |
| `RefreshToken(ctx, req)` | Refresh access token |
| `Middleware(opts...)` | HTTP middleware with options |
| `UnaryServerInterceptor()` | gRPC unary server interceptor |
| `StreamServerInterceptor()` | gRPC stream server interceptor |
| `Close()` | Release resources |

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
| `VerifyATH(token, ath)` | Verify access token hash |

## Migration from v1

Key changes in v2:
- `Option` renamed to `ClientOption` to avoid conflict with generic `Option[T]`
- DPoP support added via `WithDPoPProver` option
- Retry policy configurable via `WithRetryPolicy` option
- Observability via `WithTracer` and `WithLogger` options
- PKCE helpers added: `GeneratePKCE()`, `VerifyPKCE()`

## License

MIT
