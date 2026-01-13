defmodule MfaService.Property.TOTPPropertyTest do
  @moduledoc """
  Property-based tests for TOTP module.
  Validates universal correctness properties per spec.
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.TOTP.{Generator, Validator}
  alias MfaService.Test.Generators

  @moduletag :property

  describe "Property 1: TOTP Secret Entropy" do
    @tag property: 1
    property "generated secrets have at least 160 bits (20 bytes) of entropy" do
      check all _iteration <- StreamData.constant(:ok), max_runs: 100 do
        secret = Generator.generate_secret()

        # Decode and verify length
        {:ok, decoded} = Base.decode32(secret, padding: false)
        assert byte_size(decoded) >= 20, "Secret must be at least 20 bytes"

        # Verify it's from crypto.strong_rand_bytes (not predictable)
        secret2 = Generator.generate_secret()
        assert secret != secret2, "Secrets must be unique"
      end
    end
  end

  describe "Property 2: TOTP Encryption Round-Trip" do
    @tag property: 2
    property "encrypting then decrypting produces original secret" do
      check all secret <- Generators.totp_secret(),
                key <- Generators.encryption_key(),
                max_runs: 100 do
        encrypted = Generator.encrypt_secret(secret, key)
        {:ok, decrypted} = Generator.decrypt_secret(encrypted, key)

        assert decrypted == secret
      end
    end
  end

  describe "Property 3: TOTP Provisioning URI Format" do
    @tag property: 3
    property "provisioning URI matches RFC 6238 format" do
      check all secret <- Generators.totp_secret(),
                account <- Generators.account_name(),
                max_runs: 100 do
        uri = Generator.provisioning_uri(secret, account)

        # Must start with otpauth://totp/
        assert String.starts_with?(uri, "otpauth://totp/")

        # Must contain required parameters
        assert uri =~ "secret="
        assert uri =~ "issuer="
        assert uri =~ "algorithm="
        assert uri =~ "digits="
        assert uri =~ "period="

        # Parse and validate structure
        %URI{scheme: scheme, host: host, query: query} = URI.parse(uri)
        assert scheme == "otpauth"
        assert host == "totp"
        assert query != nil

        params = URI.decode_query(query)
        assert params["secret"] == secret
        assert params["digits"] == "6"
        assert params["period"] == "30"
      end
    end
  end

  describe "Property 4: TOTP Generate-Validate Round-Trip" do
    @tag property: 4
    property "generated code validates within same time window" do
      check all secret <- Generators.totp_secret(),
                max_runs: 100 do
        timestamp = System.system_time(:second)
        code = Validator.generate_code(secret, timestamp)

        # Code should validate at same timestamp
        assert :ok == Validator.validate(code, secret, timestamp: timestamp)

        # Code should validate within ±1 window
        assert :ok == Validator.validate(code, secret, timestamp: timestamp + 30)
        assert :ok == Validator.validate(code, secret, timestamp: timestamp - 30)
      end
    end

    property "code outside window fails validation" do
      check all secret <- Generators.totp_secret(),
                max_runs: 100 do
        timestamp = System.system_time(:second)
        code = Validator.generate_code(secret, timestamp)

        # Code should fail outside ±1 window (±2 periods = 60 seconds)
        assert {:error, :invalid_code} ==
                 Validator.validate(code, secret, timestamp: timestamp + 90)

        assert {:error, :invalid_code} ==
                 Validator.validate(code, secret, timestamp: timestamp - 90)
      end
    end
  end

  describe "TOTP Code Format" do
    property "generated codes are always 6 digits" do
      check all secret <- Generators.totp_secret(),
                timestamp <- StreamData.integer(1_000_000_000..2_000_000_000),
                max_runs: 100 do
        code = Validator.generate_code(secret, timestamp)

        assert String.length(code) == 6
        assert String.match?(code, ~r/^\d{6}$/)
      end
    end
  end

  describe "Secret Validation" do
    property "valid secrets pass validation" do
      check all secret <- Generators.totp_secret(), max_runs: 100 do
        assert Generator.valid_secret?(secret)
      end
    end

    test "short secrets fail validation" do
      short_secret = :crypto.strong_rand_bytes(10) |> Base.encode32(padding: false)
      refute Generator.valid_secret?(short_secret)
    end
  end
end
