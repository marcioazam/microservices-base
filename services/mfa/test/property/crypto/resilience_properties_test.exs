defmodule MfaService.Crypto.ResiliencePropertiesTest do
  @moduledoc """
  Property-based tests for resilience patterns (circuit breaker, retry).
  Tests correctness properties defined in the design document.
  """
  use ExUnit.Case, async: false
  use ExUnitProperties

  alias MfaService.Crypto.{CircuitBreaker, Retry}

  # Test circuit breaker in isolation without GenServer
  defmodule TestCircuitBreaker do
    @moduledoc false
    
    defstruct [
      state: :closed,
      failure_count: 0,
      threshold: 5,
      reset_timeout: 30_000,
      last_failure_time: nil
    ]

    def new(threshold \\ 5) do
      %__MODULE__{threshold: threshold}
    end

    def record_failure(%{state: :closed} = cb) do
      new_count = cb.failure_count + 1
      
      if new_count >= cb.threshold do
        %{cb | 
          state: :open, 
          failure_count: new_count,
          last_failure_time: System.monotonic_time(:millisecond)
        }
      else
        %{cb | failure_count: new_count}
      end
    end

    def record_failure(%{state: :half_open} = cb) do
      %{cb | 
        state: :open, 
        failure_count: cb.threshold,
        last_failure_time: System.monotonic_time(:millisecond)
      }
    end

    def record_failure(cb), do: cb

    def record_success(%{state: :half_open} = cb) do
      %{cb | state: :closed, failure_count: 0, last_failure_time: nil}
    end

    def record_success(cb) do
      %{cb | failure_count: 0}
    end

    def should_allow?(%{state: :closed}), do: true
    def should_allow?(%{state: :half_open}), do: true
    def should_allow?(%{state: :open} = cb) do
      case cb.last_failure_time do
        nil -> true
        time -> 
          elapsed = System.monotonic_time(:millisecond) - time
          elapsed >= cb.reset_timeout
      end
    end
  end

  describe "Property 3: Circuit Breaker Opens After Threshold" do
    @tag :property
    @tag timeout: 120_000
    property "circuit opens after threshold consecutive failures" do
      check all threshold <- integer(1..10),
                extra_failures <- integer(0..5),
                max_runs: 100 do
        
        cb = TestCircuitBreaker.new(threshold)
        
        # Apply exactly threshold failures
        cb_after_failures = Enum.reduce(1..threshold, cb, fn _, acc ->
          TestCircuitBreaker.record_failure(acc)
        end)
        
        # Circuit should be open
        assert cb_after_failures.state == :open,
          "Circuit should be open after #{threshold} failures, but was #{cb_after_failures.state}"
        
        # Additional failures should keep it open
        cb_after_more = Enum.reduce(1..extra_failures, cb_after_failures, fn _, acc ->
          TestCircuitBreaker.record_failure(acc)
        end)
        
        assert cb_after_more.state == :open,
          "Circuit should remain open after additional failures"
      end
    end

    @tag :property
    property "circuit stays closed with fewer than threshold failures" do
      check all threshold <- integer(2..10),
                failures <- integer(1..(threshold - 1)),
                max_runs: 100 do
        
        cb = TestCircuitBreaker.new(threshold)
        
        # Apply fewer than threshold failures
        cb_after_failures = Enum.reduce(1..failures, cb, fn _, acc ->
          TestCircuitBreaker.record_failure(acc)
        end)
        
        # Circuit should still be closed
        assert cb_after_failures.state == :closed,
          "Circuit should be closed with #{failures} failures (threshold: #{threshold})"
        assert cb_after_failures.failure_count == failures
      end
    end
  end

  describe "Property 4: Circuit Breaker Fail-Fast When Open" do
    @tag :property
    property "open circuit rejects requests immediately" do
      check all threshold <- integer(1..10),
                max_runs: 100 do
        
        cb = TestCircuitBreaker.new(threshold)
        
        # Open the circuit
        cb_open = Enum.reduce(1..threshold, cb, fn _, acc ->
          TestCircuitBreaker.record_failure(acc)
        end)
        
        assert cb_open.state == :open
        
        # Should not allow requests when open (unless reset timeout passed)
        # Since we just opened it, reset timeout hasn't passed
        refute TestCircuitBreaker.should_allow?(cb_open),
          "Open circuit should reject requests"
      end
    end
  end

  describe "Property 17: Retry on Transient Failures" do
    @tag :property
    property "retries up to max attempts on transient failures" do
      check all max_attempts <- integer(1..5),
                max_runs: 100 do
        
        # Track number of calls
        call_count = :counters.new(1, [:atomics])
        
        # Function that always fails with retryable error
        failing_fun = fn ->
          :counters.add(call_count, 1, 1)
          {:error, :timeout}
        end
        
        # Execute with retry
        result = Retry.with_retry(failing_fun, 
          max_attempts: max_attempts,
          base_delay: 1,  # Minimal delay for testing
          max_delay: 10
        )
        
        # Should have tried max_attempts times
        actual_calls = :counters.get(call_count, 1)
        assert actual_calls == max_attempts,
          "Expected #{max_attempts} attempts, got #{actual_calls}"
        
        # Should return error after exhausting retries
        assert {:error, :timeout} = result
      end
    end

    @tag :property
    property "succeeds immediately without retry on success" do
      check all max_attempts <- integer(1..5),
                max_runs: 100 do
        
        call_count = :counters.new(1, [:atomics])
        
        # Function that succeeds
        success_fun = fn ->
          :counters.add(call_count, 1, 1)
          {:ok, :success}
        end
        
        result = Retry.with_retry(success_fun,
          max_attempts: max_attempts,
          base_delay: 1,
          max_delay: 10
        )
        
        # Should have called only once
        assert :counters.get(call_count, 1) == 1
        assert result == {:ok, :success}
      end
    end

    @tag :property
    property "does not retry non-retryable errors" do
      check all max_attempts <- integer(2..5),
                max_runs: 100 do
        
        call_count = :counters.new(1, [:atomics])
        
        # Function that fails with non-retryable error
        non_retryable_fun = fn ->
          :counters.add(call_count, 1, 1)
          {:error, :circuit_open}
        end
        
        result = Retry.with_retry(non_retryable_fun,
          max_attempts: max_attempts,
          base_delay: 1,
          max_delay: 10
        )
        
        # Should have called only once (no retry for circuit_open)
        assert :counters.get(call_count, 1) == 1
        assert result == {:error, :circuit_open}
      end
    end

    @tag :property
    property "exponential backoff delay increases with attempts" do
      check all base_delay <- integer(10..100),
                max_runs: 100 do
        
        delay1 = Retry.calculate_delay(1, base_delay, 100_000)
        delay2 = Retry.calculate_delay(2, base_delay, 100_000)
        delay3 = Retry.calculate_delay(3, base_delay, 100_000)
        
        # Due to jitter, we check approximate ranges
        # Attempt 1: ~base_delay (±25%)
        assert delay1 >= base_delay * 0.75
        assert delay1 <= base_delay * 1.25
        
        # Attempt 2: ~base_delay * 2 (±25%)
        assert delay2 >= base_delay * 2 * 0.75
        assert delay2 <= base_delay * 2 * 1.25
        
        # Attempt 3: ~base_delay * 4 (±25%)
        assert delay3 >= base_delay * 4 * 0.75
        assert delay3 <= base_delay * 4 * 1.25
      end
    end
  end

  describe "Circuit breaker state transitions" do
    @tag :property
    property "success resets failure count in closed state" do
      check all threshold <- integer(2..10),
                failures <- integer(1..(threshold - 1)),
                max_runs: 100 do
        
        cb = TestCircuitBreaker.new(threshold)
        
        # Accumulate some failures
        cb_with_failures = Enum.reduce(1..failures, cb, fn _, acc ->
          TestCircuitBreaker.record_failure(acc)
        end)
        
        assert cb_with_failures.failure_count == failures
        
        # Success should reset count
        cb_after_success = TestCircuitBreaker.record_success(cb_with_failures)
        
        assert cb_after_success.failure_count == 0
        assert cb_after_success.state == :closed
      end
    end

    @tag :property
    property "success in half-open closes circuit" do
      check all threshold <- integer(1..10),
                max_runs: 100 do
        
        cb = %TestCircuitBreaker{
          state: :half_open,
          failure_count: threshold,
          threshold: threshold
        }
        
        cb_after_success = TestCircuitBreaker.record_success(cb)
        
        assert cb_after_success.state == :closed
        assert cb_after_success.failure_count == 0
      end
    end

    @tag :property
    property "failure in half-open reopens circuit" do
      check all threshold <- integer(1..10),
                max_runs: 100 do
        
        cb = %TestCircuitBreaker{
          state: :half_open,
          failure_count: 0,
          threshold: threshold
        }
        
        cb_after_failure = TestCircuitBreaker.record_failure(cb)
        
        assert cb_after_failure.state == :open
      end
    end
  end
end
