defmodule AuthPlatform.Resilience.RetryPropertyTest do
  @moduledoc """
  Property-based tests for Retry Policy.

  Property 13: Retry Policy Exponential Backoff
  Property 14: Retry Policy Execution
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Resilience.Retry

  @moduletag :property

  describe "Property 13: Retry Policy Exponential Backoff" do
    property "delay increases exponentially with attempt number" do
      check all(
              initial_delay <- integer(10..1000),
              multiplier <- float(min: 1.5, max: 3.0),
              max_delay <- integer(5000..50_000)
            ) do
        config = %{
          initial_delay_ms: initial_delay,
          multiplier: multiplier,
          max_delay_ms: max_delay,
          jitter: 0
        }

        delay1 = Retry.delay_for_attempt(1, config)
        delay2 = Retry.delay_for_attempt(2, config)
        delay3 = Retry.delay_for_attempt(3, config)

        # Each delay should be >= previous (exponential growth)
        assert delay2 >= delay1
        assert delay3 >= delay2

        # First delay should equal initial_delay
        assert delay1 == initial_delay
      end
    end

    property "delay never exceeds max_delay_ms" do
      check all(
              initial_delay <- integer(100..1000),
              multiplier <- float(min: 2.0, max: 10.0),
              max_delay <- integer(1000..5000),
              attempt <- integer(1..20)
            ) do
        config = %{
          initial_delay_ms: initial_delay,
          multiplier: multiplier,
          max_delay_ms: max_delay,
          jitter: 0
        }

        delay = Retry.delay_for_attempt(attempt, config)
        assert delay <= max_delay
      end
    end

    property "delay with jitter stays within expected bounds" do
      check all(
              initial_delay <- integer(100..1000),
              jitter <- float(min: 0.0, max: 0.5)
            ) do
        config = %{
          initial_delay_ms: initial_delay,
          multiplier: 1.0,
          max_delay_ms: 100_000,
          jitter: jitter
        }

        # Sample multiple delays
        delays = for _ <- 1..50, do: Retry.delay_for_attempt(1, config)

        # All delays should be non-negative
        assert Enum.all?(delays, fn d -> d >= 0 end)

        # Max delay should be within jitter range
        max_expected = initial_delay + round(initial_delay * jitter)
        assert Enum.all?(delays, fn d -> d <= max_expected + 1 end)
      end
    end

    property "delay is always non-negative" do
      check all(
              initial_delay <- integer(1..10_000),
              multiplier <- float(min: 0.5, max: 5.0),
              max_delay <- integer(1..100_000),
              jitter <- float(min: 0.0, max: 1.0),
              attempt <- integer(1..100)
            ) do
        config = %{
          initial_delay_ms: initial_delay,
          multiplier: multiplier,
          max_delay_ms: max_delay,
          jitter: jitter
        }

        delay = Retry.delay_for_attempt(attempt, config)
        assert delay >= 0
      end
    end
  end

  describe "Property 14: Retry Policy Execution" do
    property "successful function returns immediately without retry" do
      check all(value <- term()) do
        counter = :counters.new(1, [:atomics])

        result =
          Retry.execute(
            fn ->
              :counters.add(counter, 1, 1)
              {:ok, value}
            end,
            %{max_retries: 5, initial_delay_ms: 1, max_delay_ms: 10, multiplier: 1.0, jitter: 0}
          )

        assert result == {:ok, value}
        assert :counters.get(counter, 1) == 1
      end
    end

    property "retries exactly max_retries times on persistent failure" do
      check all(max_retries <- integer(0..5)) do
        counter = :counters.new(1, [:atomics])

        Retry.execute(
          fn ->
            :counters.add(counter, 1, 1)
            {:error, :timeout}
          end,
          %{
            max_retries: max_retries,
            initial_delay_ms: 1,
            max_delay_ms: 10,
            multiplier: 1.0,
            jitter: 0
          }
        )

        # Initial attempt + max_retries
        expected_calls = 1 + max_retries
        assert :counters.get(counter, 1) == expected_calls
      end
    end

    property "succeeds on Nth attempt if failure count < max_retries" do
      check all(
              succeed_on <- integer(1..5),
              max_retries <- integer(5..10)
            ) do
        counter = :counters.new(1, [:atomics])

        result =
          Retry.execute(
            fn ->
              count = :counters.get(counter, 1)
              :counters.add(counter, 1, 1)

              if count < succeed_on - 1 do
                {:error, :timeout}
              else
                {:ok, :success}
              end
            end,
            %{
              max_retries: max_retries,
              initial_delay_ms: 1,
              max_delay_ms: 10,
              multiplier: 1.0,
              jitter: 0
            }
          )

        assert result == {:ok, :success}
        assert :counters.get(counter, 1) == succeed_on
      end
    end
  end
end
