defmodule AuthPlatform.Resilience.CircuitBreakerPropertyTest do
  @moduledoc """
  Property-based tests for Circuit Breaker state machine.

  Property 12: Circuit Breaker State Machine
  - Validates state transitions follow the expected pattern
  - Ensures failure threshold triggers state change
  - Verifies timeout behavior
  """
  use ExUnit.Case, async: false
  use ExUnitProperties

  alias AuthPlatform.Resilience.CircuitBreaker
  alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor

  @moduletag :property

  # Generator for circuit breaker config
  defp config_generator do
    gen all(
          failure_threshold <- integer(1..10),
          success_threshold <- integer(1..5),
          timeout_ms <- integer(1..100),
          half_open_max <- integer(1..5)
        ) do
      %{
        failure_threshold: failure_threshold,
        success_threshold: success_threshold,
        timeout_ms: timeout_ms,
        half_open_max_requests: half_open_max
      }
    end
  end

  # Generator for operation sequences
  defp operation_generator do
    member_of([:success, :failure])
  end

  describe "Property 12: Circuit Breaker State Machine" do
    property "closed state opens after failure_threshold failures" do
      check all(config <- config_generator()) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_circuit_breaker(name, config)

        # Record exactly failure_threshold failures
        for _ <- 1..config.failure_threshold do
          CircuitBreaker.record_failure(name)
        end

        Process.sleep(10)
        assert CircuitBreaker.get_state(name) == :open

        cleanup(name)
      end
    end

    property "closed state stays closed below failure_threshold" do
      check all(config <- config_generator()) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_circuit_breaker(name, config)

        # Record one less than threshold
        failures_to_record = max(0, config.failure_threshold - 1)

        for _ <- 1..failures_to_record do
          CircuitBreaker.record_failure(name)
        end

        Process.sleep(10)
        assert CircuitBreaker.get_state(name) == :closed

        cleanup(name)
      end
    end

    property "success resets failure count in closed state" do
      check all(config <- config_generator()) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_circuit_breaker(name, config)

        # Record some failures (but not enough to open)
        failures = max(1, config.failure_threshold - 1)

        for _ <- 1..failures do
          CircuitBreaker.record_failure(name)
        end

        Process.sleep(5)
        status_before = CircuitBreaker.get_status(name)
        assert status_before.failures == failures

        # Success should reset
        CircuitBreaker.record_success(name)
        Process.sleep(5)

        status_after = CircuitBreaker.get_status(name)
        assert status_after.failures == 0

        cleanup(name)
      end
    end

    property "open state transitions to half_open after timeout" do
      check all(config <- config_generator()) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_circuit_breaker(name, config)

        # Open the circuit
        for _ <- 1..config.failure_threshold do
          CircuitBreaker.record_failure(name)
        end

        Process.sleep(10)
        assert CircuitBreaker.get_state(name) == :open

        # Wait for timeout
        Process.sleep(config.timeout_ms + 10)

        # Request should trigger half_open
        CircuitBreaker.allow_request?(name)
        assert CircuitBreaker.get_state(name) == :half_open

        cleanup(name)
      end
    end

    property "half_open closes after success_threshold successes" do
      check all(config <- config_generator()) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_circuit_breaker(name, config)

        # Open and transition to half_open
        for _ <- 1..config.failure_threshold do
          CircuitBreaker.record_failure(name)
        end

        Process.sleep(config.timeout_ms + 10)
        CircuitBreaker.allow_request?(name)
        assert CircuitBreaker.get_state(name) == :half_open

        # Record successes
        for _ <- 1..config.success_threshold do
          CircuitBreaker.record_success(name)
        end

        Process.sleep(10)
        assert CircuitBreaker.get_state(name) == :closed

        cleanup(name)
      end
    end

    property "half_open reopens on any failure" do
      check all(config <- config_generator()) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_circuit_breaker(name, config)

        # Open and transition to half_open
        for _ <- 1..config.failure_threshold do
          CircuitBreaker.record_failure(name)
        end

        Process.sleep(config.timeout_ms + 10)
        CircuitBreaker.allow_request?(name)
        assert CircuitBreaker.get_state(name) == :half_open

        # Single failure should reopen
        CircuitBreaker.record_failure(name)
        Process.sleep(10)
        assert CircuitBreaker.get_state(name) == :open

        cleanup(name)
      end
    end

    property "reset always returns to closed state" do
      check all(
              config <- config_generator(),
              operations <- list_of(operation_generator(), min_length: 1, max_length: 20)
            ) do
        name = unique_name()
        {:ok, _} = ResilienceSupervisor.start_circuit_breaker(name, config)

        # Apply random operations
        Enum.each(operations, fn
          :success -> CircuitBreaker.record_success(name)
          :failure -> CircuitBreaker.record_failure(name)
        end)

        Process.sleep(10)

        # Reset should always work
        CircuitBreaker.reset(name)
        Process.sleep(10)

        assert CircuitBreaker.get_state(name) == :closed
        assert CircuitBreaker.get_status(name).failures == 0

        cleanup(name)
      end
    end
  end

  defp unique_name do
    :"cb_prop_#{:erlang.unique_integer([:positive])}"
  end

  defp cleanup(name) do
    ResilienceSupervisor.stop_child(name)
  end
end
