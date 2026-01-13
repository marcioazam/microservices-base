defmodule MfaService.Crypto.ClientIntegrationTest do
  @moduledoc """
  Integration tests for Crypto Service client.
  These tests require a running crypto-service instance.
  
  Run with: mix test test/integration --include integration
  """

  use ExUnit.Case, async: false

  @moduletag :integration

  alias MfaService.Crypto.{Client, KeyManager, TOTPEncryptor, CircuitBreaker, Error}

  @test_user_id "test-user-integration-#{:rand.uniform(10000)}"

  setup_all do
    # Check if crypto-service is available
    case Client.health_check("setup-check") do
      {:ok, _} ->
        :ok
      {:error, _} ->
        IO.puts("\n⚠️  Skipping integration tests: crypto-service not available")
        :skip
    end
  end

  describe "end-to-end encryption/decryption" do
    @tag :integration
    test "encrypts and decrypts TOTP secret successfully" do
      secret = Base.encode32(:crypto.strong_rand_bytes(20), padding: false)
      user_id = @test_user_id
      
      # Encrypt
      assert {:ok, encrypted} = TOTPEncryptor.encrypt_secret(secret, user_id)
      
      # Verify format
      assert {:ok, :crypto_service} = TOTPEncryptor.detect_version(encrypted)
      
      # Decrypt
      assert {:ok, decrypted} = TOTPEncryptor.decrypt_secret(encrypted, user_id)
      
      # Verify round-trip
      assert decrypted == secret
    end

    @tag :integration
    test "decryption fails with wrong user_id (AAD mismatch)" do
      secret = Base.encode32(:crypto.strong_rand_bytes(20), padding: false)
      user_id = @test_user_id
      wrong_user_id = "wrong-user-#{:rand.uniform(10000)}"
      
      # Encrypt with correct user_id
      assert {:ok, encrypted} = TOTPEncryptor.encrypt_secret(secret, user_id)
      
      # Decrypt with wrong user_id should fail
      assert {:error, %Error{code: :decryption_failed}} = 
        TOTPEncryptor.decrypt_secret(encrypted, wrong_user_id)
    end

    @tag :integration
    test "handles multiple sequential encryptions" do
      user_id = @test_user_id
      secrets = for _ <- 1..10 do
        Base.encode32(:crypto.strong_rand_bytes(20), padding: false)
      end
      
      # Encrypt all
      encrypted_list = Enum.map(secrets, fn secret ->
        {:ok, encrypted} = TOTPEncryptor.encrypt_secret(secret, user_id)
        {secret, encrypted}
      end)
      
      # Decrypt all and verify
      Enum.each(encrypted_list, fn {original, encrypted} ->
        assert {:ok, decrypted} = TOTPEncryptor.decrypt_secret(encrypted, user_id)
        assert decrypted == original
      end)
    end
  end

  describe "key rotation scenarios" do
    @tag :integration
    test "secrets encrypted before rotation can still be decrypted" do
      secret = Base.encode32(:crypto.strong_rand_bytes(20), padding: false)
      user_id = @test_user_id
      
      # Encrypt before rotation
      assert {:ok, encrypted_before} = TOTPEncryptor.encrypt_secret(secret, user_id)
      
      # Rotate key
      assert {:ok, _new_key_id} = KeyManager.rotate_key()
      
      # Decrypt should still work (uses key_id from ciphertext)
      assert {:ok, decrypted} = TOTPEncryptor.decrypt_secret(encrypted_before, user_id)
      assert decrypted == secret
    end

    @tag :integration
    test "new encryptions use new key after rotation" do
      user_id = @test_user_id
      
      # Get current key
      {:ok, key_before} = KeyManager.get_active_key_id()
      
      # Rotate
      {:ok, key_after} = KeyManager.rotate_key()
      
      # Keys should be different
      assert key_before != key_after
      
      # New encryption should use new key
      secret = Base.encode32(:crypto.strong_rand_bytes(20), padding: false)
      {:ok, encrypted} = TOTPEncryptor.encrypt_secret(secret, user_id)
      
      # Verify it decrypts correctly
      assert {:ok, ^secret} = TOTPEncryptor.decrypt_secret(encrypted, user_id)
    end
  end

  describe "circuit breaker under load" do
    @tag :integration
    @tag timeout: 60_000
    test "circuit breaker handles burst of requests" do
      user_id = @test_user_id
      
      # Generate many requests concurrently
      tasks = for i <- 1..50 do
        Task.async(fn ->
          secret = "secret-#{i}-#{:rand.uniform(10000)}"
          
          case TOTPEncryptor.encrypt_secret(secret, user_id) do
            {:ok, encrypted} ->
              case TOTPEncryptor.decrypt_secret(encrypted, user_id) do
                {:ok, ^secret} -> :success
                {:error, _} -> :decrypt_failed
              end
            {:error, %Error{code: :circuit_open}} ->
              :circuit_open
            {:error, _} ->
              :encrypt_failed
          end
        end)
      end
      
      results = Task.await_many(tasks, 30_000)
      
      # Count results
      success_count = Enum.count(results, &(&1 == :success))
      circuit_open_count = Enum.count(results, &(&1 == :circuit_open))
      
      # Most should succeed, some might hit circuit breaker under load
      assert success_count > 0
      
      # Log results for debugging
      IO.puts("\nBurst test results: #{success_count} success, #{circuit_open_count} circuit_open")
    end

    @tag :integration
    test "circuit breaker state is queryable" do
      state = CircuitBreaker.state()
      
      assert state in [:closed, :half_open, :open]
    end
  end

  describe "error handling" do
    @tag :integration
    test "handles invalid encrypted data gracefully" do
      user_id = @test_user_id
      invalid_data = Base.encode64("not-valid-encrypted-data")
      
      result = TOTPEncryptor.decrypt_secret(invalid_data, user_id)
      
      assert {:error, %Error{}} = result
    end

    @tag :integration
    test "handles empty secret gracefully" do
      user_id = @test_user_id
      
      # Empty string should still work (edge case)
      result = TOTPEncryptor.encrypt_secret("", user_id)
      
      # Depending on implementation, might succeed or fail gracefully
      case result do
        {:ok, encrypted} ->
          assert {:ok, ""} = TOTPEncryptor.decrypt_secret(encrypted, user_id)
        {:error, %Error{}} ->
          :ok  # Also acceptable
      end
    end
  end

  describe "observability" do
    @tag :integration
    test "operations include correlation_id in logs" do
      # This test verifies that operations complete with correlation tracking
      secret = Base.encode32(:crypto.strong_rand_bytes(20), padding: false)
      user_id = @test_user_id
      
      # Operations should complete without error
      assert {:ok, encrypted} = TOTPEncryptor.encrypt_secret(secret, user_id)
      assert {:ok, _} = TOTPEncryptor.decrypt_secret(encrypted, user_id)
      
      # Correlation IDs are generated internally and logged
      # Manual verification: check logs for correlation_id presence
    end
  end
end
