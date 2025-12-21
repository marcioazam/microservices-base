# Validation Package

Composable validators for Go with type-safe functional programming support.

## Overview

This package provides reusable, composable validators that can be combined to create complex validation rules:

- **Generic Validators**: Type-safe validators using Go generics
- **Duration Validators**: Time duration validation
- **Float Validators**: Floating-point number validation
- **String Validators**: String validation (length, pattern, etc.)
- **Composable Results**: Aggregate multiple validation errors

## Installation

```go
import "github.com/authcorp/libs/go/src/validation"
```

## Basic Usage

### Single Field Validation

```go
// Validate an integer is in range
validator := validation.InRange(1, 100)
if err := validator(50); err != nil {
    log.Fatal(err)
}

// Validate a string is not empty
strValidator := validation.NotEmpty[string]()
if err := strValidator("hello"); err != nil {
    log.Fatal(err)
}
```

### Duration Validation

```go
// Validate duration is within range
validator := validation.DurationRange(time.Second, 5*time.Minute)
if err := validator(30 * time.Second); err != nil {
    log.Fatal(err)
}

// Validate duration is positive
positiveValidator := validation.DurationPositive()
if err := positiveValidator(time.Second); err != nil {
    log.Fatal(err)
}

// Validate minimum duration
minValidator := validation.DurationMin(100 * time.Millisecond)
```

### Float Validation

```go
// Validate float is in range
validator := validation.FloatRange(0.0, 1.0)
if err := validator(0.5); err != nil {
    log.Fatal(err)
}

// Validate float is positive
positiveValidator := validation.FloatPositive()

// Validate float is non-negative
nonNegValidator := validation.FloatNonNegative()
```

### Composable Validation

```go
// Create a validation result
result := validation.NewResult()

// Add field validations
result.Merge(validation.Field("age", user.Age,
    validation.InRange(0, 150)))

result.Merge(validation.Field("email", user.Email,
    validation.NotEmpty[string](),
    validation.MaxLength(255)))

result.Merge(validation.Field("timeout", config.Timeout,
    validation.DurationRange(time.Second, time.Hour)))

// Check if valid
if !result.IsValid() {
    return fmt.Errorf("validation failed: %v", result.ErrorMessages())
}
```

## Available Validators

### Numeric Validators

| Validator | Description |
|-----------|-------------|
| `InRange(min, max)` | Value must be between min and max |
| `Min(min)` | Value must be >= min |
| `Max(max)` | Value must be <= max |
| `Positive()` | Value must be > 0 |
| `NonNegative()` | Value must be >= 0 |
| `IntRange(min, max)` | Alias for InRange with int |
| `IntMin(min)` | Alias for Min with int |
| `IntMax(max)` | Alias for Max with int |
| `IntPositive()` | Alias for Positive with int |

### Duration Validators

| Validator | Description |
|-----------|-------------|
| `DurationRange(min, max)` | Duration must be between min and max |
| `DurationMin(min)` | Duration must be >= min |
| `DurationMax(max)` | Duration must be <= max |
| `DurationPositive()` | Duration must be > 0 |
| `DurationNonZero()` | Duration must not be zero |

### Float Validators

| Validator | Description |
|-----------|-------------|
| `FloatRange(min, max)` | Float must be between min and max |
| `FloatMin(min)` | Float must be >= min |
| `FloatMax(max)` | Float must be <= max |
| `FloatPositive()` | Float must be > 0 |
| `FloatNonNegative()` | Float must be >= 0 |

### String Validators

| Validator | Description |
|-----------|-------------|
| `NotEmpty()` | String must not be empty |
| `MinLength(n)` | String must have at least n characters |
| `MaxLength(n)` | String must have at most n characters |
| `OneOf(values...)` | String must be one of the allowed values |
| `Pattern(regex)` | String must match the regex pattern |

## Struct Validation Example

```go
type CircuitBreakerConfig struct {
    FailureThreshold int
    SuccessThreshold int
    Timeout          time.Duration
    ProbeCount       int
}

func (c *CircuitBreakerConfig) Validate() error {
    result := validation.NewResult()

    result.Merge(validation.Field("failure_threshold", c.FailureThreshold,
        validation.InRange(1, 100)))

    result.Merge(validation.Field("success_threshold", c.SuccessThreshold,
        validation.InRange(1, 10)))

    result.Merge(validation.Field("timeout", c.Timeout,
        validation.DurationRange(time.Second, 5*time.Minute)))

    result.Merge(validation.Field("probe_count", c.ProbeCount,
        validation.InRange(1, 10)))

    // Cross-field validation
    if c.SuccessThreshold > c.FailureThreshold {
        result.AddFieldError("success_threshold",
            "cannot be greater than failure_threshold",
            "cross_field")
    }

    if !result.IsValid() {
        return fmt.Errorf("validation failed: %v", result.ErrorMessages())
    }
    return nil
}
```

## Custom Validators

Create custom validators by implementing the `Validator[T]` type:

```go
type Validator[T any] func(value T) *ValidationError

// Custom email validator
func Email() Validator[string] {
    return func(s string) *ValidationError {
        if !strings.Contains(s, "@") {
            return &ValidationError{
                Message: "must be a valid email address",
                Code:    "email_invalid",
            }
        }
        return nil
    }
}

// Usage
result.Merge(validation.Field("email", user.Email, Email()))
```

## Error Handling

### ValidationError

```go
type ValidationError struct {
    Message string
    Code    string
}
```

### ValidationResult

```go
result := validation.NewResult()

// Add errors
result.AddFieldError("field", "message", "code")

// Check validity
if result.IsValid() {
    // All validations passed
}

// Get error messages
messages := result.ErrorMessages()

// Get all errors
errors := result.Errors()
```

## Best Practices

1. **Use composable validators**: Combine simple validators for complex rules
2. **Validate early**: Validate at domain boundaries (API handlers, service methods)
3. **Return Result types**: Use `functional.Result[T]` for validation methods
4. **Cross-field validation**: Add after individual field validations
5. **Meaningful error codes**: Use consistent error codes for client handling

## Thread Safety

All validators are stateless and thread-safe.

## See Also

- [functional package](../functional/README.md) - Result type for error handling
- [patterns package](../patterns/README.md) - Repository pattern with validation
