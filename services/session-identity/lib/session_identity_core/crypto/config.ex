defmodule SessionIdentityCore.Crypto.Config do
  @moduledoc """
  Configuration for crypto-service integration.
  
  Validates configuration at startup and provides typed access to settings.
  """

  @default_endpoint "localhost:50051"
  @default_timeout 5_000
  @default_cache_ttl 300
  @default_circuit_breaker_threshold 5
  @default_circuit_breaker_timeout 30_000
  @default_latency_warning_ms 1_000

  defstruct [
    :endpoint,
    :timeout,
    :enabled,
    :fallback_enabled,
    :required_for_readiness,
    :cache_ttl,
    :circuit_breaker_threshold,
    :circuit_breaker_timeout,
    :latency_warning_ms,
    :jwt_key_namespace,
    :session_key_namespace,
    :refresh_token_key_namespace,
    :jwt_algorithm
  ]

  @type t :: %__MODULE__{
    endpoint: String.t(),
    timeout: pos_integer(),
    enabled: boolean(),
    fallback_enabled: boolean(),
    required_for_readiness: boolean(),
    cache_ttl: pos_integer(),
    circuit_breaker_threshold: pos_integer(),
    circuit_breaker_timeout: pos_integer(),
    latency_warning_ms: pos_integer(),
    jwt_key_namespace: String.t(),
    session_key_namespace: String.t(),
    refresh_token_key_namespace: String.t(),
    jwt_algorithm: :ecdsa_p256 | :rsa_2048
  }

  @doc """
  Loads and validates configuration from environment.
  Raises on invalid configuration (fail-fast).
  """
  @spec load!() :: t()
  def load! do
    config = %__MODULE__{
      endpoint: get_env("CRYPTO_SERVICE_ENDPOINT", @default_endpoint),
      timeout: get_env_int("CRYPTO_SERVICE_TIMEOUT", @default_timeout),
      enabled: get_env_bool("CRYPTO_INTEGRATION_ENABLED", true),
      fallback_enabled: get_env_bool("CRYPTO_FALLBACK_ENABLED", true),
      required_for_readiness: get_env_bool("CRYPTO_REQUIRED_FOR_READINESS", false),
      cache_ttl: get_env_int("CRYPTO_CACHE_TTL", @default_cache_ttl),
      circuit_breaker_threshold: get_env_int("CRYPTO_CB_THRESHOLD", @default_circuit_breaker_threshold),
      circuit_breaker_timeout: get_env_int("CRYPTO_CB_TIMEOUT", @default_circuit_breaker_timeout),
      latency_warning_ms: get_env_int("CRYPTO_LATENCY_WARNING_MS", @default_latency_warning_ms),
      jwt_key_namespace: get_env("CRYPTO_JWT_KEY_NAMESPACE", "session_identity:jwt"),
      session_key_namespace: get_env("CRYPTO_SESSION_KEY_NAMESPACE", "session_identity:session"),
      refresh_token_key_namespace: get_env("CRYPTO_REFRESH_TOKEN_KEY_NAMESPACE", "session_identity:refresh_token"),
      jwt_algorithm: get_jwt_algorithm()
    }

    validate!(config)
    config
  end

  @doc """
  Returns the current configuration from application env.
  """
  @spec get() :: t()
  def get do
    Application.get_env(:session_identity_core, :crypto_config) || load!()
  end

  defp get_env(key, default) do
    System.get_env(key) || default
  end

  defp get_env_int(key, default) do
    case System.get_env(key) do
      nil -> default
      val -> String.to_integer(val)
    end
  end

  defp get_env_bool(key, default) do
    case System.get_env(key) do
      nil -> default
      "true" -> true
      "false" -> false
      "1" -> true
      "0" -> false
      _ -> default
    end
  end

  defp get_jwt_algorithm do
    case System.get_env("CRYPTO_JWT_ALGORITHM") do
      "RSA_2048" -> :rsa_2048
      "ECDSA_P256" -> :ecdsa_p256
      _ -> :ecdsa_p256
    end
  end

  defp validate!(config) do
    errors = []

    errors = if config.timeout <= 0, do: ["timeout must be positive" | errors], else: errors
    errors = if config.cache_ttl <= 0, do: ["cache_ttl must be positive" | errors], else: errors
    errors = if config.circuit_breaker_threshold <= 0, do: ["circuit_breaker_threshold must be positive" | errors], else: errors
    errors = if config.circuit_breaker_timeout <= 0, do: ["circuit_breaker_timeout must be positive" | errors], else: errors
    errors = if config.latency_warning_ms <= 0, do: ["latency_warning_ms must be positive" | errors], else: errors

    if errors != [] do
      raise ArgumentError, "Invalid crypto configuration: #{Enum.join(errors, ", ")}"
    end

    :ok
  end
end
