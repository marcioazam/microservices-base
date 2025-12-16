defmodule SessionIdentityCore.OAuth.IdToken do
  @moduledoc """
  OIDC ID Token generation with standard claims.
  """

  @required_claims [:sub, :iss, :aud, :exp, :iat]

  defstruct [
    :sub,      # Subject identifier
    :iss,      # Issuer
    :aud,      # Audience
    :exp,      # Expiration time
    :iat,      # Issued at
    :auth_time, # Time of authentication
    :nonce,    # Nonce from authorization request
    :acr,      # Authentication context class reference
    :amr,      # Authentication methods references
    :azp       # Authorized party
  ]

  @doc """
  Builds an ID token with standard OIDC claims.
  """
  def build(user_id, opts \\ []) do
    now = System.system_time(:second)
    ttl = Keyword.get(opts, :ttl, 3600)

    token = %__MODULE__{
      sub: user_id,
      iss: Keyword.get(opts, :issuer, "auth-platform"),
      aud: Keyword.get(opts, :audience, []),
      exp: now + ttl,
      iat: now,
      auth_time: Keyword.get(opts, :auth_time, now),
      nonce: Keyword.get(opts, :nonce),
      acr: Keyword.get(opts, :acr),
      amr: Keyword.get(opts, :amr),
      azp: Keyword.get(opts, :azp)
    }

    {:ok, token}
  end

  @doc """
  Converts ID token to a map for JWT encoding.
  """
  def to_claims(%__MODULE__{} = token) do
    %{
      sub: token.sub,
      iss: token.iss,
      aud: token.aud,
      exp: token.exp,
      iat: token.iat
    }
    |> maybe_add(:auth_time, token.auth_time)
    |> maybe_add(:nonce, token.nonce)
    |> maybe_add(:acr, token.acr)
    |> maybe_add(:amr, token.amr)
    |> maybe_add(:azp, token.azp)
  end

  @doc """
  Validates that an ID token has all required claims.
  """
  def validate(%__MODULE__{} = token) do
    missing_claims = Enum.filter(@required_claims, fn claim ->
      value = Map.get(token, claim)
      is_nil(value) or value == "" or value == []
    end)

    if Enum.empty?(missing_claims) do
      :ok
    else
      {:error, {:missing_claims, missing_claims}}
    end
  end

  @doc """
  Checks if an ID token contains all standard claims.
  """
  def has_standard_claims?(%__MODULE__{} = token) do
    validate(token) == :ok
  end

  @doc """
  Checks if nonce is present when it was provided in the request.
  """
  def has_nonce_if_provided?(token, request_nonce) do
    case request_nonce do
      nil -> true
      "" -> true
      nonce -> token.nonce == nonce
    end
  end

  defp maybe_add(map, _key, nil), do: map
  defp maybe_add(map, key, value), do: Map.put(map, key, value)
end
