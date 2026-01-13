defmodule AuthPlatform.Resilience.RateLimiterPropertyTest do
  @moduledoc """
  Property-based tests for Rate Limiter.

  Property 15: Rate Limiter Token Bucket
  """
  use ExUnit.Case, async: false
  use ExUnitProperties

  alias AuthPlatform.Resilience.RateLimiter
  alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor

  @moduletag :property

  describe "Property 15: Rate Limiter Token Bucket" do
    property "allows exactly burst_size requests initially" do
      check all(burst_size <- integer(1..20)) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_rate_limiter(name, %{rate: 1, burst_size: burst_size})

        # Should allow exactly burst_size requests
        results = for _ <- 1..burst_size, do: RateLimiter.allow?(name)
        assert Enum.all?(results, & &1)

        # Next request should fail
        assert RateLimiter.allow?(name) == false

        cleanup(name)
      end
    end

    property "tokens never exceed burst_size" do
      check all(
              rate <- integer(100..1000),
              burst_size <- integer(5..20)
            ) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_rate_limiter(name, %{rate: rate, burst_size: burst_size})

        # Wait for potential over-refill
        Process.sleep(50)

        tokens = RateLimiter.available_tokens(name)
        assert tokens <= burst_size * 1.0

        cleanup(name)
      end
    end

    property "tokens refill at configured rate" do
      check all(rate <- integer(500..2000)) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_rate_limiter(name, %{rate: rate, burst_size: 1})

        # Consume the token
        RateLimiter.allow?(name)
        tokens_after_consume = RateLimiter.available_tokens(name)
        assert tokens_after_consume < 1.0

        # Wait for refill (rate tokens/sec)
        wait_ms = ceil(1000.0 / rate) + 5
        Process.sleep(wait_ms)

        tokens_after_wait = RateLimiter.available_tokens(name)
        assert tokens_after_wait > tokens_after_consume

        cleanup(name)
      end
    end

    property "reset restores tokens to burst_size" do
      check all(burst_size <- integer(5..20)) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_rate_limiter(name, %{rate: 1, burst_size: burst_size})

        # Consume some tokens
        for _ <- 1..min(3, burst_size), do: RateLimiter.allow?(name)

        tokens_before = RateLimiter.available_tokens(name)
        assert tokens_before < burst_size

        RateLimiter.reset(name)
        Process.sleep(5)

        tokens_after = RateLimiter.available_tokens(name)
        # Should be close to burst_size (allowing for timing)
        assert tokens_after >= burst_size - 1

        cleanup(name)
      end
    end

    property "concurrent requests respect rate limit" do
      check all(burst_size <- integer(5..15)) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_rate_limiter(name, %{rate: 1, burst_size: burst_size})

        # Fire many concurrent requests
        tasks =
          for _ <- 1..(burst_size * 2) do
            Task.async(fn -> RateLimiter.allow?(name) end)
          end

        results = Task.await_many(tasks, 1000)

        # Should have exactly burst_size successes
        successes = Enum.count(results, & &1)
        assert successes == burst_size

        cleanup(name)
      end
    end
  end

  defp unique_name do
    :"rl_prop_#{:erlang.unique_integer([:positive])}"
  end

  defp cleanup(name) do
    ResilienceSupervisor.stop_child(name)
  end
end
