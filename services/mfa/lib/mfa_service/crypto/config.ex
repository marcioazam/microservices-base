defmodule MfaService.Crypto.Config do
  @moduledoc """
  Configuration for Crypto Service client.
  All settings are read from environment variables with sensible defaults.
  """

  @default_host "localhost"
  @default_port 50051
  @default_connection_timeout 5_000
  @default_request_timeout 30_000
  @default_key_namespace "mfa"
  @default_circuit_breaker_threshold 5
  @default_retry_max_attempts 3
  @default_retry_base_delay 100
  @default_cache_ttl 300

  @doc """
  Returns the Crypto Service host address.
  Default: "localhost"
  """
  @spec host() :: String.t()
  def host do
    System.get_env("CRYPTO_SERVICE_HOST", @default_host)
  end

  @doc """
  Returns the Crypto Service port.
  Default: 50051
  """
  @spec port() :: non_neg_integer()
  def port do
    "CRYPTO_SERVICE_PORT"
    |> System.get_env(Integer.to_string(@default_port))
    |> String.to_integer()
  end

  @doc """
  Returns the full Crypto Service address (host:port).
  """
  @spec address() :: String.t()
  def address do
    "#{host()}:#{port()}"
  end

  @doc """
  Returns the connection timeout in milliseconds.
  Default: 5000ms (5 seconds)
  """
  @spec connection_timeout() :: non_neg_integer()
  def connection_timeout do
    "CRYPTO_CONNECTION_TIMEOUT"
    |> System.get_env(Integer.to_string(@default_connection_timeout))
    |> String.to_integer()
  end

  @doc """
  Returns the request timeout in milliseconds.
  Default: 30000ms (30 seconds)
  """
  @spec request_timeout() :: non_neg_integer()
  def request_timeout do
    "CRYPTO_REQUEST_TIMEOUT"
    |> System.get_env(Integer.to_string(@default_request_timeout))
    |> String.to_integer()
  end

  @doc """
  Returns the key namespace prefix for MFA keys.
  Default: "mfa"
  """
  @spec key_namespace() :: String.t()
  def key_namespace do
    System.get_env("CRYPTO_KEY_NAMESPACE", @default_key_namespace)
  end

  @doc """
  Returns the full TOTP key namespace.
  Format: "{namespace}:totp"
  """
  @spec totp_key_namespace() :: String.t()
  def totp_key_namespace do
    "#{key_namespace()}:totp"
  end

  @doc """
  Returns the circuit breaker failure threshold.
  Default: 5 consecutive failures
  """
  @spec circuit_breaker_threshold() :: non_neg_integer()
  def circuit_breaker_threshold do
    "CRYPTO_CB_THRESHOLD"
    |> System.get_env(Integer.to_string(@default_circuit_breaker_threshold))
    |> String.to_integer()
  end

  @doc """
  Returns the circuit breaker reset timeout in milliseconds.
  Default: 30000ms (30 seconds)
  """
  @spec circuit_breaker_reset_timeout() :: non_neg_integer()
  def circuit_breaker_reset_timeout do
    "CRYPTO_CB_RESET_TIMEOUT"
    |> System.get_env("30000")
    |> String.to_integer()
  end

  @doc """
  Returns the maximum retry attempts for transient failures.
  Default: 3 attempts
  """
  @spec retry_max_attempts() :: non_neg_integer()
  def retry_max_attempts do
    "CRYPTO_RETRY_ATTEMPTS"
    |> System.get_env(Integer.to_string(@default_retry_max_attempts))
    |> String.to_integer()
  end

  @doc """
  Returns the base delay for exponential backoff in milliseconds.
  Default: 100ms
  """
  @spec retry_base_delay() :: non_neg_integer()
  def retry_base_delay do
    "CRYPTO_RETRY_BASE_DELAY"
    |> System.get_env(Integer.to_string(@default_retry_base_delay))
    |> String.to_integer()
  end

  @doc """
  Returns the key metadata cache TTL in seconds.
  Default: 300 seconds (5 minutes)
  """
  @spec cache_ttl() :: non_neg_integer()
  def cache_ttl do
    "CRYPTO_CACHE_TTL"
    |> System.get_env(Integer.to_string(@default_cache_ttl))
    |> String.to_integer()
  end

  @doc """
  Returns whether mTLS is enabled.
  Default: true in production, false otherwise
  """
  @spec mtls_enabled?() :: boolean()
  def mtls_enabled? do
    case System.get_env("CRYPTO_MTLS_ENABLED") do
      "true" -> true
      "false" -> false
      nil -> Mix.env() == :prod
      _ -> false
    end
  end

  @doc """
  Returns the path to the TLS certificate file.
  """
  @spec tls_cert_path() :: String.t() | nil
  def tls_cert_path do
    System.get_env("CRYPTO_TLS_CERT_PATH")
  end

  @doc """
  Returns the path to the TLS key file.
  """
  @spec tls_key_path() :: String.t() | nil
  def tls_key_path do
    System.get_env("CRYPTO_TLS_KEY_PATH")
  end

  @doc """
  Returns the path to the CA certificate file.
  """
  @spec tls_ca_path() :: String.t() | nil
  def tls_ca_path do
    System.get_env("CRYPTO_TLS_CA_PATH")
  end

  @doc """
  Returns all configuration as a map for debugging/logging.
  Sensitive values are redacted.
  """
  @spec to_map() :: map()
  def to_map do
    %{
      host: host(),
      port: port(),
      connection_timeout: connection_timeout(),
      request_timeout: request_timeout(),
      key_namespace: key_namespace(),
      circuit_breaker_threshold: circuit_breaker_threshold(),
      retry_max_attempts: retry_max_attempts(),
      cache_ttl: cache_ttl(),
      mtls_enabled: mtls_enabled?()
    }
  end
end
