defmodule MfaService.Caep.Emitter do
  @moduledoc """
  CAEP event emitter for MFA Service.
  Emits credential-change events on passkey and TOTP changes.
  Uses Logging_Service for audit logging with graceful fallback.
  """

  alias AuthPlatform.Clients.Logging

  @type event_result :: {:ok, String.t()} | {:error, term()}

  @doc """
  Emit credential-change event when passkey is added.
  """
  @spec emit_passkey_added(String.t(), String.t()) :: event_result()
  def emit_passkey_added(user_id, passkey_id) do
    emit_credential_change(user_id, "create", "passkey", %{passkey_id: passkey_id})
  end

  @doc """
  Emit credential-change event when passkey is removed.
  """
  @spec emit_passkey_removed(String.t(), String.t()) :: event_result()
  def emit_passkey_removed(user_id, passkey_id) do
    emit_credential_change(user_id, "delete", "passkey", %{passkey_id: passkey_id})
  end

  @doc """
  Emit credential-change event when TOTP is enabled.
  """
  @spec emit_totp_enabled(String.t()) :: event_result()
  def emit_totp_enabled(user_id) do
    emit_credential_change(user_id, "create", "totp", %{})
  end

  @doc """
  Emit credential-change event when TOTP is disabled.
  """
  @spec emit_totp_disabled(String.t()) :: event_result()
  def emit_totp_disabled(user_id) do
    emit_credential_change(user_id, "delete", "totp", %{})
  end

  @doc """
  Emit credential-change event when TOTP secret is rotated.
  """
  @spec emit_totp_rotated(String.t()) :: event_result()
  def emit_totp_rotated(user_id) do
    emit_credential_change(user_id, "update", "totp", %{reason: "rotation"})
  end

  defp emit_credential_change(user_id, change_type, credential_type, extra) do
    event_id = generate_event_id()

    event = %{
      event_type: "credential-change",
      event_id: event_id,
      subject: %{
        format: "iss_sub",
        iss: issuer(),
        sub: user_id
      },
      event_timestamp: DateTime.utc_now() |> DateTime.to_unix(),
      extra: Map.merge(extra, %{change_type: change_type, credential_type: credential_type})
    }

    case send_to_caep_service(event) do
      {:ok, _} ->
        log_success(event_id, user_id, change_type, credential_type)
        {:ok, event_id}

      {:error, reason} ->
        log_failure(user_id, change_type, credential_type, reason)
        {:error, reason}
    end
  end

  defp send_to_caep_service(event) do
    case Application.get_env(:mfa_service, :caep_enabled, false) do
      true ->
        caep_url = Application.get_env(:mfa_service, :caep_transmitter_url)
        send_http_event(caep_url, event)

      false ->
        Logging.debug("CAEP disabled, skipping event emission", event_id: event.event_id)
        {:ok, event.event_id}
    end
  end

  defp send_http_event(url, event) do
    headers = [
      {"content-type", "application/json"},
      {"authorization", "Bearer #{get_service_token()}"}
    ]

    body = Jason.encode!(event)

    case Req.post(url <> "/caep/emit", body: body, headers: headers) do
      {:ok, %{status: status, body: resp_body}} when status in 200..299 ->
        case Jason.decode(resp_body) do
          {:ok, %{"event_id" => eid}} -> {:ok, eid}
          _ -> {:ok, event.event_id}
        end

      {:ok, %{status: status, body: resp_body}} ->
        {:error, {:http_error, status, resp_body}}

      {:error, reason} ->
        {:error, {:network_error, reason}}
    end
  rescue
    _ -> {:error, :caep_unavailable}
  end

  defp log_success(event_id, user_id, change_type, credential_type) do
    Logging.info("CAEP credential-change emitted",
      event_id: event_id,
      user_id: user_id,
      change_type: change_type,
      credential_type: credential_type
    )
  end

  defp log_failure(user_id, change_type, credential_type, reason) do
    Logging.error("Failed to emit CAEP credential-change",
      user_id: user_id,
      change_type: change_type,
      credential_type: credential_type,
      error: inspect(reason)
    )
  end

  defp generate_event_id do
    :crypto.strong_rand_bytes(16) |> Base.encode16(case: :lower)
  end

  defp issuer do
    Application.get_env(:mfa_service, :issuer, "https://auth.example.com")
  end

  defp get_service_token do
    Application.get_env(:mfa_service, :caep_service_token, "")
  end
end
