defmodule MfaService.TOTP.GeneratorTest do
  @moduledoc """
  Unit tests for TOTP Generator module.
  """

  use ExUnit.Case, async: true

  alias MfaService.TOTP.Generator

  @moduletag :unit

  describe "generate_secret/1" do
    test "generates base32-encoded secret" do
      secret = Generator.generate_secret()

      assert is_binary(secret)
      assert {:ok, _} = Base.decode32(secret, padding: false)
    end

    test "generates secret with at least 160 bits (20 bytes)" do
      secret = Generator.generate_secret()
      {:ok, decoded} = Base.decode32(secret, padding: false)

      assert byte_size(decoded) >= 20
    end

    test "generates unique secrets" do
      secrets = for _ <- 1..100, do: Generator.generate_secret()

      assert length(Enum.uniq(secrets)) == 100
    end

    test "respects custom length" do
      secret = Generator.generate_secret(32)
      {:ok, decoded} = Base.decode32(secret, padding: false)

      assert byte_size(decoded) == 32
    end
  end

  describe "provisioning_uri/3" do
    test "generates valid otpauth URI" do
      secret = Generator.generate_secret()
      uri = Generator.provisioning_uri(secret, "user@example.com")

      assert String.starts_with?(uri, "otpauth://totp/")
    end

    test "includes required parameters" do
      secret = Generator.generate_secret()
      uri = Generator.provisioning_uri(secret, "user@example.com")

      assert uri =~ "secret=#{secret}"
      assert uri =~ "issuer="
      assert uri =~ "algorithm="
      assert uri =~ "digits="
      assert uri =~ "period="
    end

    test "uses default issuer" do
      secret = Generator.generate_secret()
      uri = Generator.provisioning_uri(secret, "user@example.com")

      assert uri =~ "issuer=AuthPlatform"
    end

    test "respects custom issuer" do
      secret = Generator.generate_secret()
      uri = Generator.provisioning_uri(secret, "user@example.com", issuer: "MyApp")

      assert uri =~ "issuer=MyApp"
    end

    test "encodes account name in label" do
      secret = Generator.generate_secret()
      uri = Generator.provisioning_uri(secret, "user@example.com")

      assert uri =~ "AuthPlatform%3Auser%40example.com"
    end
  end

  describe "encrypt_secret/2" do
    test "encrypts secret with AES-256-GCM" do
      secret = Generator.generate_secret()
      key = :crypto.strong_rand_bytes(32)

      encrypted = Generator.encrypt_secret(secret, key)

      assert is_binary(encrypted)
      assert encrypted != secret
    end

    test "produces different ciphertext each time (random IV)" do
      secret = Generator.generate_secret()
      key = :crypto.strong_rand_bytes(32)

      encrypted1 = Generator.encrypt_secret(secret, key)
      encrypted2 = Generator.encrypt_secret(secret, key)

      assert encrypted1 != encrypted2
    end
  end

  describe "decrypt_secret/2" do
    test "decrypts encrypted secret" do
      secret = Generator.generate_secret()
      key = :crypto.strong_rand_bytes(32)

      encrypted = Generator.encrypt_secret(secret, key)
      {:ok, decrypted} = Generator.decrypt_secret(encrypted, key)

      assert decrypted == secret
    end

    test "fails with wrong key" do
      secret = Generator.generate_secret()
      key1 = :crypto.strong_rand_bytes(32)
      key2 = :crypto.strong_rand_bytes(32)

      encrypted = Generator.encrypt_secret(secret, key1)
      result = Generator.decrypt_secret(encrypted, key2)

      assert result == {:error, :decryption_failed}
    end

    test "fails with corrupted ciphertext" do
      secret = Generator.generate_secret()
      key = :crypto.strong_rand_bytes(32)

      encrypted = Generator.encrypt_secret(secret, key)
      corrupted = "corrupted" <> encrypted

      result = Generator.decrypt_secret(corrupted, key)

      assert result == {:error, :decryption_failed}
    end
  end

  describe "valid_secret?/1" do
    test "returns true for valid secret" do
      secret = Generator.generate_secret()

      assert Generator.valid_secret?(secret) == true
    end

    test "returns false for short secret" do
      short = :crypto.strong_rand_bytes(10) |> Base.encode32(padding: false)

      assert Generator.valid_secret?(short) == false
    end

    test "returns false for invalid base32" do
      assert Generator.valid_secret?("not-valid-base32!") == false
    end

    test "returns false for non-binary" do
      assert Generator.valid_secret?(123) == false
      assert Generator.valid_secret?(nil) == false
    end
  end

  describe "generate_qr_code/1" do
    test "returns base64-encoded data" do
      secret = Generator.generate_secret()
      uri = Generator.provisioning_uri(secret, "user@example.com")

      {:ok, qr_data} = Generator.generate_qr_code(uri)

      assert is_binary(qr_data)
      assert {:ok, _} = Base.decode64(qr_data)
    end
  end
end
