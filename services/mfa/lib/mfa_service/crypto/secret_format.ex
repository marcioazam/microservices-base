defmodule MfaService.Crypto.SecretFormat do
  @moduledoc """
  Defines the encrypted secret format with version byte.
  Supports both legacy local encryption (v1) and crypto-service encryption (v2).
  
  ## Format
  
  ### Version 1 (Local Encryption)
  ```
  ┌─────────┬──────────────────────────────────────────────────────────────┐
  │ Version │ IV (12 bytes) + Tag (16 bytes) + Ciphertext (variable)      │
  │ (0x01)  │                                                              │
  └─────────┴──────────────────────────────────────────────────────────────┘
  ```
  
  ### Version 2 (Crypto-Service Encryption)
  ```
  ┌─────────┬──────────────────────────────────────────────────────────────┐
  │ Version │ KeyID Length (2 bytes) + KeyID (JSON) + IV (12) + Tag (16)  │
  │ (0x02)  │ + Ciphertext (variable)                                      │
  └─────────┴──────────────────────────────────────────────────────────────┘
  ```
  """

  @version_local 0x01
  @version_crypto_service 0x02

  @type version :: :local | :crypto_service | :unknown
  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}

  @type v2_payload :: %{
    key_id: key_id(),
    iv: binary(),
    tag: binary(),
    ciphertext: binary()
  }

  @doc """
  Detects the encryption version from the first byte.
  """
  @spec detect_version(binary()) :: version()
  def detect_version(<<@version_local, _rest::binary>>), do: :local
  def detect_version(<<@version_crypto_service, _rest::binary>>), do: :crypto_service
  def detect_version(_), do: :unknown

  @doc """
  Returns the version byte for local encryption.
  """
  @spec version_local() :: byte()
  def version_local, do: @version_local

  @doc """
  Returns the version byte for crypto-service encryption.
  """
  @spec version_crypto_service() :: byte()
  def version_crypto_service, do: @version_crypto_service

  @doc """
  Encodes a v1 (local) encrypted payload.
  """
  @spec encode_v1(binary(), binary(), binary()) :: binary()
  def encode_v1(iv, tag, ciphertext) when byte_size(iv) == 12 and byte_size(tag) == 16 do
    <<@version_local, iv::binary-size(12), tag::binary-size(16), ciphertext::binary>>
  end

  @doc """
  Decodes a v1 (local) encrypted payload.
  """
  @spec decode_v1(binary()) :: {:ok, {binary(), binary(), binary()}} | {:error, :invalid_format}
  def decode_v1(<<@version_local, iv::binary-size(12), tag::binary-size(16), ciphertext::binary>>) do
    {:ok, {iv, tag, ciphertext}}
  end
  def decode_v1(_), do: {:error, :invalid_format}

  @doc """
  Encodes a v2 (crypto-service) encrypted payload.
  """
  @spec encode_v2(key_id(), binary(), binary(), binary()) :: binary()
  def encode_v2(key_id, iv, tag, ciphertext) when byte_size(iv) == 12 and byte_size(tag) == 16 do
    key_id_json = Jason.encode!(key_id)
    key_id_len = byte_size(key_id_json)
    
    <<@version_crypto_service, 
      key_id_len::unsigned-big-integer-size(16), 
      key_id_json::binary,
      iv::binary-size(12), 
      tag::binary-size(16), 
      ciphertext::binary>>
  end

  @doc """
  Decodes a v2 (crypto-service) encrypted payload.
  """
  @spec decode_v2(binary()) :: {:ok, v2_payload()} | {:error, :invalid_format}
  def decode_v2(<<@version_crypto_service, 
                  key_id_len::unsigned-big-integer-size(16), 
                  key_id_json::binary-size(key_id_len),
                  iv::binary-size(12), 
                  tag::binary-size(16), 
                  ciphertext::binary>>) do
    case Jason.decode(key_id_json) do
      {:ok, key_id_map} ->
        key_id = %{
          namespace: key_id_map["namespace"],
          id: key_id_map["id"],
          version: key_id_map["version"]
        }
        {:ok, %{key_id: key_id, iv: iv, tag: tag, ciphertext: ciphertext}}

      {:error, _} ->
        {:error, :invalid_format}
    end
  end
  def decode_v2(_), do: {:error, :invalid_format}

  @doc """
  Decodes any version of encrypted payload.
  Returns the version and decoded data.
  """
  @spec decode(binary()) :: {:ok, version(), term()} | {:error, :invalid_format | :unknown_version}
  def decode(data) do
    case detect_version(data) do
      :local ->
        case decode_v1(data) do
          {:ok, {iv, tag, ciphertext}} ->
            {:ok, :local, %{iv: iv, tag: tag, ciphertext: ciphertext}}
          error -> error
        end

      :crypto_service ->
        case decode_v2(data) do
          {:ok, payload} ->
            {:ok, :crypto_service, payload}
          error -> error
        end

      :unknown ->
        {:error, :unknown_version}
    end
  end

  @doc """
  Validates that a payload has a valid version byte.
  """
  @spec valid_version?(binary()) :: boolean()
  def valid_version?(<<version, _rest::binary>>) when version in [@version_local, @version_crypto_service] do
    true
  end
  def valid_version?(_), do: false

  @doc """
  Returns the minimum valid payload size for each version.
  """
  @spec min_payload_size(version()) :: non_neg_integer()
  def min_payload_size(:local), do: 1 + 12 + 16 + 1  # version + iv + tag + min ciphertext
  def min_payload_size(:crypto_service), do: 1 + 2 + 10 + 12 + 16 + 1  # version + len + min json + iv + tag + min ciphertext
  def min_payload_size(:unknown), do: 0
end
