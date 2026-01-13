defmodule SessionIdentityCore.OAuth.IdToken do
  @moduledoc """
  OIDC ID Token generation and validation.
  
  Implements:
  - Required claims: sub, iss, aud, exp, iat
  - Optional claims: nonce, auth_time, acr, amr, azp
  - Configurable TTL (default 1 hour)
  - Claims validation before signing
  """

  alias SessionIdentityCore.Shared.{TTL, Errors}
  alias SessionIdentityCore.Shared.DateTime, as: DT

  @required_claims [:sub, :iss, :aud, :exp, :iat]
  @optional_claims [:nonce, :auth_time, :acr, :amr, :azp]

  @doc """
  Builds ID token claims from session and request data.
  
  ## Required Parameters
  
  - sub: Subject identifier (user_id)
  - iss: Issuer identifier
  - aud: Audience (client_id)
  
  ## Optional Parameters
  
  - nonce: Request nonce (MUST be included if provided in request)
  - auth_time: Time of authentication
  - acr: Authentication Context Class Reference
  - amr: Authentication Methods References
  - azp: Authorized party
  - ttl: Token TTL in seconds (default: 3600)
  """
  @spec build_claims(map()) :: {:ok, map()} | {:error, term()}
  def build_claims(params) do
    now = DateTime.utc_now()
    ttl = Map.get(params, :ttl, TTL.default_id_token_ttl())
    iat = DateTime.to_unix(now)
    exp = iat + ttl

    claims = %{
      sub: params[:sub],
      iss: params[:iss],
      aud: params[:aud],
      iat: iat,
      exp: exp
    }

    claims = add_optional_claims(claims, params)

    case validate_claims(claims) do
      :ok -> {:ok, claims}
      error -> error
    end
  end

  @doc """
  Validates that all required claims are present.
  """
  @spec validate_claims(map()) :: :ok | {:error, term()}
  def validate_claims(claims) do
    missing =
      @required_claims
      |> Enum.filter(fn claim -> is_nil(Map.get(claims, claim)) end)

    if Enum.empty?(missing) do
      :ok
    else
      Errors.missing_required_claims(missing)
    end
  end

  @doc """
  Checks if a claim is required.
  """
  @spec required_claim?(atom()) :: boolean()
  def required_claim?(claim), do: claim in @required_claims

  @doc """
  Checks if a claim is optional.
  """
  @spec optional_claim?(atom()) :: boolean()
  def optional_claim?(claim), do: claim in @optional_claims

  @doc """
  Returns the list of required claims.
  """
  @spec required_claims() :: [atom()]
  def required_claims, do: @required_claims

  @doc """
  Returns the list of optional claims.
  """
  @spec optional_claims() :: [atom()]
  def optional_claims, do: @optional_claims

  @doc """
  Checks if the token is expired.
  """
  @spec expired?(map()) :: boolean()
  def expired?(%{exp: exp}) do
    DateTime.to_unix(DateTime.utc_now()) > exp
  end

  def expired?(_), do: true

  @doc """
  Validates the nonce claim matches the expected value.
  """
  @spec validate_nonce(map(), String.t() | nil) :: :ok | {:error, :nonce_mismatch}
  def validate_nonce(_claims, nil), do: :ok

  def validate_nonce(%{nonce: nonce}, expected) when nonce == expected, do: :ok
  def validate_nonce(_, _), do: {:error, :nonce_mismatch}

  # Private functions

  defp add_optional_claims(claims, params) do
    claims
    |> maybe_add_nonce(params)
    |> maybe_add_auth_time(params)
    |> maybe_add_acr(params)
    |> maybe_add_amr(params)
    |> maybe_add_azp(params)
  end

  defp maybe_add_nonce(claims, %{nonce: nonce}) when not is_nil(nonce) do
    Map.put(claims, :nonce, nonce)
  end

  defp maybe_add_nonce(claims, _), do: claims

  defp maybe_add_auth_time(claims, %{auth_time: auth_time}) when not is_nil(auth_time) do
    Map.put(claims, :auth_time, auth_time)
  end

  defp maybe_add_auth_time(claims, _), do: claims

  defp maybe_add_acr(claims, %{acr: acr}) when not is_nil(acr) do
    Map.put(claims, :acr, acr)
  end

  defp maybe_add_acr(claims, _), do: claims

  defp maybe_add_amr(claims, %{amr: amr}) when is_list(amr) and length(amr) > 0 do
    Map.put(claims, :amr, amr)
  end

  defp maybe_add_amr(claims, _), do: claims

  defp maybe_add_azp(claims, %{azp: azp}) when not is_nil(azp) do
    Map.put(claims, :azp, azp)
  end

  defp maybe_add_azp(claims, _), do: claims
end
