defmodule SessionIdentityCore.EventStore.Aggregate do
  @moduledoc """
  Session aggregate with event sourcing support.
  Implements event replay for state reconstruction with optimistic concurrency.
  """

  alias SessionIdentityCore.EventStore.{Event, Store}
  alias SessionIdentityCore.EventStore.Events.{
    SessionCreated,
    SessionRefreshed,
    SessionInvalidated,
    DeviceBound,
    MfaVerified,
    RiskScoreUpdated
  }

  defstruct [
    :id,
    :user_id,
    :device_id,
    :device_fingerprint,
    :ip_address,
    :user_agent,
    :created_at,
    :expires_at,
    :status,
    :auth_methods,
    :risk_score,
    :mfa_verified,
    :version
  ]

  @type t :: %__MODULE__{}

  @doc """
  Loads an aggregate by replaying all its events.
  """
  def load(aggregate_id) do
    case Store.get_events(aggregate_id) do
      {:ok, events} ->
        aggregate = Enum.reduce(events, %__MODULE__{version: 0}, &apply_event/2)
        {:ok, aggregate}

      error ->
        error
    end
  end

  @doc """
  Loads an aggregate from a specific version (for snapshots).
  """
  def load_from(aggregate_id, snapshot, from_version) do
    case Store.get_events_from(aggregate_id, from_version + 1) do
      {:ok, events} ->
        aggregate = Enum.reduce(events, snapshot, &apply_event/2)
        {:ok, aggregate}

      error ->
        error
    end
  end

  @doc """
  Creates a new session and emits SessionCreated event.
  """
  def create_session(attrs, correlation_id) do
    session_id = attrs[:session_id] || UUID.uuid4()

    event_payload = %SessionCreated{
      session_id: session_id,
      user_id: attrs[:user_id],
      device_id: attrs[:device_id],
      device_fingerprint: attrs[:device_fingerprint],
      ip_address: attrs[:ip_address],
      user_agent: attrs[:user_agent],
      created_at: DateTime.utc_now(),
      expires_at: attrs[:expires_at] || default_expiry(),
      auth_methods: attrs[:auth_methods] || [],
      risk_score: attrs[:risk_score] || 0.0
    }

    event = Event.new(
      event_type: "SessionCreated",
      aggregate_id: session_id,
      aggregate_type: "Session",
      correlation_id: correlation_id,
      payload: Map.from_struct(event_payload)
    )

    case Store.append(event) do
      {:ok, stored_event} ->
        aggregate = apply_event(stored_event, %__MODULE__{version: 0})
        {:ok, aggregate, stored_event}

      error ->
        error
    end
  end

  @doc """
  Refreshes a session and emits SessionRefreshed event.
  """
  def refresh_session(aggregate, new_expires_at, correlation_id, causation_id \\ nil) do
    old_token = aggregate.id
    new_token = UUID.uuid4()

    event_payload = %SessionRefreshed{
      session_id: aggregate.id,
      old_token: old_token,
      new_token: new_token,
      refreshed_at: DateTime.utc_now(),
      new_expires_at: new_expires_at,
      reason: "renewal"
    }

    event = Event.new(
      event_type: "SessionRefreshed",
      aggregate_id: aggregate.id,
      aggregate_type: "Session",
      correlation_id: correlation_id,
      causation_id: causation_id,
      payload: Map.from_struct(event_payload)
    )

    case Store.append(event) do
      {:ok, stored_event} ->
        updated = apply_event(stored_event, aggregate)
        {:ok, updated, stored_event}

      error ->
        error
    end
  end

  @doc """
  Invalidates a session and emits SessionInvalidated event.
  """
  def invalidate_session(aggregate, reason, initiated_by, correlation_id) do
    event_payload = %SessionInvalidated{
      session_id: aggregate.id,
      user_id: aggregate.user_id,
      invalidated_at: DateTime.utc_now(),
      reason: reason,
      initiated_by: initiated_by
    }

    event = Event.new(
      event_type: "SessionInvalidated",
      aggregate_id: aggregate.id,
      aggregate_type: "Session",
      correlation_id: correlation_id,
      payload: Map.from_struct(event_payload)
    )

    case Store.append(event) do
      {:ok, stored_event} ->
        updated = apply_event(stored_event, aggregate)
        {:ok, updated, stored_event}

      error ->
        error
    end
  end

  @doc """
  Binds a device to the session.
  """
  def bind_device(aggregate, device_info, correlation_id) do
    event_payload = %DeviceBound{
      session_id: aggregate.id,
      device_id: device_info[:device_id],
      device_fingerprint: device_info[:fingerprint],
      bound_at: DateTime.utc_now(),
      device_info: device_info
    }

    event = Event.new(
      event_type: "DeviceBound",
      aggregate_id: aggregate.id,
      aggregate_type: "Session",
      correlation_id: correlation_id,
      payload: Map.from_struct(event_payload)
    )

    case Store.append(event) do
      {:ok, stored_event} ->
        updated = apply_event(stored_event, aggregate)
        {:ok, updated, stored_event}

      error ->
        error
    end
  end

  # Event Application

  defp apply_event(%Event{event_type: "SessionCreated", payload: p}, aggregate) do
    %{aggregate |
      id: p["session_id"],
      user_id: p["user_id"],
      device_id: p["device_id"],
      device_fingerprint: p["device_fingerprint"],
      ip_address: p["ip_address"],
      user_agent: p["user_agent"],
      created_at: parse_datetime(p["created_at"]),
      expires_at: parse_datetime(p["expires_at"]),
      status: :active,
      auth_methods: p["auth_methods"] || [],
      risk_score: p["risk_score"] || 0.0,
      mfa_verified: false,
      version: aggregate.version + 1
    }
  end

  defp apply_event(%Event{event_type: "SessionRefreshed", payload: p}, aggregate) do
    %{aggregate |
      expires_at: parse_datetime(p["new_expires_at"]),
      version: aggregate.version + 1
    }
  end

  defp apply_event(%Event{event_type: "SessionInvalidated"}, aggregate) do
    %{aggregate |
      status: :invalidated,
      version: aggregate.version + 1
    }
  end

  defp apply_event(%Event{event_type: "DeviceBound", payload: p}, aggregate) do
    %{aggregate |
      device_id: p["device_id"],
      device_fingerprint: p["device_fingerprint"],
      version: aggregate.version + 1
    }
  end

  defp apply_event(%Event{event_type: "MfaVerified", payload: p}, aggregate) do
    %{aggregate |
      mfa_verified: p["result"] == "success",
      version: aggregate.version + 1
    }
  end

  defp apply_event(%Event{event_type: "RiskScoreUpdated", payload: p}, aggregate) do
    %{aggregate |
      risk_score: p["new_score"],
      version: aggregate.version + 1
    }
  end

  defp apply_event(_event, aggregate), do: aggregate

  defp default_expiry do
    DateTime.utc_now() |> DateTime.add(24 * 60 * 60, :second)
  end

  defp parse_datetime(nil), do: nil
  defp parse_datetime(%DateTime{} = dt), do: dt
  defp parse_datetime(str) when is_binary(str) do
    case DateTime.from_iso8601(str) do
      {:ok, dt, _} -> dt
      _ -> nil
    end
  end
end
