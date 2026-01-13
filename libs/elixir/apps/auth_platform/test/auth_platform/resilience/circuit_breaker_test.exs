defmodule AuthPlatform.Resilience.CircuitBreakerTest do
  use ExUnit.Case, async: false

  alias AuthPlatform.Resilience.CircuitBreaker
  alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor

  @moduletag :circuit_breaker

  setup do
    # Generate unique name for each test
    name = :"cb_test_#{:erlang.unique_integer([:positive])}"
    {:ok, name: name}
  end

  describe "start_link/1" do
    test "starts with default config", %{name: name} do
      {:ok, pid} = start_circuit_breaker(name)
      assert Process.alive?(pid)
      assert CircuitBreaker.get_state(name) == :closed
    end

    test "starts with custom config", %{name: name} do
      config = %{failure_threshold: 3, timeout_ms: 5000}
      {:ok, pid} = start_circuit_breaker(name, config)

      assert Process.alive?(pid)
      status = CircuitBreaker.get_status(name)
      assert status.config.failure_threshold == 3
      assert status.config.timeout_ms == 5000
    end
  end

  describe "allow_request?/1" do
    test "allows requests when closed", %{name: name} do
      {:ok, _} = start_circuit_breaker(name)
      assert CircuitBreaker.allow_request?(name) == true
    end

    test "blocks requests when open", %{name: name} do
      config = %{failure_threshold: 2, timeout_ms: 60_000}
      {:ok, _} = start_circuit_breaker(name, config)

      # Trigger failures to open circuit
      CircuitBreaker.record_failure(name)
      CircuitBreaker.record_failure(name)

      # Wait for state to update
      Process.sleep(10)

      assert CircuitBreaker.get_state(name) == :open
      assert CircuitBreaker.allow_request?(name) == false
    end
  end

  describe "record_success/1" do
    test "resets failure count when closed", %{name: name} do
      {:ok, _} = start_circuit_breaker(name)

      CircuitBreaker.record_failure(name)
      Process.sleep(5)
      assert CircuitBreaker.get_status(name).failures == 1

      CircuitBreaker.record_success(name)
      Process.sleep(5)
      assert CircuitBreaker.get_status(name).failures == 0
    end

    test "transitions from half_open to closed after threshold", %{name: name} do
      config = %{failure_threshold: 1, success_threshold: 2, timeout_ms: 1}
      {:ok, _} = start_circuit_breaker(name, config)

      # Open the circuit
      CircuitBreaker.record_failure(name)
      Process.sleep(10)
      assert CircuitBreaker.get_state(name) == :open

      # Wait for timeout and trigger half_open
      Process.sleep(10)
      CircuitBreaker.allow_request?(name)
      assert CircuitBreaker.get_state(name) == :half_open

      # Record successes to close
      CircuitBreaker.record_success(name)
      CircuitBreaker.record_success(name)
      Process.sleep(5)

      assert CircuitBreaker.get_state(name) == :closed
    end
  end

  describe "record_failure/1" do
    test "opens circuit after threshold failures", %{name: name} do
      config = %{failure_threshold: 3}
      {:ok, _} = start_circuit_breaker(name, config)

      CircuitBreaker.record_failure(name)
      CircuitBreaker.record_failure(name)
      Process.sleep(5)
      assert CircuitBreaker.get_state(name) == :closed

      CircuitBreaker.record_failure(name)
      Process.sleep(5)
      assert CircuitBreaker.get_state(name) == :open
    end

    test "reopens circuit from half_open on failure", %{name: name} do
      config = %{failure_threshold: 1, timeout_ms: 1}
      {:ok, _} = start_circuit_breaker(name, config)

      # Open circuit
      CircuitBreaker.record_failure(name)
      Process.sleep(10)

      # Transition to half_open
      CircuitBreaker.allow_request?(name)
      assert CircuitBreaker.get_state(name) == :half_open

      # Fail again
      CircuitBreaker.record_failure(name)
      Process.sleep(5)
      assert CircuitBreaker.get_state(name) == :open
    end
  end

  describe "execute/2" do
    test "executes function when circuit closed", %{name: name} do
      {:ok, _} = start_circuit_breaker(name)

      result = CircuitBreaker.execute(name, fn -> {:ok, :success} end)
      assert result == {:ok, :success}
    end

    test "returns error when circuit open", %{name: name} do
      config = %{failure_threshold: 1, timeout_ms: 60_000}
      {:ok, _} = start_circuit_breaker(name, config)

      CircuitBreaker.record_failure(name)
      Process.sleep(10)

      result = CircuitBreaker.execute(name, fn -> {:ok, :success} end)
      assert result == {:error, :circuit_open}
    end

    test "records success on successful execution", %{name: name} do
      {:ok, _} = start_circuit_breaker(name)

      CircuitBreaker.record_failure(name)
      Process.sleep(5)
      assert CircuitBreaker.get_status(name).failures == 1

      CircuitBreaker.execute(name, fn -> {:ok, :success} end)
      Process.sleep(5)
      assert CircuitBreaker.get_status(name).failures == 0
    end

    test "records failure on failed execution", %{name: name} do
      {:ok, _} = start_circuit_breaker(name)

      CircuitBreaker.execute(name, fn -> {:error, :failed} end)
      Process.sleep(5)
      assert CircuitBreaker.get_status(name).failures == 1
    end

    test "records failure on exception", %{name: name} do
      {:ok, _} = start_circuit_breaker(name)

      result = CircuitBreaker.execute(name, fn -> raise "boom" end)
      Process.sleep(5)

      assert {:error, %RuntimeError{}} = result
      assert CircuitBreaker.get_status(name).failures == 1
    end
  end

  describe "reset/1" do
    test "resets circuit to closed state", %{name: name} do
      config = %{failure_threshold: 1}
      {:ok, _} = start_circuit_breaker(name, config)

      CircuitBreaker.record_failure(name)
      Process.sleep(5)
      assert CircuitBreaker.get_state(name) == :open

      CircuitBreaker.reset(name)
      Process.sleep(5)
      assert CircuitBreaker.get_state(name) == :closed
      assert CircuitBreaker.get_status(name).failures == 0
    end
  end

  describe "get_status/1" do
    test "returns full status information", %{name: name} do
      config = %{failure_threshold: 5, timeout_ms: 10_000}
      {:ok, _} = start_circuit_breaker(name, config)

      status = CircuitBreaker.get_status(name)

      assert status.name == name
      assert status.state == :closed
      assert status.failures == 0
      assert status.successes == 0
      assert status.config.failure_threshold == 5
      assert status.config.timeout_ms == 10_000
    end
  end

  # Helper to start circuit breaker via supervisor
  defp start_circuit_breaker(name, config \\ %{}) do
    ResilienceSupervisor.start_circuit_breaker(name, config)
  end
end
