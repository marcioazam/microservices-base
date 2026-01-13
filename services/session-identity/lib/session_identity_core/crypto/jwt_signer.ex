defmodule SessionIdentityCore.Crypto.JWTSigner do
  @moduledoc """
  JWT signing and verification using crypto-service.
  
  Provides centralized JWT operations with:
  - Signing via crypto-service (ECDSA P-256 or RSA-2048)
  - Verification via crypto-service
  - Fallback to local Joken when crypto-service unavailable
  - Key metadata caching
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

  @type claims :: map()
  @type jwt :: String.t()

  @doc """
  Signs claims and returns a JWT.
  
  Uses crypto-service for signing with centrally managed keys.
  Falls back to local Joken if crypto-service unavailable and fallback enabled.
  """
  @spec sign_jwt(claims(), keyword()) :: {:ok, jwt()} | {:error, term()}
  def sign_jwt(claims, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    opts = Keyword.merge(TraceContext.extract_current(), opts)
    
    config = Config.get()
    
    if config.enabled do
      CircuitBreaker.call_with_fallback(
        fn -> sign_via_crypto_service(claims, opts) end,
        fn -> Fallback.sign_jwt_local(claims, opts) end
      )
    else
      Fallback.sign_jwt_local(claims, opts)
    end
  end

  @doc """
  Verifies a JWT and returns the claims.
  
  Uses crypto-service for verification.
  Falls back to local Joken if crypto-service unavailable and fallback enabled.
  """
  @spec verify_jwt(jwt(), keyword()) :: {:ok, claims()} | {:error, term()}
  def verify_jwt(token, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    opts = Keyword.merge(TraceContext.extract_current(), opts)
    
    config = Config.get()
    
    if config.enabled do
      CircuitBreaker.call_with_fallback(
        fn -> verify_via_crypto_service(token, opts) end,
        fn -> Fallback.verify_jwt_local(token, opts) end
      )
    else
      Fallback.verify_jwt_local(token, opts)
    end
  end

  # Private Functions

  defp sign_via_crypto_service(claims, opts) do
    config = Config.get()
    
    with {:ok, key_id} <- KeyManager.get_active_key(config.jwt_key_namespace),
         {:ok, header} <- build_jwt_header(config.jwt_algorithm),
         {:ok, payload} <- encode_claims(claims),
         signing_input <- "#{header}.#{payload}",
         {:ok, %{signature: signature}} <- Client.sign(
           signing_input,
           key_id,
           :sha256,
           opts
         ) do
      encoded_signature = Base.url_encode64(signature, padding: false)
      jwt = "#{signing_input}.#{encoded_signature}"
      
      Logger.debug("JWT signed via crypto-service",
        Correlation.extract_for_logging(opts) ++ [key_version: key_id.version]
      )
      
      {:ok, jwt}
    else
      {:error, reason} ->
        Logger.error("JWT signing failed: #{inspect(reason)}",
          Correlation.extract_for_logging(opts)
        )
        {:error, reason}
    end
  end

  defp verify_via_crypto_service(token, opts) do
    config = Config.get()
    
    with {:ok, {header, payload, signature}} <- parse_jwt(token),
         {:ok, claims} <- decode_claims(payload),
         {:ok, key_id} <- get_verification_key(claims, config),
         signing_input <- "#{header}.#{payload}",
         {:ok, valid} <- Client.verify(
           signing_input,
           Base.url_decode64!(signature, padding: false),
           key_id,
           :sha256,
           opts
         ) do
      if valid do
        Logger.debug("JWT verified via crypto-service",
          Correlation.extract_for_logging(opts) ++ [key_version: key_id.version]
        )
        {:ok, claims}
      else
        {:error, Errors.signature_invalid()}
      end
    else
      {:error, reason} ->
        Logger.error("JWT verification failed: #{inspect(reason)}",
          Correlation.extract_for_logging(opts)
        )
        {:error, reason}
    end
  end

  defp build_jwt_header(algorithm) do
    alg = case algorithm do
      :ecdsa_p256 -> "ES256"
      :rsa_2048 -> "RS256"
      _ -> "ES256"
    end
    
    header = %{"alg" => alg, "typ" => "JWT"}
    {:ok, Base.url_encode64(Jason.encode!(header), padding: false)}
  end

  defp encode_claims(claims) do
    # Ensure claims have required JWT fields
    claims = claims
    |> Map.put_new(:iat, DateTime.to_unix(DateTime.utc_now()))
    |> Map.put_new(:exp, DateTime.to_unix(DateTime.utc_now()) + 3600)
    
    {:ok, Base.url_encode64(Jason.encode!(claims), padding: false)}
  end

  defp decode_claims(payload) do
    case Base.url_decode64(payload, padding: false) do
      {:ok, json} ->
        case Jason.decode(json) do
          {:ok, claims} -> {:ok, atomize_keys(claims)}
          error -> error
        end
      :error ->
        {:error, Errors.invalid_argument("Invalid base64 payload")}
    end
  end

  defp parse_jwt(token) when is_binary(token) do
    case String.split(token, ".") do
      [header, payload, signature] ->
        {:ok, {header, payload, signature}}
      _ ->
        {:error, Errors.invalid_argument("Invalid JWT format")}
    end
  end

  defp get_verification_key(_claims, config) do
    # For now, use the active key for the JWT namespace
    # In production, might extract key_id from JWT header
    KeyManager.get_active_key(config.jwt_key_namespace)
  end

  # SECURITY FIX: Do NOT atomize JWT claim keys to prevent atom exhaustion
  # JWT claims can come from external/untrusted sources and atomizing all keys
  # would allow an attacker to exhaust the atom table by sending JWTs with many unique keys.
  #
  # Instead, keep string keys for JWT claims. If specific known keys need to be atoms,
  # use a whitelist approach with String.to_existing_atom/1.
  #
  # This function is now a no-op for security reasons - returns data as-is with string keys.
  defp atomize_keys(data) when is_map(data) or is_list(data) or is_binary(data) do
    # Return as-is - do not convert string keys to atoms
    # Callers should access claims using string keys, e.g.: claims["sub"] instead of claims.sub
    data
  end

  defp atomize_keys(data) do
    data
  end
end
