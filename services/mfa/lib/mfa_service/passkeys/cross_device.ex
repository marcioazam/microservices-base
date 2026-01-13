defmodule MfaService.Passkeys.CrossDevice do
  @moduledoc """
  Cross-device passkey authentication using CTAP hybrid transport.
  Generates QR codes for mobile authenticator connection.
  Uses centralized Cache_Service for session storage.
  """

  alias MfaService.Challenge
  alias MfaService.Passkeys.{Authentication, Config}
  alias AuthPlatform.Clients.Cache

  @session_ttl 300

  @type qr_data :: %{
          qr_code: String.t(),
          session_id: String.t(),
          expires_at: DateTime.t()
        }

  @doc """
  Generate QR code data for cross-device authentication.
  The QR code contains CTAP hybrid transport data in FIDO:// format.
  """
  @spec generate_qr_code(String.t() | nil) :: {:ok, qr_data()}
  def generate_qr_code(user_id \\ nil) do
    session_id = generate_session_id()
    tunnel_id = generate_tunnel_id()

    {:ok, auth_options} = Authentication.create_options(user_id, mediation: "optional")

    hybrid_data = %{
      version: 1,
      tunnel_id: tunnel_id,
      session_id: session_id,
      rp_id: Config.rp_id(),
      challenge: auth_options.challenge,
      timestamp: DateTime.utc_now() |> DateTime.to_unix()
    }

    {:ok, cbor_data} = CBOR.encode(hybrid_data)
    qr_content = "FIDO://" <> Base.url_encode64(cbor_data, padding: false)

    expires_at = DateTime.utc_now() |> DateTime.add(@session_ttl, :second)

    store_session(session_id, %{
      tunnel_id: tunnel_id,
      user_id: user_id,
      challenge: auth_options.challenge,
      expires_at: DateTime.to_iso8601(expires_at),
      completed: false
    })

    {:ok,
     %{
       qr_code: qr_content,
       session_id: session_id,
       expires_at: expires_at,
       tunnel_id: tunnel_id
     }}
  end

  @doc """
  Complete cross-device authentication after mobile authenticator responds.
  """
  @spec complete_authentication(String.t(), map()) :: {:ok, map()} | {:error, term()}
  def complete_authentication(session_id, assertion_response) do
    with {:ok, session} <- get_session(session_id),
         :ok <- verify_session_not_expired(session),
         {:ok, auth_result} <- Authentication.verify_assertion(assertion_response) do
      delete_session(session_id)

      {:ok,
       Map.merge(auth_result, %{
         cross_device: true,
         offer_local_registration: true
       })}
    end
  end

  @doc """
  Get fallback methods when cross-device authentication fails.
  """
  @spec get_fallback_on_failure(String.t(), term()) :: {:ok, map()}
  def get_fallback_on_failure(user_id, error) do
    {:ok, fallback_methods} = Authentication.get_fallback_methods(user_id)

    {:ok,
     %{
       error: :cross_device_failed,
       reason: error,
       fallback_methods: fallback_methods,
       user_id: user_id
     }}
  end

  @doc """
  Check the status of a cross-device authentication session.
  """
  @spec check_session_status(String.t()) :: {:ok, :pending | :completed | :expired} | {:error, :not_found}
  def check_session_status(session_id) do
    case get_session(session_id) do
      {:ok, session} ->
        expires_at = DateTime.from_iso8601!(session.expires_at)

        cond do
          DateTime.compare(DateTime.utc_now(), expires_at) == :gt -> {:ok, :expired}
          Map.get(session, :completed, false) -> {:ok, :completed}
          true -> {:ok, :pending}
        end

      {:error, :not_found} ->
        {:error, :not_found}
    end
  end

  defp generate_session_id do
    :crypto.strong_rand_bytes(16) |> Base.url_encode64(padding: false)
  end

  defp generate_tunnel_id do
    :crypto.strong_rand_bytes(32) |> Base.url_encode64(padding: false)
  end

  defp store_session(session_id, data) do
    key = "mfa:cross_device:#{session_id}"
    Cache.set(key, Jason.encode!(data), ttl: @session_ttl)
  end

  defp get_session(session_id) do
    key = "mfa:cross_device:#{session_id}"

    case Cache.get(key) do
      {:ok, nil} -> {:error, :not_found}
      {:ok, json} -> {:ok, Jason.decode!(json, keys: :atoms)}
      error -> error
    end
  end

  defp delete_session(session_id) do
    key = "mfa:cross_device:#{session_id}"
    Cache.delete(key)
  end

  defp verify_session_not_expired(session) do
    expires_at = DateTime.from_iso8601!(session.expires_at)

    if DateTime.compare(DateTime.utc_now(), expires_at) == :lt do
      :ok
    else
      {:error, :session_expired}
    end
  end
end
