defmodule MfaService.Crypto.Logger do
  @moduledoc """
  Structured logger for Crypto Service operations.
  Ensures correlation_id is included in all log entries.
  Sanitizes sensitive data before logging.
  """

  require Logger

  @sensitive_keys [:secret, :plaintext, :key, :password, :token, :ciphertext]

  @doc """
  Logs a debug message with correlation_id.
  """
  @spec debug(String.t(), String.t(), keyword()) :: :ok
  def debug(message, correlation_id, metadata \\ []) do
    Logger.debug(message, build_metadata(correlation_id, metadata))
  end

  @doc """
  Logs an info message with correlation_id.
  """
  @spec info(String.t(), String.t(), keyword()) :: :ok
  def info(message, correlation_id, metadata \\ []) do
    Logger.info(message, build_metadata(correlation_id, metadata))
  end

  @doc """
  Logs a warning message with correlation_id.
  """
  @spec warning(String.t(), String.t(), keyword()) :: :ok
  def warning(message, correlation_id, metadata \\ []) do
    Logger.warning(message, build_metadata(correlation_id, metadata))
  end

  @doc """
  Logs an error message with correlation_id.
  """
  @spec error(String.t(), String.t(), keyword()) :: :ok
  def error(message, correlation_id, metadata \\ []) do
    Logger.error(message, build_metadata(correlation_id, metadata))
  end

  @doc """
  Logs an operation start.
  """
  @spec log_operation_start(atom(), String.t(), keyword()) :: :ok
  def log_operation_start(operation, correlation_id, metadata \\ []) do
    debug("Starting #{operation}", correlation_id, [{:operation, operation} | metadata])
  end

  @doc """
  Logs an operation completion.
  """
  @spec log_operation_complete(atom(), String.t(), non_neg_integer(), keyword()) :: :ok
  def log_operation_complete(operation, correlation_id, duration_ms, metadata \\ []) do
    debug("Completed #{operation}", correlation_id, 
      [{:operation, operation}, {:duration_ms, duration_ms} | metadata])
  end

  @doc """
  Logs an operation failure.
  """
  @spec log_operation_failure(atom(), String.t(), term(), keyword()) :: :ok
  def log_operation_failure(operation, correlation_id, error, metadata \\ []) do
    sanitized_error = sanitize_error(error)
    error("Failed #{operation}", correlation_id,
      [{:operation, operation}, {:error, sanitized_error} | metadata])
  end

  # Private functions

  defp build_metadata(correlation_id, metadata) do
    sanitized = sanitize_metadata(metadata)
    
    [
      correlation_id: correlation_id,
      service: :crypto_client,
      timestamp: DateTime.utc_now() |> DateTime.to_iso8601()
    ] ++ sanitized
  end

  defp sanitize_metadata(metadata) do
    Enum.map(metadata, fn {key, value} ->
      if key in @sensitive_keys do
        {key, "[REDACTED]"}
      else
        {key, sanitize_value(value)}
      end
    end)
  end

  defp sanitize_value(value) when is_binary(value) do
    if looks_like_secret?(value) do
      "[REDACTED]"
    else
      truncate_if_long(value)
    end
  end

  # SECURITY FIX: Keep map keys as original type to prevent atom exhaustion
  # Previously converted all keys to atoms, which is dangerous for external data
  defp sanitize_value(value) when is_map(value) do
    Map.new(value, fn {k, v} ->
      # Check if key (as atom or string) matches sensitive keys
      is_sensitive = cond do
        is_atom(k) -> k in @sensitive_keys
        is_binary(k) -> String.to_existing_atom(k) in @sensitive_keys
        true -> false
      end

      if is_sensitive do
        {k, "[REDACTED]"}
      else
        {k, sanitize_value(v)}
      end
    rescue
      # String.to_existing_atom raises if atom doesn't exist - that's fine, not sensitive
      ArgumentError -> {k, sanitize_value(v)}
    end)
  end

  defp sanitize_value(value), do: value

  defp looks_like_secret?(value) do
    # Check if value looks like base64-encoded secret or key
    byte_size(value) >= 32 and String.match?(value, ~r/^[A-Za-z0-9+\/=]+$/)
  end

  defp truncate_if_long(value) when byte_size(value) > 100 do
    String.slice(value, 0, 100) <> "...[truncated]"
  end

  defp truncate_if_long(value), do: value

  defp sanitize_error(%{message: message} = error) do
    %{error | message: sanitize_value(message)}
  end

  defp sanitize_error(error) when is_binary(error) do
    sanitize_value(error)
  end

  defp sanitize_error(error), do: inspect(error)
end
