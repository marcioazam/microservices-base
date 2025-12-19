# Functional Types Module

Unified functional programming types for Go with consistent interfaces.

## Types

### Option[T]
Represents an optional value that may or may not be present.

```go
opt := functional.Some(42)
none := functional.None[int]()

// Pattern matching
opt.Match(
    func(v int) { fmt.Println("Got:", v) },
    func() { fmt.Println("Nothing") },
)

// Safe access
value := opt.UnwrapOr(0)
```

### Result[T]
Represents the outcome of an operation that may fail.

```go
result := functional.Ok(42)
err := functional.Err[int](errors.New("failed"))

// Pattern matching
result.Match(
    func(v int) { fmt.Println("Success:", v) },
    func(e error) { fmt.Println("Error:", e) },
)

// Chaining
mapped := functional.MapResult(result, func(v int) string {
    return fmt.Sprintf("Value: %d", v)
})
```

### Either[L, R]
Represents a value that can be one of two types.

```go
right := functional.Right[error](42)
left := functional.Left[error, int](errors.New("error"))

// Conversion
result := functional.EitherToResult(right)
either := functional.ResultToEither(result)
```

### Iterator[T]
Lazy iteration using Go 1.23+ range functions.

```go
iter := functional.FromSlice([]int{1, 2, 3, 4, 5})
doubled := functional.Map(iter, func(x int) int { return x * 2 })
filtered := functional.Filter(doubled, func(x int) bool { return x > 4 })
result := functional.Collect(filtered) // [6, 8, 10]
```

### Stream[T]
Lazy, potentially infinite sequences with memoization.

```go
stream := functional.Iterate(1, func(x int) int { return x * 2 })
first5 := functional.TakeStream(stream, 5)
values := functional.CollectStream(first5) // [1, 2, 4, 8, 16]
```

## Functor Interface

All types implement the Functor interface for consistent mapping:

```go
type Functor[A any] interface {
    Map(fn func(A) A) Functor[A]
}
```
