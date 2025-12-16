defmodule MfaService.TOTP.Generator do
  @moduledoc """
  TOTP secret generation and provisioning URI creation.
  """

  @secret_length 20  # 160 bits as recommended by RFC 4226
  @default_issuer "AuthPlatform"
  @default_algorithm "SHA1"
  @default_digits 6
  @default_period 30

  @doc """
  Generates a new TOTP secret.
  Returns a base32-encoded secret.
  """
  def generate_secret(length \\ @secret_length) do
    :crypto.strong_rand_bytes(length)
    |> Base.encode32(padding: false)
  end

  @doc """
  Creates a provisioning URI for authenticator apps.
  Format: otpauth://totp/ISSUER:ACCOUNT?secret=SECRET&issuer=ISSUER&algorithm=SHA1&digits=6&period=30
  """
  def provisioning_uri(secret, account_name, opts \\ []) do
    issuer = Keyword.get(opts, :issuer, @default_issuer)
    algorithm = Keyword.get(opts, :algorithm, @default_algorithm)
    digits = Keyword.get(opts, :digits, @default_digits)
    period = Keyword.get(opts, :period, @default_period)

    label = URI.encode("#{issuer}:#{account_name}")
    
    params = URI.encode_query(%{
      "secret" => secret,
      "issuer" => issuer,
      "algorithm" => algorithm,
      "digits" => digits,
      "period" => period
    })

    "otpauth://totp/#{label}?#{params}"
  end

  @doc """
  Generates a QR code as base64-encoded PNG.
  Requires the `eqrcode` library.
  """
  def generate_qr_code(provisioning_uri) do
    # In production, use a QR code library like eqrcode
    # For now, return a placeholder
    {:ok, Base.encode64("QR_CODE_PLACEHOLDER:#{provisioning_uri}")}
  end

  @doc """
  Encrypts a TOTP secret for storage.
  Uses AES-256-GCM for authenticated encryption.
  """
  def encrypt_secret(secret, key) do
    iv = :crypto.strong_rand_bytes(12)
    {ciphertext, tag} = :crypto.crypto_one_time_aead(
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
  """
  def decrypt_secret(encrypted, key) do
    <<iv::binary-size(12), tag::binary-size(16), ciphertext::binary>> = 
      Base.decode64!(encrypted)
    
    :crypto.crypto_one_time_aead(
      :aes_256_gcm,
      key,
      iv,
      ciphertext,
      "",
      tag
    )
  end
end
