# Codec

Unified encoding/decoding with multiple formats. This is the single authoritative codec implementation (consolidated from utils/).

## Features

- JSON encoding/decoding with pretty print options
- YAML encoding/decoding for configuration
- Base64 encoding (standard and URL-safe variants)
- Generic type-safe `TypedCodec[T]` interface
- `functional.Result[T]` integration for error handling

## Usage

### Basic Codecs

```go
import "github.com/authcorp/libs/go/src/codec"

// JSON encoding
jsonCodec := codec.NewJSONCodec().WithPretty()
data, err := jsonCodec.Encode(myStruct)

// YAML encoding
yamlCodec := codec.NewYAMLCodec().WithIndent(4)
data, err := yamlCodec.Encode(config)

// Base64 encoding
b64 := codec.NewBase64Codec().WithURLSafe()
encoded := b64.Encode([]byte("data"))
```

### Type-Safe Codecs

```go
// Generic type-safe JSON codec
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

userCodec := codec.NewTypedJSONCodec[User]()
data, err := userCodec.Encode(user)
decoded, err := userCodec.Decode(data)  // Returns User, not any
```

### Result-Based Error Handling

```go
import "github.com/authcorp/libs/go/src/functional"

userCodec := codec.NewTypedJSONCodec[User]()

// Encode with Result
result := codec.EncodeResult(userCodec, user)
if result.IsOk() {
    data := result.Unwrap()
}

// Decode with Result
result := codec.DecodeResult(userCodec, jsonData)
result.Match(
    func(u User) { /* success */ },
    func(err error) { /* failure */ },
)
```

### Convenience Functions

```go
// JSON
data, err := codec.EncodeJSON(v)
v, err := codec.DecodeJSON[MyType](data)

// YAML
data, err := codec.EncodeYAML(v)
v, err := codec.DecodeYAML[MyType](data)

// Base64
encoded := codec.Base64Encode(data)
decoded, err := codec.Base64Decode(encoded)

// URL-safe Base64
encoded := codec.Base64URLEncode(data)
decoded, err := codec.Base64URLDecode(encoded)
```

## Interfaces

### Codec (untyped)

```go
type Codec interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

### TypedCodec[T] (generic)

```go
type TypedCodec[T any] interface {
    Encode(T) ([]byte, error)
    Decode([]byte) (T, error)
}
```
