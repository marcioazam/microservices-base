# Functional

Functional programming primitives for Go with full generics support.

## Features

- **Either** - Left/Right discriminated union
- **Iterator** - Lazy iteration with Go 1.23+ `iter.Seq` support
- **Lazy** - Deferred evaluation
- **Option** - Optional values (Some/None)
- **Pipeline** - Function composition
- **Result** - Success/failure handling (Ok/Err)
- **Stream** - Lazy stream processing
- **Tuple** - Type-safe tuples (Pair, Triple, Quad)
- **Validated** - Error accumulation (applicative functor)

## Tuple Types

```go
import "github.com/authcorp/libs/go/src/functional"

// Pair - two values
pair := functional.NewPair("key", 42)
key, value := pair.Unpack()
swapped := pair.Swap() // Pair[int, string]

// Triple - three values
triple := functional.NewTriple("a", 1, true)
a, b, c := triple.Unpack()
pair := triple.ToPair() // drops third element

// Quad - four values
quad := functional.NewQuad("w", "x", "y", "z")
w, x, y, z := quad.Unpack()

// Transformations
mapped := functional.MapPairFirst(pair, strings.ToUpper)
mapped2 := functional.MapPairSecond(pair, func(n int) int { return n * 2 })
mapped3 := functional.MapPairBoth(pair, strings.ToUpper, func(n int) int { return n * 2 })
```

## Zip/Unzip Operations

```go
// Zip slices into pairs
names := []string{"Alice", "Bob"}
ages := []int{30, 25}
pairs := functional.Zip(names, ages) // []Pair[string, int]

// Zip with custom function
sums := functional.ZipWith([]int{1, 2}, []int{10, 20}, func(a, b int) int { return a + b })

// Zip three slices
triples := functional.Zip3(as, bs, cs) // []Triple[A, B, C]

// Unzip back to slices
names, ages := functional.Unzip(pairs)
as, bs, cs := functional.Unzip3(triples)

// Enumerate with indices
indexed := functional.EnumerateSlice(items) // []Pair[int, T]
```

## Go 1.23+ Iterator Support

```go
// Lazy iteration over zipped pairs
for pair := range functional.ZipIter(names, ages) {
    fmt.Printf("%s: %d\n", pair.First, pair.Second)
}

// Enumerate with index
for i, item := range functional.EnumerateIter(items) {
    fmt.Printf("[%d] %v\n", i, item)
}
```

## Validated (Error Accumulation)

`Validated` is an applicative functor that accumulates errors instead of short-circuiting on the first failure. Use it when you want to collect all validation errors at once.

```go
import "github.com/authcorp/libs/go/src/functional"

// Create valid/invalid results
valid := functional.Valid[error, int](42)
invalid := functional.Invalid[error, int](errors.New("must be positive"))

// Check validity
if valid.IsValid() {
    value := valid.GetValue()
}

// Get errors from invalid result
errors := invalid.GetErrors()

// Safe value access with default
value := invalid.GetOrElse(0)
```

### Combining Validated Values

```go
// Combine two validated values (accumulates errors from both)
nameV := functional.Valid[string, string]("Alice")
ageV := functional.Valid[string, int](30)

result := functional.CombineValidated(nameV, ageV, func(name string, age int) Person {
    return Person{Name: name, Age: age}
})

// Combine three validated values
result3 := functional.CombineValidated3(va, vb, vc, func(a, b, c A) D {
    return combine(a, b, c)
})
```

### Transformations

```go
// Map over valid value
doubled := functional.MapValidated(valid, func(n int) int { return n * 2 })

// Map over errors
mapped := functional.MapValidatedErrors(invalid, func(e error) string { return e.Error() })

// Fold to handle both cases
message := functional.FoldValidated(result,
    func(errs []error) string { return "Failed: " + errs[0].Error() },
    func(value int) string { return fmt.Sprintf("Success: %d", value) },
)
```

### Sequence and Traverse

```go
// Convert slice of Validated to Validated of slice
validations := []functional.Validated[error, int]{
    functional.Valid[error, int](1),
    functional.Valid[error, int](2),
    functional.Invalid[error, int](errors.New("bad")),
}
result := functional.SequenceValidated(validations) // Invalid with accumulated errors

// Apply validation function to each item
items := []string{"1", "2", "bad"}
result := functional.TraverseValidated(items, func(s string) functional.Validated[error, int] {
    n, err := strconv.Atoi(s)
    if err != nil {
        return functional.Invalid[error, int](err)
    }
    return functional.Valid[error, int](n)
})
```

### Converting Between Result and Validated

```go
// Result to Validated
result := functional.Ok(42)
validated := functional.ResultToValidated(result)

// Validated to Result (uses first error if invalid)
validated := functional.Valid[error, int](42)
result := functional.ValidatedToResult(validated)
```

### When to Use Validated vs Result

| Use Case | Type |
|----------|------|
| Fail fast on first error | `Result[T]` |
| Collect all validation errors | `Validated[E, A]` |
| Form validation | `Validated[E, A]` |
| Config validation | `Validated[E, A]` |
| Sequential operations | `Result[T]` |
| Parallel validation | `Validated[E, A]` |
