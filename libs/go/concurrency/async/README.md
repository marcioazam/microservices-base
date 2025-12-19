# Async

Async utilities for concurrent operations in Go.

## Category: Concurrency

## Installation

```go
import "github.com/auth-platform/libs/go/async"
```

## Usage

### Parallel Execution

```go
// Run multiple operations in parallel
results, err := async.Parallel(ctx,
    func() (int, error) { return fetchA() },
    func() (int, error) { return fetchB() },
    func() (int, error) { return fetchC() },
)
// results = []int{resultA, resultB, resultC}
```

### Race

```go
// Return first successful result
result, err := async.Race(ctx,
    func() (string, error) { return fetchFromPrimary() },
    func() (string, error) { return fetchFromBackup() },
)
```

### WithTimeout

```go
// Execute with timeout
result, err := async.WithTimeout(ctx, 5*time.Second, func() (string, error) {
    return slowOperation()
})
```

### Go (Fire and Forget)

```go
// Start goroutine with panic recovery
async.Go(func() {
    processInBackground()
})
```

### Collect

```go
// Collect results from channel
results := async.Collect(ctx, resultChan, 10) // collect up to 10
```

### FanOut

```go
// Distribute work across workers
results := async.FanOut(ctx, items, 5, func(item Item) (Result, error) {
    return process(item)
})
```

## API

| Function | Description |
|----------|-------------|
| `Parallel[T](ctx, fns...)` | Run functions in parallel, collect all results |
| `Race[T](ctx, fns...)` | Return first successful result |
| `WithTimeout[T](ctx, duration, fn)` | Execute with timeout |
| `Go(fn)` | Start goroutine with panic recovery |
| `Collect[T](ctx, ch, max)` | Collect from channel |
| `FanOut[T,R](ctx, items, workers, fn)` | Parallel map with worker pool |
