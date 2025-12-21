# Collections

Generic data structures and collection utilities for Go.

## Packages

### LRU Cache (`collections.LRUCache`)

Full-featured thread-safe LRU cache with TTL, statistics, and Go 1.23+ iterator support:

```go
import "github.com/authcorp/libs/go/src/collections"

cache := collections.NewLRUCache[string, int](100).
    WithTTL(5 * time.Minute).
    WithEvictCallback(func(k string, v int) {
        log.Printf("evicted: %s", k)
    })

cache.Put("key", 42)
cache.PutWithTTL("temp", 100, time.Second)

if opt := cache.Get("key"); opt.IsSome() {
    value := opt.Unwrap() // 42
}

// Lazy initialization
value := cache.GetOrCompute("computed", func() int {
    return expensiveComputation()
})

// Statistics
stats := cache.Stats() // hits, misses, evictions, hit_rate

// Go 1.23+ iterator
for k, v := range cache.All() {
    fmt.Printf("%s: %d\n", k, v)
}

// Cleanup expired entries
removed := cache.Cleanup()
```

### Simple LRU Cache (`lru.Cache`)

Lightweight thread-safe LRU cache without TTL or statistics:

```go
import "github.com/authcorp/libs/go/src/collections/lru"

cache := lru.New[string, int](100)
cache.WithEvictCallback(func(k string, v int) {
    log.Printf("evicted: %s", k)
})

cache.Set("key", 42)

if value, ok := cache.Get("key"); ok {
    fmt.Println(value) // 42
}

// Peek without updating LRU order
if value, ok := cache.Peek("key"); ok {
    fmt.Println(value)
}

// Get or set atomically
value, existed := cache.GetOrSet("new", 100)

// Resize cache
cache.Resize(50)

// Get all keys/values (most to least recently used)
keys := cache.Keys()
values := cache.Values()
```

### Priority Queue

Heap-based generic priority queue:

```go
import "github.com/authcorp/libs/go/src/collections/pqueue"

pq := pqueue.New[string](func(a, b string) bool {
    return a < b // min-heap by string comparison
})

pq.Push("banana")
pq.Push("apple")
pq.Push("cherry")

item := pq.Pop() // "apple" (smallest)
```

### Queue (FIFO)

Thread-safe FIFO queue:

```go
import "github.com/authcorp/libs/go/src/collections/queue"

q := queue.New[int]()
q.Enqueue(1)
q.Enqueue(2)

value, ok := q.Dequeue() // 1, true
```

### Set

Thread-safe generic hash set with comprehensive set operations:

```go
import "github.com/authcorp/libs/go/src/collections"

// Create sets
set := collections.NewSet[string]()
set.Add("a")
set.Add("b")

// Variadic constructor
set2 := collections.SetOf("x", "y", "z")

if set.Contains("a") {
    // ...
}

// Set operations
union := set.Union(otherSet)
intersection := set.Intersection(otherSet)
difference := set.Difference(otherSet)
symDiff := set.SymmetricDifference(otherSet) // elements in either but not both

// Set relationships
isSubset := set.IsSubset(otherSet)
isSuperset := set.IsSuperset(otherSet)
isEqual := set.Equal(otherSet)

// Functional operations
clone := set.Clone()
filtered := set.Filter(func(s string) bool {
    return len(s) > 1
})

// Transform elements (returns new set with different type)
lengths := collections.SetMap(set, func(s string) int {
    return len(s)
})

// Iterate
set.ForEach(func(item string) {
    fmt.Println(item)
})

// Go 1.23+ iterator
for item := range set.All() {
    fmt.Println(item)
}

// Convert to slice
items := set.ToSlice()
```

### Maps Utilities

Generic map operations:

```go
import "github.com/authcorp/libs/go/src/collections/maps"

m := map[string]int{"a": 1, "b": 2}

keys := maps.Keys(m)
values := maps.Values(m)
filtered := maps.Filter(m, func(k string, v int) bool {
    return v > 1
})
```

## Choosing an LRU Implementation

| Feature | `collections.LRUCache` | `lru.Cache` |
|---------|------------------------|-------------|
| TTL support | ✅ Per-entry and default | ❌ |
| Statistics | ✅ Hits, misses, evictions | ❌ |
| `Option[V]` returns | ✅ Type-safe | ❌ `(V, bool)` |
| Go 1.23+ iterators | ✅ `All()` | ✅ `All()` |
| `GetOrCompute` | ✅ | ❌ |
| `GetOrSet` | ❌ | ✅ |
| `Peek` | ✅ `Option[V]` | ✅ `(V, bool)` |
| `Resize` | ✅ | ✅ |
| Dependencies | `functional` package | None |

Use `collections.LRUCache` for production caches needing TTL, monitoring, and functional patterns.
Use `lru.Cache` for simple caching without external dependencies.

## Go 1.23+ Iterator Support

All collections support the new Go 1.23 range-over-func iterators:

```go
// LRU Cache
for k, v := range cache.All() {
    fmt.Printf("%s: %d\n", k, v)
}

// Set
for item := range set.All() {
    fmt.Println(item)
}

// Queue
for item := range queue.All() {
    fmt.Println(item)
}

// Priority Queue (in priority order)
for item := range pqueue.All() {
    fmt.Println(item)
}
```
