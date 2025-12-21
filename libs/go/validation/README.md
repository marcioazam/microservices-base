# Validation Library

Generic validation utilities for configuration and input validation in Go 1.23+.

## Features

- **Generic Validators**: Type-safe validators using Go generics
- **Composable**: Combine multiple validators with `Compose`
- **Builder Pattern**: Collect multiple validation errors
- **Common Validators**: Positive, NonNegative, InRange, NonEmpty, MinLength, MaxLength, OneOf, NotNil
- **Duration Validators**: PositiveDuration, DurationInRange

## Installation

```go
import "github.com/auth-platform/libs/go/validation"
```

## Usage

### Basic Validators

```go
// Positive number validator
err := validation.Positive[int]()("age", 25)

// Range validator
err := validation.InRange[int](1, 100)("score", 50)

// Non-empty string
err := validation.NonEmpty()("name", "John")

// One of allowed values
err := validation.OneOf("active", "inactive", "pending")("status", "active")
```

### Duration Validators

```go
// Positive duration
err := validation.PositiveDuration()("timeout", 30*time.Second)

// Duration in range
err := validation.DurationInRange(time.Second, time.Minute)("interval", 30*time.Second)
```

### Composing Validators

```go
validator := validation.Compose(
    validation.Positive[int](),
    validation.InRange[int](1, 100),
)
err := validator("score", 50)
```

### Builder Pattern

```go
builder := validation.NewBuilder()
builder.Validate(validation.Positive[int]()("age", user.Age))
builder.Validate(validation.NonEmpty()("name", user.Name))
builder.Validate(validation.InRange[int](0, 100)("score", user.Score))

if err := builder.Build(); err != nil {
    return err
}
```

### Custom Validators

```go
// Create a custom validator
func Email() validation.Validator[string] {
    return func(field string, value string) error {
        if !strings.Contains(value, "@") {
            return &validation.ValidationError{
                Field:   field,
                Message: "must be a valid email address",
            }
        }
        return nil
    }
}
```

## Dependencies

- Standard library only
