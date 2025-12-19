# Collections Module

Generic, thread-safe collection types with unified interfaces.

## Set[T]

Thread-safe set implementation.

```go
set := collections.NewSet[int]()
set.Add(1)
set.Add(2)

if set.Contains(1) {
    fmt.Println("Found!")
}

// Set operations
other := collections.SetFrom([]int{2, 3})
union := set.Union(other)
intersection := set.Intersection(other)
```

## LRUCache[K, V]

Thread-safe LRU cache with TTL support.

```go
cache := collections.NewLRUCache[string, int](100).
    WithTTL(5 * time.Minute).
    WithEvictCallback(func(k string, v int) {
        log.Printf("Evicted: %s", k)
    })

cache.Put("key", 42)

if opt := cache.Get("key"); opt.IsSome() {
    fmt.Println("Value:", opt.Unwrap())
}

// Compute if absent
value := cache.GetOrCompute("key2", func() int {
    return expensiveComputation()
})
```

## Queue[T] and Stack[T]

Thread-safe FIFO queue and LIFO stack.

```go
queue := collections.NewQueue[int]()
queue.Enqueue(1)
queue.Enqueue(2)
opt := queue.Dequeue() // Some(1)

stack := collections.NewStack[int]()
stack.Push(1)
stack.Push(2)
opt := stack.Pop() // Some(2)
```

## PriorityQueue[T]

Thread-safe priority queue.

```go
pq := collections.NewPriorityQueue(func(a, b int) bool {
    return a < b // Min heap
})

pq.Push(3)
pq.Push(1)
pq.Push(2)

opt := pq.Pop() // Some(1)
```

## Iterator[T]

Lazy iteration with functional operations.

```go
iter := collections.FromSlice([]int{1, 2, 3, 4, 5})
result := collections.Collect(
    collections.Filter(
        collections.Map(iter, func(x int) int { return x * 2 }),
        func(x int) bool { return x > 4 },
    ),
) // [6, 8, 10]
```
