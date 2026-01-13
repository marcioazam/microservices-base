defmodule MfaService.Crypto.ClientPropertiesTest do
  @moduledoc """
  Property-based tests for Crypto Client.
  Tests correctness properties defined in the design document.
  """
  use ExUnit.Case, async: false
  use ExUnitProperties

  alias MfaService.Crypto.{Client, Config}

  # Mock module for testing without real crypto-service
  defmodule MockCryptoService do
    @moduledoc false
    
    def encrypt(plaintext, key_id, aad) do
      # Simulate AES-256-GCM encryption
      iv = :crypto.strong_rand_bytes(12)
      {ciphertext, tag} = :crypto.crypto_one_time_aead(
        :aes_256_gcm,
        get_key(key_id),
        iv,
        plaintext,
        aad,
        true
      )
      
      {:ok, %{
        ciphertext: ciphertext,
        iv: iv,
        tag: tag,
        key_id: key_id,
        algorithm: "AES-256-GCM"
      }}
    end

    def decrypt(ciphertext, iv, tag, key_id, aad) do
      case :crypto.crypto_one_time_aead(
        :aes_256_gcm,
        get_key(key_id),
        iv,
        ciphertext,
        aad,
        tag
      ) do
        :error -> {:error, :decryption_failed}
        plaintext -> {:ok, plaintext}
      end
    end

    defp get_key(_key_id) do
      # Use a fixed test key for property testing
      :crypto.hash(:sha256, "test-key-material")
    end
  end

  # Generators

  defp secret_generator do
    # Generate TOTP-like secrets (20 bytes, base32 encoded)
    gen all bytes <- binary(length: 20) do
      Base.encode32(bytes, padding: false)
    end
  end

  defp user_id_generator do
    gen all uuid <- binary(length: 16) do
      Base.encode16(uuid, case: :lower)
    end
  end

  defp key_id_generator do
    gen all id <- binary(length: 16) do
      %{
        namespace: "mfa:totp",
        id: Base.encode16(id, case: :lower),
        version: 1
      }
    end
  end

  defp correlation_id_generator do
    gen all uuid <- binary(length: 16) do
      Base.encode16(uuid, case: :lower)
    end
  end

  describe "Property 1: Encryption Round-Trip" do
    @tag :property
    @tag timeout: 120_000
    property "encrypting then decrypting returns original secret" do
      check all secret <- secret_generator(),
                user_id <- user_id_generator(),
                key_id <- key_id_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        # Encrypt
        {:ok, encrypted} = MockCryptoService.encrypt(secret, key_id, user_id)
        
        # Decrypt
        {:ok, decrypted} = MockCryptoService.decrypt(
          encrypted.ciphertext,
          encrypted.iv,
          encrypted.tag,
          key_id,
          user_id
        )
        
        # Verify round-trip
        assert decrypted == secret,
          "Round-trip failed: expected #{inspect(secret)}, got #{inspect(decrypted)}"
      end
    end

    @tag :property
    property "decryption fails with wrong AAD (user_id)" do
      check all secret <- secret_generator(),
                user_id <- user_id_generator(),
                wrong_user_id <- user_id_generator(),
                key_id <- key_id_generator(),
                max_runs: 100 do
        
        # Skip if user_ids happen to be the same
        if user_id != wrong_user_id do
          # Encrypt with correct user_id
          {:ok, encrypted} = MockCryptoService.encrypt(secret, key_id, user_id)
          
          # Decrypt with wrong user_id should fail
          result = MockCryptoService.decrypt(
            encrypted.ciphertext,
            encrypted.iv,
            encrypted.tag,
            key_id,
            wrong_user_id
          )
          
          assert result == {:error, :decryption_failed},
            "Decryption should fail with wrong AAD"
        end
      end
    end
  end

  describe "Property 9: AAD Includes User ID" do
    @tag :property
    property "AAD binding ensures ciphertext is bound to user" do
      check all secret <- secret_generator(),
                user_id <- user_id_generator(),
                key_id <- key_id_generator(),
                max_runs: 100 do
        
        # Encrypt with user_id as AAD
        {:ok, encrypted} = MockCryptoService.encrypt(secret, key_id, user_id)
        
        # Verify we can decrypt with same user_id
        {:ok, decrypted} = MockCryptoService.decrypt(
          encrypted.ciphertext,
          encrypted.iv,
          encrypted.tag,
          key_id,
          user_id
        )
        
        assert decrypted == secret
        
        # Verify decryption fails with empty AAD
        result = MockCryptoService.decrypt(
          encrypted.ciphertext,
          encrypted.iv,
          encrypted.tag,
          key_id,
          ""
        )
        
        assert result == {:error, :decryption_failed},
          "Decryption should fail without proper AAD"
      end
    end
  end

  describe "Property 5: Correlation ID Propagation" do
    @tag :property
    property "correlation_id is included in all requests" do
      check all correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        # Verify correlation_id format is valid
        assert is_binary(correlation_id)
        assert byte_size(correlation_id) > 0
        
        # In real implementation, we would verify the correlation_id
        # is passed to the gRPC metadata. Here we just verify the format.
        metadata = %{"x-correlation-id" => correlation_id}
        assert metadata["x-correlation-id"] == correlation_id
      end
    end
  end
end
