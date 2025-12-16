defmodule MfaService.Passkeys.Authentication do
  @moduledoc """
  WebAuthn authentication with discoverable credentials (passkeys) support.
  Supports Conditional UI and cross-device authentication.
  """

  alias MfaService.Passkeys.{Config, PostgresProvider}

  @type authentication_options :: map()
  @type assertion_response :: map()
  @type auth_result :: %{
          credential_id: binary(),
          user_id: String.t(),
          sign_count: non_neg_integer(),
          backed_up: boolean(),
          user_verified: boolean()
        }

  @doc """
  Generate authentication options for passkey login.
  Supports both username-based and usernameless (discoverable) flows.
  """
  @spec create_options(String.t() | nil, keyword()) :: {:ok, authentication_options()}
  def create_options(user_id \\ nil, opts \\ []) do
    challenge = generate_challenge()
    mediation = Keyword.get(opts, :mediation, "optional")

    allow_credentials =
      if user_id do
        case PostgresProvider.get_credentials_for_user(user_id) do
          {:ok, creds} -> Enum.map(creds, &credential_descriptor/1)
          _ -> []
        end
      else
        # Empty for discoverable credentials (usernameless)
        []
      end

    options = %{
      challenge: Base.url_encode64(challenge, padding: false),
      rpId: Config.rp_id(),
      userVerification: "required",
      timeout: Config.timeout(),
      allowCredentials: allow_credentials,
      # For Conditional UI support
      mediation: mediation
    }

    # Store challenge for verification
    store_challenge(challenge, user_id)

    {:ok, options}
  end

  @doc """
  Verify the assertion response and return authentication result.
  """
  @spec verify_assertion(assertion_response(), keyword()) :: {:ok, auth_result()} | {:error, term()}
  def verify_assertion(response, _opts \\ []) do
    credential_id = decode_credential_id(response["id"])

    with {:ok, credential} <- PostgresProvider.get_credential(credential_id),
         {:ok, challenge} <- get_stored_challenge(credential_id),
         {:ok, client_data} <- parse_client_data(response["clientDataJSON"]),
         :ok <- verify_client_data(client_data, challenge, "webauthn.get"),
         {:ok, auth_data} <- parse_authenticator_data(response["authenticatorData"]),
         :ok <- verify_rp_id_hash(auth_data.rp_id_hash),
         :ok <- verify_user_verification(auth_data.flags),
         :ok <- verify_signature(credential, auth_data, client_data, response["signature"]),
         :ok <- verify_sign_count(credential.sign_count, auth_data.sign_count),
         {:ok, updated_credential} <-
           PostgresProvider.increment_sign_count(credential_id, auth_data.sign_count) do
      clear_challenge(credential_id)

      {:ok,
       %{
         credential_id: credential_id,
         user_id: credential.user_id,
         sign_count: auth_data.sign_count,
         backed_up: auth_data.flags.backup_state,
         user_verified: auth_data.flags.user_verified,
         authenticator_type: determine_authenticator_type(updated_credential)
       }}
    end
  end

  @doc """
  Get available fallback methods for a user when passkey auth fails.
  """
  @spec get_fallback_methods(String.t()) :: {:ok, [String.t()]}
  def get_fallback_methods(user_id) do
    # Query available MFA methods for user
    methods = []

    # Check TOTP
    methods = if has_totp?(user_id), do: ["totp" | methods], else: methods

    # Check backup codes
    methods = if has_backup_codes?(user_id), do: ["backup_codes" | methods], else: methods

    {:ok, methods}
  end

  # Private functions

  defp generate_challenge do
    :crypto.strong_rand_bytes(32)
  end

  defp credential_descriptor(credential) do
    %{
      type: "public-key",
      id: Base.url_encode64(credential.credential_id, padding: false),
      transports: credential.transports
    }
  end

  defp decode_credential_id(id) when is_binary(id) do
    case Base.url_decode64(id, padding: false) do
      {:ok, decoded} -> decoded
      :error -> id
    end
  end

  defp store_challenge(challenge, user_id) do
    key = if user_id, do: "passkey_auth:#{user_id}", else: "passkey_auth:#{Base.encode16(challenge)}"
    :ets.insert(:passkey_challenges, {key, challenge, System.system_time(:second) + 300})
    :ok
  end

  defp get_stored_challenge(credential_id) do
    # Try to find challenge by credential_id or iterate
    case :ets.match(:passkey_challenges, {:"$1", :"$2", :"$3"}) do
      matches when is_list(matches) ->
        now = System.system_time(:second)

        case Enum.find(matches, fn [_key, _challenge, expires] -> expires > now end) do
          [_key, challenge, _expires] -> {:ok, challenge}
          nil -> {:error, :challenge_not_found}
        end

      _ ->
        {:error, :challenge_not_found}
    end
  end

  defp clear_challenge(_credential_id) do
    # Clear all expired challenges
    now = System.system_time(:second)

    :ets.match(:passkey_challenges, {:"$1", :_, :"$2"})
    |> Enum.filter(fn [_key, expires] -> expires <= now end)
    |> Enum.each(fn [key, _] -> :ets.delete(:passkey_challenges, key) end)

    :ok
  end

  defp parse_client_data(client_data_json) when is_binary(client_data_json) do
    case Base.url_decode64(client_data_json, padding: false) do
      {:ok, json} -> Jason.decode(json)
      :error -> {:error, :invalid_client_data}
    end
  end

  defp verify_client_data(client_data, challenge, expected_type) do
    with :ok <- verify_type(client_data["type"], expected_type),
