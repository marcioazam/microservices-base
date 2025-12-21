# Patterns Package

Generic design patterns for Go applications with type-safe functional programming support.

## Overview

This package provides reusable design patterns that leverage Go generics for type safety:

- **Repository Pattern**: Generic CRUD operations with `Option[T]` and `Result[T]` returns
- **Cached Repository**: Repository wrapper with LRU caching support
- **Pagination**: Type-safe page handling for list operations

## Installation

```go
import "github.com/authcorp/libs/go/src/patterns"
```

## Repository Pattern

### Basic Interface

```go
type Repository[T any, ID comparable] interface {
    Get(ctx context.Context, id ID) functional.Option[T]
    Save(ctx context.Context, entity T) functional.Result[T]
    Delete(ctx context.Context, id ID) error
    List(ctx context.Context) functional.Result[[]T]
    Exists(ctx context.Context, id ID) bool
}
```

### Usage Example

```go
// Define your entity
type User struct {
    ID    string
    Name  string
    Email string
}

// Create an in-memory repository for testing
repo := patterns.NewInMemoryRepository(func(u User) string {
    return u.ID
})

// Save an entity
result := repo.Save(ctx, User{ID: "1", Name: "John", Email: "[email]"})
if result.IsErr() {
    log.Fatal(result.UnwrapErr())
}

// Get an entity - returns Option[T]
opt := repo.Get(ctx, "1")
if opt.IsSome() {
    user := opt.Unwrap()
    fmt.Printf("Found user: %s\n", user.Name)
} else {
    fmt.Println("User not found")
}

// Check existence
if repo.Exists(ctx, "1") {
    fmt.Println("User exists")
}

// List all entities
listResult := repo.List(ctx)
if listResult.IsOk() {
    users := listResult.Unwrap()
    fmt.Printf("Total users: %d\n", len(users))
}
```

## Cached Repository

Wraps any repository with LRU caching for improved read performance.

### Usage

```go
// Create inner repository (e.g., database-backed)
innerRepo := NewDatabaseRepository(db)

// Create cache
cache := collections.NewLRUCache[string, *User](1000).
    WithTTL(5 * time.Minute)

// Wrap with caching
cachedRepo := patterns.NewCachedRepository(
    innerRepo,
    cache,
    func(u *User) string { return u.ID },
)

// Use like any repository - cache is transparent
opt := cachedRepo.Get(ctx, "user-123") // Cache miss -> DB lookup -> cache
opt = cachedRepo.Get(ctx, "user-123")  // Cache hit

// Check cache statistics
stats := cachedRepo.Stats()
fmt.Printf("Hit rate: %.2f%%\n", stats.HitRate * 100)

// Invalidate specific entry
cachedRepo.Invalidate("user-123")

// Clear entire cache
cachedRepo.InvalidateAll()
```

## Pagination

Type-safe pagination support for list operations.

### Page Structure

```go
type Page[T any] struct {
    Items      []T
    Page       int
    PageSize   int
    TotalItems int64
    TotalPages int
}
```

### Usage

```go
// Create a page of results
items := []User{{ID: "1"}, {ID: "2"}, {ID: "3"}}
page := patterns.NewPage(items, 1, 10, 100)

fmt.Printf("Page %d of %d\n", page.Page, page.TotalPages)
fmt.Printf("Showing %d items\n", len(page.Items))

if page.HasNext() {
    fmt.Println("More pages available")
}

if page.HasPrev() {
    fmt.Println("Previous pages available")
}

if page.IsEmpty() {
    fmt.Println("No items on this page")
}
```

## Read/Write Separation

For CQRS-style architectures, use the separated interfaces:

```go
type ReadRepository[T any, ID comparable] interface {
    Get(ctx context.Context, id ID) functional.Option[T]
    List(ctx context.Context) functional.Result[[]T]
    Exists(ctx context.Context, id ID) bool
}

type WriteRepository[T any, ID comparable] interface {
    Save(ctx context.Context, entity T) functional.Result[T]
    Delete(ctx context.Context, id ID) error
}
```

## Best Practices

1. **Use Option for nullable returns**: `Get()` returns `Option[T]` instead of `(T, error)` for cleaner null handling
2. **Use Result for fallible operations**: `Save()` returns `Result[T]` for functional error handling
3. **Implement ID extraction**: Provide an `IDExtractor` function when creating repositories
4. **Configure cache appropriately**: Set TTL and size based on your access patterns
5. **Monitor cache stats**: Track hit rates to optimize cache configuration

## Testing

Use `InMemoryRepository` for unit tests:

```go
func TestUserService(t *testing.T) {
    repo := patterns.NewInMemoryRepository(func(u User) string {
        return u.ID
    })
    
    service := NewUserService(repo)
    
    // Test your service...
    
    // Clear between tests
    repo.Clear()
}
```

## Thread Safety

- `InMemoryRepository`: Not thread-safe (use for testing only)
- `CachedRepository`: Thread-safe if underlying cache is thread-safe
- `LRUCache` from collections package: Thread-safe

## See Also

- [functional package](../functional/README.md) - Option and Result types
- [collections package](../collections/README.md) - LRU cache implementation
- [validation package](../validation/README.md) - Input validation
