# Concurrency Module

Generic concurrency primitives with Result[T] integration.

## Future[T]

Asynchronous computation with context support.

```go
// Create future from async operation
future := concurrency.NewFuture(func() (string, error) {
    return fetchData()
})

// Wait for result
result := future.Wait()
if result.IsOk() {
    fmt.Println("Got:", result.Unwrap())
}

// Wait with context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
result := future.WaitContext(ctx)

// Check if done without blocking
if future.IsDone() {
    opt := future.Result() // Option[Result[T]]
}
```

### Future Combinators

```go
// Map transforms the value
mapped := concurrency.Map(future, func(s string) int {
    return len(s)
})

// FlatMap chains futures
chained := concurrency.FlatMap(future, func(s string) *concurrency.Future[int] {
    return concurrency.NewFuture(func() (int, error) {
        return processString(s)
    })
})

// Wait for all futures
results := concurrency.All(future1, future2, future3)

// Race - first to complete wins
result := concurrency.Race(future1, future2, future3)

// Create completed futures
resolved := concurrency.Resolve(42)
rejected := concurrency.Reject[int](errors.New("failed"))
```

## WorkerPool[T, R]

Generic worker pool with backpressure.

```go
pool := concurrency.NewWorkerPool(4, func(task string) (int, error) {
    return processTask(task)
})

pool.Start()

// Submit tasks
pool.Submit("task1")
pool.Submit("task2")

// Read results
for result := range pool.Results() {
    if result.IsOk() {
        fmt.Println("Result:", result.Unwrap())
    }
}

pool.StopAndWait()
```

## ErrGroup

Error group for coordinated goroutines.

```go
g := concurrency.NewErrGroup(ctx)

g.Go(func() error {
    return task1()
})

g.Go(func() error {
    return task2()
})

// Wait returns first error
if err := g.Wait(); err != nil {
    log.Printf("Failed: %v", err)
}
```
