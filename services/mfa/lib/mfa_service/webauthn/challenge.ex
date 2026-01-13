defmodule MfaService.WebAuthn.Challenge do
  @moduledoc """
  WebAuthn challenge generation and management.
  Delegates to centralized MfaService.Challenge module.

  This module is kept for backward compatibility but all operations
  are now handled by the centralized challenge storage.
  """

  alias MfaService.Challenge

  @challenge_length 32

  @doc """
  Generates a cryptographically secure random challenge.
  """
  @spec generate(pos_integer()) :: binary()
  def generate(_length \\ @challenge_length) do
    Challenge.generate()
  end

  @doc """
  Encodes a challenge for transmission to the client.
  """
  @spec encode(binary()) :: String.t()
  def encode(challenge) do
    Challenge.encode(challenge)
  end

  @doc """
  Decodes a challenge received from the client.
  """
  @spec decode(String.t()) :: binary()
  def decode(encoded_challenge) do
    case Challenge.decode(encoded_challenge) do
      {:ok, challenge} -> challenge
      {:error, _} -> raise ArgumentError, "Invalid challenge encoding"
    end
  end

  @doc """
  Stores a challenge with expiration for later verification.
  Uses centralized Cache_Service storage.
  """
  @spec store(binary(), String.t(), pos_integer()) :: :ok | {:error, term()}
  def store(challenge, user_id, ttl_seconds \\ 300) do
    Challenge.store("webauthn:#{user_id}", challenge, ttl: ttl_seconds)
  end

  @doc """
  Retrieves and deletes a stored challenge (one-time use).
  """
  @spec retrieve_and_delete(String.t()) :: {:ok, binary()} | {:error, :challenge_not_found | term()}
  def retrieve_and_delete(user_id) do
    case Challenge.retrieve_and_delete("webauthn:#{user_id}") do
      {:ok, challenge} -> {:ok, challenge}
      {:error, :not_found} -> {:error, :challenge_not_found}
      error -> error
    end
  end

  @doc """
  Verifies that a challenge matches the expected value.
  Uses constant-time comparison to prevent timing attacks.
  """
  @spec verify(binary(), binary()) :: :ok | {:error, :challenge_mismatch}
  def verify(received, expected) do
    Challenge.verify(received, expected)
  end
end
