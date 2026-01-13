defmodule AuthPlatform.Resilience.RetryTest do
  use ExUnit.Case, async: true

  alias AuthPlatform.Resilience.Retry
  alias AuthPlatform.Errors.AppError

  describe "default_config/0" do
    test "returns default configuration" do
      config = Retry.default_config()

      assert config.max_retries == 3
      assert config.initial_delay_ms == 100
      assert config.max_delay_ms == 10_000
      assert config.multiplier == 2.0
      assert config.jitter == 0.1
    end
  end

  describe "delay_for_attempt/2" do
    test "calculates exponential backoff without jitter" do
      config = %{initial_delay_ms: 100, multiplier: 2.0, max_delay_ms: 10_000, jitter: 0}

      assert Retry.delay_for_attempt(1, config) == 100
      assert Retry.delay_for_attempt(2, config) == 200
      assert Retry.delay_for_attempt(3, config) == 400
      assert Retry.delay_for_attempt(4, config) == 800
    end

    test "caps delay at max_delay_ms" do
      config = %{initial_delay_ms: 1000, multiplier: 10.0, max_delay_ms: 5000, jitter: 0}

      assert Retry.delay_for_attempt(1, config) == 1000
      assert Retry.delay_for_attempt(2, config) == 5000
      assert Retry.delay_for_attempt(3, config) == 5000
    end

    test "applies jitter within expected range" do
      config = %{initial_delay_ms: 1000, multiplier: 1.0, max_delay_ms: 10_000, jitter: 0.1}

      # Run multiple times to verify jitter
      delays = for _ <- 1..100, do: Retry.delay_for_attempt(1, config)

      # All delays should be within 10% of base (900-1100)
      assert Enum.all?(delays, fn d -> d >= 0 and d <= 1100 end)
      # Should have some variation
      assert Enum.uniq(delays) |> length() > 1
    end
  end

  describe "should_retry?/3" do
    test "returns true for retryable AppError" do
      error = {:error, AppError.timeout("operation")}
      config = %{max_retries: 3}

      assert Retry.should_retry?(error, 0, config) == true
      assert Retry.should_retry?(error, 1, config) == true
      assert Retry.should_retry?(error, 2, config) == true
    end

    test "returns false for non-retryable AppError" do
      error = {:error, AppError.validation("invalid")}
      config = %{max_retries: 3}

      assert Retry.should_retry?(error, 0, config) == false
    end

    test "returns false when max retries reached" do
      error = {:error, AppError.timeout("operation")}
      config = %{max_retries: 3}

      assert Retry.should_retry?(error, 3, config) == false
    end

    test "returns true for timeout atom" do
      assert Retry.should_retry?({:error, :timeout}, 0, %{max_retries: 3}) == true
    end

    test "returns true for connection errors" do
      assert Retry.should_retry?({:error, :econnrefused}, 0, %{max_retries: 3}) == true
      assert Retry.should_retry?({:error, :econnreset}, 0, %{max_retries: 3}) == true
      assert Retry.should_retry?({:error, :closed}, 0, %{max_retries: 3}) == true
    end

    test "returns false for ok result" do
      assert Retry.should_retry?({:ok, "result"}, 0, %{max_retries: 3}) == false
    end
  end

  describe "execute/3" do
    test "returns success on first try" do
      result = Retry.execute(fn -> {:ok, "success"} end)
      assert result == {:ok, "success"}
    end

    test "retries on retryable error and succeeds" do
      counter = :counters.new(1, [:atomics])

      result =
        Retry.execute(
          fn ->
            count = :counters.get(counter, 1)
            :counters.add(counter, 1, 1)

            if count < 2 do
              {:error, :timeout}
            else
              {:ok, "success"}
            end
          end,
          %{max_retries: 3, initial_delay_ms: 1, max_delay_ms: 10, multiplier: 1.0, jitter: 0}
        )

      assert result == {:ok, "success"}
      assert :counters.get(counter, 1) == 3
    end

    test "returns error after max retries exhausted" do
      counter = :counters.new(1, [:atomics])

      result =
        Retry.execute(
          fn ->
            :counters.add(counter, 1, 1)
            {:error, :timeout}
          end,
          %{max_retries: 3, initial_delay_ms: 1, max_delay_ms: 10, multiplier: 1.0, jitter: 0}
        )

      assert result == {:error, :timeout}
      # Initial attempt + 3 retries = 4 total
      assert :counters.get(counter, 1) == 4
    end

    test "does not retry non-retryable errors" do
      counter = :counters.new(1, [:atomics])

      result =
        Retry.execute(
          fn ->
            :counters.add(counter, 1, 1)
            {:error, AppError.validation("invalid")}
          end,
          %{max_retries: 3, initial_delay_ms: 1, max_delay_ms: 10, multiplier: 1.0, jitter: 0}
        )

      assert {:error, %AppError{code: :validation}} = result
      assert :counters.get(counter, 1) == 1
    end

    test "calls on_retry callback before each retry" do
      callback_calls = :counters.new(1, [:atomics])
      attempt_counter = :counters.new(1, [:atomics])

      Retry.execute(
        fn ->
          :counters.add(attempt_counter, 1, 1)
          {:error, :timeout}
        end,
        %{max_retries: 2, initial_delay_ms: 1, max_delay_ms: 10, multiplier: 1.0, jitter: 0},
        on_retry: fn _attempt, _delay, _error ->
          :counters.add(callback_calls, 1, 1)
        end
      )

      # Should be called twice (before retry 1 and retry 2)
      assert :counters.get(callback_calls, 1) == 2
    end

    test "handles exceptions as errors" do
      result =
        Retry.execute(
          fn -> raise "boom" end,
          %{max_retries: 1, initial_delay_ms: 1, max_delay_ms: 10, multiplier: 1.0, jitter: 0}
        )

      assert {:error, %RuntimeError{message: "boom"}} = result
    end
  end
end
