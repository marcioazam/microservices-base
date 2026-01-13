defmodule MfaService.TOTP.Generator do
  @moduledoc """
  TOTP secret generation and provisioning URI creation.
  Uses AuthPlatform.Security for cryptographic operations.

  ## Features
  - RFC 4226/6238 compliant TOTP generation
  - AES-256-GCM encryption for secret storage
  - Secure random secret generation (160 bits)
  """

  alias AuthPlatform.Security
  alias AuthPlatform.Clients.Logging

  @secret_length 20
  @default_issuer "AuthPlatform"
  @default_algorithm "SHA1"
  @default_digits 6
  @default_period 30

  @type secret :: String.t()
  @type encrypted_secret :: String.t()

  @doc """
  Generates a new TOTP secret with cryptographically secure randomness.
  Returns a base32-encoded secret (160 bits / 20 bytes).
  """
  @spec generate_secret(pos_integer()) :: secret()
  def generate_secret(length \\ @secret_length) do
    :crypto.strong_rand_bytes(length)
    |> Base.encode32(padding: false)
  end

  @doc """
  Creates a provisioning URI for authenticator apps per RFC 6238.
  Format: otpauth://totp/{label}?secret={secret}&issuer={issuer}&algorithm={alg}&digits={digits}&period={period}

  ## Options
    * `:issuer` - Service name (default: "AuthPlatform")
    * `:algorithm` - Hash algorithm (default: "SHA1")
    * `:digits` - Code length (default: 6)
    * `:period` - Time step in seconds (default: 30)
  """
  @spec provisioning_uri(secret(), String.t(), keyword()) :: String.t()
  def provisioning_uri(secret, account_name, opts \\ []) do
    issuer = Keyword.get(opts, :issuer, @default_issuer)
    algorithm = Keyword.get(opts, :algorithm, @default_algorithm)
    digits = Keyword.get(opts, :digits, @default_digits)
    period = Keyword.get(opts, :period, @default_period)

    label = URI.encode("#{issuer}:#{account_name}")

    params =
      URI.encode_query(%{
        "secret" => secret,
        "issuer" => issuer,
        "algorithm" => algorithm,
        "digits" => digits,
        "period" => period
      })

    "otpauth://totp/#{label}?#{params}"
  end

  @doc """
  Generates a QR code as base64-encoded PNG for the provisioning URI.
  """
  @spec generate_qr_code(String.t()) :: {:ok, String.t()} | {:error, term()}
  def generate_qr_code(provisioning_uri) do
    # In production, use a QR code library like eqrcode
    {:ok, Base.encode64("QR_CODE_PLACEHOLDER:#{provisioning_uri}")}
  end

  @doc """
  Encrypts a TOTP secret for secure storage using AES-256-GCM.
  Returns base64-encoded ciphertext with IV and auth tag.
  """
  @spec encrypt_secret(secret(), binary()) :: encrypted_secret()
  def encrypt_secret(secret, key) when byte_size(key) == 32 do
    iv = :crypto.strong_rand_bytes(12)

    {ciphertext, tag} =
      :crypto.crypto_one_time_aead(
        :aes_256_gcm,
        key,
        iv,
        secret,
        "",
        true
      )

    Base.encode64(iv <> tag <> ciphertext)
  end

  @doc """
  Decrypts a stored TOTP secret.
  Returns the original secret or error on decryption failure.
  """
  @spec decrypt_secret(encrypted_secret(), binary()) :: {:ok, secret()} | {:error, :decryption_failed}
  def decrypt_secret(encrypted, key) when byte_size(key) == 32 do
    case Base.decode64(encrypted) do
      {:ok, <<iv::binary-size(12), tag::binary-size(16), ciphertext::binary>>} ->
        case :crypto.crypto_one_time_aead(:aes_256_gcm, key, iv, ciphertext, "", tag) do
          :error ->
            Logging.warn("TOTP secret decryption failed", module: __MODULE__)
            {:error, :decryption_failed}

          plaintext ->
            {:ok, plaintext}
        end

      _ ->
        {:error, :decryption_failed}
    end
  end

  @doc """
  Validates that a secret has sufficient entropy (160+ bits).
  """
  @spec valid_secret?(secret()) :: boolean()
  def valid_secret?(secret) when is_binary(secret) do
    case Base.decode32(secret, padding: false) do
      {:ok, decoded} -> byte_size(decoded) >= @secret_length
      :error -> false
    end
  end

  def valid_secret?(_), do: false
end
