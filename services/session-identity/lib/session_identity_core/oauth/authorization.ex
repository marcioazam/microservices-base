defmodule SessionIdentityCore.OAuth.Authorization do
  @moduledoc """
  OAuth 2.0 authorization code generation and management.
  """

  alias SessionIdentityCore.OAuth.PKCE

  @code_length 32
  @code_ttl 600  # 10 minutes

  defstruct [
    :code,
    :client_id,
    :redirect_uri,
    :user_id,
    :scopes,
    :code_challenge,
    :code_challenge_method,
    :nonce,
    :created_at,
    :expires_at
  ]

  @doc """
  Validates an OAuth authorization request.
  """
  def validate_request(params, client) do
    with :ok <- validate_client_id(params.client_id, client),
         :ok <- validate_redirect_uri(params.redirect_uri, client),
         :ok <- validate_pkce_for_public_client(params, client) do
      {:ok, params}
    end
  end

  @doc """
  Generates an authorization code bound to the PKCE challenge.
  """
  def generate_code(user_id, params) do
    code = :crypto.strong_rand_bytes(@code_length) |> Base.url_encode64(padding: false)
    now = DateTime.utc_now()

    auth = %__MODULE__{
      code: code,
      client_id: params.client_id,
      redirect_uri: params.redirect_uri,
      user_id: user_id,
      scopes: params.scopes || [],
      code_challenge: params.code_challenge,
      code_challenge_method: params.code_challenge_method || "S256",
      nonce: params.nonce,
      created_at: now,
      expires_at: DateTime.add(now, @code_ttl, :second)
    }

    {:ok, auth}
  end

  @doc """
  Exchanges an authorization code for tokens.
  Verifies the PKCE code_verifier.
  """
  def exchange_code(stored_auth, code_verifier) do
    with :ok <- verify_not_expired(stored_auth),
         :ok <- PKCE.verify(code_verifier, stored_auth.code_challenge, stored_auth.code_challenge_method) do
      {:ok, stored_auth}
    else
      {:error, :invalid_code_verifier} ->
        # Log potential attack attempt
        log_attack_attempt(stored_auth, "PKCE verification failed")
        {:error, :invalid_code_verifier}

      error ->
        error
    end
  end

  @doc """
  Stores an authorization code in Redis.
  """
  def store_code(auth) do
    key = code_key(auth.code)
    value = Jason.encode!(auth_to_map(auth))

    case Redix.command(:redix, ["SETEX", key, @code_ttl, value]) do
      {:ok, _} -> {:ok, auth}
      error -> error
    end
  end

  @doc """
  Retrieves and deletes an authorization code (one-time use).
  """
  def retrieve_and_delete_code(code) do
    key = code_key(code)

    case Redix.command(:redix, ["GETDEL", key]) do
      {:ok, nil} -> {:error, :code_not_found}
      {:ok, value} -> {:ok, map_to_auth(Jason.decode!(value))}
      error -> error
    end
  end

  # Private functions

  defp validate_client_id(client_id, client) do
    if client_id == client.client_id do
      :ok
    else
      {:error, :invalid_client}
    end
  end

  defp validate_redirect_uri(redirect_uri, client) do
    if redirect_uri in client.redirect_uris do
      :ok
    else
      {:error, :invalid_redirect_uri}
    end
  end

  defp validate_pkce_for_public_client(params, client) do
    if client.client_type == "public" do
      if params.code_challenge && params.code_challenge != "" do
        PKCE.validate_code_challenge(params.code_challenge)
      else
        {:error, :pkce_required}
      end
    else
      :ok
    end
  end

  defp verify_not_expired(auth) do
    if DateTime.compare(DateTime.utc_now(), auth.expires_at) == :lt do
      :ok
    else
      {:error, :code_expired}
    end
  end

  defp log_attack_attempt(auth, reason) do
    # In production, this would emit a security event
    require Logger
    Logger.warning("Potential attack: #{reason}, client_id: #{auth.client_id}, user_id: #{auth.user_id}")
  end

  defp code_key(code), do: "oauth_code:#{code}"

  defp auth_to_map(auth) do
    %{
      code: auth.code,
      client_id: auth.client_id,
      redirect_uri: auth.redirect_uri,
      user_id: auth.user_id,
      scopes: auth.scopes,
      code_challenge: auth.code_challenge,
      code_challenge_method: auth.code_challenge_method,
      nonce: auth.nonce,
      created_at: DateTime.to_iso8601(auth.created_at),
      expires_at: DateTime.to_iso8601(auth.expires_at)
    }
  end

  defp map_to_auth(map) do
    %__MODULE__{
      code: map["code"],
      client_id: map["client_id"],
      redirect_uri: map["redirect_uri"],
      user_id: map["user_id"],
      scopes: map["scopes"],
      code_challenge: map["code_challenge"],
      code_challenge_method: map["code_challenge_method"],
      nonce: map["nonce"],
      created_at: DateTime.from_iso8601!(map["created_at"]),
      expires_at: DateTime.from_iso8601!(map["expires_at"])
    }
  end
end
