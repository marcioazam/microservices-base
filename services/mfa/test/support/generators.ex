defmodule MfaService.Test.Generators do
  @moduledoc """
  StreamData generators for property-based testing.
  """

  use ExUnitProperties

  @doc """
  Generates a random UUID v4.
  """
  def generate_uuid do
    <<a::32, b::16, c::16, d::16, e::48>> = :crypto.strong_rand_bytes(16)

    # Set version (4) and variant (10xx)
    c = (c &&& 0x0FFF) ||| 0x4000
    d = (d &&& 0x3FFF) ||| 0x8000

    :io_lib.format("~8.16.0b-~4.16.0b-~4.16.0b-~4.16.0b-~12.16.0b", [a, b, c, d, e])
    |> IO.iodata_to_binary()
  end

  @doc """
  Generates a valid TOTP secret (base32 encoded, 20+ bytes).
  """
  def totp_secret do
    gen all bytes <- StreamData.binary(length: 20) do
      Base.encode32(bytes, padding: false)
    end
  end

  @doc """
  Generates a valid 6-digit TOTP code.
  """
  def totp_code do
    gen all code <- StreamData.integer(0..999_999) do
      code |> Integer.to_string() |> String.pad_leading(6, "0")
    end
  end

  @doc """
  Generates a valid WebAuthn challenge (32 bytes).
  """
  def webauthn_challenge do
    StreamData.binary(length: 32)
  end

  @doc """
  Generates a valid user ID.
  """
  def user_id do
    gen all id <- StreamData.binary(min_length: 16, max_length: 36) do
      Base.url_encode64(id, padding: false)
    end
  end

  @doc """
  Generates a valid passkey name (1-255 characters).
  """
  def passkey_name do
    StreamData.string(:alphanumeric, min_length: 1, max_length: 255)
  end

  @doc """
  Generates an invalid passkey name (>255 characters).
  """
  def invalid_passkey_name do
    StreamData.string(:alphanumeric, min_length: 256, max_length: 500)
  end

  @doc """
  Generates device fingerprint attributes.
  """
  def device_attributes do
    gen all user_agent <- StreamData.string(:alphanumeric, min_length: 10, max_length: 200),
            accept_language <- StreamData.member_of(["en-US", "pt-BR", "es-ES", "fr-FR"]),
            timezone <- StreamData.member_of(["UTC", "America/Sao_Paulo", "Europe/London"]),
            screen_resolution <- StreamData.member_of(["1920x1080", "1366x768", "2560x1440"]),
            platform <- StreamData.member_of(["Win32", "MacIntel", "Linux x86_64"]) do
      %{
        user_agent: user_agent,
        accept_language: accept_language,
        accept_encoding: "gzip, deflate, br",
        timezone: timezone,
        screen_resolution: screen_resolution,
        platform: platform,
        plugins: [],
        canvas_hash: "",
        webgl_hash: ""
      }
    end
  end

  @doc """
  Generates a sign count pair where new > old.
  """
  def sign_count_pair do
    gen all old <- StreamData.integer(0..1_000_000) do
      new = old + Enum.random(1..1000)
      {old, new}
    end
  end

  @doc """
  Generates a sign count pair where new <= old (invalid).
  """
  def invalid_sign_count_pair do
    gen all old <- StreamData.integer(1..1_000_000) do
      new = Enum.random(0..old)
      {old, new}
    end
  end

  @doc """
  Generates a recent DateTime (within 5 minutes).
  """
  def recent_datetime do
    gen all seconds_ago <- StreamData.integer(0..299) do
      DateTime.utc_now() |> DateTime.add(-seconds_ago, :second)
    end
  end

  @doc """
  Generates an old DateTime (more than 5 minutes ago).
  """
  def old_datetime do
    gen all seconds_ago <- StreamData.integer(301..3600) do
      DateTime.utc_now() |> DateTime.add(-seconds_ago, :second)
    end
  end

  @doc """
  Generates AES-256 encryption key (32 bytes).
  """
  def encryption_key do
    StreamData.binary(length: 32)
  end

  @doc """
  Generates a valid account name for TOTP provisioning URI.
  """
  def account_name do
    gen all name <- StreamData.string(:alphanumeric, min_length: 3, max_length: 50),
            domain <- StreamData.member_of(["example.com", "test.org", "auth.io"]) do
      "#{name}@#{domain}"
    end
  end
end
