defmodule SessionIdentityCore.Telemetry.Instrumenter do
  @moduledoc """
  Telemetry event emission helpers for Session Identity Core.
  
  Provides functions to emit telemetry events that are captured
  by the metrics module for Prometheus exposition.
  """

  @prefix [:session_identity]

  # Session events

  @doc "Emit session created event"
  def session_created(status \\ :success) do
    :telemetry.execute(
      @prefix ++ [:session, :created],
      %{count: 1},
      %{status: status}
    )
  end

  @doc "Emit session deleted event"
  def session_deleted(reason \\ :user_logout) do
    :telemetry.execute(
      @prefix ++ [:session, :deleted],
      %{count: 1},
      %{reason: reason}
    )
  end

  @doc "Emit session refreshed event"
  def session_refreshed do
    :telemetry.execute(@prefix ++ [:session, :refreshed], %{count: 1}, %{})
  end

  @doc "Emit session duration metric"
  def session_duration(duration_seconds) do
    :telemetry.execute(
      @prefix ++ [:session, :duration],
      %{seconds: duration_seconds},
      %{}
    )
  end

  # OAuth events

  @doc "Emit OAuth authorize event"
  def oauth_authorize(status) do
    :telemetry.execute(
      @prefix ++ [:oauth, :authorize],
      %{count: 1},
      %{status: status}
    )
  end

  @doc "Emit OAuth token event"
  def oauth_token(grant_type, status) do
    :telemetry.execute(
      @prefix ++ [:oauth, :token],
      %{count: 1},
      %{grant_type: grant_type, status: status}
    )
  end

  @doc "Emit OAuth refresh event"
  def oauth_refresh(status) do
    :telemetry.execute(
      @prefix ++ [:oauth, :refresh],
      %{count: 1},
      %{status: status}
    )
  end

  @doc "Emit token generation duration"
  def oauth_token_duration(duration_ms) do
    :telemetry.execute(
      @prefix ++ [:oauth, :token, :duration],
      %{milliseconds: duration_ms},
      %{}
    )
  end

  # PKCE events

  @doc "Emit PKCE verification event"
  def pkce_verification(status) do
    :telemetry.execute(
      @prefix ++ [:pkce, :verification],
      %{count: 1},
      %{status: status}
    )
  end

  # Risk scoring events

  @doc "Emit risk score metric"
  def risk_score(score) do
    :telemetry.execute(@prefix ++ [:risk, :score], %{score: score}, %{})
  end

  @doc "Emit step-up required event"
  def step_up_required do
    :telemetry.execute(@prefix ++ [:risk, :step_up_required], %{count: 1}, %{})
  end

  # CAEP events

  @doc "Emit CAEP event emission metric"
  def caep_event_emitted(event_type, status) do
    :telemetry.execute(
      @prefix ++ [:caep, :event, :emitted],
      %{count: 1},
      %{event_type: event_type, status: status}
    )
  end

  # Cache events

  @doc "Emit cache hit event"
  def cache_hit do
    :telemetry.execute(@prefix ++ [:cache, :hit], %{count: 1}, %{})
  end

  @doc "Emit cache miss event"
  def cache_miss do
    :telemetry.execute(@prefix ++ [:cache, :miss], %{count: 1}, %{})
  end

  @doc "Emit cache operation duration"
  def cache_operation_duration(duration_ms) do
    :telemetry.execute(
      @prefix ++ [:cache, :operation, :duration],
      %{milliseconds: duration_ms},
      %{}
    )
  end

  # Event store events

  @doc "Emit event appended metric"
  def event_appended(event_type) do
    :telemetry.execute(
      @prefix ++ [:events, :appended],
      %{count: 1},
      %{event_type: event_type}
    )
  end

  @doc "Emit current sequence number"
  def event_sequence_number(sequence) do
    :telemetry.execute(
      @prefix ++ [:events, :sequence_number],
      %{value: sequence},
      %{}
    )
  end

  # Health events

  @doc "Emit health status (1=healthy, 0=unhealthy)"
  def health_status(healthy?) do
    :telemetry.execute(
      @prefix ++ [:health, :status],
      %{value: if(healthy?, do: 1, else: 0)},
      %{}
    )
  end

  # Timing helper

  @doc "Measure execution time and emit metric"
  defmacro measure(metric_fn, do: block) do
    quote do
      start = System.monotonic_time(:millisecond)
      result = unquote(block)
      duration = System.monotonic_time(:millisecond) - start
      unquote(metric_fn).(duration)
      result
    end
  end
end
