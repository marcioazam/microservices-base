defmodule SessionIdentityCore.OAuth.PKCE do
  @moduledoc """
  PKCE (Proof Key for Code Exchange) implementation per OAuth 2.1.
  
  Supports S256 method ONLY as required by OAuth 2.1/RFC 9700.
  Uses constant-time comparison from AuthPlatform.Security.
  
  ## Security
  
  - Plain method is rejected (OAuth 2.1 requirement)
  - Code verifier must be 43-128 characters
  - Valid charset: [A-Za-z0-9\\-._~]
  - Uses constant-time comparison to prevent timing attacks
  """

  alias AuthPlatform.Security
  alias SessionIdentityCore.Shared.Errors

  @verifier_min_length 43
  @verifier_max_length 128
  @challenge_length 43
  @valid_chars ~r/^[A-Za-z0-9\-._~]+$/
  @valid_base64url ~r/^[A-Za-z0-9_-]+$/

  @doc """
  Verifies that the code_verifier matches the code_challenge using S256 method.
  
  S256: BASE64URL(SHA256(code_verifier)) == code_challenge
  
  Uses constant-time comparison to prevent timing attacks.
  """
  @spec verify(String.t(), String.t(), String.t()) :: :ok | {:error, atom()}
  def verify(code_verifier, code_challenge, "S256") do
    computed = compute_s256_challenge(code_verifier)

    if Security.constant_time_compare(computed, code_challenge) do
      :ok
    else
      Errors.invalid_code_verifier()
    end
  end

  def verify(_code_verifier, _code_challenge, "plain") do
    # Plain method is NOT allowed per OAuth 2.1
    {:error, :plain_method_not_allowed}
  end

  def verify(_code_verifier, _code_challenge, _method) do
    Errors.unsupported_pkce_method()
  end

  @doc """
  Computes the S256 challenge from a code_verifier.
  
  S256 = BASE64URL(SHA256(code_verifier))
  """
  @spec compute_s256_challenge(String.t()) :: String.t()
  def compute_s256_challenge(code_verifier) do
    :crypto.hash(:sha256, code_verifier)
    |> Base.url_encode64(padding: false)
  end

  @doc """
  Validates code_verifier format per RFC 7636.
  
  Must be 43-128 characters, using only [A-Z], [a-z], [0-9], "-", ".", "_", "~"
  """
  @spec validate_code_verifier(String.t()) :: :ok | {:error, atom()}
  def validate_code_verifier(verifier) when is_binary(verifier) do
    len = String.length(verifier)

    cond do
      len < @verifier_min_length -> Errors.code_verifier_too_short()
      len > @verifier_max_length -> Errors.code_verifier_too_long()
      not Regex.match?(@valid_chars, verifier) -> {:error, :invalid_characters}
      true -> :ok
    end
  end

  def validate_code_verifier(_), do: Errors.invalid_code_verifier()

  @doc """
  Validates code_challenge format.
  
  Must be 43 characters (BASE64URL encoded SHA256 without padding)
  """
  @spec validate_code_challenge(String.t()) :: :ok | {:error, atom()}
  def validate_code_challenge(challenge) when is_binary(challenge) do
    if String.length(challenge) == @challenge_length and Regex.match?(@valid_base64url, challenge) do
      :ok
    else
      Errors.invalid_code_challenge()
    end
  end

  def validate_code_challenge(_), do: Errors.invalid_code_challenge()

  @doc """
  Generates a cryptographically secure code_verifier.
  
  Returns a 43-character URL-safe base64 encoded string.
  """
  @spec generate_code_verifier() :: String.t()
  def generate_code_verifier do
    Security.generate_token(32, encoding: :url_safe_base64)
  end

  @doc """
  Generates a code_challenge from a code_verifier using S256.
  """
  @spec generate_code_challenge(String.t()) :: String.t()
  def generate_code_challenge(code_verifier) do
    compute_s256_challenge(code_verifier)
  end
end
