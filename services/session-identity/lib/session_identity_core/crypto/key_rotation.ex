defmodule SessionIdentityCore.Crypto.KeyRotation do
  @moduledoc """
  Key rotation support for encrypted data.
  
  Provides:
  - Multi-version decryption during rotation period
  - Re-encryption with latest key on deprecated key access
  - Key version tracking and logging
  """

  require Logger

  alias SessionIdentityCore.Crypto.{
    KeyManager,
    EncryptedStore,
    Correlation
  }

  @doc """
  Checks if data needs re-encryption due to deprecated key.
  """
  @spec needs_reencryption?(map()) :: boolean()
  def needs_reencryption?(envelope) do
    case extract_key_id(envelope) do
      {:ok, key_id} -> KeyManager.deprecated?(key_id)
      _ -> false
    end
  end

  @doc """
  Re-encrypts data with the latest key version.
  
  Used when data encrypted with deprecated key is accessed.
  Returns the new envelope with updated key version.
  """
  @spec reencrypt(atom(), binary(), binary(), keyword()) :: {:ok, map()} | {:error, term()}
  def reencrypt(namespace, plaintext, aad, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    old_key_version = Keyword.get(opts, :old_key_version)
    
    Logger.info("Re-encrypting data with latest key",
      Correlation.extract_for_logging(opts) ++ [
        namespace: namespace,
        old_key_version: old_key_version
      ]
    )
    
    emit_reencryption_metric(namespace, old_key_version)
    
    EncryptedStore.encrypt(namespace, plaintext, aad, opts)
  end

  @doc """
  Decrypts and optionally re-encrypts if key is deprecated.
  
  Returns {:ok, plaintext, new_envelope} if re-encryption occurred,
  or {:ok, plaintext, nil} if no re-encryption needed.
  
  The caller is responsible for persisting the new_envelope if returned.
  """
  @spec decrypt_and_maybe_reencrypt(atom(), map(), binary(), keyword()) :: 
    {:ok, binary(), map() | nil} | {:error, term()}
  def decrypt_and_maybe_reencrypt(namespace, envelope, aad, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    
    case EncryptedStore.decrypt(namespace, envelope, aad, opts) do
      {:ok, plaintext} ->
        handle_potential_reencryption(namespace, envelope, plaintext, aad, opts)
      
      error ->
        error
    end
  end

  defp handle_potential_reencryption(namespace, envelope, plaintext, aad, opts) do
    if needs_reencryption?(envelope) do
      old_version = get_key_version(envelope)
      reencrypt_opts = Keyword.put(opts, :old_key_version, elem(old_version, 1))
      
      case reencrypt(namespace, plaintext, aad, reencrypt_opts) do
        {:ok, new_envelope} ->
          log_key_usage(:reencryption, new_envelope, opts)
          {:ok, plaintext, new_envelope}
        
        {:error, reason} ->
          Logger.warning("Re-encryption failed, returning decrypted data",
            Correlation.extract_for_logging(opts) ++ [error: inspect(reason)]
          )
          {:ok, plaintext, nil}
      end
    else
      {:ok, plaintext, nil}
    end
  end

  @doc """
  Gets the key version from an envelope.
  """
  @spec get_key_version(map()) :: {:ok, non_neg_integer()} | {:error, term()}
  def get_key_version(envelope) do
    case extract_key_id(envelope) do
      {:ok, %{version: version}} -> {:ok, version}
      error -> error
    end
  end

  @doc """
  Checks if envelope was encrypted with a specific key version.
  """
  @spec encrypted_with_version?(map(), non_neg_integer()) :: boolean()
  def encrypted_with_version?(envelope, version) do
    case get_key_version(envelope) do
      {:ok, ^version} -> true
      _ -> false
    end
  end

  @doc """
  Gets all available key versions for a namespace.
  """
  @spec available_versions(String.t()) :: {:ok, [map()]} | {:error, term()}
  def available_versions(namespace) do
    case KeyManager.get_active_key(namespace) do
      {:ok, key_id} ->
        KeyManager.get_key_versions(namespace, key_id.id)
      error ->
        error
    end
  end

  @doc """
  Logs key version used for an operation.
  """
  @spec log_key_usage(atom(), map(), keyword()) :: :ok
  def log_key_usage(operation, envelope, opts \\ []) do
    case extract_key_id(envelope) do
      {:ok, key_id} ->
        Logger.debug("Crypto operation completed",
          Correlation.extract_for_logging(opts) ++ [
            operation: operation,
            key_namespace: key_id.namespace,
            key_id: key_id.id,
            key_version: key_id.version
          ]
        )
      _ ->
        :ok
    end
  end

  # Private Functions

  defp extract_key_id(%{"key_id" => %{"namespace" => ns, "id" => id, "version" => v}}) do
    {:ok, %{namespace: ns, id: id, version: v}}
  end

  defp extract_key_id(_), do: {:error, :invalid_envelope}

  defp emit_reencryption_metric(namespace, old_key_version) do
    :telemetry.execute(
      [:session_identity, :crypto, :reencryption],
      %{count: 1},
      %{namespace: namespace, old_key_version: old_key_version}
    )
  end
end
