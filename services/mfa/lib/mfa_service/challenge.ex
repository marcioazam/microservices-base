defmodule MfaService.Challenge do
  @moduledoc """
  Centralized challenge storage using Cache_Service.
  Replaces all direct Redis/ETS challenge storage across MFA modules.

  ## Features
  - Cryptographically secure challenge generation (32 bytes)
  - Cache_Service integration with TTL support
  - Constant-time comparison for verification
  - One-time use with retrieve_and_delete
  """

  alias AuthPlatform.Clients.Cache
  alias AuthPlatform.Security

  @challenge_length 32
  @default_ttl 300

  @type challenge :: binary()
  @type key :: String.t()

  @doc """
  Generates a cryptographically secure random challenge.
  Returns 32 bytes of random data.
  """
  @spec generate() :: challenge()
  def generate do
    :crypto.strong_rand_bytes(@challenge_length)
  end

  @doc """
  Stores a challenge in Cache_Service with TTL.

  ## Options
    * `:ttl` - Time to live in seconds (default: 300)
  """
  @spec store(key(), challenge(), keyword()) :: :ok | {:error, term()}
  def store(key, challenge, opts \\ []) do
    ttl = Keyword.get(opts, :ttl, @default_ttl)
    cache_key = build_key(key)
    encoded = Base.url_encode64(challenge, padding: false)

    case Cache.set(cache_key, encoded, ttl: ttl) do
      :ok -> :ok
      {:error, _} = error -> error
    end
  end

  @doc """
  Retrieves a challenge from Cache_Service.
  """
  @spec retrieve(key()) :: {:ok, challenge()} | {:error, :not_found | term()}
  def retrieve(key) do
    cache_key = build_key(key)

    case Cache.get(cache_key) do
      {:ok, nil} -> {:error, :not_found}
      {:ok, encoded} -> decode_challenge(encoded)
      {:error, _} = error -> error
    end
  end

  @doc """
  Retrieves and deletes a challenge (one-time use).
  """
  @spec retrieve_and_delete(key()) :: {:ok, challenge()} | {:error, :not_found | term()}
  def retrieve_and_delete(key) do
    cache_key = build_key(key)

    case Cache.get(cache_key) do
      {:ok, nil} ->
        {:error, :not_found}

      {:ok, encoded} ->
        Cache.delete(cache_key)
        decode_challenge(encoded)

      {:error, _} = error ->
        error
    end
  end

  @doc """
  Verifies a challenge using constant-time comparison.
  """
  @spec verify(challenge(), challenge()) :: :ok | {:error, :challenge_mismatch}
  def verify(received, expected) when is_binary(received) and is_binary(expected) do
    if Security.constant_time_compare(received, expected) do
      :ok
    else
      {:error, :challenge_mismatch}
    end
  end

  def verify(_, _), do: {:error, :challenge_mismatch}

  @doc """
  Encodes a challenge for transmission to client.
  """
  @spec encode(challenge()) :: String.t()
  def encode(challenge) do
    Base.url_encode64(challenge, padding: false)
  end

  @doc """
  Decodes a challenge received from client.
  """
  @spec decode(String.t()) :: {:ok, challenge()} | {:error, :invalid_encoding}
  def decode(encoded) do
    case Base.url_decode64(encoded, padding: false) do
      {:ok, challenge} -> {:ok, challenge}
      :error -> {:error, :invalid_encoding}
    end
  end

  # Private

  defp build_key(key), do: "mfa:challenge:#{key}"

  defp decode_challenge(encoded) do
    case Base.url_decode64(encoded, padding: false) do
      {:ok, challenge} -> {:ok, challenge}
      :error -> {:error, :invalid_encoding}
    end
  end
end
