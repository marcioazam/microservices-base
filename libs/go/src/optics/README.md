# Optics Package

Functional optics for immutable data manipulation in Go with generics.

## Overview

This package provides composable optics for accessing and modifying nested data structures immutably:

- **Lens**: Focus on a specific field of a product type (struct)
- **Prism**: Focus on a variant of a sum type (union/enum)
- **Optional**: Focus on a value that may not exist
- **Iso**: Bidirectional transformation between two types

## Installation

```go
import "github.com/authcorp/libs/go/src/optics"
```

## Lens

A Lens focuses on a specific part of a data structure, providing get and set operations.

```go
type User struct {
    Name    string
    Address Address
}

type Address struct {
    City string
}

// Create a lens for User.Name
nameLens := optics.NewLens(
    func(u User) string { return u.Name },
    func(u User, name string) User { u.Name = name; return u },
)

// Get value
name := nameLens.Get(user)

// Set value (returns new struct)
updated := nameLens.Set(user, "Alice")

// Modify value
uppercased := nameLens.Modify(user, strings.ToUpper)
```

### Lens Composition

```go
addressLens := optics.NewLens(
    func(u User) Address { return u.Address },
    func(u User, a Address) User { u.Address = a; return u },
)

cityLens := optics.NewLens(
    func(a Address) string { return a.City },
    func(a Address, city string) Address { a.City = city; return a },
)

// Compose to focus on User.Address.City
userCityLens := optics.Compose(addressLens, cityLens)
city := userCityLens.Get(user)
```

### Utility Lenses

Built-in lenses for common patterns:

```go
// Identity lens - returns the value unchanged
idLens := optics.Identity[User]()
user := idLens.Get(originalUser) // returns originalUser

// First/Second - access pair elements
type Pair = struct{ First int; Second string }
firstLens := optics.First[int, string]()
secondLens := optics.Second[int, string]()

pair := Pair{First: 42, Second: "hello"}
first := firstLens.Get(pair)   // 42
second := secondLens.Get(pair) // "hello"

// MapAt - access map value at key with default
mapLens := optics.MapAt("key", 0) // default value 0
m := map[string]int{"key": 42}
val := mapLens.Get(m)              // 42
updated := mapLens.Set(m, 100)     // {"key": 100}
missing := mapLens.Get(map[string]int{}) // 0 (default)

// SliceAt - access slice element at index with default
sliceLens := optics.SliceAt(1, -1) // default value -1
s := []int{10, 20, 30}
val := sliceLens.Get(s)            // 20
updated := sliceLens.Set(s, 99)    // [10, 99, 30]
outOfBounds := sliceLens.Get([]int{}) // -1 (default)
```

## Prism

A Prism focuses on a variant of a sum type, returning Option for safe access.

```go
// Example: Focus on the "success" case of a Result-like type
type Response struct {
    Success *SuccessData
    Error   *ErrorData
}

successPrism := optics.NewPrism(
    func(r Response) functional.Option[*SuccessData] {
        if r.Success != nil {
            return functional.Some(r.Success)
        }
        return functional.None[*SuccessData]()
    },
    func(s *SuccessData) Response {
        return Response{Success: s}
    },
)

// Get returns Option (safe access)
opt := successPrism.GetOption(response)
if opt.IsSome() {
    data := opt.Unwrap()
}

// Modify only if variant matches
modified := successPrism.Modify(response, transformSuccess)

// Set only if variant matches
updated := successPrism.Set(response, newSuccessData)
```

### SomePrism

Built-in prism for `Option[T]` focusing on the Some case:

```go
somePrism := optics.SomePrism[int]()
opt := functional.Some(42)

// Get the inner value as Option
inner := somePrism.GetOption(opt) // Some(42)

// Modify the inner value
doubled := somePrism.Modify(opt, func(x int) int { return x * 2 })
```

### Prism Composition

```go
outerPrism := optics.NewPrism(...)
innerPrism := optics.NewPrism(...)

composed := optics.ComposePrism(outerPrism, innerPrism)
```

## Iso (Isomorphism)

An Iso represents a bidirectional, lossless transformation between two types.

```go
// Celsius <-> Fahrenheit conversion
tempIso := optics.NewIso(
    func(c float64) float64 { return c*9/5 + 32 },  // Celsius to Fahrenheit
    func(f float64) float64 { return (f - 32) * 5/9 }, // Fahrenheit to Celsius
)

fahrenheit := tempIso.Get(100.0)    // 212.0
celsius := tempIso.Reverse(212.0)   // 100.0

// Convert Iso to Lens or Prism
lens := tempIso.ToLens()
prism := tempIso.ToPrism()
```

### Iso Composition

```go
iso1 := optics.NewIso(aToB, bToA)
iso2 := optics.NewIso(bToC, cToB)

composed := optics.ComposeIso(iso1, iso2) // A <-> C
```

## Optional

An Optional is like a Lens but the value may not exist.

```go
// Access slice element by index
indexOpt := optics.Index[int](2)

slice := []int{1, 2, 3, 4, 5}
opt := indexOpt.GetOption(slice) // Some(3)

// Modify if exists
modified := indexOpt.Modify(slice, func(x int) int { return x * 10 })
// Result: [1, 2, 30, 4, 5]

// Out of bounds returns None
emptySlice := []int{}
opt := indexOpt.GetOption(emptySlice) // None
```

### Map Access with At

```go
atKey := optics.At[string, int]("count")

m := map[string]int{"count": 42}
opt := atKey.Get(m) // Some(42)

// Set value
updated := atKey.Set(m, functional.Some(100))

// Remove key
removed := atKey.Set(m, functional.None[int]())
```

## Conversions

```go
// Lens to Optional (always succeeds)
opt := optics.LensToOptional(lens)

// Iso to Lens
lens := iso.ToLens()

// Iso to Prism
prism := iso.ToPrism()
```

## Best Practices

1. **Immutability**: All operations return new values, never mutate
2. **Composition**: Build complex optics from simple ones
3. **Type Safety**: Use generics for compile-time type checking
4. **Option for Safety**: Prism and Optional use Option for safe access

## See Also

- [functional package](../functional/README.md) - Option and Result types
- [collections package](../collections/README.md) - Generic collections
