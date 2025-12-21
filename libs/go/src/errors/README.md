# Errors

Typed error handling with HTTP/gRPC mapping. Supports Go 1.25+ features including generic error type assertions.

## Features

- `AppError` - Standard error type with code, message, details
- Error wrapping with `Wrap()`, `RootCause()`, `Chain()`
- HTTP status mapping via `HTTPStatus()`
- gRPC code mapping via `GRPCCode()`
- API response with PII redaction via `ToAPIResponse()`
- Go 1.25+ generic `AsType[T]` for type-safe error assertions
- Re-exported standard library functions (`Is`, `As`, `Unwrap`, `Join`)

## Usage

### Basic Errors

```go
import "github.com/authcorp/libs/go/src/errors"

// Create typed errors
err := errors.NotFound("user")
err = errors.Validation("invalid input").WithDetail("field", "email")

// HTTP/gRPC mapping
status := err.HTTPStatus() // 404
code := err.GRPCCode()     // codes.NotFound
```

### Error Wrapping

```go
// Wrap errors with context
wrapped := errors.Wrap(err, "failed to fetch user")

// Get root cause
root := errors.RootCause(wrapped)

// Build error chain
chain := errors.Chain(err1, err2, err3)
```

### Generic Type Assertions (Go 1.25+)

```go
// Type-safe error assertion using AsType[T]
if appErr, ok := errors.AsType[*errors.AppError](err); ok {
    log.Error("app error", "code", appErr.Code)
}

// Re-exported standard library functions
if errors.Is(err, ErrNotFound) {
    // handle not found
}

var target *MyError
if errors.As(err, &target) {
    // use target
}
```

### Must Helper

```go
// Panic on error (useful for initialization)
config := errors.Must(LoadConfig())
```

### API Response

```go
// Convert to API response with PII redaction
response := err.ToAPIResponse()
// Returns sanitized error for client consumption
```

## Error Types

| Constructor | HTTP Status | gRPC Code | Use Case |
|-------------|-------------|-----------|----------|
| `Validation()` | 400 | InvalidArgument | Input validation failed |
| `BadRequest()` | 400 | InvalidArgument | Malformed request |
| `NotFound()` | 404 | NotFound | Resource not found |
| `Unauthorized()` | 401 | Unauthenticated | Authentication required |
| `Forbidden()` | 403 | PermissionDenied | Access denied |
| `Conflict()` | 409 | AlreadyExists | Resource conflict |
| `TooManyRequests()` | 429 | ResourceExhausted | Rate limit exceeded |
| `Internal()` | 500 | Internal | Server error |
| `Unavailable()` | 503 | Unavailable | Service unavailable |
| `Timeout()` | 504 | DeadlineExceeded | Operation timeout |
| `NotImplemented()` | 501 | Unimplemented | Feature not implemented |
| `Dependency()` | 502 | Unavailable | Downstream service error |
| `BusinessRule()` | 422 | FailedPrecondition | Business rule violation |
| `InvalidState()` | 409 | FailedPrecondition | Invalid state transition |

## Re-exported Functions

For convenience, standard library error functions are re-exported:

```go
errors.Is(err, target)     // Check error equality
errors.As(err, &target)    // Type assertion
errors.Unwrap(err)         // Get wrapped error
errors.Join(err1, err2)    // Combine errors
```
