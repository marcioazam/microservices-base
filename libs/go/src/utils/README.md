# Utils Module

Common utility functions and types.

## Validation

Composable validation with error accumulation.

```go
validator := utils.NewValidator[string]("email").
    AddRule(utils.Required()).
    AddRule(utils.MinLength(5)).
    AddRule(utils.MaxLength(100))

result := validator.Validate(email)
if !result.IsValid() {
    for _, err := range result.Errors() {
        log.Printf("%s: %s", err.Field, err.Message)
    }
}
```

### Composing Validators

```go
v1 := utils.NewValidator[int]("age").AddRule(utils.Min(0))
v2 := utils.NewValidator[int]("age").AddRule(utils.Max(150))
combined := v1.And(v2)
```

### Custom Rules

```go
isEven := utils.Custom(
    func(v int) bool { return v%2 == 0 },
    "must be even",
    "even",
)

validator := utils.NewValidator[int]("number").AddRule(isEven)
```

## UUID

UUID v4 generation and parsing.

```go
uuid, err := utils.NewUUID()
fmt.Println(uuid.String()) // "550e8400-e29b-41d4-a716-446655440000"

parsed, err := utils.ParseUUID("550e8400-e29b-41d4-a716-446655440000")
```

## Codec

Generic encoding/decoding.

```go
codec := utils.NewJSONCodec[User]()

data, err := codec.Encode(user)
decoded, err := codec.Decode(data)

// With Result type
result := utils.EncodeResult(codec, user)
if result.IsOk() {
    data := result.Unwrap()
}
```

### Base64

```go
encoded := utils.Base64Encode(data)
decoded, err := utils.Base64Decode(encoded)

// URL-safe
urlEncoded := utils.Base64URLEncode(data)
urlDecoded, err := utils.Base64URLDecode(urlEncoded)
```
