defmodule SessionIdentityCore.Crypto.SessionEncryption do
  @moduledoc """
  Session data encryption integration.
  
  Wraps session serialization with encryption using crypto-service.
  Handles transparent encryption/decryption for SessionStore.
  """

  require Logger

  alias SessionIdentityCore.Crypto.{
    EncryptedStore,
    KeyRotation,
    Correlation
  }

  @doc """
  Encrypts session data for storage.
  
  Uses session_id as AAD for binding integrity.
  """
  @spec encrypt_session(binary(), String.t(), keyword()) :: {:ok, binary()} | {:error, term()}
  def encrypt_session(serialized_data, session_id, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    aad = EncryptedStore.build_session_aad(session_id)
    
    case EncryptedStore.encrypt(:session, serialized_data, aad, opts) do
      {:ok, envelope} ->
        KeyRotation.log_key_usage(:session_encrypt, envelope, opts)
        {:ok, Jason.encode!(envelope)}
      
      {:error, _} = error ->
        Logger.error("Failed to encrypt session",
          Correlation.extract_for_logging(opts) ++ [session_id: session_id]
        )
        error
    end
  end

  @doc """
  Decrypts session data from storage.
  
  Handles key rotation by re-encrypting if needed.
  Returns {:ok, data, should_update} where should_update indicates
  if the stored value should be updated with new encryption.
  """
  @spec decrypt_session(binary(), String.t(), keyword()) :: 
    {:ok, binary(), boolean()} | {:error, term()}
  def decrypt_session(encrypted_data, session_id, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    aad = EncryptedStore.build_session_aad(session_id)
    
    with {:ok, envelope} <- Jason.decode(encrypted_data),
         {:ok, plaintext, new_envelope} <- 
           KeyRotation.decrypt_and_maybe_reencrypt(:session, envelope, aad, opts) do
      KeyRotation.log_key_usage(:session_decrypt, envelope, opts)
      {:ok, plaintext, new_envelope != nil}
    else
      {:error, %Jason.DecodeError{}} ->
        # Not encrypted - return as-is for backward compatibility
        {:ok, encrypted_data, false}
      
      {:error, _} = error ->
        Logger.error("Failed to decrypt session",
          Correlation.extract_for_logging(opts) ++ [session_id: session_id]
        )
        error
    end
  end

  @doc """
  Checks if data is encrypted (has envelope structure).
  """
  @spec encrypted?(binary()) :: boolean()
  def encrypted?(data) do
    case Jason.decode(data) do
      {:ok, %{"v" => _, "key_id" => _, "ciphertext" => _}} -> true
      _ -> false
    end
  end
end
