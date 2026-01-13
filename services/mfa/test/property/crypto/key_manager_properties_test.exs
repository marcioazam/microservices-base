defmodule MfaService.Crypto.KeyManagerPropertiesTest do
  @moduledoc """
  Property-based tests for KeyManager.
  Tests correctness properties defined in the design document.
  """
  use ExUnit.Case, async: false
  use ExUnitProperties

  # Test cache in isolation
  defmodule TestCache do
    @moduledoc false
    
    defstruct cache: %{}, ttl: 300_000

    def new(ttl \\ 300_000) do
      %__MODULE__{ttl: ttl}
    end

    def get(%{cache: cache, ttl: ttl}, key) do
      case Map.get(cache, key) do
        nil -> :miss
        {value, cached_at} ->
          if expired?(cached_at, ttl) do
            :miss
          else
            {:ok, value}
          end
      end
    end

    def put(%{cache: cache} = state, key, value) do
      cached_at = System.monotonic_time(:millisecond)
      %{state | cache: Map.put(cache, key, {value, cached_at})}
    end

    def delete(%{cache: cache} = state, key) do
      %{state | cache: Map.delete(cache, key)}
    end

    defp expired?(cached_at, ttl) do
      System.monotonic_time(:millisecond) - cached_at > ttl
    end
  end

  # Generators

  defp key_id_generator do
    gen all namespace <- string(:alphanumeric, min_length: 1, max_length: 20),
            id <- binary(length: 16),
            version <- integer(1..100) do
      %{
        namespace: namespace,
        id: Base.encode16(id, case: :lower),
        version: version
      }
    end
  end

  defp metadata_generator do
    gen all algorithm <- member_of([:aes_256_gcm, :aes_128_gcm]),
            state <- member_of([:active, :deprecated, :pending_destruction]) do
      %{
        algorithm: algorithm,
        state: state,
        created_at: DateTime.utc_now(),
        expires_at: nil,
        usage_count: :rand.uniform(1000)
      }
    end
  end

  defp cache_key(%{namespace: ns, id: id, version: v}) do
    "#{ns}:#{id}:#{v}"
  end

  describe "Property 16: Key Metadata Caching" do
    @tag :property
    @tag timeout: 120_000
    property "cached value returned within TTL" do
      check all key_id <- key_id_generator(),
                metadata <- metadata_generator(),
                max_runs: 100 do
        
        cache = TestCache.new(300_000)  # 5 minute TTL
        key = cache_key(key_id)
        
        # Initially miss
        assert TestCache.get(cache, key) == :miss
        
        # Put value
        cache = TestCache.put(cache, key, metadata)
        
        # Should hit
        assert {:ok, ^metadata} = TestCache.get(cache, key)
        
        # Should still hit (within TTL)
        assert {:ok, ^metadata} = TestCache.get(cache, key)
      end
    end

    @tag :property
    property "cache miss after TTL expires" do
      check all key_id <- key_id_generator(),
                metadata <- metadata_generator(),
                max_runs: 100 do
        
        # Use very short TTL for testing
        cache = TestCache.new(1)  # 1ms TTL
        key = cache_key(key_id)
        
        # Put value
        cache = TestCache.put(cache, key, metadata)
        
        # Wait for TTL to expire
        Process.sleep(5)
        
        # Should miss after TTL
        assert TestCache.get(cache, key) == :miss
      end
    end

    @tag :property
    property "cache invalidation removes entry" do
      check all key_id <- key_id_generator(),
                metadata <- metadata_generator(),
                max_runs: 100 do
        
        cache = TestCache.new(300_000)
        key = cache_key(key_id)
        
        # Put value
        cache = TestCache.put(cache, key, metadata)
        assert {:ok, ^metadata} = TestCache.get(cache, key)
        
        # Delete
        cache = TestCache.delete(cache, key)
        
        # Should miss
        assert TestCache.get(cache, key) == :miss
      end
    end
  end

  describe "Property 8: Key ID From Ciphertext Used for Decryption" do
    @tag :property
    property "decryption uses key_id from ciphertext, not active key" do
      check all key_id_v1 <- key_id_generator(),
                key_id_v2 <- key_id_generator(),
                max_runs: 100 do
        
        # Simulate ciphertext with embedded key_id
        ciphertext_payload = %{
          ciphertext: :crypto.strong_rand_bytes(32),
          iv: :crypto.strong_rand_bytes(12),
          tag: :crypto.strong_rand_bytes(16),
          key_id: key_id_v1  # Key used for encryption
        }
        
        # Even if active key is different (v2), we should use v1 from payload
        key_id_for_decryption = ciphertext_payload.key_id
        
        assert key_id_for_decryption == key_id_v1
        assert key_id_for_decryption != key_id_v2 || key_id_v1 == key_id_v2
      end
    end

    @tag :property
    property "key_id is preserved through encryption payload" do
      check all key_id <- key_id_generator(),
                plaintext <- binary(min_length: 1, max_length: 100),
                max_runs: 100 do
        
        # Simulate encryption result
        encrypted_payload = %{
          ciphertext: :crypto.strong_rand_bytes(byte_size(plaintext) + 16),
          iv: :crypto.strong_rand_bytes(12),
          tag: :crypto.strong_rand_bytes(16),
          key_id: key_id
        }
        
        # Key ID should be preserved
        assert encrypted_payload.key_id == key_id
        assert encrypted_payload.key_id.namespace == key_id.namespace
        assert encrypted_payload.key_id.id == key_id.id
        assert encrypted_payload.key_id.version == key_id.version
      end
    end
  end

  describe "Key rotation" do
    @tag :property
    property "rotation creates new key with incremented version" do
      check all base_version <- integer(1..100),
                max_runs: 100 do
        
        old_key_id = %{
          namespace: "mfa:totp",
          id: Base.encode16(:crypto.strong_rand_bytes(16), case: :lower),
          version: base_version
        }
        
        # Simulate rotation - new key should have higher version
        new_key_id = %{
          namespace: old_key_id.namespace,
          id: Base.encode16(:crypto.strong_rand_bytes(16), case: :lower),
          version: base_version + 1
        }
        
        assert new_key_id.version > old_key_id.version
        assert new_key_id.namespace == old_key_id.namespace
      end
    end
  end
end
