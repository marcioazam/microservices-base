defmodule MfaService.Property.WebAuthnPropertyTest do
  @moduledoc """
  Property-based tests for WebAuthn/Challenge modules.
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.Challenge
  alias MfaService.WebAuthn.Authentication
  alias MfaService.Test.Generators

  @moduletag :property

  describe "Property 5: WebAuthn Challenge Entropy" do
    @tag property: 5
    property "challenges are exactly 32 bytes with balanced bit distribution" do
      check all _iteration <- StreamData.constant(:ok), max_runs: 100 do
        challenge = Challenge.generate()

        # Must be exactly 32 bytes
        assert byte_size(challenge) == 32

        # Check bit distribution (30-70% ones is reasonable for random data)
        ones = count_ones(challenge)
        total_bits = 32 * 8
        ratio = ones / total_bits

        assert ratio >= 0.30 and ratio <= 0.70,
               "Bit distribution should be 30-70% ones, got #{Float.round(ratio * 100, 1)}%"
      end
    end

    property "challenges are unique" do
      check all _iteration <- StreamData.constant(:ok), max_runs: 100 do
        c1 = Challenge.generate()
        c2 = Challenge.generate()
        assert c1 != c2
      end
    end
  end

  describe "Property 6: WebAuthn Challenge Encode-Decode Round-Trip" do
    @tag property: 6
    property "encoding then decoding produces original challenge" do
      check all challenge <- Generators.webauthn_challenge(), max_runs: 100 do
        encoded = Challenge.encode(challenge)
        {:ok, decoded} = Challenge.decode(encoded)

        assert decoded == challenge
      end
    end

    property "encoded challenges are valid base64url" do
      check all challenge <- Generators.webauthn_challenge(), max_runs: 100 do
        encoded = Challenge.encode(challenge)

        # Base64URL should not contain + or /
        refute String.contains?(encoded, "+")
        refute String.contains?(encoded, "/")

        # Should not have padding
        refute String.ends_with?(encoded, "=")
      end
    end
  end

  describe "Property 7: WebAuthn Sign Count Monotonicity" do
    @tag property: 7
    property "authentication succeeds only when new sign count > stored" do
      check all {stored, new} <- Generators.sign_count_pair(), max_runs: 100 do
        # Build minimal authenticator data with sign count
        auth_data = build_auth_data(new)

        # This should pass - new > stored
        assert new > stored
        assert {:ok, _} = Authentication.parse_authenticator_data(auth_data)
      end
    end

    property "authentication fails when sign count not increased" do
      check all {stored, new} <- Generators.invalid_sign_count_pair(), max_runs: 100 do
        # new <= stored should fail
        assert new <= stored
      end
    end
  end

  describe "Property 8: WebAuthn Authenticator Data Parsing" do
    @tag property: 8
    property "parsing extracts correct RP ID hash, flags, and sign count" do
      check all rp_id_hash <- StreamData.binary(length: 32),
                flags <- StreamData.integer(0..255),
                sign_count <- StreamData.integer(0..0xFFFFFFFF),
                max_runs: 100 do
        auth_data = <<
          rp_id_hash::binary-size(32),
          flags::8,
          sign_count::unsigned-big-integer-size(32)
        >>

        {:ok, parsed} = Authentication.parse_authenticator_data(auth_data)

        assert parsed.rp_id_hash == rp_id_hash
        assert parsed.sign_count == sign_count

        # Verify flag parsing
        assert parsed.flags.user_present == ((flags &&& 0x01) == 0x01)
        assert parsed.flags.user_verified == ((flags &&& 0x04) == 0x04)
      end
    end
  end

  describe "Challenge Verification" do
    property "verify succeeds for matching challenges" do
      check all challenge <- Generators.webauthn_challenge(), max_runs: 100 do
        assert :ok == Challenge.verify(challenge, challenge)
      end
    end

    property "verify fails for different challenges" do
      check all c1 <- Generators.webauthn_challenge(),
                c2 <- Generators.webauthn_challenge(),
                c1 != c2,
                max_runs: 100 do
        assert {:error, :challenge_mismatch} == Challenge.verify(c1, c2)
      end
    end
  end

  # Helper functions

  defp count_ones(binary) do
    binary
    |> :binary.bin_to_list()
    |> Enum.reduce(0, fn byte, acc ->
      acc + count_byte_ones(byte)
    end)
  end

  defp count_byte_ones(byte) do
    Enum.reduce(0..7, 0, fn bit, acc ->
      if (byte >>> bit &&& 1) == 1, do: acc + 1, else: acc
    end)
  end

  defp build_auth_data(sign_count) do
    rp_id_hash = :crypto.hash(:sha256, "localhost")
    flags = 0x05

    <<
      rp_id_hash::binary-size(32),
      flags::8,
      sign_count::unsigned-big-integer-size(32)
    >>
  end
end
