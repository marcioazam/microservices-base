defmodule SessionIdentityCore.Crypto.IdTokenSigner do
  @moduledoc """
  ID Token signing integration with crypto-service.
  
  Replaces direct Joken usage with JWTSigner for centralized
  key management and signing via crypto-service.
  """

  require Logger

  alias SessionIdentityCore.Crypto.{JWTSigner, Correlation}
  alias SessionIdentityCore.OAuth.IdToken

  @doc """
  Signs ID token claims using crypto-service.
  
  Builds claims from params and signs via JWTSigner.
  """
  @spec sign(map(), keyword()) :: {:ok, String.t()} | {:error, term()}
  def sign(params, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    
    with {:ok, claims} <- IdToken.build_claims(params),
         {:ok, token} <- JWTSigner.sign(claims, opts) do
      Logger.debug("ID token signed",
        Correlation.extract_for_logging(opts) ++ [
          sub: params[:sub],
          aud: params[:aud]
        ]
      )
      {:ok, token}
    end
  end

  @doc """
  Verifies an ID token and returns claims.
  """
  @spec verify(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def verify(token, opts \\ []) do
    opts = Correlation.ensure_correlation_id(opts)
    
    with {:ok, claims} <- JWTSigner.verify(token, opts),
         :ok <- validate_token_claims(claims) do
      {:ok, claims}
    end
  end

  @doc """
  Verifies token and validates nonce if provided.
  """
  @spec verify_with_nonce(String.t(), String.t() | nil, keyword()) :: 
    {:ok, map()} | {:error, term()}
  def verify_with_nonce(token, expected_nonce, opts \\ []) do
    with {:ok, claims} <- verify(token, opts),
         :ok <- IdToken.validate_nonce(claims, expected_nonce) do
      {:ok, claims}
    end
  end

  defp validate_token_claims(claims) do
    cond do
      IdToken.expired?(claims) -> {:error, :token_expired}
      true -> :ok
    end
  end
end
