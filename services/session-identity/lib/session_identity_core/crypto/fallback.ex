defmodule SessionIdentityCore.Crypto.Fallback do
  @moduledoc """
  Local fallback implementations for crypto operations.
  
  Used when crypto-service is unavailable and fallback is enabled.
  Emits warning logs and metrics when fallback is used.
  """

  require Logger

  alias SessionIdentityCore.Crypto.Config

  @doc """
  Local JWT signing fallback using Joken.
  
  WARNING: Uses local keys, not centrally managed keys.
  """
  @spec sign_jwt_local(map(), keyword()) :: {:ok, String.t()} | {:error, term()}
  def sign_jwt_local(claims, opts \\ []) do
    Logger.warning("Using local JWT signing fallback - keys not centrally managed",
      correlation_id: Keyword.get(opts, :correlation_id)
    )
    emit_fallback_metric(:sign_jwt)

    try do
      signer = get_local_signer()
      {:ok, token, _claims} = Joken.encode_and_sign(claims, signer)
      {:ok, token}
    rescue
      e ->
        Logger.error("Local JWT signing failed: #{inspect(e)}")
        {:error, %{error_code: :signing_failed, message: Exception.message(e)}}
    end
  end

  @doc """
  Local JWT verification fallback using Joken.
  """
  @spec verify_jwt_local(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def verify_jwt_local(token, opts \\ []) do
    Logger.warning("Using local JWT verification fallback",
      correlation_id: Keyword.get(opts, :correlation_id)
    )
    emit_fallback_metric(:verify_jwt)

    try do
      signer = get_local_signer()
      case Joken.verify_and_validate(token, signer) do
        {:ok, claims} -> {:ok, claims}
        {:error, reason} -> {:error, %{error_code: :signature_invalid, message: inspect(reason)}}
      end
    rescue
      e ->
        Logger.error("Local JWT verification failed: #{inspect(e)}")
        {:error, %{error_code: :signature_invalid, message: Exception.message(e)}}
    end
  end

  @doc """
  Local encryption fallback - stores data unencrypted with warning.
  
  WARNING: Data is NOT encrypted when using fallback.
  """
  @spec encrypt_local(binary(), map(), binary() | nil, keyword()) :: {:ok, map()} | {:error, term()}
  def encrypt_local(plaintext, key_id, _aad, opts \\ []) do
    Logger.warning("Using local encryption fallback - DATA IS NOT ENCRYPTED",
      correlation_id: Keyword.get(opts, :correlation_id),
      key_id: key_id
    )
    emit_fallback_metric(:encrypt)

    # Return plaintext wrapped in a structure that indicates it's unencrypted
    {:ok, %{
      ciphertext: plaintext,
      iv: <<0::96>>,
      tag: <<0::128>>,
      key_id: key_id,
      fallback: true
    }}
  end

  @doc """
  Local decryption fallback - returns data as-is if marked as fallback.
  """
  @spec decrypt_local(binary(), binary(), binary(), map(), binary() | nil, keyword()) :: {:ok, binary()} | {:error, term()}
  def decrypt_local(ciphertext, iv, tag, _key_id, _aad, opts \\ []) do
    Logger.warning("Using local decryption fallback",
      correlation_id: Keyword.get(opts, :correlation_id)
    )
    emit_fallback_metric(:decrypt)

    # Check if this was encrypted with fallback (null IV and tag)
    if iv == <<0::96>> and tag == <<0::128>> do
      {:ok, ciphertext}
    else
      {:error, %{error_code: :decryption_failed, message: "Cannot decrypt data encrypted by crypto-service in fallback mode"}}
    end
  end

  @doc """
  Checks if fallback is enabled in configuration.
  """
  @spec enabled?() :: boolean()
  def enabled? do
    Config.get().fallback_enabled
  end

  @doc """
  Wraps a crypto operation with fallback support.
  """
  @spec with_fallback((() -> {:ok, term()} | {:error, term()}), (() -> {:ok, term()} | {:error, term()})) :: {:ok, term()} | {:error, term()}
  def with_fallback(primary_fn, fallback_fn) do
    if enabled?() do
      case primary_fn.() do
        {:ok, result} -> {:ok, result}
        {:error, %{error_code: :crypto_service_unavailable}} -> fallback_fn.()
        {:error, %{error_code: :crypto_operation_timeout}} -> fallback_fn.()
        error -> error
      end
    else
      primary_fn.()
    end
  end

  # Private Functions

  defp get_local_signer do
    # Get local signing key from config or generate ephemeral
    case Application.get_env(:session_identity_core, :local_jwt_secret) do
      nil ->
        Logger.warning("No local JWT secret configured, using ephemeral key")
        Joken.Signer.create("HS256", :crypto.strong_rand_bytes(32))
      
      secret ->
        Joken.Signer.create("HS256", secret)
    end
  end

  defp emit_fallback_metric(operation) do
    :telemetry.execute(
      [:session_identity, :crypto, :fallback],
      %{count: 1},
      %{operation: operation}
    )
  end
end
