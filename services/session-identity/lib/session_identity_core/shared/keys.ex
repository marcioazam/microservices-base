defmodule SessionIdentityCore.Shared.Keys do
  @moduledoc """
  Centralized Redis key generation for session identity service.
  
  All cache keys MUST be generated through this module to ensure:
  - Consistent namespacing
  - Key isolation in shared cache infrastructure
  - Easy key pattern changes
  
  ## Key Prefixes
  
  - `session:` - Individual session data
  - `user_sessions:` - Set of session IDs per user
  - `oauth_code:` - OAuth authorization codes
  - `events:` - Event store entries
  - `refresh_token:` - Refresh token data
  """

  @session_prefix "session:"
  @user_sessions_prefix "user_sessions:"
  @oauth_code_prefix "oauth_code:"
  @events_prefix "events:"
  @refresh_token_prefix "refresh_token:"
  @aggregate_prefix "aggregate:"

  @doc """
  Generates a session cache key.
  
  ## Examples
  
      iex> Keys.session_key("abc-123")
      "session:abc-123"
  """
  @spec session_key(String.t()) :: String.t()
  def session_key(session_id) when is_binary(session_id) do
    "#{@session_prefix}#{session_id}"
  end

  @doc """
  Generates a user sessions set key.
  
  ## Examples
  
      iex> Keys.user_sessions_key("user-456")
      "user_sessions:user-456"
  """
  @spec user_sessions_key(String.t()) :: String.t()
  def user_sessions_key(user_id) when is_binary(user_id) do
    "#{@user_sessions_prefix}#{user_id}"
  end

  @doc """
  Generates an OAuth authorization code key.
  
  ## Examples
  
      iex> Keys.oauth_code_key("code-789")
      "oauth_code:code-789"
  """
  @spec oauth_code_key(String.t()) :: String.t()
  def oauth_code_key(code) when is_binary(code) do
    "#{@oauth_code_prefix}#{code}"
  end

  @doc """
  Generates an event store key.
  
  ## Examples
  
      iex> Keys.event_key("evt-001")
      "events:evt-001"
  """
  @spec event_key(String.t()) :: String.t()
  def event_key(event_id) when is_binary(event_id) do
    "#{@events_prefix}#{event_id}"
  end

  @doc """
  Generates a refresh token key.
  
  ## Examples
  
      iex> Keys.refresh_token_key("rt-abc")
      "refresh_token:rt-abc"
  """
  @spec refresh_token_key(String.t()) :: String.t()
  def refresh_token_key(token_id) when is_binary(token_id) do
    "#{@refresh_token_prefix}#{token_id}"
  end

  @doc """
  Generates an aggregate key for event sourcing.
  
  ## Examples
  
      iex> Keys.aggregate_key("Session", "sess-123")
      "aggregate:Session:sess-123"
  """
  @spec aggregate_key(String.t(), String.t()) :: String.t()
  def aggregate_key(aggregate_type, aggregate_id)
      when is_binary(aggregate_type) and is_binary(aggregate_id) do
    "#{@aggregate_prefix}#{aggregate_type}:#{aggregate_id}"
  end

  @doc """
  Returns the session key prefix for pattern matching.
  """
  @spec session_prefix() :: String.t()
  def session_prefix, do: @session_prefix

  @doc """
  Returns the user sessions key prefix for pattern matching.
  """
  @spec user_sessions_prefix() :: String.t()
  def user_sessions_prefix, do: @user_sessions_prefix
end
