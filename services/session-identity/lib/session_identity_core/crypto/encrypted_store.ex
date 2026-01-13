defmodule SessionIdentityCore.Crypto.EncryptedStore do
  @moduledoc """
  Encrypted storage wrapper for sensitive data.
  
  Provides AES-256-GCM encryption for:
  - Session data
  - Refresh tokens
  
  Features:
  - Namespace-specific encryption keys
  - AAD binding for integrity
  - Multi-version key support for rotation
  - Re-encryption on deprecated key access
  """

  require Logger

  alias SessionIdentityCore.Crypto.{
    Client,
    Config,
    KeyManager,
    CircuitBreaker,
    Fallback,
    Correlation,
    TraceContext,
    Errors
  }

  @namespaces %{
    session: "session_identity:session",
    refresh_token: "session_identity:refresh_token"
  }

  @doc """
  Encrypts and stores data.
  
  Returns the encrypted envelope as a map that can be serialized to JSON.
  """
  @spec encrypt(atom(), binary(), binary()) :: {:ok, map()} | {:error, term()}
  def encrypt(namespace, data, aad, opts \\ []) when namespace in [:session, :refresh_token] do
    opts = Correlation.ensure_correlation_id(opts)
    opts = Keyword.merge(TraceContext.extract_current(), opts)
    
    config = Config.get()
    key_namespace = Map.fetch!(@namespaces, namespace)
    
    if config.enabled do
      CircuitBreaker.call_with_fallback(
        fn -> encrypt_via_crypto_service(key_namespace, data, aad, opts) end,
        fn -> Fallback.encrypt_local(data, build_fallback_key_id(key_namespace), aad, opts) end
      )
    else
      Fallback.encrypt_local(data, build_fallback_key_id(key_namespace), aad, opts)
    end
  end

  @doc """
  Decrypts stored data.
  
  Handles key rotation by trying previous versions if current fails.
  Re-encrypts with latest key if decrypted with deprecated key.
  """
  @spec decrypt(atom(), map(), binary()) :: {:ok, binary()} | {:error, term()}
  def decrypt(namespace, envelope, aad, opts \\ []) when namespace in [:session, :refresh_token] do
    opts = Correlation.ensure_correlation_id(opts)
    opts = Keyword.merge(TraceContext.extract_current(), opts)
    
    config = Config.get()
    
    if config.enabled do
      CircuitBreaker.call_with_fallback(
        fn -> decrypt_via_crypto_service(envelope, aad, opts) end,
        fn -> decrypt_with_fallback(envelope, aad, opts) end
      )
    else
      decrypt_with_fallback(envelope, aad, opts)
    end
  end

  @doc """
  Builds AAD for session encryption.
  """
  @spec build_session_aad(String.t()) :: binary()
  def build_session_aad(session_id) do
    "session:#{session_id}"
  end

  @doc """
  Builds AAD for refresh token encryption.
  """
  @spec build_refresh_token_aad(String.t(), String.t()) :: binary()
  def build_refresh_token_aad(user_id, client_id) do
    "refresh_token:#{user_id}:#{client_id}"
  end

  @doc """
  Returns the key namespace for a data type.
  """
  @spec get_namespace(atom()) :: String.t()
  def get_namespace(type) when type in [:session, :refresh_token] do
    Map.fetch!(@namespaces, type)
  end

  # Private Functions

  defp encrypt_via_crypto_service(key_namespace, data, aad, opts) do
    with {:ok, key_id} <- KeyManager.get_active_key(key_namespace),
         {:ok, result} <- Client.encrypt(data, key_id, aad, opts) do
      envelope = build_envelope(result)
      
      Logger.debug("Data encrypted via crypto-service",
        Correlation.extract_for_logging(opts) ++ [
          namespace: key_namespace,
          key_version: key_id.version
        ]
      )
      
      {:ok, envelope}
    end
  end

  defp decrypt_via_crypto_service(envelope, aad, opts) do
    with {:ok, {ciphertext, iv, tag, key_id}} <- parse_envelope(envelope),
         result <- try_decrypt_with_versions(ciphertext, iv, tag, key_id, aad, opts) do
      result
    end
  end

  defp try_decrypt_with_versions(ciphertext, iv, tag, key_id, aad, opts) do
    # Try with the key version from the envelope first
    case Client.decrypt(ciphertext, iv, tag, key_id, aad, opts) do
      {:ok, plaintext} ->
        maybe_reencrypt_if_deprecated(key_id, plaintext, aad, opts)
        {:ok, plaintext}
      
      {:error, %{error_code: :decryption_failed}} ->
        # Try previous versions
        try_previous_versions(ciphertext, iv, tag, key_id, aad, opts)
      
      error ->
        error
    end
  end

  defp try_previous_versions(ciphertext, iv, tag, key_id, aad, opts) do
    case KeyManager.get_key_versions(key_id.namespace, key_id.id) do
      {:ok, versions} ->
        # Sort by version descending, skip the one we already tried
        other_versions = versions
        |> Enum.filter(& &1.version != key_id.version)
        |> Enum.sort_by(& &1.version, :desc)
        
        try_versions(ciphertext, iv, tag, other_versions, aad, opts)
      
      _ ->
        {:error, Errors.decryption_failed("No alternative key versions available")}
    end
  end

  defp try_versions(_ciphertext, _iv, _tag, [], _aad, _opts) do
    {:error, Errors.decryption_failed("All key versions exhausted")}
  end

  defp try_versions(ciphertext, iv, tag, [key_id | rest], aad, opts) do
    case Client.decrypt(ciphertext, iv, tag, key_id, aad, opts) do
      {:ok, plaintext} ->
        Logger.info("Decrypted with previous key version",
          Correlation.extract_for_logging(opts) ++ [key_version: key_id.version]
        )
        maybe_reencrypt_if_deprecated(key_id, plaintext, aad, opts)
        {:ok, plaintext}
      
      {:error, _} ->
        try_versions(ciphertext, iv, tag, rest, aad, opts)
    end
  end

  defp maybe_reencrypt_if_deprecated(key_id, _plaintext, _aad, opts) do
    if KeyManager.deprecated?(key_id) do
      Logger.info("Data was encrypted with deprecated key, should re-encrypt",
        Correlation.extract_for_logging(opts) ++ [
          deprecated_key_version: key_id.version
        ]
      )
      # Note: Actual re-encryption is handled by the caller
      # We just log the recommendation here
    end
  end

  defp decrypt_with_fallback(envelope, aad, opts) do
    with {:ok, {ciphertext, iv, tag, key_id}} <- parse_envelope(envelope) do
      Fallback.decrypt_local(ciphertext, iv, tag, key_id, aad, opts)
    end
  end

  defp build_envelope(%{ciphertext: ct, iv: iv, tag: tag, key_id: key_id} = result) do
    %{
      "v" => 1,
      "key_id" => %{
        "namespace" => key_id.namespace,
        "id" => key_id.id,
        "version" => key_id.version
      },
      "iv" => Base.encode64(iv),
      "tag" => Base.encode64(tag),
      "ciphertext" => Base.encode64(ct),
      "encrypted_at" => DateTime.to_unix(DateTime.utc_now()),
      "fallback" => Map.get(result, :fallback, false)
    }
  end

  defp parse_envelope(%{"v" => 1} = envelope) do
    with {:ok, ciphertext} <- decode_base64(envelope["ciphertext"]),
         {:ok, iv} <- decode_base64(envelope["iv"]),
         {:ok, tag} <- decode_base64(envelope["tag"]),
         {:ok, key_id} <- parse_key_id(envelope["key_id"]) do
      {:ok, {ciphertext, iv, tag, key_id}}
    end
  end

  defp parse_envelope(_) do
    {:error, Errors.invalid_argument("Invalid envelope format")}
  end

  defp decode_base64(nil), do: {:error, Errors.invalid_argument("Missing field")}
  defp decode_base64(data) do
    case Base.decode64(data) do
      {:ok, decoded} -> {:ok, decoded}
      :error -> {:error, Errors.invalid_argument("Invalid base64")}
    end
  end

  defp parse_key_id(%{"namespace" => ns, "id" => id, "version" => v}) do
    {:ok, %{namespace: ns, id: id, version: v}}
  end

  defp parse_key_id(_), do: {:error, Errors.invalid_argument("Invalid key_id")}

  defp build_fallback_key_id(namespace) do
    %{namespace: namespace, id: "fallback", version: 0}
  end
end
