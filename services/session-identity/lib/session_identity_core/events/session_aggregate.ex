defmodule SessionIdentityCore.Events.SessionAggregate do
  @moduledoc """
  Session aggregate for event sourcing.
  
  Ensures:
  - SessionCreated events have correlation_id
  - SessionInvalidated events have reason field
  - Proper event replay for state reconstruction
  """

  alias SessionIdentityCore.Events.{Event, EventStore}
  alias SessionIdentityCore.Shared.DateTime, as: DT

  @aggregate_type "Session"

  defstruct [
    :session_id,
    :user_id,
    :device_fingerprint,
    :ip_address,
    :created_at,
    :invalidated_at,
    :invalidation_reason,
    :mfa_verified,
    :status
  ]

  @type t :: %__MODULE__{}

  # Event types
  @session_created "SessionCreated"
  @session_invalidated "SessionInvalidated"
  @session_mfa_verified "SessionMfaVerified"
  @session_activity_updated "SessionActivityUpdated"

  @doc """
  Creates a new session and emits SessionCreated event.
  """
  @spec create(map()) :: {:ok, t(), Event.t()} | {:error, term()}
  def create(attrs) do
    correlation_id = attrs[:correlation_id] || generate_correlation_id()

    event = Event.new(%{
      event_type: @session_created,
      aggregate_id: attrs[:session_id],
      aggregate_type: @aggregate_type,
      correlation_id: correlation_id,
      payload: %{
        "user_id" => attrs[:user_id],
        "device_fingerprint" => attrs[:device_fingerprint],
        "ip_address" => attrs[:ip_address],
        "user_agent" => attrs[:user_agent],
        "risk_score" => attrs[:risk_score]
      },
      metadata: %{
        "source" => "session_manager"
      }
    })

    with {:ok, stored_event} <- EventStore.append(event) do
      state = apply_event(%__MODULE__{}, stored_event)
      {:ok, state, stored_event}
    end
  end

  @doc """
  Invalidates a session and emits SessionInvalidated event with reason.
  """
  @spec invalidate(String.t(), atom(), String.t() | nil) ::
          {:ok, t(), Event.t()} | {:error, term()}
  def invalidate(session_id, reason, correlation_id \\ nil) do
    correlation_id = correlation_id || generate_correlation_id()

    event = Event.new(%{
      event_type: @session_invalidated,
      aggregate_id: session_id,
      aggregate_type: @aggregate_type,
      correlation_id: correlation_id,
      payload: %{
        "reason" => Atom.to_string(reason),
        "invalidated_at" => DT.to_iso8601(DateTime.utc_now())
      },
      metadata: %{
        "source" => "session_manager"
      }
    })

    with {:ok, state} <- load(session_id),
         {:ok, stored_event} <- EventStore.append(event) do
      updated_state = apply_event(state, stored_event)
      {:ok, updated_state, stored_event}
    end
  end

  @doc """
  Records MFA verification and emits event.
  """
  @spec verify_mfa(String.t(), String.t() | nil) :: {:ok, t(), Event.t()} | {:error, term()}
  def verify_mfa(session_id, correlation_id \\ nil) do
    correlation_id = correlation_id || generate_correlation_id()

    event = Event.new(%{
      event_type: @session_mfa_verified,
      aggregate_id: session_id,
      aggregate_type: @aggregate_type,
      correlation_id: correlation_id,
      payload: %{
        "verified_at" => DT.to_iso8601(DateTime.utc_now())
      },
      metadata: %{}
    })

    with {:ok, state} <- load(session_id),
         {:ok, stored_event} <- EventStore.append(event) do
      updated_state = apply_event(state, stored_event)
      {:ok, updated_state, stored_event}
    end
  end

  @doc """
  Loads aggregate state from events.
  """
  @spec load(String.t()) :: {:ok, t()} | {:error, term()}
  def load(session_id) do
    EventStore.load_with_snapshot(
      @aggregate_type,
      session_id,
      &apply_event/2,
      %__MODULE__{}
    )
  end

  @doc """
  Applies an event to the aggregate state.
  """
  @spec apply_event(t(), Event.t()) :: t()
  def apply_event(state, %Event{event_type: @session_created} = event) do
    %__MODULE__{
      state
      | session_id: event.aggregate_id,
        user_id: event.payload["user_id"],
        device_fingerprint: event.payload["device_fingerprint"],
        ip_address: event.payload["ip_address"],
        created_at: event.timestamp,
        status: :active,
        mfa_verified: false
    }
  end

  def apply_event(state, %Event{event_type: @session_invalidated} = event) do
    %__MODULE__{
      state
      | invalidated_at: DT.from_iso8601(event.payload["invalidated_at"]),
        invalidation_reason: parse_invalidation_reason(event.payload["reason"]),
        status: :invalidated
    }
  end

  # SECURITY: Whitelist of valid invalidation reasons to prevent atom exhaustion
  # Uses String.to_existing_atom/1 which only converts to pre-existing atoms
  @valid_invalidation_reasons ~w(
    user_logout
    timeout
    security
    admin_action
    token_revoked
    max_sessions_exceeded
    device_changed
    ip_changed
    suspicious_activity
  )

  defp parse_invalidation_reason(reason_str) when reason_str in @valid_invalidation_reasons do
    String.to_existing_atom(reason_str)
  end

  defp parse_invalidation_reason(_unknown_reason) do
    # Default to :unknown for any reason not in whitelist
    # This prevents atom exhaustion from malicious/unexpected values
    :unknown
  end

  def apply_event(state, %Event{event_type: @session_mfa_verified}) do
    %__MODULE__{state | mfa_verified: true}
  end

  def apply_event(state, %Event{event_type: @session_activity_updated}) do
    state
  end

  def apply_event(state, _event), do: state

  # Private functions

  defp generate_correlation_id do
    :crypto.strong_rand_bytes(16) |> Base.url_encode64(padding: false)
  end
end
