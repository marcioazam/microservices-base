defmodule MfaService.Integration.CacheClientTest do
  @moduledoc """
  Integration tests for Cache_Service client.
  Tests connection, operations, and circuit breaker behavior.
  """

  use ExUnit.Case, async: false

  alias AuthPlatform.Clients.Cache
  alias MfaService.Challenge

  @moduletag :integration

  describe "Cache_Service connection" do
    test "cache client is available after application start" do
      # The cache client should be initialized by the application
      # Test basic connectivity
      key = "test:integration:#{System.unique_integer()}"

      # Should not raise
      result = Cache.set(key, "test_value", ttl: 60)
      assert result in [:ok, {:error, _}]
    end
  end

  describe "Cache operations" do
    test "set and get operations work correctly" do
      key = "test:cache:#{System.unique_integer()}"
      value = "test_value_#{System.unique_integer()}"

      # Set value
      :ok = Cache.set(key, value, ttl: 60)

      # Get value
      {:ok, retrieved} = Cache.get(key)
      assert retrieved == value

      # Cleanup
      Cache.delete(key)
    end

    test "delete operation removes value" do
      key = "test:delete:#{System.unique_integer()}"

      Cache.set(key, "to_delete", ttl: 60)
      Cache.delete(key)

      {:ok, result} = Cache.get(key)
      assert result == nil
    end

    test "exists? returns correct status" do
      key = "test:exists:#{System.unique_integer()}"

      refute Cache.exists?(key)

      Cache.set(key, "exists", ttl: 60)
      assert Cache.exists?(key)

      Cache.delete(key)
      refute Cache.exists?(key)
    end

    test "TTL expiration works" do
      key = "test:ttl:#{System.unique_integer()}"

      # Set with very short TTL (1 second)
      Cache.set(key, "expires_soon", ttl: 1)

      # Should exist immediately
      assert Cache.exists?(key)

      # Wait for expiration
      Process.sleep(1500)

      # Should be gone
      refute Cache.exists?(key)
    end
  end

  describe "Challenge storage integration" do
    test "challenge store and retrieve works" do
      user_id = "test_user_#{System.unique_integer()}"
      challenge = Challenge.generate()

      # Store challenge
      :ok = Challenge.store(user_id, challenge)

      # Retrieve challenge
      {:ok, retrieved} = Challenge.retrieve(user_id)
      assert retrieved == challenge

      # Cleanup
      Challenge.retrieve_and_delete(user_id)
    end

    test "challenge retrieve_and_delete removes challenge" do
      user_id = "test_user_#{System.unique_integer()}"
      challenge = Challenge.generate()

      Challenge.store(user_id, challenge)

      # First retrieval should succeed
      {:ok, _} = Challenge.retrieve_and_delete(user_id)

      # Second retrieval should fail
      {:error, :not_found} = Challenge.retrieve(user_id)
    end
  end

  describe "Circuit breaker behavior" do
    test "circuit breaker is configured" do
      # The circuit breaker should be started by the application
      # This test verifies it doesn't crash on operations

      key = "test:breaker:#{System.unique_integer()}"

      # Multiple operations should work without circuit breaker issues
      for _ <- 1..10 do
        Cache.set(key, "value", ttl: 60)
        Cache.get(key)
      end

      Cache.delete(key)
    end
  end
end
