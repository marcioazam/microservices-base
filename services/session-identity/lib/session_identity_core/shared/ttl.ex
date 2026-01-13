defmodule SessionIdentityCore.Shared.TTL do
  @moduledoc """
  Centralized TTL (Time To Live) calculations for session identity service.
  
  All TTL values MUST be calculated through this module to ensure:
  - Consistent expiration policies
  - Easy configuration changes
  - Proper handling of edge cases
  
  ## Default Values
  
  - Session TTL: 24 hours (86,400 seconds)
  - OAuth Code TTL: 10 minutes (600 seconds)
  - Refresh Token TTL: 30 days (2,592,000 seconds)
  - ID Token TTL: 1 hour (3,600 seconds)
  """

  @default_session_ttl 86_400
  @default_code_ttl 600
  @default_refresh_token_ttl 2_592_000
  @default_id_token_ttl 3_600
  @minimum_ttl 1

  @doc """
  Calculates TTL in seconds from an expiration DateTime.
  
  Returns at least 1 second to avoid immediate expiration.
  
  ## Examples
  
      iex> future = DateTime.add(DateTime.utc_now(), 3600, :second)
      iex> TTL.calculate(future)
      3600  # approximately
  """
  @spec calculate(DateTime.t()) :: pos_integer()
  def calculate(%DateTime{} = expires_at) do
    diff = DateTime.diff(expires_at, DateTime.utc_now())
    max(diff, @minimum_ttl)
  end

  def calculate(nil), do: @default_session_ttl

  @doc """
  Returns the default session TTL in seconds (24 hours).
  """
  @spec default_session_ttl() :: pos_integer()
  def default_session_ttl, do: @default_session_ttl

  @doc """
  Returns the default OAuth authorization code TTL in seconds (10 minutes).
  """
  @spec default_code_ttl() :: pos_integer()
  def default_code_ttl, do: @default_code_ttl

  @doc """
  Returns the default refresh token TTL in seconds (30 days).
  """
  @spec default_refresh_token_ttl() :: pos_integer()
  def default_refresh_token_ttl, do: @default_refresh_token_ttl

  @doc """
  Returns the default ID token TTL in seconds (1 hour).
  """
  @spec default_id_token_ttl() :: pos_integer()
  def default_id_token_ttl, do: @default_id_token_ttl

  @doc """
  Calculates the default session expiry DateTime.
  
  ## Examples
  
      iex> expiry = TTL.default_expiry()
      iex> DateTime.diff(expiry, DateTime.utc_now()) > 86000
      true
  """
  @spec default_expiry() :: DateTime.t()
  def default_expiry do
    DateTime.utc_now() |> DateTime.add(@default_session_ttl, :second)
  end

  @doc """
  Calculates the default OAuth code expiry DateTime.
  """
  @spec default_code_expiry() :: DateTime.t()
  def default_code_expiry do
    DateTime.utc_now() |> DateTime.add(@default_code_ttl, :second)
  end

  @doc """
  Calculates the default refresh token expiry DateTime.
  """
  @spec default_refresh_token_expiry() :: DateTime.t()
  def default_refresh_token_expiry do
    DateTime.utc_now() |> DateTime.add(@default_refresh_token_ttl, :second)
  end

  @doc """
  Calculates expiry DateTime from a custom TTL in seconds.
  """
  @spec expiry_from_ttl(pos_integer()) :: DateTime.t()
  def expiry_from_ttl(ttl_seconds) when is_integer(ttl_seconds) and ttl_seconds > 0 do
    DateTime.utc_now() |> DateTime.add(ttl_seconds, :second)
  end

  @doc """
  Checks if a TTL value is valid (positive integer).
  """
  @spec valid_ttl?(any()) :: boolean()
  def valid_ttl?(ttl) when is_integer(ttl) and ttl > 0, do: true
  def valid_ttl?(_), do: false
end
