defmodule SessionIdentityCore.Crypto.Correlation do
  @moduledoc """
  Correlation ID management for crypto operations.
  
  Ensures all crypto operations include a correlation_id for audit trail
  and distributed tracing correlation.
  """

  @doc """
  Gets or generates a correlation ID.
  
  If a correlation_id is provided in opts, returns it.
  Otherwise generates a new one.
  """
  @spec get_or_generate(keyword()) :: String.t()
  def get_or_generate(opts \\ []) do
    case Keyword.get(opts, :correlation_id) do
      nil -> generate()
      "" -> generate()
      id when is_binary(id) -> id
    end
  end

  @doc """
  Generates a new correlation ID.
  
  Format: 32 character hex string (128 bits of entropy)
  """
  @spec generate() :: String.t()
  def generate do
    :crypto.strong_rand_bytes(16)
    |> Base.encode16(case: :lower)
  end

  @doc """
  Validates a correlation ID format.
  
  Must be a non-empty string.
  """
  @spec valid?(term()) :: boolean()
  def valid?(id) when is_binary(id) and byte_size(id) > 0, do: true
  def valid?(_), do: false

  @doc """
  Ensures opts contain a valid correlation_id.
  
  Returns updated opts with correlation_id guaranteed to be present.
  """
  @spec ensure_correlation_id(keyword()) :: keyword()
  def ensure_correlation_id(opts) do
    correlation_id = get_or_generate(opts)
    Keyword.put(opts, :correlation_id, correlation_id)
  end

  @doc """
  Extracts correlation_id from opts for logging.
  """
  @spec extract_for_logging(keyword()) :: keyword()
  def extract_for_logging(opts) do
    case Keyword.get(opts, :correlation_id) do
      nil -> []
      id -> [correlation_id: id]
    end
  end
end
