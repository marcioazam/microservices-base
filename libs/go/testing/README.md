# Testing

Test utilities and helpers for writing effective unit and property-based tests.

## Packages

| Package | Description |
|---------|-------------|
| `testutil` | Test generators, helpers, and mocks |

## Consolidated Module (src/testing)

The `src/testing` module provides a unified testing toolkit with assertions, builders, generators, and sync utilities.

### Assert

Fluent assertion helpers for cleaner test code:

```go
import testhelpers "github.com/auth-platform/libs/go/src/testing"

func TestExample(t *testing.T) {
    assert := testhelpers.NewAssert(t)
    
    // Value assertions
    assert.Equal(expected, actual)
    assert.NotEqual(a, b)
    assert.True(condition, "optional message")
    assert.False(condition)
    
    // Nil checks
    assert.Nil(value)
    assert.NotNil(value)
    
    // Error assertions
    assert.NoError(err)
    assert.Error(err)
    
    // Collection assertions
    assert.Len(slice, 5)
    assert.Contains(str, "substring")
    
    // Panic assertions
    assert.Panics(func() { panic("boom") })
    assert.NoPanic(func() { /* safe code */ })
}
```

### Builder

Generic fixture builder for test data:

```go
import testhelpers "github.com/auth-platform/libs/go/src/testing"

type User struct {
    ID    string
    Name  string
    Email string
}

func TestWithBuilder(t *testing.T) {
    user := testhelpers.NewBuilder[User]().
        With(func(u *User) {
            u.ID = "123"
            u.Name = "Test User"
            u.Email = "test@example.com"
        }).
        Build()
    
    // Or start from an existing value
    modified := testhelpers.NewBuilderFrom(user).
        With(func(u *User) { u.Name = "Modified" }).
        Build()
}
```

### TestFixture

Manage test setup and cleanup:

```go
import testhelpers "github.com/auth-platform/libs/go/src/testing"

func TestWithFixture(t *testing.T) {
    config := testhelpers.DefaultTestConfig()
    fixture := testhelpers.NewTestFixture(config)
    defer fixture.Cleanup()
    
    // Register cleanup functions
    fixture.AddCleanup(func() {
        // cleanup resources
    })
}
```

## Legacy Package Usage

```go
import "github.com/auth-platform/libs/go/testing/testutil"
```
