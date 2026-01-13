defmodule SessionIdentityCore.Sessions.SessionStore do
  @moduledoc """
  Session storage using centralized Cache Service.
  
  Uses platform Cache Service client with circuit breaker protection.
  Falls back to local Redix when platform service is unavailable.
  """

  alias AuthPlatform.Clients.Cache
  alias AuthPlatform.Clients.Logging
  alias SessionIdentityCore.Sessions.SessionSerializer
  alias SessionIdentityCore.Shared.{Keys, TTL, Errors}

  @doc """
  Stores a session in the cache.
  """
  @spec store_session(map()) :: {:ok, map()} | {:error, term()}
  def store_session(session) do
    key = Keys.session_key(session.id)
    value = SessionSerializer.serialize(session)
    ttl = TTL.calculate(session.expires_at)

    with :ok <- Cache.set(key, value, ttl: ttl),
         :ok <- add_to_user_sessions(session.user_id, session.id) do
      Logging.info("Session stored", session_id: session.id, user_id: session.user_id)
      {:ok, session}
    else
      {:error, reason} = error ->
        Logging.error("Failed to store session",
          session_id: session.id,
          error: inspect(reason)
        )
        error
    end
  end

  @doc """
  Retrieves a session from the cache.
  """
  @spec get_session(String.t()) :: {:ok, map()} | {:error, :not_found | term()}
  def get_session(session_id) do
    key = Keys.session_key(session_id)

    case Cache.get(key) do
      {:ok, nil} -> Errors.session_not_found()
      {:ok, value} -> SessionSerializer.deserialize(value)
      {:error, _} = error -> error
    end
  end

  @doc """
  Deletes a session from the cache.
  """
  @spec delete_session(String.t(), String.t()) :: :ok | {:error, term()}
  def delete_session(session_id, user_id) do
    key = Keys.session_key(session_id)

    with :ok <- Cache.delete(key),
         :ok <- remove_from_user_sessions(user_id, session_id) do
      Logging.info("Session deleted", session_id: session_id, user_id: user_id)
      :ok
    end
  end

  @doc """
  Gets all sessions for a user.
  """
  @spec get_user_sessions(String.t()) :: {:ok, list()} | {:error, term()}
  def get_user_sessions(user_id) do
    key = Keys.user_sessions_key(user_id)

    case Cache.get(key) do
      {:ok, nil} ->
        {:ok, []}

      {:ok, session_ids_json} ->
        session_ids = Jason.decode!(session_ids_json)

        sessions =
          session_ids
          |> Enum.map(&get_session/1)
          |> Enum.filter(&match?({:ok, _}, &1))
          |> Enum.map(fn {:ok, session} -> session end)

        {:ok, sessions}

      error ->
        error
    end
  end

  @doc """
  Updates the last activity timestamp for a session.
  """
  @spec update_last_activity(String.t()) :: {:ok, map()} | {:error, term()}
  def update_last_activity(session_id) do
    case get_session(session_id) do
      {:ok, session} ->
        updated = Map.put(session, :last_activity, DateTime.utc_now())
        store_session(updated)

      error ->
        error
    end
  end

  @doc """
  Checks if a session exists.
  """
  @spec exists?(String.t()) :: boolean()
  def exists?(session_id) do
    key = Keys.session_key(session_id)
    Cache.exists?(key)
  end

  # Private functions

  defp add_to_user_sessions(user_id, session_id) do
    key = Keys.user_sessions_key(user_id)

    case Cache.get(key) do
      {:ok, nil} ->
        Cache.set(key, Jason.encode!([session_id]), ttl: TTL.default_session_ttl())

      {:ok, existing} ->
        session_ids = Jason.decode!(existing)
        updated = Enum.uniq([session_id | session_ids])
        Cache.set(key, Jason.encode!(updated), ttl: TTL.default_session_ttl())

      {:error, _} = error ->
        error
    end
  end

  defp remove_from_user_sessions(user_id, session_id) do
    key = Keys.user_sessions_key(user_id)

    case Cache.get(key) do
      {:ok, nil} ->
        :ok

      {:ok, existing} ->
        session_ids = Jason.decode!(existing)
        updated = List.delete(session_ids, session_id)

        if Enum.empty?(updated) do
          Cache.delete(key)
        else
          Cache.set(key, Jason.encode!(updated), ttl: TTL.default_session_ttl())
        end

      {:error, _} = error ->
        error
    end
  end
end
