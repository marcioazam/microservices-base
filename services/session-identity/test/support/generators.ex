defmodule SessionIdentityCore.Test.Generators do
  @moduledoc """
  Centralized StreamData generators for property-based testing.
  """

  use ExUnitProperties

  alias SessionIdentityCore.Sessions.Session

  @valid_pkce_chars ~c"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

  @doc """
  Generates a valid code_verifier (43-128 chars, valid charset).
  """
  def code_verifier do
    gen all(
          length <- integer(43..128),
          chars <- list_of(member_of(@valid_pkce_chars), length: length)
        ) do
      List.to_string(chars)
    end
  end

  @doc """
  Generates an invalid code_verifier (too short).
  """
  def short_code_verifier do
    gen all(
          length <- integer(1..42),
          chars <- list_of(member_of(@valid_pkce_chars), length: length)
        ) do
      List.to_string(chars)
    end
  end

  @doc """
  Generates an invalid code_verifier (too long).
  """
  def long_code_verifier do
    gen all(
          length <- integer(129..200),
          chars <- list_of(member_of(@valid_pkce_chars), length: length)
        ) do
      List.to_string(chars)
    end
  end

  @doc """
  Generates a valid session struct.
  """
  def session do
    gen all(
          id <- uuid(),
          user_id <- uuid(),
          device_id <- uuid(),
          ip_address <- ip_address(),
          device_fingerprint <- string(:alphanumeric, min_length: 32, max_length: 64),
          risk_score <- float(min: 0.0, max: 1.0),
          mfa_verified <- boolean()
        ) do
      now = DateTime.utc_now()

      %Session{
        id: id,
        user_id: user_id,
        device_id: device_id,
        ip_address: ip_address,
        user_agent: "Mozilla/5.0",
        device_fingerprint: device_fingerprint,
        risk_score: risk_score,
        mfa_verified: mfa_verified,
        expires_at: DateTime.add(now, 86_400, :second),
        last_activity: now,
        inserted_at: now,
        updated_at: now
      }
    end
  end

  @doc """
  Generates a UUID string.
  """
  def uuid do
    gen all(bytes <- binary(length: 16)) do
      <<a::32, b::16, c::16, d::16, e::48>> = bytes

      :io_lib.format(
        "~8.16.0b-~4.16.0b-~4.16.0b-~4.16.0b-~12.16.0b",
        [a, b, c, d, e]
      )
      |> IO.iodata_to_binary()
    end
  end

  @doc """
  Generates a valid IP address string.
  """
  def ip_address do
    gen all(
          a <- integer(1..255),
          b <- integer(0..255),
          c <- integer(0..255),
          d <- integer(1..254)
        ) do
      "#{a}.#{b}.#{c}.#{d}"
    end
  end

  @doc """
  Generates a risk score in valid range [0.0, 1.0].
  """
  def risk_score do
    float(min: 0.0, max: 1.0)
  end

  @doc """
  Generates a redirect URI.
  """
  def redirect_uri do
    gen all(
          scheme <- member_of(["https"]),
          domain <- string(:alphanumeric, min_length: 3, max_length: 20),
          path <- string(:alphanumeric, min_length: 0, max_length: 10)
        ) do
      "#{scheme}://#{domain}.example.com/#{path}"
    end
  end

  @doc """
  Generates session creation attributes.
  """
  def session_attrs do
    gen all(
          user_id <- uuid(),
          device_id <- uuid(),
          ip_address <- ip_address(),
          device_fingerprint <- string(:alphanumeric, min_length: 32, max_length: 64)
        ) do
      %{
        user_id: user_id,
        device_id: device_id,
        ip_address: ip_address,
        user_agent: "Mozilla/5.0",
        device_fingerprint: device_fingerprint
      }
    end
  end

  @doc """
  Generates ID token claims parameters.
  """
  def id_token_params do
    gen all(
          sub <- uuid(),
          aud <- string(:alphanumeric, min_length: 10, max_length: 30),
          nonce <- one_of([constant(nil), string(:alphanumeric, min_length: 16, max_length: 32)]),
          ttl <- integer(60..7200)
        ) do
      %{
        sub: sub,
        iss: "https://auth.example.com",
        aud: aud,
        nonce: nonce,
        ttl: ttl
      }
    end
  end
end
