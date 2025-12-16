defmodule SessionIdentityCore.Sessions.SessionStore do
  @moduledoc """
  Redis-backed session storage with TTL support.
  """

  @session_prefix "session:"
  @user_sessions_prefix "user_sessions:"
  @default_ttl 86_400  # 24 hours

  def store_session(session) do
    key = session_key(session.id)
    value = Jason.encode!(session_to_map(session))
    ttl = calculate_ttl(session.expires_at)

    with {:ok, _} <- Redix.command(:redix, ["SETEX", key, ttl, value]),
         {:ok, _} <- add_to_user_sessions(session.user_id, session.id) do
      {:ok, session}
    end
  end

  def get_session(session_id) do
    key = session_key(session_id)

    case Redix.command(:redix, ["GET", key]) do
      {:ok, nil} -> {:error, :not_found}
      {:ok, value} -> {:ok, Jason.decode!(value)}
      error -> error
    end
  end

  def delete_session(session_id, user_id) do
    key = session_key(session_id)

    with {:ok, _} <- Redix.command(:redix, ["DEL", key]),
         {:ok, _} <- remove_from_user_sessions(user_id, session_id) do
      :ok
    end
  end

  def get_user_sessions(user_id) do
    key = user_sessions_key(user_id)

    case Redix.command(:redix, ["SMEMBERS", key]) do
      {:ok, session_ids} ->
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

  def update_last_activity(session_id) do
    case get_session(session_id) do
      {:ok, session} ->
        updated = Map.put(session, "last_activity", DateTime.to_unix(DateTime.utc_now()))
        key = session_key(session_id)
        ttl = Map.get(session, "expires_at", @default_ttl) - DateTime.to_unix(DateTime.utc_now())
        Redix.command(:redix, ["SETEX", key, max(ttl, 1), Jason.encode!(updated)])

      error ->
        error
    end
  end

  defp session_key(session_id), do: "#{@session_prefix}#{session_id}"
  defp user_sessions_key(user_id), do: "#{@user_sessions_prefix}#{user_id}"

  defp add_to_user_sessions(user_id, session_id) do
    key = user_sessions_key(user_id)
    Redix.command(:redix, ["SADD", key, session_id])
  end

  defp remove_from_user_sessions(user_id, session_id) do
    key = user_sessions_key(user_id)
    Redix.command(:redix, ["SREM", key, session_id])
  end

  defp calculate_ttl(expires_at) do
    diff = DateTime.diff(expires_at, DateTime.utc_now())
    max(diff, 1)
  end

  defp session_to_map(session) do
    %{
      id: session.id,
      user_id: session.user_id,
      device_id: session.device_id,
      ip_address: session.ip_address,
      user_agent: session.user_agent,
      device_fingerprint: session.device_fingerprint,
      risk_score: session.risk_score,
      mfa_verified: session.mfa_verified,
      created_at: DateTime.to_unix(session.inserted_at),
      last_activity: DateTime.to_unix(session.last_activity),
      expires_at: DateTime.to_unix(session.expires_at)
    }
  end
end
