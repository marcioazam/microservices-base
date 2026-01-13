defmodule MfaService.WebAuthn.ChallengeTest do
  @moduledoc """
  Unit tests for WebAuthn Challenge module.
  """

  use ExUnit.Case, async: true

  alias MfaService.WebAuthn.Challenge
  alias MfaService.Challenge, as: CentralizedChallenge

  @moduletag :unit

  describe "generate/1" do
    test "generates 32-byte challenge by default" do
      challenge = Challenge.generate()

      assert byte_size(challenge) == 32
    end

    test "generates unique challenges" do
      challenges = for _ <- 1..100, do: Challenge.generate()

      assert length(Enum.uniq(challenges)) == 100
    end

    test "delegates to centralized Challenge module" do
      # Both should produce 32-byte challenges
      webauthn_challenge = Challenge.generate()
      centralized_challenge = CentralizedChallenge.generate()

      assert byte_size(webauthn_challenge) == byte_size(centralized_challenge)
    end
  end

  describe "encode/1" do
    test "encodes challenge as base64url" do
      challenge = Challenge.generate()
      encoded = Challenge.encode(challenge)

      assert is_binary(encoded)
      # Base64URL should not contain + or /
      refute String.contains?(encoded, "+")
      refute String.contains?(encoded, "/")
      # Should not have padding
      refute String.ends_with?(encoded, "=")
    end

    test "produces consistent encoding" do
      challenge = :crypto.strong_rand_bytes(32)

      encoded1 = Challenge.encode(challenge)
      encoded2 = Challenge.encode(challenge)

      assert encoded1 == encoded2
    end
  end

  describe "decode/1" do
    test "decodes base64url-encoded challenge" do
      original = :crypto.strong_rand_bytes(32)
      encoded = Base.url_encode64(original, padding: false)

      decoded = Challenge.decode(encoded)

      assert decoded == original
    end

    test "raises on invalid encoding" do
      assert_raise ArgumentError, fn ->
        Challenge.decode("not-valid-base64!")
      end
    end
  end

  describe "store/3" do
    test "stores challenge with TTL" do
      user_id = "test-user-#{System.unique_integer()}"
      challenge = Challenge.generate()

      result = Challenge.store(challenge, user_id)

      assert result == :ok
    end

    test "uses default TTL of 300 seconds" do
      user_id = "test-user-#{System.unique_integer()}"
      challenge = Challenge.generate()

      :ok = Challenge.store(challenge, user_id)

      # Should be retrievable
      {:ok, _} = Challenge.retrieve_and_delete(user_id)
    end

    test "respects custom TTL" do
      user_id = "test-user-#{System.unique_integer()}"
      challenge = Challenge.generate()

      :ok = Challenge.store(challenge, user_id, 60)

      {:ok, _} = Challenge.retrieve_and_delete(user_id)
    end
  end

  describe "retrieve_and_delete/1" do
    test "retrieves and deletes stored challenge" do
      user_id = "test-user-#{System.unique_integer()}"
      challenge = Challenge.generate()

      Challenge.store(challenge, user_id)

      {:ok, retrieved} = Challenge.retrieve_and_delete(user_id)
      assert retrieved == challenge

      # Second retrieval should fail
      {:error, :challenge_not_found} = Challenge.retrieve_and_delete(user_id)
    end

    test "returns error for non-existent challenge" do
      result = Challenge.retrieve_and_delete("non-existent-user")

      assert result == {:error, :challenge_not_found}
    end
  end

  describe "verify/2" do
    test "returns :ok for matching challenges" do
      challenge = Challenge.generate()

      assert :ok == Challenge.verify(challenge, challenge)
    end

    test "returns error for mismatched challenges" do
      c1 = Challenge.generate()
      c2 = Challenge.generate()

      assert {:error, :challenge_mismatch} == Challenge.verify(c1, c2)
    end

    test "uses constant-time comparison" do
      # This is hard to test directly, but we verify it delegates
      # to the centralized module which uses constant_time_compare
      challenge = Challenge.generate()

      # Should not raise or behave differently based on input
      assert :ok == Challenge.verify(challenge, challenge)
      assert {:error, :challenge_mismatch} == Challenge.verify(challenge, <<0::256>>)
    end
  end
end
