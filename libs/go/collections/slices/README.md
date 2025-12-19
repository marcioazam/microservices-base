# Slices

Generic slice utilities for Go.

## Category: Collections

## Installation

```go
import "github.com/auth-platform/libs/go/slices"
```

## Usage

```go
numbers := []int{1, 2, 3, 4, 5}

// Map - transform each element
doubled := slices.Map(numbers, func(x int) int { return x * 2 })
// [2, 4, 6, 8, 10]

// Filter - keep matching elements
evens := slices.Filter(numbers, func(x int) bool { return x%2 == 0 })
// [2, 4]

// Reduce - aggregate to single value
sum := slices.Reduce(numbers, 0, func(acc, x int) int { return acc + x })
// 15

// Find - get first matching element
found := slices.Find(numbers, func(x int) bool { return x > 3 })
// Some(4)

// Any/All - check conditions
hasEven := slices.Any(numbers, func(x int) bool { return x%2 == 0 })
// true

// GroupBy - group by key
grouped := slices.GroupBy(users, func(u User) string { return u.Role })
// map[string][]User

// Chunk - split into chunks
chunks := slices.Chunk(numbers, 2)
// [[1,2], [3,4], [5]]

// Flatten - flatten nested slices
flat := slices.Flatten([][]int{{1,2}, {3,4}})
// [1, 2, 3, 4]
```

## API

| Function | Description |
|----------|-------------|
| `Map[T,R](slice, fn)` | Transform each element |
| `Filter[T](slice, predicate)` | Keep matching elements |
| `Reduce[T,R](slice, initial, fn)` | Aggregate to single value |
| `Find[T](slice, predicate)` | Find first match (returns Option) |
| `Any[T](slice, predicate)` | Check if any match |
| `All[T](slice, predicate)` | Check if all match |
| `GroupBy[T,K](slice, keyFn)` | Group by key |
| `Partition[T](slice, predicate)` | Split into matching/non-matching |
| `Chunk[T](slice, size)` | Split into chunks |
| `Flatten[T](slices)` | Flatten nested slices |
