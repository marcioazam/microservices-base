# Design: Architecture and Target Structure

## Target Architecture

```
sdk/go/
├── src/                          # Source code
│   ├── sdk.go                    # Public API entry point
│   ├── client/                   # Client package
│   │   ├── client.go             # Main client implementation
│   │   ├── config.go             # Configuration
│   │   └── options.go            # Functional options
│   ├── auth/                     # Authentication package
│   │   ├── pkce.go               # PKCE implementation
│   │   ├── dpop.go               # DPoP implementation
│   │   ├── dpop_jwk.go           # JWK utilities for DPoP
│   │   └── flows.go              # OAuth flows
│   ├── token/                    # Token handling package
│   │   ├── extractor.go          # Unified token extraction
│   │   ├── jwks.go               # JWKS cache
│   │   └── validator.go          # Token validation
│   ├── middleware/               # Middleware package
│   │   ├── http.go               # HTTP middleware
│   │   └── grpc.go               # gRPC interceptors
│   ├── errors/                   # Error handling package
│   │   ├── errors.go             # SDKError and codes
│   │   └── sanitize.go           # Error sanitization
│   ├── types/                    # Shared types package
│   │   ├── result.go             # Result[T] type
│   │   ├── option.go             # Option[T] type
│   │   └── claims.go             # JWT claims types
│   ├── retry/                    # Retry package
│   │   └── retry.go              # Retry logic with backoff
│   └── internal/                 # Internal packages
│       └── observability/        # Tracing and logging
│           ├── tracing.go
│           └── logging.go
├── tests/                        # Test code (mirrors src/)
│   ├── client/
│   ├── auth/
│   ├── token/
│   ├── middleware/
│   ├── errors/
│   ├── types/
│   ├── retry/
│   ├── integration/
│   └── property/                 # Property-based tests
└── examples/                     # Usage examples
```

## Package Responsibilities

### client/
- Main SDK client implementation
- Configuration management with environment variable support
- Functional options pattern for customization
- OAuth flow orchestration

### auth/
- PKCE code verifier/challenge generation (RFC 7636)
- DPoP proof generation and validation (RFC 9449)
- JWK thumbprint computation (RFC 7638)
- Key pair generation (ES256, RS256)

### token/
- Unified token extraction from HTTP/gRPC/cookies
- JWKS caching with automatic refresh
- JWT validation with configurable options
- Token scheme detection (Bearer, DPoP)

### middleware/
- HTTP middleware with skip patterns
- gRPC unary and stream interceptors
- Claims context propagation
- Error to status code mapping

### errors/
- Unified SDKError type with error codes
- Type-safe error checking helpers
- Error message sanitization
- Error chain support with wrapping

### types/
- Generic Result[T] for operation outcomes
- Generic Option[T] for optional values
- Functional transformations (Map, FlatMap)
- JWT claims types

### retry/
- Configurable retry policy
- Exponential backoff with jitter
- Retry-After header parsing
- Context cancellation support

### internal/observability/
- OpenTelemetry tracing integration
- Structured logging with log/slog
- Sensitive data filtering
- Span creation for major operations
