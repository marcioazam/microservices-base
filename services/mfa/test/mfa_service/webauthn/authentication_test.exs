defmodule MfaService.WebAuthn.AuthenticationTest do
  @moduledoc """
  Unit tests for WebAuthn Authentication module.
  """

  use ExUnit.Case, async: true

  alias MfaService.WebAuthn.Authentication
  alias MfaService.Challenge

  @moduletag :unit

  describe "begin_authentication/3" do
    test "generates challenge and options" do
      user_id = "user-123"
      credentials = [
        %{credential_id: :crypto.strong_rand_bytes(32), transports: ["internal"]}
      ]

      {:ok, options, challenge} = Authentication.begin_authentication(user_id, credentials)

      assert is_binary(challenge)
      assert byte_size(challenge) == 32
      assert is_map(options)
      assert options.challenge == challenge
    end

    test "includes allow_credentials" do
      user_id = "user-123"
      cred_id = :crypto.strong_rand_bytes(32)
      credentials = [%{credential_id: cred_id, transports: ["internal", "hybrid"]}]

      {:ok, options, _} = Authentication.begin_authentication(user_id, credentials)

      assert length(options.allow_credentials) == 1
      [cred] = options.allow_credentials
      assert cred.type == "public-key"
      assert cred.id == cred_id
      assert cred.transports == ["internal", "hybrid"]
    end

    test "uses default options" do
      {:ok, options, _} = Authentication.begin_authentication("user", [])

      assert options.timeout == 60_000
      assert options.rp_id == "localhost"
      assert options.user_verification == "preferred"
    end

    test "respects custom options" do
      {:ok, options, _} = Authentication.begin_authentication("user", [],
        rp_id: "example.com",
        timeout: 120_000,
        user_verification: "required"
      )

      assert options.rp_id == "example.com"
      assert options.timeout == 120_000
      assert options.user_verification == "required"
    end
  end

  describe "parse_authenticator_data/1" do
    test "parses valid authenticator data" do
      rp_id_hash = :crypto.hash(:sha256, "localhost")
      flags = 0x05  # UP + UV
      sign_count = 42

      auth_data = <<
        rp_id_hash::binary-size(32),
        flags::8,
        sign_count::unsigned-big-integer-size(32)
      >>

      {:ok, parsed} = Authentication.parse_authenticator_data(auth_data)

      assert parsed.rp_id_hash == rp_id_hash
      assert parsed.sign_count == 42
      assert parsed.flags.user_present == true
      assert parsed.flags.user_verified == true
    end

    test "parses flags correctly" do
      rp_id_hash = :crypto.hash(:sha256, "localhost")

      # Test various flag combinations
      test_cases = [
        {0x01, %{user_present: true, user_verified: false}},
        {0x04, %{user_present: false, user_verified: true}},
        {0x05, %{user_present: true, user_verified: true}},
        {0x00, %{user_present: false, user_verified: false}}
      ]

      for {flags, expected} <- test_cases do
        auth_data = <<rp_id_hash::binary-size(32), flags::8, 0::32>>
        {:ok, parsed} = Authentication.parse_authenticator_data(auth_data)

        assert parsed.flags.user_present == expected.user_present
        assert parsed.flags.user_verified == expected.user_verified
      end
    end

    test "returns error for invalid data" do
      # Too short
      assert {:error, :invalid_auth_data} = Authentication.parse_authenticator_data(<<1, 2, 3>>)
    end

    test "parses sign count as big-endian" do
      rp_id_hash = :crypto.hash(:sha256, "localhost")
      # 0x01020304 = 16909060
      auth_data = <<rp_id_hash::binary-size(32), 0x05::8, 0x01, 0x02, 0x03, 0x04>>

      {:ok, parsed} = Authentication.parse_authenticator_data(auth_data)

      assert parsed.sign_count == 16_909_060
    end
  end

  describe "Challenge integration" do
    test "stores challenge for later verification" do
      user_id = "test-user-#{System.unique_integer()}"

      {:ok, _options, challenge} = Authentication.begin_authentication(user_id, [])

      # Challenge should be stored
      {:ok, stored} = Challenge.retrieve("webauthn:auth:#{user_id}")
      assert stored == challenge
    end
  end
end
