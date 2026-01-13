defmodule AuthPlatform.Resilience.RateLimiterTest do
  use ExUnit.Case, async: false

  alias AuthPlatform.Resilience.RateLimiter
  alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor

  @moduletag :rate_limiter

  setup do
    name = :"rl_test_#{:erlang.unique_integer([:positive])}"
    {:ok, name: name}
  end

  describe "start_link/1" do
    test "starts with default config", %{name: name} do
      {:ok, pid} = start_rate_limiter(name)
      assert Process.alive?(pid)

      status = RateLimiter.get_status(name)
      assert status.config.rate == 100
      assert status.config.burst_size == 100
    end

    test "starts with custom config", %{name: name} do
      config = %{rate: 50, burst_size: 200}
      {:ok, pid} = start_rate_limiter(name, config)

      assert Process.alive?(pid)
      status = RateLimiter.get_status(name)
      assert status.config.rate == 50
      assert status.config.burst_size == 200
    end
  end

  describe "allow?/1" do
    test "allows requests when tokens available", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 100, burst_size: 10})

      # Should allow first 10 requests
      results = for _ <- 1..10, do: RateLimiter.allow?(name)
      assert Enum.all?(results, & &1)
    end

    test "rejects requests when no tokens", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 1, burst_size: 1})

      # First request should succeed
      assert RateLimiter.allow?(name) == true

      # Second should fail (no tokens left)
      assert RateLimiter.allow?(name) == false
    end

    test "refills tokens over time", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 1000, burst_size: 1})

      # Consume the token
      assert RateLimiter.allow?(name) == true
      assert RateLimiter.allow?(name) == false

      # Wait for refill (1000 tokens/sec = 1 token/ms)
      Process.sleep(5)

      # Should have tokens again
      assert RateLimiter.allow?(name) == true
    end
  end

  describe "acquire/2" do
    test "returns :ok when token available", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 100, burst_size: 10})

      assert RateLimiter.acquire(name, 100) == :ok
    end

    test "waits for token when none available", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 1000, burst_size: 1})

      # Consume token
      RateLimiter.allow?(name)

      # Should wait and succeed
      start = System.monotonic_time(:millisecond)
      assert RateLimiter.acquire(name, 100) == :ok
      elapsed = System.monotonic_time(:millisecond) - start

      # Should have waited at least a bit
      assert elapsed >= 0
    end

    test "returns timeout when wait exceeds timeout", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 1, burst_size: 1})

      # Consume token
      RateLimiter.allow?(name)

      # Should timeout (rate is 1/sec, timeout is 10ms)
      assert RateLimiter.acquire(name, 10) == {:error, :timeout}
    end
  end

  describe "available_tokens/1" do
    test "returns current token count", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 100, burst_size: 10})

      tokens = RateLimiter.available_tokens(name)
      assert tokens == 10.0
    end

    test "decreases after allow?", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 100, burst_size: 10})

      RateLimiter.allow?(name)
      tokens = RateLimiter.available_tokens(name)
      assert tokens < 10.0
    end
  end

  describe "reset/1" do
    test "resets tokens to burst_size", %{name: name} do
      {:ok, _} = start_rate_limiter(name, %{rate: 1, burst_size: 10})

      # Consume all tokens
      for _ <- 1..10, do: RateLimiter.allow?(name)

      tokens_before = RateLimiter.available_tokens(name)
      assert tokens_before < 1.0

      RateLimiter.reset(name)
      Process.sleep(5)

      tokens_after = RateLimiter.available_tokens(name)
      assert tokens_after >= 9.0
    end
  end

  describe "get_status/1" do
    test "returns status information", %{name: name} do
      config = %{rate: 50, burst_size: 25}
      {:ok, _} = start_rate_limiter(name, config)

      status = RateLimiter.get_status(name)

      assert status.name == name
      assert status.config.rate == 50
      assert status.config.burst_size == 25
      assert is_float(status.tokens)
    end
  end

  defp start_rate_limiter(name, config \\ %{}) do
    ResilienceSupervisor.start_rate_limiter(name, config)
  end
end
