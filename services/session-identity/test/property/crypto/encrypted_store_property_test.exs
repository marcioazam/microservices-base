defmodule SessionIdentityCore.Crypto.EncryptedStorePropertyTest do
  @moduledoc """
  Property tests for EncryptedStore.
  
  **Property 7: Session Data Encryption Round-Trip**
  **Validates: Requirements 3.1, 3.2**
  
  **Property 8: Key Namespace Isolation**
  **Validates: Requirements 3.3, 4.3**
  
  **Property 9: AAD Binding Integrity**
  **Validates: Requirements 3.4, 4.4**
  
  **Property 11: Refresh Token Encryption Round-Trip**
  **Validates: Requirements 4.1, 4.2**
  """

  use ExUnit.Case, async: false
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.{EncryptedStore, Fallback}

  setup do
    # Configure to use fallback for testing
    Application.put_env(:session_identity_core, :crypto_config, %{
      enabled: false,
      fallback_enabled: true,
      session_key_namespace: "session_identity:session",
      refresh_token_key_namespace: "session_identity:refresh_token"
    })
    
    :ok
  end

  # Generators

  defp session_data do
    gen all data <- binary(min_length: 1, max_length: 1000) do
      data
    end
  end

  defp session_id do
    gen all id <- string(:alphanumeric, min_length: 10, max_length: 50) do
      id
    end
  end

  defp user_id do
    gen all id <- string(:alphanumeric, min_length: 10, max_length: 50) do
      id
    end
  end

  defp client_id do
    gen all id <- string(:alphanumeric, min_length: 5, max_length: 30) do
      id
    end
  end

  defp namespace do
    member_of([:session, :refresh_token])
  end

  # Property Tests - Session Encryption Round-Trip

  @tag property: true
  @tag validates: "Requirements 3.1, 3.2"
  property "session data encryption round-trip preserves data (fallback mode)" do
    check all data <- session_data(),
              sid <- session_id(),
              max_runs: 100 do
      aad = EncryptedStore.build_session_aad(sid)
      
      {:ok, envelope} = EncryptedStore.encrypt(:session, data, aad)
      {:ok, decrypted} = EncryptedStore.decrypt(:session, envelope, aad)
      
      assert decrypted == data
    end
  end

  # Property Tests - Refresh Token Round-Trip

  @tag property: true
  @tag validates: "Requirements 4.1, 4.2"
  property "refresh token encryption round-trip preserves data (fallback mode)" do
    check all data <- session_data(),
              uid <- user_id(),
              cid <- client_id(),
              max_runs: 100 do
      aad = EncryptedStore.build_refresh_token_aad(uid, cid)
      
      {:ok, envelope} = EncryptedStore.encrypt(:refresh_token, data, aad)
      {:ok, decrypted} = EncryptedStore.decrypt(:refresh_token, envelope, aad)
      
      assert decrypted == data
    end
  end

  # Property Tests - Key Namespace Isolation

  @tag property: true
  @tag validates: "Requirements 3.3, 4.3"
  property "session namespace returns correct key namespace" do
    check all _ <- constant(:ok), max_runs: 100 do
      namespace = EncryptedStore.get_namespace(:session)
      assert namespace == "session_identity:session"
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.3, 4.3"
  property "refresh_token namespace returns correct key namespace" do
    check all _ <- constant(:ok), max_runs: 100 do
      namespace = EncryptedStore.get_namespace(:refresh_token)
      assert namespace == "session_identity:refresh_token"
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.3, 4.3"
  property "different namespaces produce different key namespaces" do
    check all _ <- constant(:ok), max_runs: 100 do
      session_ns = EncryptedStore.get_namespace(:session)
      refresh_ns = EncryptedStore.get_namespace(:refresh_token)
      
      assert session_ns != refresh_ns
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.3, 4.3"
  property "envelope contains correct namespace in key_id" do
    check all data <- session_data(),
              ns <- namespace(),
              max_runs: 100 do
      aad = case ns do
        :session -> EncryptedStore.build_session_aad("test-session")
        :refresh_token -> EncryptedStore.build_refresh_token_aad("user", "client")
      end
      
      {:ok, envelope} = EncryptedStore.encrypt(ns, data, aad)
      
      expected_namespace = EncryptedStore.get_namespace(ns)
      assert envelope["key_id"]["namespace"] == expected_namespace
    end
  end

  # Property Tests - AAD Binding

  @tag property: true
  @tag validates: "Requirements 3.4, 4.4"
  property "session AAD includes session_id" do
    check all sid <- session_id(), max_runs: 100 do
      aad = EncryptedStore.build_session_aad(sid)
      
      assert String.contains?(aad, sid)
      assert String.starts_with?(aad, "session:")
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.4, 4.4"
  property "refresh token AAD includes user_id and client_id" do
    check all uid <- user_id(),
              cid <- client_id(),
              max_runs: 100 do
      aad = EncryptedStore.build_refresh_token_aad(uid, cid)
      
      assert String.contains?(aad, uid)
      assert String.contains?(aad, cid)
      assert String.starts_with?(aad, "refresh_token:")
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.4, 4.4"
  property "different session_ids produce different AADs" do
    check all sid1 <- session_id(),
              sid2 <- session_id(),
              sid1 != sid2,
              max_runs: 100 do
      aad1 = EncryptedStore.build_session_aad(sid1)
      aad2 = EncryptedStore.build_session_aad(sid2)
      
      assert aad1 != aad2
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.4, 4.4"
  property "different user_ids produce different refresh token AADs" do
    check all uid1 <- user_id(),
              uid2 <- user_id(),
              cid <- client_id(),
              uid1 != uid2,
              max_runs: 100 do
      aad1 = EncryptedStore.build_refresh_token_aad(uid1, cid)
      aad2 = EncryptedStore.build_refresh_token_aad(uid2, cid)
      
      assert aad1 != aad2
    end
  end

  # Property Tests - Envelope Structure

  @tag property: true
  @tag validates: "Requirements 3.1, 4.1"
  property "envelope contains all required fields" do
    check all data <- session_data(),
              sid <- session_id(),
              max_runs: 100 do
      aad = EncryptedStore.build_session_aad(sid)
      {:ok, envelope} = EncryptedStore.encrypt(:session, data, aad)
      
      assert Map.has_key?(envelope, "v")
      assert Map.has_key?(envelope, "key_id")
      assert Map.has_key?(envelope, "iv")
      assert Map.has_key?(envelope, "tag")
      assert Map.has_key?(envelope, "ciphertext")
      assert Map.has_key?(envelope, "encrypted_at")
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.1, 4.1"
  property "envelope version is 1" do
    check all data <- session_data(),
              ns <- namespace(),
              max_runs: 100 do
      aad = case ns do
        :session -> EncryptedStore.build_session_aad("test")
        :refresh_token -> EncryptedStore.build_refresh_token_aad("user", "client")
      end
      
      {:ok, envelope} = EncryptedStore.encrypt(ns, data, aad)
      
      assert envelope["v"] == 1
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.1, 4.1"
  property "encrypted_at is a valid unix timestamp" do
    check all data <- session_data(),
              sid <- session_id(),
              max_runs: 100 do
      aad = EncryptedStore.build_session_aad(sid)
      {:ok, envelope} = EncryptedStore.encrypt(:session, data, aad)
      
      encrypted_at = envelope["encrypted_at"]
      assert is_integer(encrypted_at)
      assert encrypted_at > 0
      
      # Should be within last minute
      now = DateTime.to_unix(DateTime.utc_now())
      assert encrypted_at <= now
      assert encrypted_at >= now - 60
    end
  end
end
