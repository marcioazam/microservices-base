# Design Document: Go SDK State-of-Art 2025

## Overview

This design document specifies the modernization of the Auth Platform Go SDK to state-of-the-art standards as of December 2025. The modernization eliminates redundancy, centralizes logic, improves architecture, and ensures comprehensive test coverage with property-based testing.

### Goals

1. **Zero Redundancy**: Single source of truth for all logic
2. **Clean Architecture**: Domain-driven design with clear boundaries
3. **Type Safety**: Leverage Go 1.25 generics for compile-time safety
4. **Security First**: DPoP, PKCE, and sensitive data protection
5. **Comprehensive Testing**: Property-based tests for all core algorithms

### Technology Stack

- **Go Version**: 1.25 (latest stable)
- **JWT Library**: `github.com/lestrrat-go/jwx/v3` (latest)
- **Tracing**: OpenTelemetry 1.33+
- **Logging**: `log/slog` (standard library)
- **Testing**: `pgregory.net/rapid` for property-based testing
- **gRPC**: `google.golang.org/grpc` 1.70+

## Design Files

The design is split into focused documents:

- #[[file:design-architecture.md]] - Architecture and Target Structure
- #[[file:design-interfaces.md]] - Components and Interfaces
- #[[file:design-properties.md]] - Correctness Properties for Testing

## Current State Analysis

The current SDK has the following issues:
1. **Redundant error definitions**: Both sentinel errors and SDKError codes exist
2. **Mixed concerns**: Client, auth, and middleware logic interleaved
3. **Scattered tests**: Tests not mirroring source structure
4. **Duplicate token extraction**: Similar logic in HTTP and gRPC extractors

## Data Models

### Claims

```go
// Claims represents JWT claims.
type Claims struct {
    Subject   string   `json:"sub"`
    Issuer    string   `json:"iss"`
    Audience  []string `json:"aud"`
    ExpiresAt int64    `json:"exp"`
    IssuedAt  int64    `json:"iat"`
    Scope     string   `json:"scope,omitempty"`
    ClientID  string   `json:"client_id,omitempty"`
}
```

### Token Response

```go
// TokenResponse represents an OAuth token response.
type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token,omitempty"`
    Scope        string `json:"scope,omitempty"`
}
```

### Validation Options

```go
// ValidationOptions holds custom validation options.
type ValidationOptions struct {
    Audience       string
    Issuer         string
    RequiredClaims []string
    SkipExpiry     bool
}
```

## Error Handling

### Error Code to gRPC Status Mapping

| ErrorCode | gRPC Status Code |
|-----------|------------------|
| ErrCodeUnauthorized | codes.Unauthenticated |
| ErrCodeTokenMissing | codes.Unauthenticated |
| ErrCodeTokenInvalid | codes.Unauthenticated |
| ErrCodeTokenExpired | codes.Unauthenticated |
| ErrCodeValidation | codes.InvalidArgument |
| ErrCodeNetwork | codes.Unavailable |
| ErrCodeRateLimited | codes.ResourceExhausted |
| ErrCodeInvalidConfig | codes.InvalidArgument |
| ErrCodeDPoPRequired | codes.Unauthenticated |
| ErrCodeDPoPInvalid | codes.Unauthenticated |
| ErrCodePKCEInvalid | codes.InvalidArgument |

### Error Code to HTTP Status Mapping

| ErrorCode | HTTP Status Code |
|-----------|------------------|
| ErrCodeUnauthorized | 401 Unauthorized |
| ErrCodeTokenMissing | 401 Unauthorized |
| ErrCodeTokenInvalid | 401 Unauthorized |
| ErrCodeTokenExpired | 401 Unauthorized |
| ErrCodeValidation | 400 Bad Request |
| ErrCodeNetwork | 503 Service Unavailable |
| ErrCodeRateLimited | 429 Too Many Requests |
| ErrCodeInvalidConfig | 500 Internal Server Error |
| ErrCodeDPoPRequired | 401 Unauthorized |
| ErrCodeDPoPInvalid | 401 Unauthorized |
| ErrCodePKCEInvalid | 400 Bad Request |

## Testing Strategy

### Dual Testing Approach

The SDK uses both unit tests and property-based tests:

1. **Unit Tests**: Verify specific examples, edge cases, and error conditions
2. **Property Tests**: Verify universal properties across all inputs

### Property-Based Testing Configuration

- **Library**: `pgregory.net/rapid` v1.1.0
- **Minimum Iterations**: 100 per property test
- **Tag Format**: `Feature: go-sdk-state-of-art-2025, Property N: [property_text]`

### Test Organization

```
tests/
├── client/
│   ├── client_test.go           # Unit tests
│   └── client_prop_test.go      # Property tests
├── auth/
│   ├── pkce_test.go
│   ├── pkce_prop_test.go
│   ├── dpop_test.go
│   └── dpop_prop_test.go
├── token/
│   ├── extractor_test.go
│   ├── extractor_prop_test.go
│   ├── jwks_test.go
│   └── jwks_prop_test.go
├── middleware/
│   ├── http_test.go
│   ├── http_prop_test.go
│   ├── grpc_test.go
│   └── grpc_prop_test.go
├── errors/
│   ├── errors_test.go
│   └── errors_prop_test.go
├── types/
│   ├── result_test.go
│   ├── result_prop_test.go
│   ├── option_test.go
│   └── option_prop_test.go
├── retry/
│   ├── retry_test.go
│   └── retry_prop_test.go
└── integration/
    └── oauth_flows_test.go
```

### Coverage Requirements

- Core modules (errors, types, auth, token): 80%+ coverage
- Middleware: 75%+ coverage
- Integration tests: Cover all OAuth flows
