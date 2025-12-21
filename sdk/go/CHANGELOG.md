# Changelog

All notable changes to the Auth Platform Go SDK will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-12-20

### Added

- **PKCE Support**: Full PKCE (Proof Key for Code Exchange) implementation
  - `GeneratePKCE()` generates verifier/challenge pairs
  - `VerifyPKCE()` validates verifier against challenge
  - `BuildAuthorizationURL()` supports PKCE parameters
  - `ExchangeCode()` supports code verifier

- **DPoP Support**: Sender-constrained token binding (RFC 9449)
  - `GenerateES256KeyPair()` and `GenerateRS256KeyPair()` for key generation
  - `NewDPoPProver()` creates DPoP proof generator
  - `WithDPoPProver()` client option for custom prover
  - `DPoPEnabled` config option for automatic DPoP
  - Automatic DPoP proof generation in token requests

- **Retry Policy**: Configurable retry with exponential backoff
  - `NewRetryPolicy()` with functional options
  - `WithRetryPolicy()` client option
  - Respects `Retry-After` header
  - Configurable jitter, delays, and retry counts

- **Observability Integration**: OpenTelemetry tracing and structured logging
  - `WithTracer()` for custom OpenTelemetry tracer
  - `WithLogger()` for custom structured logger
  - Automatic span creation for token operations
  - Sensitive data filtering in logs and traces

- **Generic Result Types**: Functional error handling
  - `Result[T]` with `Ok()`, `Err()`, `Map()`, `FlatMap()`, `Match()`
  - `Option[T]` with `Some()`, `None()`, `UnwrapOr()`

- **Structured Errors**: Enhanced error handling
  - `SDKError` type with error codes
  - Error codes: `ErrCodeDPoPRequired`, `ErrCodeDPoPInvalid`, `ErrCodePKCEInvalid`
  - `IsDPoPRequired()`, `IsDPoPInvalid()`, `IsPKCEInvalid()` helpers

- **Token Extraction**: Centralized token extraction
  - `TokenExtractor` interface
  - `HTTPTokenExtractor`, `GRPCTokenExtractor`, `CookieTokenExtractor`
  - Support for Bearer and DPoP token schemes

- **Middleware Enhancements**
  - `WithSkipPatterns()` for path-based authentication bypass
  - `WithErrorHandler()` for custom error responses
  - `WithCookieExtraction()` for cookie-based tokens

- **gRPC Interceptor Enhancements**
  - `WithGRPCSkipMethods()` for method-based bypass
  - `ShouldSkipMethod()` exported helper function
  - `MapToGRPCError()` exported for custom error mapping

### Changed

- **BREAKING**: Renamed `Option func(*Client)` to `ClientOption func(*Client)`
  - Avoids conflict with generic `Option[T]` type
  - Update: `authplatform.Option` → `authplatform.ClientOption`

- **Dependencies Updated**
  - Go 1.25 minimum version
  - `github.com/lestrrat-go/jwx/v3` (from v2)
  - `github.com/golang-jwt/jwt/v5` v5.2.2+
  - `google.golang.org/grpc` v1.70.0+
  - `golang.org/x/crypto` v0.31.0+
  - `go.opentelemetry.io/otel` for tracing

- **JWKS Cache**: Modernized with jwx/v3 API
  - Uses `jwk.Cache` with auto-refresh
  - Configurable refresh intervals
  - Fallback to cached keys on fetch failure

### Deprecated

- `MiddlewareFunc()` - Use `Middleware()` with functional options instead

### Migration Guide

#### Rename Option to ClientOption

```go
// Before
func WithCustomOption() authplatform.Option {
    return func(c *authplatform.Client) { ... }
}

// After
func WithCustomOption() authplatform.ClientOption {
    return func(c *authplatform.Client) { ... }
}
```

#### Update Middleware Usage

```go
// Before
handler := client.MiddlewareFunc(myHandler)

// After
handler := client.Middleware()(myHandler)

// Or with options
handler := client.Middleware(
    authplatform.WithSkipPatterns("/health"),
)(myHandler)
```

#### Enable DPoP

```go
// Via config
client, _ := authplatform.New(authplatform.Config{
    BaseURL:     "https://auth.example.com",
    ClientID:    "client-id",
    DPoPEnabled: true,
})

// Or via option
keyPair, _ := authplatform.GenerateES256KeyPair()
client, _ := authplatform.New(config,
    authplatform.WithDPoPProver(authplatform.NewDPoPProver(keyPair)),
)
```

## [1.0.0] - 2024-01-15

### Added

- Initial release
- Client credentials flow
- Token validation with JWKS
- HTTP middleware
- gRPC interceptors
- Functional options configuration
