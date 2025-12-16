defmodule MfaService.WebAuthn.Challenge do
  @moduledoc """
  WebAuthn challenge generation and management.
  """

  @challenge_length 32  # 256 bits

  @doc """
  Generates a cryptographically secure random challenge.
  """
  def generate(length \\ @challenge_length) do
    :crypto.strong_rand_bytes(length)
  end

  @doc """
  Encodes a challenge for transmission to the client.
  """
  def encode(challenge) do
    Base.url_encode64(challenge, padding: false)
  end

  @doc """
  Decodes a challenge received from the client.
  """
  def decode(encoded_challenge) do
    Base.url_decode64!(encoded_challenge, padding: false)
  end

  @doc """
  Stores a challenge with expiration for later verification.
  """
  def store(challenge, user_id, ttl_seconds \\ 300) do
    key = challenge_key(user_id)
    encoded = encode(challenge)
    
    Redix.command(:redix, ["SETEX", key, ttl_seconds, encoded])
  end

  @doc """
  Retrieves and deletes a stored challenge (one-time use).
  """
  def retrieve_and_delete(user_id) do
    key = challenge_key(user_id)
    
    case Redix.command(:redix, ["GETDEL", key]) do
      {:ok, nil} -> {:error, :challenge_not_found}
      {:ok, encoded} -> {:ok, decode(encoded)}
      error -> error
    end
  end

  @doc """
  Verifies that a challenge matches the expected value.
  Uses constant-time comparison to prevent timing attacks.
  """
  def verify(received, expected) do
    if :crypto.hash_equals(received, expected) do
      :ok
    else
      {:error, :challenge_mismatch}
    end
  end

  defp challenge_key(user_id), do: "webauthn_challenge:#{user_id}"
end
