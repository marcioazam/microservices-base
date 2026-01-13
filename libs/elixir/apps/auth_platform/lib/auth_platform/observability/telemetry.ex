defmodule AuthPlatform.Observability.Telemetry do
  @moduledoc """
  Telemetry event definitions and handler utilities.

  Defines all telemetry events emitted by the Auth Platform library and
  provides helpers for attaching handlers.

  ## Events

  ### Circuit Breaker
  - `[:auth_platform, :circuit_breaker, :state_change]`
  - `[:auth_platform, :circuit_breaker, :request_blocked]`

  ### Retry
  - `[:auth_platform, :retry, :attempt]`
  - `[:auth_platform, :retry, :exhausted]`

  ### Rate Limiter
  - `[:auth_platform, :rate_limiter, :allowed]`
  - `[:auth_platform, :rate_limiter, :rejected]`

  ### Bulkhead
  - `[:auth_platform, :bulkhead, :acquired]`
  - `[:auth_platform, :bulkhead, :released]`
  - `[:auth_platform, :bulkhead, :rejected]`
  - `[:auth_platform, :bulkhead, :queued]`

  ## Usage

      # Attach a handler for circuit breaker events
      Telemetry.attach_circuit_breaker_handler(fn event, measurements, metadata ->
        IO.inspect({event, measurements, metadata})
      end)

  """

  @circuit_breaker_events [
    [:auth_platform, :circuit_breaker, :state_change],
    [:auth_platform, :circuit_breaker, :request_blocked]
  ]

  @retry_events [
    [:auth_platform, :retry, :attempt],
    [:auth_platform, :retry, :exhausted]
  ]

  @rate_limiter_events [
    [:auth_platform, :rate_limiter, :allowed],
    [:auth_platform, :rate_limiter, :rejected]
  ]

  @bulkhead_events [
    [:auth_platform, :bulkhead, :acquired],
    [:auth_platform, :bulkhead, :released],
    [:auth_platform, :bulkhead, :rejected],
    [:auth_platform, :bulkhead, :queued]
  ]

  @doc """
  Returns all circuit breaker telemetry events.
  """
  @spec circuit_breaker_events() :: [list(atom())]
  def circuit_breaker_events, do: @circuit_breaker_events

  @doc """
  Returns all retry telemetry events.
  """
  @spec retry_events() :: [list(atom())]
  def retry_events, do: @retry_events

  @doc """
  Returns all rate limiter telemetry events.
  """
  @spec rate_limiter_events() :: [list(atom())]
  def rate_limiter_events, do: @rate_limiter_events

  @doc """
  Returns all bulkhead telemetry events.
  """
  @spec bulkhead_events() :: [list(atom())]
  def bulkhead_events, do: @bulkhead_events

  @doc """
  Returns all Auth Platform telemetry events.
  """
  @spec all_events() :: [list(atom())]
  def all_events do
    @circuit_breaker_events ++ @retry_events ++ @rate_limiter_events ++ @bulkhead_events
  end

  @doc """
  Attaches a handler for circuit breaker events.
  """
  @spec attach_circuit_breaker_handler(function()) :: :ok | {:error, :already_exists}
  def attach_circuit_breaker_handler(handler) do
    attach_handler("auth_platform_circuit_breaker", @circuit_breaker_events, handler)
  end

  @doc """
  Attaches a handler for retry events.
  """
  @spec attach_retry_handler(function()) :: :ok | {:error, :already_exists}
  def attach_retry_handler(handler) do
    attach_handler("auth_platform_retry", @retry_events, handler)
  end

  @doc """
  Attaches a handler for rate limiter events.
  """
  @spec attach_rate_limiter_handler(function()) :: :ok | {:error, :already_exists}
  def attach_rate_limiter_handler(handler) do
    attach_handler("auth_platform_rate_limiter", @rate_limiter_events, handler)
  end

  @doc """
  Attaches a handler for bulkhead events.
  """
  @spec attach_bulkhead_handler(function()) :: :ok | {:error, :already_exists}
  def attach_bulkhead_handler(handler) do
    attach_handler("auth_platform_bulkhead", @bulkhead_events, handler)
  end

  @doc """
  Attaches a handler for all Auth Platform events.
  """
  @spec attach_all_handlers(function()) :: :ok | {:error, :already_exists}
  def attach_all_handlers(handler) do
    attach_handler("auth_platform_all", all_events(), handler)
  end

  @doc """
  Detaches a handler by ID.
  """
  @spec detach_handler(String.t()) :: :ok | {:error, :not_found}
  def detach_handler(handler_id) do
    :telemetry.detach(handler_id)
  end

  # Private functions

  defp attach_handler(handler_id, events, handler) do
    :telemetry.attach_many(
      handler_id,
      events,
      fn event, measurements, metadata, _config ->
        handler.(event, measurements, metadata)
      end,
      nil
    )
  end
end
