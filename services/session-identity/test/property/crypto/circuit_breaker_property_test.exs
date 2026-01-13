defmodule SessionIdentityCore.Crypto.CircuitBreakerPropertyTest do
  @moduledoc """
  Property tests for circuit breaker and fallback behavior.
  
  **Property 1: Circuit Breaker Fallback Behavior**
  **Validates: Requirements 1.2, 2.5**
  """

  use ExUnit.Case, async: false
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.{CircuitBreaker, Errors}

  setup do
    # Reset circuit breaker before each test
    CircuitBreaker.init()
    CircuitBreaker.reset()
    :ok
  end

  # Generators

  defp success_result do
    gen all value <- term() do
      {:ok, value}
    end
  end

  defp transient_error do
    member_of([
      {:error, Errors.service_unavailable()},
      {:error, Errors.operation_timeout()}
    ])
  end

  defp non_transient_error do
    gen all message <- string(:alphanumeric, min_length: 1, max_length: 50) do
      {:error, Errors.invalid_argument(message)}
    end
  end

  # Property Tests

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "successful operations pass through circuit breaker" do
    check all result <- success_result(), max_runs: 100 do
      CircuitBreaker.reset()
      
      outcome = CircuitBreaker.call(fn -> result end)
      
      assert outcome == result
      assert CircuitBreaker.state() == :ok
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "non-transient errors do not trip circuit breaker" do
    check all error <- non_transient_error(), max_runs: 100 do
      CircuitBreaker.reset()
      
      # Execute multiple non-transient errors
      for _ <- 1..10 do
        CircuitBreaker.call(fn -> error end)
      end
      
      # Circuit should still be closed
      assert CircuitBreaker.state() == :ok
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "transient errors trip circuit after threshold" do
    check all error <- transient_error(), max_runs: 50 do
      CircuitBreaker.reset()
      
      # Trip the circuit with transient errors
      for _ <- 1..6 do
        CircuitBreaker.call(fn -> error end)
      end
      
      # Circuit should be open
      assert CircuitBreaker.state() == :blown
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "open circuit returns service_unavailable error" do
    check all _ <- constant(:ok), max_runs: 100 do
      CircuitBreaker.reset()
      CircuitBreaker.melt()  # Force circuit open
      
      result = CircuitBreaker.call(fn -> {:ok, :should_not_execute} end)
      
      assert {:error, %{error_code: :crypto_service_unavailable}} = result
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "fallback is called when circuit is open" do
    check all primary_result <- success_result(),
              fallback_result <- success_result(),
              max_runs: 100 do
      CircuitBreaker.reset()
      CircuitBreaker.melt()  # Force circuit open
      
      result = CircuitBreaker.call_with_fallback(
        fn -> primary_result end,
        fn -> fallback_result end
      )
      
      assert result == fallback_result
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "primary function is called when circuit is closed" do
    check all primary_result <- success_result(),
              fallback_result <- success_result(),
              max_runs: 100 do
      CircuitBreaker.reset()
      
      result = CircuitBreaker.call_with_fallback(
        fn -> primary_result end,
        fn -> fallback_result end
      )
      
      assert result == primary_result
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "reset restores circuit to closed state" do
    check all _ <- constant(:ok), max_runs: 100 do
      CircuitBreaker.melt()
      assert CircuitBreaker.state() == :blown
      
      CircuitBreaker.reset()
      assert CircuitBreaker.state() == :ok
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.2, 2.5"
  property "melt opens the circuit" do
    check all _ <- constant(:ok), max_runs: 100 do
      CircuitBreaker.reset()
      assert CircuitBreaker.state() == :ok
      
      CircuitBreaker.melt()
      assert CircuitBreaker.state() == :blown
    end
  end
end
