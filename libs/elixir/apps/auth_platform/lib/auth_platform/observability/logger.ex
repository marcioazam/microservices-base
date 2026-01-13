defmodule AuthPlatform.Observability.Logger do
  @moduledoc """
  Structured logging utilities with correlation ID support.

  Provides JSON-formatted logging with automatic correlation ID propagation
  and PII redaction.

  ## Usage

      # Set correlation ID for current process
      Logger.put_correlation_id("req-123")

      # Log with automatic correlation ID
      Logger.info("User logged in", user_id: 123)

      # Use with_correlation_id for scoped operations
      Logger.with_correlation_id("req-456", fn ->
        Logger.info("Processing request")
        do_work()
      end)

  """

  require Logger

  @correlation_id_key :auth_platform_correlation_id
  @sensitive_fields ~w(password token secret api_key credit_card ssn)a

  @doc """
  Sets the correlation ID for the current process.
  """
  @spec put_correlation_id(String.t()) :: :ok
  def put_correlation_id(correlation_id) when is_binary(correlation_id) do
    Process.put(@correlation_id_key, correlation_id)
    :ok
  end

  @doc """
  Gets the correlation ID for the current process.
  """
  @spec get_correlation_id() :: String.t() | nil
  def get_correlation_id do
    Process.get(@correlation_id_key)
  end

  @doc """
  Generates a new correlation ID.
  """
  @spec generate_correlation_id() :: String.t()
  def generate_correlation_id do
    :crypto.strong_rand_bytes(8)
    |> Base.encode16(case: :lower)
  end

  @doc """
  Executes a function with a correlation ID set for the duration.

  ## Examples

      Logger.with_correlation_id("req-123", fn ->
        Logger.info("Processing")
        do_work()
      end)

  """
  @spec with_correlation_id(String.t(), (() -> result)) :: result when result: any()
  def with_correlation_id(correlation_id, fun) when is_binary(correlation_id) do
    old_id = get_correlation_id()
    put_correlation_id(correlation_id)

    try do
      fun.()
    after
      if old_id, do: put_correlation_id(old_id), else: Process.delete(@correlation_id_key)
    end
  end

  @doc """
  Logs a debug message with structured metadata.
  """
  @spec debug(String.t(), keyword()) :: :ok
  def debug(message, metadata \\ []) do
    log(:debug, message, metadata)
  end

  @doc """
  Logs an info message with structured metadata.
  """
  @spec info(String.t(), keyword()) :: :ok
  def info(message, metadata \\ []) do
    log(:info, message, metadata)
  end

  @doc """
  Logs a warning message with structured metadata.
  """
  @spec warn(String.t(), keyword()) :: :ok
  def warn(message, metadata \\ []) do
    log(:warning, message, metadata)
  end

  @doc """
  Logs an error message with structured metadata.
  """
  @spec error(String.t(), keyword()) :: :ok
  def error(message, metadata \\ []) do
    log(:error, message, metadata)
  end

  @doc """
  Formats log entry as JSON.
  """
  @spec format_json(map()) :: String.t()
  def format_json(entry) when is_map(entry) do
    entry
    |> redact_sensitive()
    |> Jason.encode!()
  end

  # Private functions

  defp log(level, message, metadata) do
    entry = build_entry(level, message, metadata)

    case level do
      :debug -> Logger.debug(fn -> format_json(entry) end)
      :info -> Logger.info(fn -> format_json(entry) end)
      :warning -> Logger.warning(fn -> format_json(entry) end)
      :error -> Logger.error(fn -> format_json(entry) end)
    end

    :ok
  end

  defp build_entry(level, message, metadata) do
    base = %{
      timestamp: DateTime.utc_now() |> DateTime.to_iso8601(),
      level: level,
      message: message,
      correlation_id: get_correlation_id()
    }

    metadata
    |> Enum.into(%{})
    |> Map.merge(base)
  end

  defp redact_sensitive(entry) when is_map(entry) do
    Enum.reduce(entry, %{}, fn {key, value}, acc ->
      redacted_value =
        cond do
          is_sensitive_key?(key) -> "[REDACTED]"
          is_map(value) -> redact_sensitive(value)
          is_list(value) -> Enum.map(value, &redact_if_map/1)
          true -> value
        end

      Map.put(acc, key, redacted_value)
    end)
  end

  defp redact_if_map(value) when is_map(value), do: redact_sensitive(value)
  defp redact_if_map(value), do: value

  defp is_sensitive_key?(key) when is_atom(key), do: key in @sensitive_fields

  defp is_sensitive_key?(key) when is_binary(key) do
    String.downcase(key) in Enum.map(@sensitive_fields, &Atom.to_string/1)
  end

  defp is_sensitive_key?(_), do: false
end
