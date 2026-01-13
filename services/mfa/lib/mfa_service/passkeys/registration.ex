defmodule MfaService.Passkeys.Registration do
  @moduledoc """
  WebAuthn registration with discoverable credentials (passkeys) support.
  Implements FIDO2/WebAuthn Level 2 specification.
  Uses centralized Challenge module for challenge storage.
  """

  alias MfaService.Challenge
  alias MfaService.Passkeys.{Config, PostgresProvider, Credential}
  alias AuthPlatform.Security

  @type user :: %{id: String.t(), name: String.t(), display_name: String.t()}
  @type registration_options :: map()
  @type attestation_response :: map()

  @doc """
  Generate registration options for creating a new passkey.
  Always sets residentKey to "required" for discoverable credentials.
  """
  @spec create_options(user(), keyword()) :: {:ok, registration_options()} | {:error, term()}
  def create_options(user, opts \\ []) do
    challenge = Challenge.generate()

    exclude_credentials =
      case PostgresProvider.get_credentials_for_user(user.id) do
        {:ok, creds} -> Enum.map(creds, &credential_descriptor/1)
        _ -> []
      end

    authenticator_attachment = Keyword.get(opts, :authenticator_attachment)

    options = %{
      challenge: Challenge.encode(challenge),
      rp: %{id: Config.rp_id(), name: Config.rp_name()},
      user: %{
        id: Base.url_encode64(user.id, padding: false),
        name: user.name,
        displayName: user.display_name
      },
      pubKeyCredParams: Config.pub_key_cred_params(),
      authenticatorSelection: build_authenticator_selection(authenticator_attachment),
      attestation: Config.attestation(),
      timeout: Config.timeout(),
      excludeCredentials: exclude_credentials
    }

    Challenge.store("passkey:reg:#{user.id}", challenge)
    {:ok, options}
  end

  @doc """
  Verify the attestation response and store the credential.
  """
  @spec verify_attestation(String.t(), attestation_response(), keyword()) ::
          {:ok, Credential.t()} | {:error, term()}
  def verify_attestation(user_id, response, opts \\ []) do
    with {:ok, challenge} <- Challenge.retrieve_and_delete("passkey:reg:#{user_id}"),
         {:ok, client_data} <- parse_client_data(response["clientDataJSON"]),
         :ok <- verify_client_data(client_data, challenge, "webauthn.create"),
         {:ok, attestation_object} <- parse_attestation_object(response["attestationObject"]),
         {:ok, auth_data} <- parse_authenticator_data(attestation_object["authData"]),
         :ok <- verify_rp_id_hash(auth_data.rp_id_hash),
         :ok <- verify_flags(auth_data.flags),
         {:ok, credential_data} <- extract_credential_data(auth_data) do
      credential_attrs = %{
        credential_id: credential_data.credential_id,
        public_key: credential_data.public_key,
        public_key_alg: credential_data.public_key_alg,
        sign_count: auth_data.sign_count,
        transports: response["transports"] || [],
        attestation_format: attestation_object["fmt"],
        attestation_statement: attestation_object["attStmt"],
        aaguid: credential_data.aaguid,
        device_name: Keyword.get(opts, :device_name, "Passkey"),
        is_discoverable: true,
        backed_up: auth_data.flags.backup_state
      }

      PostgresProvider.store_credential(user_id, credential_attrs)
    end
  end

  defp build_authenticator_selection(nil) do
    %{residentKey: "required", userVerification: "required"}
  end

  defp build_authenticator_selection(attachment) when attachment in ["platform", "cross-platform"] do
    %{residentKey: "required", userVerification: "required", authenticatorAttachment: attachment}
  end

  defp credential_descriptor(credential) do
    %{
      type: "public-key",
      id: Base.url_encode64(credential.credential_id, padding: false),
      transports: credential.transports
    }
  end

  defp parse_client_data(client_data_json) when is_binary(client_data_json) do
    case Base.url_decode64(client_data_json, padding: false) do
      {:ok, json} -> Jason.decode(json)
      :error -> {:error, :invalid_client_data}
    end
  end

  defp verify_client_data(client_data, challenge, expected_type) do
    with :ok <- verify_type(client_data["type"], expected_type),
         :ok <- verify_challenge(client_data["challenge"], challenge),
         :ok <- verify_origin(client_data["origin"]) do
      :ok
    end
  end

  defp verify_type(type, expected) when type == expected, do: :ok
  defp verify_type(_, _), do: {:error, :invalid_type}

  defp verify_challenge(challenge_b64, expected_challenge) do
    expected_b64 = Base.url_encode64(expected_challenge, padding: false)

    if Security.constant_time_compare(challenge_b64, expected_b64) do
      :ok
    else
      {:error, :challenge_mismatch}
    end
  end

  defp verify_origin(origin) do
    if origin in Config.origins(), do: :ok, else: {:error, :invalid_origin}
  end

  defp parse_attestation_object(attestation_object_b64) do
    with {:ok, cbor_data} <- Base.url_decode64(attestation_object_b64, padding: false),
         {:ok, decoded, _} <- CBOR.decode(cbor_data) do
      {:ok, decoded}
    else
      _ -> {:error, :invalid_attestation_object}
    end
  end

  defp parse_authenticator_data(auth_data) when is_binary(auth_data) do
    <<rp_id_hash::binary-size(32), flags::8, sign_count::unsigned-big-integer-size(32),
      rest::binary>> = auth_data

    parsed_flags = %{
      user_present: (flags &&& 0x01) == 0x01,
      user_verified: (flags &&& 0x04) == 0x04,
      attested_credential_data: (flags &&& 0x40) == 0x40,
      extension_data: (flags &&& 0x80) == 0x80,
      backup_eligibility: (flags &&& 0x08) == 0x08,
      backup_state: (flags &&& 0x10) == 0x10
    }

    {:ok, %{rp_id_hash: rp_id_hash, flags: parsed_flags, sign_count: sign_count, attested_credential_data: rest}}
  end

  defp verify_rp_id_hash(rp_id_hash) do
    expected_hash = :crypto.hash(:sha256, Config.rp_id())
    if Security.constant_time_compare(rp_id_hash, expected_hash), do: :ok, else: {:error, :rp_id_mismatch}
  end

  defp verify_flags(%{user_present: true, user_verified: true}), do: :ok
  defp verify_flags(_), do: {:error, :invalid_flags}

  defp extract_credential_data(%{attested_credential_data: data, flags: %{attested_credential_data: true}}) do
    <<aaguid::binary-size(16), cred_id_len::unsigned-big-integer-size(16),
      credential_id::binary-size(cred_id_len), public_key_cbor::binary>> = data

    with {:ok, public_key, _} <- CBOR.decode(public_key_cbor) do
      alg = public_key[3] || -7

      {:ok, %{
        aaguid: format_uuid(aaguid),
        credential_id: credential_id,
        public_key: public_key_cbor,
        public_key_alg: alg
      }}
    end
  end

  defp extract_credential_data(_), do: {:error, :no_credential_data}

  defp format_uuid(<<a::binary-size(4), b::binary-size(2), c::binary-size(2), d::binary-size(2), e::binary-size(6)>>) do
    [a, b, c, d, e] |> Enum.map(&Base.encode16(&1, case: :lower)) |> Enum.join("-")
  end
end
