defmodule SessionIdentityCore.Config do
  @moduledoc """
  Environment-based configuration with validation.
  
  Reads all config from environment variables with sensible defaults.
  Validates config at startup and fails fast on invalid values.
  """

  @doc """
  Loads and validates all configuration at startup.
  """
  @spec load!() :: map()
  def load! do
    config = %{
      issuer: get_env("ISSUER", "https://auth.example.com"),
      session_ttl: get_env_int("SESSION_TTL", 86_400),
      code_ttl: get_env_int("CODE_TTL", 600),
      id_token_ttl: get_env_int("ID_TOKEN_TTL", 3_600),
      refresh_token_ttl: get_env_int("REFRESH_TOKEN_TTL", 2_592_000),
      caep_enabled: get_env_bool("CAEP_ENABLED", false),
      caep_receiver_url: get_env("CAEP_RECEIVER_URL", nil),
      cache_endpoint: get_env("CACHE_ENDPOINT", "localhost:50052"),
      logging_endpoint: get_env("LOGGING_ENDPOINT", "localhost:50051"),
      risk_step_up_threshold: get_env_float("RISK_STEP_UP_THRESHOLD", 0.7),
      risk_high_threshold: get_env_float("RISK_HIGH_THRESHOLD", 0.9)
    }

    validate!(config)
    config
  end

  @doc """
  Validates configuration values.
  """
  @spec validate!(map()) :: :ok
  def validate!(config) do
    errors = []

    errors = if config.session_ttl <= 0, do: ["SESSION_TTL must be positive" | errors], else: errors
    errors = if config.code_ttl <= 0, do: ["CODE_TTL must be positive" | errors], else: errors
    errors = if config.id_token_ttl <= 0, do: ["ID_TOKEN_TTL must be positive" | errors], else: errors

    errors =
      if config.risk_step_up_threshold < 0 or config.risk_step_up_threshold > 1,
        do: ["RISK_STEP_UP_THRESHOLD must be between 0 and 1" | errors],
        else: errors

    errors =
      if config.risk_high_threshold < 0 or config.risk_high_threshold > 1,
        do: ["RISK_HIGH_THRESHOLD must be between 0 and 1" | errors],
        else: errors

    if Enum.empty?(errors) do
      :ok
    else
      raise "Configuration validation failed: #{Enum.join(errors, ", ")}"
    end
  end

  @doc """
  Gets a configuration value.
  """
  @spec get(atom()) :: any()
  def get(key) do
    Application.get_env(:session_identity_core, key)
  end

  # Private helpers

  defp get_env(key, default), do: System.get_env(key, default)

  defp get_env_int(key, default) do
    case System.get_env(key) do
      nil -> default
      val -> String.to_integer(val)
    end
  end

  defp get_env_float(key, default) do
    case System.get_env(key) do
      nil -> default
      val -> String.to_float(val)
    end
  end

  defp get_env_bool(key, default) do
    case System.get_env(key) do
      nil -> default
      "true" -> true
      "1" -> true
      _ -> false
    end
  end
end