ok <- verify_challenge(client_data["challenge"], challenge),
         :ok <- verify_origin(client_data["origin"]) do
      :ok
    end
  end

  defp verify_type(type, expected) when type == expected, do: :ok
  defp verify_type(_, _), do: {:error, :invalid_type}

  defp verify_challenge(challenge_b64, expected_challenge) do
    case Base.url_decode64(challenge_b64, padding: false) do
      {:ok, ^expected_challenge} -> :ok
      _ -> {:error, :challenge_mismatch}
    end
  end

  defp verify_origin(origin) do
    if origin in Config.origins(), do: :ok, else: {:error, :invalid_origin}
  end

  defp parse_authenticator_data(auth_data_b64) when is_binary(auth_data_b64) do
    with {:ok, auth_data} <- Base.url_decode64(auth_data_b64, padding: false) do
      <<
        rp_id_hash::binary-size(32),
        flags::8,
        sign_count::unsigned-big-integer-size(32),
        _rest::binary
      >> = auth_data

      parsed_flags = %{
        user_present: (flags &&& 0x01) == 0x01,
        user_verified: (flags &&& 0x04) == 0x04,
        backup_eligibility: (flags &&& 0x08) == 0x08,
        backup_state: (flags &&& 0x10) == 0x10
      }

      {:ok,
       %{
         rp_id_hash: rp_id_hash,
         flags: parsed_flags,
         sign_count: sign_count,
         raw: auth_data
       }}
    end
  end

  defp verify_rp_id_hash(rp_id_hash) do
    expected_hash = :crypto.hash(:sha256, Config.rp_id())
    if rp_id_hash == expected_hash, do: :ok, else: {:error, :rp_id_mismatch}
  end

  defp verify_user_verification(%{user_verified: true}), do: :ok
  defp verify_user_verification(_), do: {:error, :user_verification_required}

  defp verify_signature(credential, auth_data, client_data, signature_b64) do
    with {:ok, signature} <- Base.url_decode64(signature_b64, padding: false),
         {:ok, public_key} <- decode_public_key(credential.public_key, credential.public_key_alg) do
      client_data_hash = :crypto.hash(:sha256, Jason.encode!(client_data))
      signed_data = auth_data.raw <> client_data_hash

      if :public_key.verify(signed_data, :sha256, signature, public_key) do
        :ok
      else
        {:error, :invalid_signature}
      end
    end
  end

  defp decode_public_key(public_key_cbor, alg) do
    with {:ok, cose_key, _} <- CBOR.decode(public_key_cbor) do
      case alg do
        -7 -> decode_ec_key(cose_key)
        -257 -> decode_rsa_key(cose_key)
        _ -> {:error, :unsupported_algorithm}
      end
    end
  end

  defp decode_ec_key(cose_key) do
    x = cose_key[-2]
    y = cose_key[-3]
    {:ok, {:ECPoint, <<0x04>> <> x <> y}}
  end

  defp decode_rsa_key(cose_key) do
    n = cose_key[-1]
    e = cose_key[-2]
    {:ok, {:RSAPublicKey, :binary.decode_unsigned(n), :binary.decode_unsigned(e)}}
  end

  defp verify_sign_count(stored_count, new_count) do
    if new_count > stored_count do
      :ok
    else
      # Sign count didn't increase - possible cloned authenticator
      {:error, :sign_count_not_increased}
    end
  end

  defp determine_authenticator_type(credential) do
    cond do
      "internal" in credential.transports -> "platform"
      "hybrid" in credential.transports -> "cross-platform"
      true -> "unknown"
    end
  end

  defp has_totp?(_user_id), do: false
  defp has_backup_codes?(_user_id), do: false
end
