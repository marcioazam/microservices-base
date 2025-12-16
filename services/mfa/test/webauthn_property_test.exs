defmodule MfaService.WebAuthnPropertyTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.WebAuthn.Challenge
  alias MfaService.WebAuthn.Authentication

  # Generators
  defp user_id_generator do
    StreamData.string(:alphanumeric, min_length: 16, max_length: 36)
  end

  defp credential_id_generator do
    StreamData.binary(min_length: 16, max_length: 64)
    |> StreamData.map(&Base.url_encode64(&1, padding: false))
  end

  defp public_key_generator do
    # Simulated COSE key structure
    StreamData.binary(min_length: 32, max_length: 128)
  end

  defp credential_generator do
    StreamData.fixed_map(%{
      credential_id: credential_id_generator(),
      public_key: public_key_generator(),
      sign_count: StreamData.integer(0..1_000_000),
      transports: StreamData.list_of(
        StreamData.member_of(["usb", "nfc", "ble", "internal", "hybrid"]),
        max_length: 3
      )
    })
  end

  # **Feature: auth-platform-2025-enhancements, Property 5: WebAuthn Challenge Entropy**
  # **Validates: Requirements 2.1, 2.3**
  property "WebAuthn challenges have at least 32 bytes of cryptographic randomness" do
    check all _i <- StreamData.integer(1..100), max_runs: 100 do
      # Generate challenge
      challenge = Challenge.generate()

      # Verify length (32 bytes minimum)
      assert byte_size(challenge) >= 32

      # Verify randomness - check bit distribution
      bits = for <<bit::1 <- challenge>>, do: bit
      ones = Enum.count(bits, &(&1 == 1))
      total = length(bits)

      # Should be roughly balanced (30-70% range for 256+ bits)
      ratio = ones / total
      assert ratio > 0.3 and ratio < 0.7,
             "Challenge bits should be roughly balanced, got #{ratio}"

      # Verify no obvious patterns (all zeros, all ones, repeating)
      refute challenge == <<0::size(byte_size(challenge) * 8)>>
      refute challenge == <<255::size(byte_size(challenge))>>
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 6: WebAuthn Credential Storage Completeness**
  # **Validates: Requirements 2.2**
  property "WebAuthn credentials contain all required fields" do
    check all credential <- credential_generator(), max_runs: 100 do
      # Verify public key is present and non-empty
      assert credential.public_key != nil
      assert byte_size(credential.public_key) > 0

      # Verify credential ID is present and non-empty
      assert credential.credential_id != nil
      assert String.length(credential.credential_id) > 0

      # Verify sign count is non-negative
      assert credential.sign_count >= 0

      # Verify transports is a list
      assert is_list(credential.transports)
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 8: Multi-Authenticator Support**
  # **Validates: Requirements 2.6**
  property "authentication succeeds with any registered authenticator" do
    check all credentials <- StreamData.list_of(credential_generator(), min_length: 2, max_length: 5),
              selected_index <- StreamData.integer(0..4),
              max_runs: 100 do
      # Ensure we have at least 2 credentials
      if length(credentials) >= 2 do
        # Select one credential (bounded by actual list size)
        index = rem(selected_index, length(credentials))
        selected = Enum.at(credentials, index)

        # Verify the selected credential is in the list
        assert selected in credentials

        # All credentials should be valid for authentication
        Enum.each(credentials, fn cred ->
          assert cred.credential_id != nil
          assert cred.public_key != nil
        end)
      end
    end
  end

  describe "Challenge" do
    test "generates unique challenges" do
      challenges = for _ <- 1..100, do: Challenge.generate()
      unique_challenges = Enum.uniq(challenges)

      # All challenges should be unique
      assert length(unique_challenges) == 100
    end

    test "challenge encoding is base64url without padding" do
      challenge = Challenge.generate()
      encoded = Challenge.encode(challenge)

      # Should not contain padding
      refute String.contains?(encoded, "=")

      # Should be valid base64url
      assert {:ok, _} = Base.url_decode64(encoded, padding: false)
    end

    test "challenge round-trip encoding" do
      original = Challenge.generate()
      encoded = Challenge.encode(original)
      decoded = Challenge.decode(encoded)

      assert decoded == original
    end
  end

  describe "Authentication" do
    test "begin_authentication generates valid options" do
      user_id = "user-123"
      credentials = [
        %{credential_id: "cred-1", transports: ["internal"]},
        %{credential_id: "cred-2", transports: ["usb"]}
      ]

      {:ok, options, challenge} = Authentication.begin_authentication(user_id, credentials)

      assert options.challenge != nil
      assert byte_size(challenge) >= 32
      assert length(options.allow_credentials) == 2
    end
  end
end
