defmodule SessionIdentityCore.Crypto.RefreshTokenEncryption do
  @moduledoc """
  Refresh token encryption integration.
  
  Wraps refresh token payload with encryption using crypto-service.
  Uses user_id + client_id as AAD for binding integrity.
  """

  require Logger

  alias SessionIdentityCore.Crypto.{
    EncryptedStore,
    KeyRotation,
    Correlation
  }

  @doc """
  Encrypts refresh token payload for storage.
  """
  @spec encrypt_token(binary(), String.t(), String.t(), keyword()) :: 
    {:ok, binary()} | {:error, term()}
  def encrypt_token(payload, user_id, client_id, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    aad = EncryptedStore.build_refresh_token_aad(user_id, client_id)
    
    case EncryptedStore.encrypt(:refresh_token, payload, aad, opts) do
      {:ok, envelope} ->
        KeyRotation.log_key_usage(:refresh_token_encrypt, envelope, opts)
        {:ok, Jason.encode!(envelope)}
      
      {:error, _} = error ->
        Logger.error("Failed to encrypt refresh token",
          Correlation.extract_for_logging(opts) ++ [user_id: user_id]
        )
        error
    end
  end

  @doc """
  Decrypts refresh token payload from storage.
  
  Returns {:ok, payload, should_update} where should_update indicates
  if the stored value should be updated with new encryption.
  """
  @spec decrypt_token(binary(), String.t(), String.t(), keyword()) :: 
    {:ok, binary(), boolean()} | {:error, term()}
  def decrypt_token(encrypted_data, user_id, client_id, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    aad = EncryptedStore.build_refresh_token_aad(user_id, client_id)
    
    with {:ok, envelope} <- Jason.decode(encrypted_data),
         {:ok, plaintext, new_envelope} <- 
           KeyRotation.decrypt_and_maybe_reencrypt(:refresh_token, envelope, aad, opts) do
      KeyRotation.log_key_usage(:refresh_token_decrypt, envelope, opts)
      {:ok, plaintext, new_envelope != nil}
    else
      {:error, %Jason.DecodeError{}} ->
        {:ok, encrypted_data, false}
      
      {:error, _} = error ->
        Logger.error("Failed to decrypt refresh token",
          Correlation.extract_for_logging(opts) ++ [user_id: user_id]
        )
        error
    end
  end

  @doc """
  Checks if data is encrypted.
  """
  @spec encrypted?(binary()) :: boolean()
  def encrypted?(data) do
    case Jason.decode(data) do
      {:ok, %{"v" => _, "key_id" => _, "ciphertext" => _}} -> true
      _ -> false
    end
  end
end
