# Auth Platform Go SDK

Official Go SDK for the Auth Platform. Provides HTTP middleware, gRPC interceptors, and functional options configuration.

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

## Functional Options

```go
client, err := authplatform.New(
    authplatform.Config{
        BaseURL:  "https://auth.example.com",
        ClientID: "your-client-id",
    },
    authplatform.WithTimeout(10*time.Second),
    authplatform.WithJWKSCacheTTL(30*time.Minute),
    authplatform.WithHTTPClient(customHTTPClient),
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

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `New(config, opts...)` | Create new client |
| `ClientCredentials(ctx)` | Obtain token via client credentials |
| `ValidateToken(ctx, token)` | Validate JWT and return claims |
| `GetAccessToken(ctx)` | Get valid access token |
| `Middleware()` | HTTP middleware |
| `UnaryServerInterceptor()` | gRPC unary server interceptor |
| `StreamServerInterceptor()` | gRPC stream server interceptor |
| `UnaryClientInterceptor()` | gRPC unary client interceptor |
| `StreamClientInterceptor()` | gRPC stream client interceptor |
| `Close()` | Release resources |

## License

MIT
