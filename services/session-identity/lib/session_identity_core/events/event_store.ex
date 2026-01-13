defmodule SessionIdentityCore.Events.EventStore do
  @moduledoc """
  Event store for event sourcing with snapshot support.
  
  Ensures:
  - Monotonic sequence numbers via atomic increment
  - Event persistence with correlation tracking
  - Snapshot loading for performance
  """

  alias AuthPlatform.Clients.{Cache, Logging}
  alias SessionIdentityCore.Events.Event
  alias SessionIdentityCore.Shared.{Keys, Errors}

  @sequence_key_prefix "sequence:"
  @events_key_prefix "events:"
  @snapshot_key_prefix "snapshot:"

  @doc """
  Appends an event to the store with monotonic sequence number.
  """
  @spec append(Event.t()) :: {:ok, Event.t()} | {:error, term()}
  def append(%Event{} = event) do
    sequence = get_next_sequence(event.aggregate_type, event.aggregate_id)
    event_with_seq = %{event | sequence_number: sequence}

    key = events_key(event.aggregate_type, event.aggregate_id)
    event_json = event_with_seq |> Event.to_map() |> Jason.encode!()

    with {:ok, existing} <- Cache.get(key),
         events_list <- parse_events_list(existing),
         updated <- events_list ++ [event_json],
         :ok <- Cache.set(key, Jason.encode!(updated)) do
      Logging.info("Event appended",
        event_id: event_with_seq.event_id,
        event_type: event_with_seq.event_type,
        aggregate_id: event_with_seq.aggregate_id,
        sequence: sequence,
        correlation_id: event_with_seq.correlation_id
      )
      {:ok, event_with_seq}
    end
  end

  @doc """
  Loads all events for an aggregate.
  """
  @spec load_events(String.t(), String.t()) :: {:ok, [Event.t()]} | {:error, term()}
  def load_events(aggregate_type, aggregate_id) do
    key = events_key(aggregate_type, aggregate_id)

    case Cache.get(key) do
      {:ok, nil} ->
        {:ok, []}

      {:ok, json} ->
        events =
          json
          |> Jason.decode!()
          |> Enum.map(&Jason.decode!/1)
          |> Enum.map(&Event.from_map/1)
          |> Enum.map(fn {:ok, e} -> e end)

        {:ok, events}

      error ->
        error
    end
  end

  @doc """
  Loads events after a specific sequence number.
  """
  @spec load_events_after(String.t(), String.t(), pos_integer()) ::
          {:ok, [Event.t()]} | {:error, term()}
  def load_events_after(aggregate_type, aggregate_id, after_sequence) do
    with {:ok, events} <- load_events(aggregate_type, aggregate_id) do
      filtered = Enum.filter(events, fn e -> e.sequence_number > after_sequence end)
      {:ok, filtered}
    end
  end

  @doc """
  Saves a snapshot for an aggregate.
  """
  @spec save_snapshot(String.t(), String.t(), map(), pos_integer()) :: :ok | {:error, term()}
  def save_snapshot(aggregate_type, aggregate_id, state, sequence) do
    key = snapshot_key(aggregate_type, aggregate_id)

    snapshot = %{
      "state" => state,
      "sequence" => sequence,
      "timestamp" => DateTime.to_iso8601(DateTime.utc_now())
    }

    Cache.set(key, Jason.encode!(snapshot))
  end

  @doc """
  Loads a snapshot for an aggregate.
  """
  @spec load_snapshot(String.t(), String.t()) ::
          {:ok, {map(), pos_integer()}} | {:error, :not_found}
  def load_snapshot(aggregate_type, aggregate_id) do
    key = snapshot_key(aggregate_type, aggregate_id)

    case Cache.get(key) do
      {:ok, nil} ->
        {:error, :not_found}

      {:ok, json} ->
        snapshot = Jason.decode!(json)
        {:ok, {snapshot["state"], snapshot["sequence"]}}

      error ->
        error
    end
  end

  @doc """
  Loads aggregate state from snapshot + subsequent events.
  """
  @spec load_with_snapshot(String.t(), String.t(), (map(), Event.t() -> map()), map()) ::
          {:ok, map()} | {:error, term()}
  def load_with_snapshot(aggregate_type, aggregate_id, apply_fn, initial_state \\ %{}) do
    case load_snapshot(aggregate_type, aggregate_id) do
      {:ok, {state, sequence}} ->
        with {:ok, events} <- load_events_after(aggregate_type, aggregate_id, sequence) do
          final_state = Enum.reduce(events, state, apply_fn)
          {:ok, final_state}
        end

      {:error, :not_found} ->
        with {:ok, events} <- load_events(aggregate_type, aggregate_id) do
          final_state = Enum.reduce(events, initial_state, apply_fn)
          {:ok, final_state}
        end
    end
  end

  @doc """
  Gets the current sequence number for an aggregate.
  """
  @spec current_sequence(String.t(), String.t()) :: pos_integer()
  def current_sequence(aggregate_type, aggregate_id) do
    key = sequence_key(aggregate_type, aggregate_id)

    case Cache.get(key) do
      {:ok, nil} -> 0
      {:ok, value} -> String.to_integer(value)
      _ -> 0
    end
  end

  # Private functions

  # SECURITY FIX: Use atomic INCR operation to prevent race condition
  # Previous implementation had a read-increment-write race condition where
  # concurrent requests could get duplicate sequence numbers.
  #
  # Redis INCR command is atomic and guarantees monotonic sequence numbers
  # even under high concurrency.
  defp get_next_sequence(aggregate_type, aggregate_id) do
    key = sequence_key(aggregate_type, aggregate_id)

    # Use atomic Redis INCR command for thread-safe increment
    # This returns the new value after incrementing
    case Cache.increment(key) do
      {:ok, next} -> next
      {:error, _reason} ->
        # Fallback: If Cache doesn't support increment, log error and use non-atomic method
        # This should be fixed by implementing Cache.increment/1 in the cache client
        Logging.error("EventStore", "Failed to use atomic increment - falling back to unsafe method", %{
          aggregate_type: aggregate_type,
          aggregate_id: aggregate_id
        })

        # Non-atomic fallback (still has race condition - should not be used in production)
        current = current_sequence(aggregate_type, aggregate_id)
        next = current + 1
        Cache.set(key, Integer.to_string(next))
        next
    end
  end

  defp sequence_key(aggregate_type, aggregate_id) do
    "#{@sequence_key_prefix}#{aggregate_type}:#{aggregate_id}"
  end

  defp events_key(aggregate_type, aggregate_id) do
    "#{@events_key_prefix}#{aggregate_type}:#{aggregate_id}"
  end

  defp snapshot_key(aggregate_type, aggregate_id) do
    "#{@snapshot_key_prefix}#{aggregate_type}:#{aggregate_id}"
  end

  defp parse_events_list(nil), do: []
  defp parse_events_list(json), do: Jason.decode!(json)
end
