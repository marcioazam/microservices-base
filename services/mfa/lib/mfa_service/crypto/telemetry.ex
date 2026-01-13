defmodule MfaService.Crypto.Telemetry do
  @moduledoc """
  Telemetry events for Crypto Service operations.
  Emits events for RPC calls, circuit breaker state changes, and errors.
  Integrates with Prometheus for metrics collection.
  """

  require Logger

  @prefix [:mfa_service, :crypto]

  # Prometheus metric names
  @rpc_duration_histogram "mfa_crypto_rpc_duration_milliseconds"
  @rpc_total_counter "mfa_crypto_rpc_total"
  @circuit_breaker_state_gauge "mfa_crypto_circuit_breaker_state"
  @circuit_breaker_rejections_counter "mfa_crypto_circuit_breaker_rejections_total"
  @failures_counter "mfa_crypto_failures_total"

  @doc """
  Emits a telemetry event for an RPC call.
  Includes operation type, latency, and status.
  """
  @spec emit_rpc_call(atom(), {:ok, term()} | {:error, term()}, non_neg_integer(), String.t()) :: :ok
  def emit_rpc_call(operation, result, duration_ms, correlation_id) do
    status = case result do
      {:ok, _} -> :success
      {:error, _} -> :failure
    end

    :telemetry.execute(
      @prefix ++ [:rpc, :call],
      %{duration_ms: duration_ms, count: 1},
      %{
        operation: operation,
        status: status,
        correlation_id: correlation_id,
        timestamp: DateTime.utc_now()
      }
    )

    log_rpc_call(operation, status, duration_ms, correlation_id)
    :ok
  end

  @doc """
  Emits a telemetry event for circuit breaker state change.
  """
  @spec emit_circuit_breaker_state_change(atom()) :: :ok
  def emit_circuit_breaker_state_change(new_state) do
    :telemetry.execute(
      @prefix ++ [:circuit_breaker, :state_change],
      %{},
      %{new_state: new_state}
    )

    Logger.info("Crypto service circuit breaker state changed",
      new_state: new_state)
    :ok
  end

  @doc """
  Emits a telemetry event when circuit breaker rejects a request.
  """
  @spec emit_circuit_breaker_rejection(atom(), String.t()) :: :ok
  def emit_circuit_breaker_rejection(operation, correlation_id) do
    :telemetry.execute(
      @prefix ++ [:circuit_breaker, :rejection],
      %{},
      %{
        operation: operation,
        correlation_id: correlation_id
      }
    )

    Logger.warning("Crypto service request rejected by circuit breaker",
      operation: operation, correlation_id: correlation_id)
    :ok
  end

  @doc """
  Emits a telemetry event for a failure.
  """
  @spec emit_failure(atom(), term(), String.t()) :: :ok
  def emit_failure(operation, error_code, correlation_id) do
    :telemetry.execute(
      @prefix ++ [:failure],
      %{},
      %{
        operation: operation,
        error_code: error_code,
        correlation_id: correlation_id
      }
    )

    Logger.error("Crypto service operation failed",
      operation: operation, error_code: error_code, correlation_id: correlation_id)
    :ok
  end

  @doc """
  Attaches default telemetry handlers for Prometheus metrics.
  """
  @spec attach_handlers() :: :ok
  def attach_handlers do
    :telemetry.attach_many(
      "mfa-crypto-metrics",
      [
        @prefix ++ [:rpc, :call],
        @prefix ++ [:circuit_breaker, :state_change],
        @prefix ++ [:circuit_breaker, :rejection],
        @prefix ++ [:failure]
      ],
      &handle_event/4,
      nil
    )
    :ok
  end

  defp handle_event([@prefix, :rpc, :call] = _event, measurements, metadata, _config) do
    # Update Prometheus metrics
    labels = [operation: metadata.operation, status: metadata.status]
    
    # Histogram for duration
    :telemetry_metrics_prometheus_core.observe(
      @rpc_duration_histogram,
      measurements.duration_ms,
      labels
    )
    
    # Counter for total calls
    :telemetry_metrics_prometheus_core.increment(
      @rpc_total_counter,
      labels
    )
  end

  defp handle_event([@prefix, :circuit_breaker, :state_change] = _event, _measurements, metadata, _config) do
    state_value = case metadata.new_state do
      :closed -> 0
      :half_open -> 1
      :open -> 2
    end
    
    :telemetry_metrics_prometheus_core.set(
      @circuit_breaker_state_gauge,
      state_value,
      []
    )
  end

  defp handle_event([@prefix, :circuit_breaker, :rejection] = _event, _measurements, metadata, _config) do
    :telemetry_metrics_prometheus_core.increment(
      @circuit_breaker_rejections_counter,
      [operation: metadata.operation]
    )
  end

  defp handle_event([@prefix, :failure] = _event, _measurements, metadata, _config) do
    :telemetry_metrics_prometheus_core.increment(
      @failures_counter,
      [operation: metadata.operation, error_code: metadata.error_code]
    )
  end

  defp handle_event(_event, _measurements, _metadata, _config), do: :ok

  defp log_rpc_call(operation, status, duration_ms, correlation_id) do
    level = if status == :success, do: :debug, else: :warning
    
    Logger.log(level, "Crypto service RPC call",
      operation: operation,
      status: status,
      duration_ms: duration_ms,
      correlation_id: correlation_id)
  end

  @doc """
  Returns the Prometheus metrics definitions.
  """
  @spec metrics() :: list()
  def metrics do
    [
      Telemetry.Metrics.distribution(@rpc_duration_histogram,
        event_name: @prefix ++ [:rpc, :call],
        measurement: :duration_ms,
        tags: [:operation, :status],
        unit: {:native, :millisecond},
        reporter_options: [buckets: [5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000]]
      ),
      Telemetry.Metrics.counter(@rpc_total_counter,
        event_name: @prefix ++ [:rpc, :call],
        tags: [:operation, :status]
      ),
      Telemetry.Metrics.last_value(@circuit_breaker_state_gauge,
        event_name: @prefix ++ [:circuit_breaker, :state_change],
        measurement: fn _measurements -> 1 end
      ),
      Telemetry.Metrics.counter(@circuit_breaker_rejections_counter,
        event_name: @prefix ++ [:circuit_breaker, :rejection],
        tags: [:operation]
      ),
      Telemetry.Metrics.counter(@failures_counter,
        event_name: @prefix ++ [:failure],
        tags: [:operation, :error_code]
      )
    ]
  end

  @doc """
  Returns health metrics for the crypto service.
  """
  @spec health_metrics() :: map()
  def health_metrics do
    %{
      circuit_breaker_state: get_circuit_breaker_state(),
      last_successful_call: get_last_successful_call(),
      error_rate_1m: calculate_error_rate(60),
      avg_latency_1m: calculate_avg_latency(60)
    }
  end

  defp get_circuit_breaker_state do
    # Would query actual circuit breaker state
    :closed
  end

  defp get_last_successful_call do
    # Would query from ETS or similar
    DateTime.utc_now()
  end

  defp calculate_error_rate(_window_seconds) do
    # Would calculate from metrics
    0.0
  end

  defp calculate_avg_latency(_window_seconds) do
    # Would calculate from metrics
    0.0
  end
end
