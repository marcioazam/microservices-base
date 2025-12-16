defmodule SessionIdentityCore.OAuth.PKCE do
  @moduledoc """
  PKCE (Proof Key for Code Exchange) implementation for OAuth 2.0.
  Supports S256 method as required by the spec.
  """

  @doc """
  Validates that the code_verifier matches the code_challenge using S256 method.
  
  S256: BASE64URL(SHA256(code_verifier)) == code_challenge
  """
  def verify(code_verifier, code_challenge, "S256") do
    computed_challenge = compute_s256_challenge(code_verifier)
    
    if secure_compare(computed_challenge, code_challenge) do
      :ok
    else
      {:error, :invalid_code_verifier}
    end
  end

  def verify(_code_verifier, _code_challenge, "plain") do
    # Plain method is not recommended and disabled by default
    {:error, :plain_method_not_allowed}
  end

  def verify(_code_verifier, _code_challenge, _method) do
    {:error, :unsupported_method}
  end

  @doc """
  Computes the S256 challenge from a code_verifier.
  """
  def compute_s256_challenge(code_verifier) do
    :crypto.hash(:sha256, code_verifier)
    |> Base.url_encode64(padding: false)
  end

  @doc """
  Validates code_verifier format.
  Must be 43-128 characters, using only [A-Z], [a-z], [0-9], "-", ".", "_", "~"
  """
  def validate_code_verifier(code_verifier) when is_binary(code_verifier) do
    len = String.length(code_verifier)
    
    cond do
      len < 43 -> {:error, :code_verifier_too_short}
      len > 128 -> {:error, :code_verifier_too_long}
      not valid_characters?(code_verifier) -> {:error, :invalid_characters}
      true -> :ok
    end
  end

  def validate_code_verifier(_), do: {:error, :invalid_code_verifier}

  @doc """
  Validates code_challenge format.
  Must be 43 characters (BASE64URL encoded SHA256 without padding)
  """
  def validate_code_challenge(code_challenge) when is_binary(code_challenge) do
    if String.length(code_challenge) == 43 and valid_base64url?(code_challenge) do
      :ok
    else
      {:error, :invalid_code_challenge}
    end
  end

  def validate_code_challenge(_), do: {:error, :invalid_code_challenge}

  # Constant-time comparison to prevent timing attacks
  defp secure_compare(a, b) when byte_size(a) == byte_size(b) do
    :crypto.hash_equals(a, b)
  end

  defp secure_compare(_, _), do: false

  defp valid_characters?(string) do
    Regex.match?(~r/^[A-Za-z0-9\-._~]+$/, string)
  end

  defp valid_base64url?(string) do
    Regex.match?(~r/^[A-Za-z0-9_-]+$/, string)
  end
end
