defmodule MfaService.Crypto.TOTPEncryptor do
  @moduledoc """
  TOTP secret encryption/decryption using crypto-service.
  Supports both legacy local encryption and crypto-service encryption.
  Implements lazy migration from local to crypto-service.
  """

  require Logger

  alias MfaService.Crypto.{Client, KeyManager, SecretFormat, Error}

  @type encrypted_secret :: String.t()
  @type secret :: String.t()

  @doc """
  Encrypts a TOTP secret using crypto-service.
  The user_id is used as AAD (Additional Authenticated Data) to bind
  the ciphertext to the user.
  
  Returns base64-encoded encrypted payload with version byte.
  """
  @spec encrypt_secret(secret(), String.t()) :: {:ok, encrypted_secret()} | {:error, Error.t()}
  def encrypt_secret(secret, user_id) when is_binary(secret) and is_binary(user_id) do
    correlation_id = generate_correlation_id()
    
    with {:ok, key_id} <- KeyManager.get_active_key_id(),
         {:ok, result} <- Client.encrypt(secret, key_id, user_id, correlation_id) do
      
      # Encode to v2 format
      payload = SecretFormat.encode_v2(
        result.key_id,
        result.iv,
        result.tag,
        result.ciphertext
      )
      
      {:ok, Base.encode64(payload)}
    else
      {:error, %Error{} = error} ->
        Logger.error("TOTP secret encryption failed",
          error_code: error.code, correlation_id: correlation_id)
        {:error, error}

      {:error, reason} ->
        Logger.error("TOTP secret encryption failed",
          reason: inspect(reason), correlation_id: correlation_id)
        {:error, Error.new(:encryption_failed, reason, correlation_id)}
    end
  end

  @doc """
  Decrypts a TOTP secret.
  Automatically detects the encryption version and uses the appropriate method.
  The user_id must match the AAD used during encryption.
  """
  @spec decrypt_secret(encrypted_secret(), String.t()) :: {:ok, secret()} | {:error, Error.t()}
  def decrypt_secret(encrypted, user_id) when is_binary(encrypted) and is_binary(user_id) do
    correlation_id = generate_correlation_id()
    
    with {:ok, payload} <- Base.decode64(encrypted),
         {:ok, version, data} <- SecretFormat.decode(payload) do
      
      case version do
        :crypto_service ->
          decrypt_v2(data, user_id, correlation_id)

        :local ->
          decrypt_v1(data, user_id, correlation_id)
      end
    else
      :error ->
        {:error, Error.new(:decryption_failed, "Invalid base64 encoding", correlation_id)}

      {:error, :unknown_version} ->
        {:error, Error.new(:decryption_failed, "Unknown encryption version", correlation_id)}

      {:error, :invalid_format} ->
        {:error, Error.new(:decryption_failed, "Invalid payload format", correlation_id)}
    end
  end

  @doc """
  Detects the encryption version of an encrypted secret.
  """
  @spec detect_version(encrypted_secret()) :: {:ok, :local | :crypto_service} | {:error, :invalid}
  def detect_version(encrypted) when is_binary(encrypted) do
    case Base.decode64(encrypted) do
      {:ok, payload} ->
        case SecretFormat.detect_version(payload) do
          :unknown -> {:error, :invalid}
          version -> {:ok, version}
        end

      :error ->
        {:error, :invalid}
    end
  end

  @doc """
  Checks if a secret needs migration to crypto-service.
  Returns true if the secret is encrypted with local encryption (v1).
  """
  @spec needs_migration?(encrypted_secret()) :: boolean()
  def needs_migration?(encrypted) do
    case detect_version(encrypted) do
      {:ok, :local} -> true
      _ -> false
    end
  end

  @doc """
  Migrates a locally-encrypted secret to crypto-service encryption.
  Decrypts with local key, re-encrypts with crypto-service.
  """
  @spec migrate_secret(encrypted_secret(), String.t(), binary()) :: 
    {:ok, encrypted_secret()} | {:error, Error.t()}
  def migrate_secret(encrypted, user_id, local_key) when byte_size(local_key) == 32 do
    correlation_id = generate_correlation_id()
    
    with {:ok, payload} <- Base.decode64(encrypted),
         {:ok, :local, data} <- SecretFormat.decode(payload),
         {:ok, secret} <- decrypt_local(data, local_key),
         {:ok, new_encrypted} <- encrypt_secret(secret, user_id) do
      
      Logger.info("TOTP secret migrated to crypto-service",
        user_id: sanitize_user_id(user_id), correlation_id: correlation_id)
      
      {:ok, new_encrypted}
    else
      {:error, %Error{} = error} ->
        {:error, error}

      {:error, reason} ->
        {:error, Error.new(:migration_failed, reason, correlation_id)}
    end
  end

  # Private functions

  defp decrypt_v2(data, user_id, correlation_id) do
    case Client.decrypt(
      data.ciphertext,
      data.iv,
      data.tag,
      data.key_id,
      user_id,
      correlation_id
    ) do
      {:ok, plaintext} ->
        {:ok, plaintext}

      {:error, %Error{} = error} ->
        Logger.error("TOTP secret decryption (v2) failed",
          error_code: error.code, correlation_id: correlation_id)
        {:error, error}
    end
  end

  defp decrypt_v1(data, _user_id, correlation_id) do
    # For v1 (local encryption), we need the local key
    # This is a fallback for legacy secrets
    Logger.warning("Decrypting legacy v1 secret - migration recommended",
      correlation_id: correlation_id)
    
    # In production, get the local key from secure storage
    case get_local_encryption_key() do
      {:ok, local_key} ->
        decrypt_local(data, local_key)

      {:error, _} ->
        {:error, Error.new(:decryption_failed, "Local key not available", correlation_id)}
    end
  end

  defp decrypt_local(%{iv: iv, tag: tag, ciphertext: ciphertext}, key) do
    case :crypto.crypto_one_time_aead(:aes_256_gcm, key, iv, ciphertext, "", tag) do
      :error ->
        {:error, :decryption_failed}

      plaintext ->
        {:ok, plaintext}
    end
  end

  defp get_local_encryption_key do
    # In production, this would retrieve the key from secure storage
    # For now, return error to force migration
    case System.get_env("MFA_LOCAL_ENCRYPTION_KEY") do
      nil -> {:error, :key_not_configured}
      key_b64 -> Base.decode64(key_b64)
    end
  end

  defp generate_correlation_id do
    UUID.uuid4()
  end

  defp sanitize_user_id(user_id) do
    # Only show first 8 chars for logging
    String.slice(user_id, 0, 8) <> "..."
  end
end
