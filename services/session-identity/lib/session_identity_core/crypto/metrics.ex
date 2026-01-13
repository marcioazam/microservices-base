defmodule SessionIdentityCore.Crypto.Metrics do
  @moduledoc """
  Prometheus metrics for crypto operations.
  
  Provides:
  - Counter for operations by type and status
  - Histogram for operation latency
  - Gauge for circuit breaker state
  """

  use Prometheus.Metric

  @counter_name :crypto_operations_total
  @histogram_name :crypto_operation_duration_seconds
  @gauge_name :crypto_circuit_breaker_state

  @doc """
  Sets up all crypto metrics.
  """
  def setup do
    Counter.declare(
      name: @counter_name,
      help: "Total crypto operations",
      labels: [:operation, :status, :namespace]
    )

    Histogram.declare(
      name: @histogram_name,
      help: "Crypto operation duration in seconds",
      labels: [:operation, :namespace],
      buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0]
    )

    Gauge.declare(
      name: @gauge_name,
      help: "Circuit breaker state (0=closed, 1=open, 2=half_open)",
      labels: [:service]
    )

    :ok
  end

  @doc """
  Increments operation counter.
  """
  @spec inc_operation(atom(), atom(), String.t()) :: :ok
  def inc_operation(operation, status, namespace) do
    Counter.inc(
      name: @counter_name,
      labels: [operation, status, namespace]
    )
  end

  @doc """
  Records operation duration.
  """
  @spec observe_duration(atom(), String.t(), number()) :: :ok
  def observe_duration(operation, namespace, duration_ms) do
    Histogram.observe(
      [name: @histogram_name, labels: [operation, namespace]],
      duration_ms / 1000
    )
  end

  @doc """
  Updates circuit breaker state gauge.
  """
  @spec set_circuit_state(atom()) :: :ok
  def set_circuit_state(state) do
    value = case state do
      :closed -> 0
      :open -> 1
      :half_open -> 2
    end
    
    Gauge.set([name: @gauge_name, labels: [:crypto_service]], value)
  end

  @doc """
  Measures and records operation duration.
  """
  defmacro measure(operation, namespace, do: block) do
    quote do
      start = System.monotonic_time(:millisecond)
      result = unquote(block)
      duration = System.monotonic_time(:millisecond) - start
      
      status = case result do
        {:ok, _} -> :success
        {:error, _} -> :error
        :ok -> :success
        _ -> :unknown
      end
      
      Metrics.inc_operation(unquote(operation), status, unquote(namespace))
      Metrics.observe_duration(unquote(operation), unquote(namespace), duration)
      
      result
    end
  end
end
