defmodule MfaService.WebAuthn.Authentication do
  @moduledoc """
  WebAuthn authentication (assertion) handling.
  Uses centralized Challenge module and AuthPlatform for security.
  """

  alias MfaService.Challenge
  alias AuthPlatform.Security

  @type assertion :: map()
  @type credential :: map()
  @type auth_result :: %{sign_count: non_neg_integer()}

  @doc """
  Begins WebAuthn authentication by generating a challenge.
  """
  @spec begin_authentication(String.t(), [map()], keyword()) :: {:ok, map(), binary()}
  def begin_authentication(user_id, credentials, opts \\ []) do
    rp_id = Keyword.get(opts, :rp_id, "localhost")
    timeout = Keyword.get(opts, :timeout, 60_000)
    user_verification = Keyword.get(opts, :user_verification, "preferred")

    challenge = Challenge.generate()

    allow_credentials =
      Enum.map(credentials, fn cred ->
        %{
          type: "public-key",
          id: cred.credential_id,
          transports: cred.transports || []
        }
      end)

    options = %{
      challenge: challenge,
      timeout: timeout,
      rp_id: rp_id,
      allow_credentials: allow_credentials,
      user_verification: user_verification
    }

    Challenge.store("webauthn:auth:#{user_id}", challenge)

    {:ok, options, challenge}
  end

  @doc """
  Completes WebAuthn authentication by verifying the assertion.
  """
  @spec complete_authentication(assertion(), credential(), binary(), keyword()) ::
          {:ok, auth_result()} | {:error, term()}
  def complete_authentication(assertion, stored_credential, expected_challenge, opts \\ []) do
    rp_id = Keyword.get(opts, :rp_id, "localhost")
    origin = Keyword.get(opts, :origin, "https://#{rp_id}")

    with :ok <- verify_client_data(assertion.client_data_json, expected_challenge, origin),
         :ok <- verify_authenticator_data(assertion.authenticator_data, rp_id),
         :ok <- verify_signature(assertion, stored_credential),
         :ok <- verify_sign_count(assertion.authenticator_data, stored_credential.sign_count) do
      new_sign_count = extract_sign_count(assertion.authenticator_data)
      {:ok, %{sign_count: new_sign_count}}
    end
  end

  @doc """
  Parses authenticator data binary into structured map.
  """
  @spec parse_authenticator_data(binary()) :: {:ok, map()} | {:error, :invalid_auth_data}
  def parse_authenticator_data(auth_data) when byte_size(auth_data) >= 37 do
    <<rp_id_hash::binary-size(32), flags::8, sign_count::unsigned-big-integer-size(32),
      rest::binary>> = auth_data

    {:ok,
     %{
       rp_id_hash: rp_id_hash,
       flags: parse_flags(flags),
       sign_count: sign_count,
       rest: rest
     }}
  end

  def parse_authenticator_data(_), do: {:error, :invalid_auth_data}

  defp parse_flags(flags) do
    %{
      user_present: (flags &&& 0x01) == 0x01,
      user_verified: (flags &&& 0x04) == 0x04,
      backup_eligibility: (flags &&& 0x08) == 0x08,
      backup_state: (flags &&& 0x10) == 0x10,
      attested_credential_data: (flags &&& 0x40) == 0x40,
      extension_data: (flags &&& 0x80) == 0x80
    }
  end

  defp verify_client_data(client_data_json, expected_challenge, expected_origin) do
    with {:ok, client_data} <- Jason.decode(client_data_json) do
      cond do
        client_data["type"] != "webauthn.get" ->
          {:error, :invalid_type}

        not challenge_matches?(client_data["challenge"], expected_challenge) ->
          {:error, :challenge_mismatch}

        client_data["origin"] != expected_origin ->
          {:error, :origin_mismatch}

        true ->
          :ok
      end
    end
  end

  defp challenge_matches?(received_b64, expected) do
    expected_b64 = Base.url_encode64(expected, padding: false)
    Security.constant_time_compare(received_b64, expected_b64)
  end

  defp verify_authenticator_data(auth_data, expected_rp_id) do
    <<rp_id_hash::binary-size(32), flags::8, _rest::binary>> = auth_data
    expected_hash = :crypto.hash(:sha256, expected_rp_id)

    cond do
      not Security.constant_time_compare(rp_id_hash, expected_hash) ->
        {:error, :rp_id_mismatch}

      (flags &&& 0x01) == 0 ->
        {:error, :user_not_present}

      true ->
        :ok
    end
  end

  defp verify_signature(assertion, stored_credential) do
    client_data_hash = :crypto.hash(:sha256, assertion.client_data_json)
    signed_data = assertion.authenticator_data <> client_data_hash

    case :public_key.verify(
           signed_data,
           :sha256,
           assertion.signature,
           decode_public_key(stored_credential.public_key)
         ) do
      true -> :ok
      false -> {:error, :invalid_signature}
    end
  end

  defp verify_sign_count(auth_data, stored_sign_count) do
    new_sign_count = extract_sign_count(auth_data)

    if new_sign_count > stored_sign_count do
      :ok
    else
      {:error, :sign_count_not_increased}
    end
  end

  defp extract_sign_count(auth_data) do
    <<_rp_id_hash::binary-size(32), _flags::8, sign_count::unsigned-big-integer-size(32),
      _rest::binary>> = auth_data

    sign_count
  end

  defp decode_public_key(public_key_cbor), do: public_key_cbor
end
