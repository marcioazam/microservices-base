defmodule AuthPlatform.Clients.Logging do
  @moduledoc """
  Client for the platform Logging Service.

  Provides a resilient interface to the centralized logging service with
  circuit breaker protection.

  ## Usage

      # Log a message
      Logging.log(:info, "User logged in", user_id: 123)

      # Log with different levels
      Logging.debug("Debug message")
      Logging.info("Info message")
      Logging.warn("Warning message")
      Logging.error("Error message", error: "details")

  ## Configuration

      config :auth_platform_clients, :logging,
        endpoint: "localhost:50051",
        circuit_breaker: :logging_breaker

  """

  alias AuthPlatform.Resilience.CircuitBreaker

  @type level :: :debug | :info | :warn | :error
  @type metadata :: keyword() | map()

  @circuit_breaker_name :logging_service_breaker

  @doc """
  Logs a message with the specified level and metadata.

  ## Examples

      Logging.log(:info, "User action", user_id: 123, action: "login")

  """
  @spec log(level(), String.t(), metadata()) :: :ok | {:error, any()}
  def log(level, message, metadata \\ []) when level in [:debug, :info, :warn, :error] do
    entry = build_log_entry(level, message, metadata)

    case CircuitBreaker.execute(@circuit_breaker_name, fn ->
           send_log(entry)
         end) do
      {:ok, :ok} -> :ok
      {:error, :circuit_open} -> fallback_log(entry)
      {:error, reason} -> {:error, reason}
    end
  end

  @doc """
  Logs a debug message.
  """
  @spec debug(String.t(), metadata()) :: :ok | {:error, any()}
  def debug(message, metadata \\ []), do: log(:debug, message, metadata)

  @doc """
  Logs an info message.
  """
  @spec info(String.t(), metadata()) :: :ok | {:error, any()}
  def info(message, metadata \\ []), do: log(:info, message, metadata)

  @doc """
  Logs a warning message.
  """
  @spec warn(String.t(), metadata()) :: :ok | {:error, any()}
  def warn(message, metadata \\ []), do: log(:warn, message, metadata)

  @doc """
  Logs an error message.
  """
  @spec error(String.t(), metadata()) :: :ok | {:error, any()}
  def error(message, metadata \\ []), do: log(:error, message, metadata)

  @doc """
  Starts the logging client circuit breaker.

  Should be called during application startup.
  """
  @spec start_circuit_breaker(map()) :: {:ok, pid()} | {:error, any()}
  def start_circuit_breaker(config \\ %{}) do
    alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor
    ResilienceSupervisor.start_circuit_breaker(@circuit_breaker_name, config)
  end

  # Private functions

  defp build_log_entry(level, message, metadata) do
    %{
      level: level,
      message: message,
      metadata: Enum.into(metadata, %{}),
      timestamp: DateTime.utc_now() |> DateTime.to_iso8601(),
      service: Application.get_env(:auth_platform_clients, :service_name, "unknown")
    }
  end

  defp send_log(_entry) do
    # In a real implementation, this would send to gRPC service
    # For now, we simulate success
    {:ok, :ok}
  end

  defp fallback_log(entry) do
    # Fallback to local logging when circuit is open
    require Logger

    case entry.level do
      :debug -> Logger.debug(entry.message, entry.metadata)
      :info -> Logger.info(entry.message, entry.metadata)
      :warn -> Logger.warning(entry.message, entry.metadata)
      :error -> Logger.error(entry.message, entry.metadata)
    end

    :ok
  end
end
