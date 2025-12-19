# Result[T]

A generic Result monad for explicit error handling in Go.

## Category: Functional Types

## Installation

```go
import "github.com/auth-platform/libs/go/result"
```

## Usage

```go
// Create success result
r := result.Ok(42)

// Create error result
r := result.Err[int](errors.New("failed"))

// Map over success value
doubled := r.Map(func(x int) int { return x * 2 })

// Chain operations
result := result.Ok(10).
    FlatMap(func(x int) result.Result[int] {
        if x > 0 {
            return result.Ok(x * 2)
        }
        return result.Err[int](errors.New("negative"))
    })

// Get value with default
value := r.GetOrElse(0)

// Check state
if r.IsOk() {
    fmt.Println(r.Unwrap())
}
```

## API

| Function | Description |
|----------|-------------|
| `Ok[T](value T)` | Create success result |
| `Err[T](err error)` | Create error result |
| `Map(fn)` | Transform success value |
| `FlatMap(fn)` | Chain Result-returning functions |
| `GetOrElse(default)` | Get value or default |
| `IsOk()` | Check if success |
| `IsErr()` | Check if error |
| `Unwrap()` | Get value (panics on error) |
