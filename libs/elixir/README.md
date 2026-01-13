# Auth Platform Elixir Library

Production-ready Elixir library for building resilient microservices in the Auth Platform ecosystem.

## Features

- **Functional Types** - Type-safe `Result` and `Option` types for explicit error handling
- **Error Handling** - Structured errors with HTTP/gRPC mapping
- **Validation** - Composable validators with error accumulation
- **Domain Primitives** - Type-safe value objects (Email, UUID, ULID, Money, PhoneNumber, URL)
- **Resilience Patterns** - Circuit Breaker, Retry, Rate Limiter, Bulkhead
- **Codecs** - JSON and Base64 encoding/decoding
- **Security** - Timing-safe comparison, token generation, sanitization
- **Observability** - Structured logging with correlation IDs and telemetry

## Requirements

- Elixir 1.15+
- OTP 26+

## Installation

Add to your `mix.exs`:

```elixir
def deps do
  [
    {:auth_platform, path: "libs/elixir/apps/auth_platform"},
    {:auth_platform_testing, path: "libs/elixir/apps/auth_platform_testing", only: :test}
  ]
end
```

## Quick Start

### Result Type

```elixir
alias AuthPlatform.Functional.Result

case fetch_user(id) do
  {:ok, user} -> process_user(user)
  {:error, reason} -> handle_error(reason)
end

# Chaining operations
Result.ok(user)
|> Result.map(&update_profile/1)
|> Result.flat_map(&save_user/1)
```

### Domain Primitives

```elixir
alias AuthPlatform.Domain.Email

case Email.new("user@example.com") do
  {:ok, email} -> send_welcome_email(email)
  {:error, reason} -> {:error, :invalid_email}
end
```

### Circuit Breaker

```elixir
alias AuthPlatform.Resilience.CircuitBreaker

# Start circuit breaker
AuthPlatform.Resilience.Supervisor.start_circuit_breaker(:external_api, %{
  failure_threshold: 5,
  timeout_ms: 30_000
})

# Execute with protection
CircuitBreaker.execute(:external_api, fn ->
  ExternalAPI.call()
end)
```

### Rate Limiter

```elixir
alias AuthPlatform.Resilience.RateLimiter

# Start rate limiter
AuthPlatform.Resilience.Supervisor.start_rate_limiter(:api_limiter, %{
  rate: 100,      # tokens per second
  burst_size: 150
})

# Check if allowed
if RateLimiter.allow?(:api_limiter) do
  process_request()
end
```

### Validation

```elixir
alias AuthPlatform.Validation

result = Validation.validate_all([
  Validation.validate_field("email", email, [Validation.required(), Validation.matches_regex(~r/@/)]),
  Validation.validate_field("age", age, [Validation.positive(), Validation.in_range(18, 120)])
])

case result do
  {:ok, _} -> :valid
  {:errors, errors} -> {:invalid, errors}
end
```

## Project Structure

```
libs/elixir/
├── apps/
│   ├── auth_platform/           # Core library
│   │   ├── lib/
│   │   │   ├── auth_platform/
│   │   │   │   ├── functional/  # Result, Option types
│   │   │   │   ├── errors/      # AppError
│   │   │   │   ├── domain/      # Email, UUID, Money, etc.
│   │   │   │   ├── resilience/  # CircuitBreaker, Retry, RateLimiter, Bulkhead
│   │   │   │   ├── codec/       # JSON, Base64
│   │   │   │   ├── observability/ # Logger, Telemetry
│   │   │   │   └── security.ex
│   │   │   └── auth_platform.ex
│   │   └── test/
│   ├── auth_platform_clients/   # Platform service clients
│   └── auth_platform_testing/   # Test utilities and generators
├── config/
└── mix.exs
```

## Testing

```bash
# Run all tests
mix test

# Run with coverage
mix coveralls

# Run property tests only
mix test --only property

# Run specific module tests
mix test test/auth_platform/resilience/circuit_breaker_test.exs
```

## Development

```bash
# Install dependencies
mix deps.get

# Compile
mix compile

# Run static analysis
mix dialyzer

# Run linter
mix credo --strict

# Generate documentation
mix docs
```

## Telemetry Events

The library emits telemetry events for observability:

- `[:auth_platform, :circuit_breaker, :state_change]`
- `[:auth_platform, :circuit_breaker, :request_blocked]`
- `[:auth_platform, :retry, :attempt]`
- `[:auth_platform, :rate_limiter, :allowed]`
- `[:auth_platform, :rate_limiter, :rejected]`
- `[:auth_platform, :bulkhead, :acquired]`
- `[:auth_platform, :bulkhead, :rejected]`

## License

MIT
