# Functional

Functional programming types and utilities for Go.

## Packages

| Package | Description |
|---------|-------------|
| `either` | Either type (Left/Right) |
| `iterator` | Iterator pattern implementation |
| `lazy` | Lazy evaluation |
| `option` | Option type (Some/None) |
| `pipeline` | Pipeline pattern for data processing |
| `result` | Result type (Ok/Err) |
| `stream` | Stream processing |
| `tuple` | Tuple types |

## Unified Module (src/functional)

The consolidated `src/functional` module provides all functional types in a single package with a unified `Functor` interface.

### Functor Interface

All functional types (`Option`, `Result`, `Either`) implement the `Functor` interface for consistent mapping operations:

```go
// Functor represents types that can be mapped over.
type Functor[A any] interface {
    Map(fn func(A) A) Functor[A]
}
```

### Type Conversions

Seamless conversion between `Either[error, T]` and `Result[T]`:

```go
// Either to Result
result := EitherToResult(either)

// Result to Either  
either := ResultToEither(result)
```

### Usage (Consolidated Module)

```go
import "github.com/auth-platform/libs/go/src/functional"

// Option
opt := functional.Some(42)
none := functional.None[int]()

// Result
ok := functional.Ok("success")
err := functional.Err[string](errors.New("failed"))

// Either
right := functional.Right[error, int](42)
left := functional.Left[error, int](errors.New("error"))

// Type conversions
result := functional.EitherToResult(right)
either := functional.ResultToEither(ok)
```

## Tuple Types

The `tuple` package provides generic tuple types with public fields for direct access:

```go
import "github.com/auth-platform/libs/go/functional/tuple"

// Create a pair
pair := tuple.NewPair("key", 42)
fmt.Println(pair.First, pair.Second) // "key" 42

// Unpack values
key, value := pair.Unpack()

// Swap values
swapped := pair.Swap() // Pair[int, string]{42, "key"}

// Zip slices into pairs
names := []string{"a", "b"}
values := []int{1, 2}
pairs := tuple.Zip(names, values) // []Pair[string, int]

// Enumerate with index
items := []string{"x", "y", "z"}
indexed := tuple.Enumerate(items) // []Pair[int, string]{{0,"x"}, {1,"y"}, {2,"z"}}
```

## Usage (Legacy Packages)

```go
import "github.com/auth-platform/libs/go/functional/option"
import "github.com/auth-platform/libs/go/functional/result"
```
