defmodule MfaService.Crypto.MigrationPropertiesTest do
  @moduledoc """
  Property-based tests for TOTP secret migration.
  Validates that migration from local encryption to crypto-service preserves values.
  """

  use ExUnit.Case, async: false
  use ExUnitProperties

  alias MfaService.Crypto.{TOTPEncryptor, SecretFormat, Client, Error}

  import Mox

  setup :verify_on_exit!

  @local_key :crypto.strong_rand_bytes(32)

  # Generators

  defp totp_secret_generator do
    gen all length <- integer(16..32),
            bytes <- binary(length: length) do
      Base.encode32(bytes, padding: false)
    end
  end

  defp user_id_generator do
    gen all prefix <- string(:alphanumeric, length: 8),
            suffix <- string(:alphanumeric, length: 24) do
      "user_#{prefix}_#{suffix}"
    end
  end

  # Helper to create v1 (local) encrypted secret
  defp create_v1_encrypted(secret, _user_id) do
    iv = :crypto.strong_rand_bytes(12)
    {ciphertext, tag} = :crypto.crypto_one_time_aead(
      :aes_256_gcm, @local_key, iv, secret, "", true
    )
    
    payload = SecretFormat.encode_v1(iv, tag, ciphertext)
    Base.encode64(payload)
  end

  describe "Property 2: Migration Preserves Value" do
    @tag :property
    property "migrated secrets decrypt to original value" do
      check all secret <- totp_secret_generator(),
                user_id <- user_id_generator(),
                max_runs: 100 do
        
        # Create v1 encrypted secret
        v1_encrypted = create_v1_encrypted(secret, user_id)
        
        # Mock crypto-service for encryption
        mock_key_id = "key-#{:rand.uniform(1000)}"
        mock_iv = :crypto.strong_rand_bytes(12)
        mock_tag = :crypto.strong_rand_bytes(16)
        mock_ciphertext = :crypto.strong_rand_bytes(byte_size(secret) + 16)
        
        # Setup mocks for migration
        MfaService.Crypto.ClientMock
        |> expect(:encrypt, fn ^secret, _key_id, ^user_id, _corr_id ->
          {:ok, %{
            key_id: mock_key_id,
            iv: mock_iv,
            tag: mock_tag,
            ciphertext: mock_ciphertext
          }}
        end)
        
        MfaService.Crypto.KeyManagerMock
        |> expect(:get_active_key_id, fn -> {:ok, mock_key_id} end)
        
        # Set local key for migration
        System.put_env("MFA_LOCAL_ENCRYPTION_KEY", Base.encode64(@local_key))
        
        # Perform migration
        result = TOTPEncryptor.migrate_secret(v1_encrypted, user_id, @local_key)
        
        # Verify migration succeeded
        assert {:ok, v2_encrypted} = result
        
        # Verify new format is v2
        assert {:ok, :crypto_service} = TOTPEncryptor.detect_version(v2_encrypted)
        
        # Mock decryption to return original secret
        MfaService.Crypto.ClientMock
        |> expect(:decrypt, fn _ct, _iv, _tag, ^mock_key_id, ^user_id, _corr_id ->
          {:ok, secret}
        end)
        
        # Verify decryption returns original
        assert {:ok, ^secret} = TOTPEncryptor.decrypt_secret(v2_encrypted, user_id)
        
        # Cleanup
        System.delete_env("MFA_LOCAL_ENCRYPTION_KEY")
      end
    end

    @tag :property
    property "migration fails gracefully with wrong local key" do
      check all secret <- totp_secret_generator(),
                user_id <- user_id_generator(),
                max_runs: 50 do
        
        # Create v1 encrypted secret
        v1_encrypted = create_v1_encrypted(secret, user_id)
        
        # Try migration with wrong key
        wrong_key = :crypto.strong_rand_bytes(32)
        result = TOTPEncryptor.migrate_secret(v1_encrypted, user_id, wrong_key)
        
        # Should fail
        assert {:error, %Error{code: :migration_failed}} = result
      end
    end

    @tag :property
    property "v1 secrets are detected as needing migration" do
      check all secret <- totp_secret_generator(),
                user_id <- user_id_generator(),
                max_runs: 100 do
        
        v1_encrypted = create_v1_encrypted(secret, user_id)
        
        assert TOTPEncryptor.needs_migration?(v1_encrypted)
        assert {:ok, :local} = TOTPEncryptor.detect_version(v1_encrypted)
      end
    end

    @tag :property
    property "v2 secrets do not need migration" do
      check all secret <- totp_secret_generator(),
                user_id <- user_id_generator(),
                max_runs: 100 do
        
        # Create v2 format directly
        key_id = "key-#{:rand.uniform(1000)}"
        iv = :crypto.strong_rand_bytes(12)
        tag = :crypto.strong_rand_bytes(16)
        ciphertext = :crypto.strong_rand_bytes(byte_size(secret))
        
        payload = SecretFormat.encode_v2(key_id, iv, tag, ciphertext)
        v2_encrypted = Base.encode64(payload)
        
        refute TOTPEncryptor.needs_migration?(v2_encrypted)
        assert {:ok, :crypto_service} = TOTPEncryptor.detect_version(v2_encrypted)
      end
    end
  end

  describe "Migration edge cases" do
    @tag :property
    property "migration preserves secret length" do
      check all secret <- totp_secret_generator(),
                user_id <- user_id_generator(),
                max_runs: 50 do
        
        original_length = byte_size(secret)
        v1_encrypted = create_v1_encrypted(secret, user_id)
        
        # Mock for migration
        mock_key_id = "key-test"
        
        MfaService.Crypto.KeyManagerMock
        |> expect(:get_active_key_id, fn -> {:ok, mock_key_id} end)
        
        MfaService.Crypto.ClientMock
        |> expect(:encrypt, fn received_secret, _key_id, ^user_id, _corr_id ->
          # Verify the decrypted secret has correct length
          assert byte_size(received_secret) == original_length
          
          {:ok, %{
            key_id: mock_key_id,
            iv: :crypto.strong_rand_bytes(12),
            tag: :crypto.strong_rand_bytes(16),
            ciphertext: :crypto.strong_rand_bytes(original_length + 16)
          }}
        end)
        
        System.put_env("MFA_LOCAL_ENCRYPTION_KEY", Base.encode64(@local_key))
        
        result = TOTPEncryptor.migrate_secret(v1_encrypted, user_id, @local_key)
        assert {:ok, _} = result
        
        System.delete_env("MFA_LOCAL_ENCRYPTION_KEY")
      end
    end
  end
end
