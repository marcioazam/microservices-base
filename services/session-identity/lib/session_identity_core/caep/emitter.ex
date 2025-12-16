defmodule SessionIdentityCore.Caep.Emitter do
  @moduledoc """
  CAEP event emitter for Session Identity Core.
  Emits session-revoked events on logout and admin termination.
  """

  require Logger

  @type event_result :: {:ok, String.t()} | {:error, term()}

  @doc """
  Emit session-revoked event when user logs out.
  """
  @spec emit_session_revoked_logout(String.t(), String.t()) :: event_result()
  def emit_session_revoked_logout(user_id, session_id) do
    emit_session_revoked(user_id, session_id, "User initiated logout")
  end

  @doc """
  Emit session-revoked event when admin terminates session.
  """
  @spec emit_session_revoked_admin(String.t(), String.t(), String.t()) :: event_result()
  def emit_session_revoked_admin(user_id, session_id, admin_id) do
    emit_session_revoked(user_id, session_id, "Admin termination by #{admin_id}")
  end

  @doc """
  Emit session-revoked event for security policy violation.
  """
  @spec emit_session_revoked_security(String.t(), String.t(), String.t()) :: event_result()
  def emit_session_revoked_security(user_id, session_id, reason) do
    emit_session_revoked(user_id, session_id, "Security policy: #{reason}")
  end

  defp emit_session_revoked(user_id, session_id, reason) do
    event = %{
      event_type: "session-revoked",
      subject: %{
        format: "iss_sub",
        iss: issuer(),
        sub: user_id
      },
      event_timestamp: DateTime.utc_now() |> DateTime.to_unix(),
      reason_admin: %{
        en: reason
      },
      extra: %{
        session_id: session_id
      }
    }

    case send_to_caep_service(event) do
      {:ok, event_id} ->
        Logger.info("CAEP session-revoked emitted",
          event_id: event_id,
          user_id: user_id,
          session_id: session_id
        )
        {:ok, event_id}

      {:error, reason} = error ->
        Logger.error("Failed to emit CAEP session-revoked",
          user_id: user_id,
          session_id: session_id,
          error: inspect(reason)
        )
        error
    end
  end

  defp send_to_caep_service(event) do
    # In production, this would call the CAEP transmitter service
    # For now, we'll use a GenServer or HTTP client
    case Application.get_env(:session_identity_core, :caep_enabled, false) do
      true ->
        caep_url = Application.get_env(:session_identity_core, :caep_transmitter_url)
        send_http_event(caep_url, event)

      false ->
        # CAEP disabled, just log
        Logger.debug("CAEP disabled, skipping event emission")
        {:ok, UUID.uuid4()}
    end
  end

  defp send_http_event(url, event) do
    headers = [
      {"Content-Type", "application/json"},
      {"Authorization", "Bearer #{get_service_token()}"}
    ]

    body = Jason.encode!(event)

    case HTTPoison.post(url <> "/caep/emit", body, headers) do
      {:ok, %{status_code: status, body: resp_body}} when status in 200..299 ->
        case Jason.decode(resp_body) do
          {:ok, %{"event_id" => event_id}} -> {:ok, event_id}
          _ -> {:ok, UUID.uuid4()}
        end

      {:ok, %{status_code: status, body: resp_body}} ->
        {:error, {:http_error, status, resp_body}}

      {:error, reason} ->
        {:error, {:network_error, reason}}
    end
  end

  defp issuer do
    Application.get_env(:session_identity_core, :issuer, "https://auth.example.com")
  end

  defp get_service_token do
    Application.get_env(:session_identity_core, :caep_service_token, "")
  end
end
