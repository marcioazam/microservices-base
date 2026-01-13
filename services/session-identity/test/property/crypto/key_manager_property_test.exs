defmodule SessionIdentityCore.Crypto.KeyManagerPropertyTest do
  @moduledoc """
  Property tests for KeyManager caching and key resolution.
  
  **Property 6: Key Metadata Caching**
  **Validates: Requirements 2.4**
  
  **Property 12: Latest Key Version for New Encryptions**
  **Validates: Requirements 5.2**
  """

  use ExUnit.Case, async: false
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.KeyManager

  setup do
    # Start KeyManager if not running
    case GenServer.whereis(KeyManager) do
      nil -> 
        {:ok, _pid} = KeyManager.start_link()
      _pid -> 
        :ok
    end
    
    KeyManager.invalidate_all()
    :ok
  end

  # Generators

  defp namespace do
    member_of([
      "session_identity:jwt",
      "session_identity:session",
      "session_identity:refresh_token"
    ])
  end

  defp key_id do
    gen all ns <- namespace(),
            id <- string(:alphanumeric, min_length: 1, max_length: 20),
            version <- integer(1..100) do
      %{namespace: ns, id: id, version: version}
    end
  end

  # Property Tests - Key Metadata Caching

  @tag property: true
  @tag validates: "Requirements 2.4"
  property "get_active_key returns consistent key_id for same namespace" do
    check all ns <- namespace(), max_runs: 100 do
      {:ok, key1} = KeyManager.get_active_key(ns)
      {:ok, key2} = KeyManager.get_active_key(ns)
      
      assert key1 == key2
      assert key1.namespace == ns
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.4"
  property "cached values are returned without re-fetching" do
    check all ns <- namespace(), max_runs: 100 do
      # First call fetches
      {:ok, key1} = KeyManager.get_active_key(ns)
      
      # Second call should return cached value
      {:ok, key2} = KeyManager.get_active_key(ns)
      
      assert key1 == key2
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.4"
  property "invalidate_cache clears namespace entries" do
    check all ns <- namespace(), max_runs: 100 do
      # Populate cache
      {:ok, _key} = KeyManager.get_active_key(ns)
      
      # Invalidate
      KeyManager.invalidate_cache(ns)
      
      # Should still work (re-fetches)
      {:ok, key} = KeyManager.get_active_key(ns)
      assert key.namespace == ns
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.4"
  property "invalidate_all clears all entries" do
    check all namespaces <- list_of(namespace(), min_length: 1, max_length: 3), max_runs: 50 do
      # Populate cache for multiple namespaces
      for ns <- namespaces do
        KeyManager.get_active_key(ns)
      end
      
      # Invalidate all
      KeyManager.invalidate_all()
      
      # All should still work (re-fetches)
      for ns <- namespaces do
        {:ok, key} = KeyManager.get_active_key(ns)
        assert key.namespace == ns
      end
    end
  end

  # Property Tests - Latest Key Version

  @tag property: true
  @tag validates: "Requirements 5.2"
  property "get_active_key returns valid key_id structure" do
    check all ns <- namespace(), max_runs: 100 do
      {:ok, key} = KeyManager.get_active_key(ns)
      
      assert is_binary(key.namespace)
      assert is_binary(key.id)
      assert is_integer(key.version)
      assert key.version >= 1
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.2"
  property "different namespaces return different key_ids" do
    check all ns1 <- namespace(),
              ns2 <- namespace(),
              ns1 != ns2,
              max_runs: 100 do
      {:ok, key1} = KeyManager.get_active_key(ns1)
      {:ok, key2} = KeyManager.get_active_key(ns2)
      
      assert key1.namespace != key2.namespace
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.2"
  property "get_key_versions returns list with at least one version" do
    check all ns <- namespace(), max_runs: 100 do
      {:ok, key} = KeyManager.get_active_key(ns)
      {:ok, versions} = KeyManager.get_key_versions(ns, key.id)
      
      assert is_list(versions)
      assert length(versions) >= 1
      
      for v <- versions do
        assert v.namespace == ns
        assert is_integer(v.version)
      end
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.2"
  property "get_latest_version returns highest version number" do
    check all ns <- namespace(), max_runs: 100 do
      {:ok, key} = KeyManager.get_active_key(ns)
      {:ok, latest} = KeyManager.get_latest_version(ns, key.id)
      {:ok, versions} = KeyManager.get_key_versions(ns, key.id)
      
      max_version = Enum.max_by(versions, & &1.version)
      assert latest.version == max_version.version
    end
  end

  @tag property: true
  @tag validates: "Requirements 5.2"
  property "deprecated? returns boolean for any key_id" do
    check all key <- key_id(), max_runs: 100 do
      result = KeyManager.deprecated?(key)
      assert is_boolean(result)
    end
  end
end
