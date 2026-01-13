defmodule SessionIdentityCore.Crypto.KeyRotationPropertyTest do
  @moduledoc """
  Property tests for key rotation support.
  
  **Property 10: Multi-Version Key Decryption**
  **Validates: Requirements 3.5, 5.1, 5.3**
  
  **Property 13: Re-encryption on Deprecated Key Access**
  **Validates: Requirements 5.5**
  """

  use ExUnit.Case, async: false
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.{KeyRotation, EncryptedStore}

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
    gen all data <- binary(min_length: 1, max_length: 500) do
      data
    end
  end

  defp session_id do
    gen all id <- string(:alphanumeric, min_length: 10, max_length: 30) do
      id
    end
  end

  defp key_version do
    integer(1..100)
  end

  defp valid_envelope do
    gen all data <- session_data(),
            sid <- session_id() do
      aad = EncryptedStore.build_session_aad(sid)
      {:ok, envelope} = EncryptedStore.encrypt(:session, data, aad)
      {envelope, data, aad}
    end
  end

  # Property Tests - Multi-Version Decryption

  @tag property: true
  @tag validates: "Requirements 3.5, 5.1, 5.3"
  property "get_key_version extracts version from valid envelope" do
    check all {envelope, _data, _aad} <- valid_envelope(), max_runs: 100 do
      {:ok, version} = KeyRotation.get_key_version(envelope)
      
      assert is_integer(version)
      assert version >= 0
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.5, 5.1, 5.3"
  property "encrypted_with_version? returns true for matching version" do
    check all {envelope, _data, _aad} <- valid_envelope(), max_runs: 100 do
      {:ok, version} = KeyRotation.get_key_version(envelope)
      
      assert KeyRotation.encrypted_with_version?(envelope, version)
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.5, 5.1, 5.3"
  property "encrypted_with_version? returns false for non-matching version" do
    check all {envelope, _data, _aad} <- valid_envelope(),
              wrong_version <- key_version(),
              max_runs: 100 do
      {:ok, actual_version} = KeyRotation.get_key_version(envelope)
      
      if wrong_version != actual_version do
        refute KeyRotation.encrypted_with_version?(envelope, wrong_version)
      end
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.5, 5.1, 5.3"
  property "decrypt_and_maybe_reencrypt returns plaintext" do
    check all {envelope, original_data, aad} <- valid_envelope(), max_runs: 100 do
      {:ok, plaintext, _new_envelope} = KeyRotation.decrypt_and_maybe_reencrypt(:session, envelope, aad)
      
      assert plaintext == original_data
    end
  end

  @tag property: true
  @tag validates: "Requirements 3.5, 5.1, 5.3"
  property "decrypt_and_maybe_reencrypt returns nil new_envelope when key not deprecated" do
    check all {envelope, _data, aad} <- valid_envelope(), max_runs: 100 do
      # In fallback mode, keys are never deprecated
      {:ok, _plaintext, new_envelope} = KeyRotation.decrypt_and_maybe_reencrypt(:session, envelope, aad)
      
      assert is_nil(new_envelope)
    end
  end

  # Property Tests - Re-encryption

  @tag property: true
  @tag validates: "Requirements 5.5"
  property "reencrypt produces valid envelope" do
    check all data <- session_data(),
              sid <- session_id(),
              max_runs: 100 do
      aad = EncryptedStore.build_session_aad(sid)
      {:ok, envelope} = KeyRotation.reencrypt(:session, data, aad)
      
      assert Map.has_key?(envelope, "v")
      assert Map.has_key?(envelope, "key_id")
      assert Map.has_key?(envelope, "ciphertext")
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.5"
  property "reencrypted data can be decrypted" do
    check all data <- session_data(),
              sid <- session_id(),
              max_runs: 100 do
      aad = EncryptedStore.build_session_aad(sid)
      {:ok, envelope} = KeyRotation.reencrypt(:session, data, aad)
      {:ok, decrypted} = EncryptedStore.decrypt(:session, envelope, aad)
      
      assert decrypted == data
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.5"
  property "needs_reencryption? returns boolean for any envelope" do
    check all {envelope, _data, _aad} <- valid_envelope(), max_runs: 100 do
      result = KeyRotation.needs_reencryption?(envelope)
      assert is_boolean(result)
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.5"
  property "needs_reencryption? returns false for invalid envelope" do
    check all invalid <- one_of([
              constant(%{}),
              constant(%{"v" => 1}),
              constant(nil)
            ]),
            max_runs: 100 do
      refute KeyRotation.needs_reencryption?(invalid)
    end
  end

  # Property Tests - Key Version Logging

  @tag property: true
  @tag validates: "Requirements 5.4"
  property "log_key_usage does not raise for valid envelope" do
    check all {envelope, _data, _aad} <- valid_envelope(), max_runs: 100 do
      # Should not raise
      assert :ok = KeyRotation.log_key_usage(:encrypt, envelope)
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.4"
  property "log_key_usage handles invalid envelope gracefully" do
    check all invalid <- one_of([
              constant(%{}),
              constant(%{"v" => 1}),
              constant(nil)
            ]),
            max_runs: 100 do
      # Should not raise
      assert :ok = KeyRotation.log_key_usage(:encrypt, invalid)
    end
  end
end
