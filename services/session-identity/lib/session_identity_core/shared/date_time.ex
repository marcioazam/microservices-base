defmodule SessionIdentityCore.Shared.DateTime do
  @moduledoc """
  Centralized DateTime operations for session identity service.
  
  All datetime serialization/deserialization MUST use this module to ensure:
  - Consistent ISO 8601 UTC format
  - Round-trip guarantee (serialize then deserialize produces equivalent value)
  - Proper handling of nil and edge cases
  
  ## Format
  
  All datetimes are serialized to ISO 8601 format with UTC timezone.
  Example: "2025-12-24T10:30:00Z"
  """

  @doc """
  Converts a DateTime to ISO 8601 string format.
  
  ## Examples
  
      iex> dt = ~U[2025-12-24 10:30:00Z]
      iex> SessionIdentityCore.Shared.DateTime.to_iso8601(dt)
      "2025-12-24T10:30:00Z"
      
      iex> SessionIdentityCore.Shared.DateTime.to_iso8601(nil)
      nil
  """
  @spec to_iso8601(DateTime.t() | NaiveDateTime.t() | nil) :: String.t() | nil
  def to_iso8601(nil), do: nil

  def to_iso8601(%DateTime{} = dt) do
    DateTime.to_iso8601(dt)
  end

  def to_iso8601(%NaiveDateTime{} = dt) do
    dt
    |> DateTime.from_naive!("Etc/UTC")
    |> DateTime.to_iso8601()
  end

  @doc """
  Parses an ISO 8601 string to DateTime.
  
  ## Examples
  
      iex> SessionIdentityCore.Shared.DateTime.from_iso8601("2025-12-24T10:30:00Z")
      ~U[2025-12-24 10:30:00Z]
      
      iex> SessionIdentityCore.Shared.DateTime.from_iso8601(nil)
      nil
  """
  @spec from_iso8601(String.t() | DateTime.t() | nil) :: DateTime.t() | nil
  def from_iso8601(nil), do: nil

  def from_iso8601(%DateTime{} = dt), do: dt

  def from_iso8601(str) when is_binary(str) do
    case DateTime.from_iso8601(str) do
      {:ok, dt, _offset} -> dt
      {:error, _} -> nil
    end
  end

  @doc """
  Returns the current UTC DateTime.
  """
  @spec utc_now() :: DateTime.t()
  def utc_now, do: DateTime.utc_now()

  @doc """
  Adds seconds to a DateTime.
  """
  @spec add_seconds(DateTime.t(), integer()) :: DateTime.t()
  def add_seconds(%DateTime{} = dt, seconds) when is_integer(seconds) do
    DateTime.add(dt, seconds, :second)
  end

  @doc """
  Calculates the difference in seconds between two DateTimes.
  """
  @spec diff_seconds(DateTime.t(), DateTime.t()) :: integer()
  def diff_seconds(%DateTime{} = dt1, %DateTime{} = dt2) do
    DateTime.diff(dt1, dt2)
  end

  @doc """
  Checks if a DateTime is in the past.
  """
  @spec is_past?(DateTime.t() | nil) :: boolean()
  def is_past?(nil), do: true

  def is_past?(%DateTime{} = dt) do
    DateTime.compare(dt, DateTime.utc_now()) == :lt
  end

  @doc """
  Checks if a DateTime is in the future.
  """
  @spec is_future?(DateTime.t() | nil) :: boolean()
  def is_future?(nil), do: false

  def is_future?(%DateTime{} = dt) do
    DateTime.compare(dt, DateTime.utc_now()) == :gt
  end

  @doc """
  Converts Unix timestamp to DateTime.
  """
  @spec from_unix(integer() | nil) :: DateTime.t() | nil
  def from_unix(nil), do: nil

  def from_unix(timestamp) when is_integer(timestamp) do
    DateTime.from_unix!(timestamp)
  end

  @doc """
  Converts DateTime to Unix timestamp.
  """
  @spec to_unix(DateTime.t() | nil) :: integer() | nil
  def to_unix(nil), do: nil

  def to_unix(%DateTime{} = dt) do
    DateTime.to_unix(dt)
  end
end
