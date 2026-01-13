defmodule AuthPlatform do
  @moduledoc """
  Auth Platform Elixir Library - Production-ready modules for building resilient microservices.

  This library provides:

  - **Functional Types** - Type-safe Result and Option types for explicit error handling
  - **Error Handling** - Structured errors with HTTP/gRPC mapping
  - **Validation** - Composable validators with error accumulation
  - **Domain Primitives** - Type-safe value objects (Email, UUID, Money, etc.)
  - **Resilience Patterns** - Circuit Breaker, Retry, Rate Limiter, Bulkhead
  - **Codecs** - JSON and Base64 encoding/decoding
  - **Security** - Timing-safe comparison, token generation, sanitization
  - **Observability** - Structured logging and telemetry

  ## Quick Start

      # Result type for error handling
      alias AuthPlatform.Functional.Result

      case fetch_user(id) do
        {:ok, user} -> process_user(user)
        {:error, reason} -> handle_error(reason)
      end

      # Domain primitives with validation
      alias AuthPlatform.Domain.Email

      case Email.new("user@example.com") do
        {:ok, email} -> send_welcome_email(email)
        {:error, reason} -> {:error, :invalid_email}
      end

      # Circuit breaker for resilience
      alias AuthPlatform.Resilience.CircuitBreaker

      CircuitBreaker.execute(:external_api, fn ->
        ExternalAPI.call()
      end)

      # Retry with exponential backoff
      alias AuthPlatform.Resilience.Retry

      Retry.execute(fn ->
        ExternalAPI.call()
      end)

  ## Configuration

  Configure the library in your `config/config.exs`:

      config :auth_platform,
        circuit_breaker: [
          failure_threshold: 5,
          timeout_ms: 30_000
        ],
        rate_limiter: [
          rate: 100,
          burst_size: 100
        ]

  """

  @doc """
  Returns the library version.
  """
  @spec version() :: String.t()
  def version, do: "0.1.0"
end
