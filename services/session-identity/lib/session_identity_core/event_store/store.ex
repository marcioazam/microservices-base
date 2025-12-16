defmodule SessionIdentityCore.EventStore.Store do
  @moduledoc """
  Append-only event store with monotonically increasing sequence numbers.
  Provides event persistence and replay capabilities.
  """

  use GenServer
  alias SessionIdentityCore.EventStore.Event

  @event_prefix "events:"
  @sequence_key "event_store:sequence"
  @aggregate_events_prefix "aggregate_events:"

  # Client API

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Appends an event to the store with a monotonically increasing sequence number.
  """
  def append(event) do
    GenServer.call(__MODULE__, {:append, event})
  end

  @doc """
  Appends multiple events atomically.
  """
  def append_batch(events) when is_list(events) do
    GenServer.call(__MODULE__, {:append_batch, events})
  end

  @doc """
  Retrieves all events for an aggregate in sequence order.
  """
  def get_events(aggregate_id) do
    GenServer.call(__MODULE__, {:get_events, aggregate_id})
  end

  @doc """
  Retrieves events for an aggregate starting from a specific sequence number.
  """
  def get_events_from(aggregate_id, from_sequence) do
    GenServer.call(__MODULE__, {:get_events_from, aggregate_id, from_sequence})
  end

  @doc """
  Gets the current global sequence number.
  """
  def get_sequence_number do
    GenServer.call(__MODULE__, :get_sequence)
  end

  # Server Callbacks

  @impl true
  def init(_opts) do
    {:ok, %{}}
  end

  @impl true
  def handle_call({:append, event}, _from, state) do
    result = do_append_event(event)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:append_batch, events}, _from, state) do
    result = do_append_batch(events)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:get_events, aggregate_id}, _from, state) do
    result = do_get_events(aggregate_id)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:get_events_from, aggregate_id, from_seq}, _from, state) do
    result = do_get_events_from(aggregate_id, from_seq)
    {:reply, result, state}
  end

  @impl true
  def handle_call(:get_sequence, _from, state) do
    result = get_current_sequence()
    {:reply, result, state}
  end

  # Private Functions

  defp do_append_event(event) do
    with {:ok, sequence} <- increment_sequence(),
         event_with_seq = %{event | sequence_number: sequence},
         :ok <- store_event(event_with_seq),
         :ok <- index_aggregate_event(event_with_seq) do
      {:ok, event_with_seq}
    end
  end

  defp do_append_batch(events) do
    Enum.reduce_while(events, {:ok, []}, fn event, {:ok, acc} ->
      case do_append_event(event) do
        {:ok, stored_event} -> {:cont, {:ok, [stored_event | acc]}}
        error -> {:halt, error}
      end
    end)
    |> case do
      {:ok, stored} -> {:ok, Enum.reverse(stored)}
      error -> error
    end
  end

  defp do_get_events(aggregate_id) do
    key = aggregate_events_key(aggregate_id)

    case Redix.command(:redix, ["LRANGE", key, "0", "-1"]) do
      {:ok, event_ids} ->
        events =
          event_ids
          |> Enum.map(&fetch_event/1)
          |> Enum.filter(&match?({:ok, _}, &1))
          |> Enum.map(fn {:ok, e} -> e end)
          |> Enum.sort_by(& &1.sequence_number)

        {:ok, events}

      error ->
        error
    end
  end

  defp do_get_events_from(aggregate_id, from_sequence) do
    case do_get_events(aggregate_id) do
      {:ok, events} ->
        filtered = Enum.filter(events, &(&1.sequence_number >= from_sequence))
        {:ok, filtered}

      error ->
        error
    end
  end

  defp increment_sequence do
    case Redix.command(:redix, ["INCR", @sequence_key]) do
      {:ok, seq} -> {:ok, seq}
      error -> error
    end
  end

  defp get_current_sequence do
    case Redix.command(:redix, ["GET", @sequence_key]) do
      {:ok, nil} -> {:ok, 0}
      {:ok, seq} -> {:ok, String.to_integer(seq)}
      error -> error
    end
  end

  defp store_event(event) do
    key = event_key(event.event_id)
    json = Event.serialize(event)

    case Redix.command(:redix, ["SET", key, json]) do
      {:ok, _} -> :ok
      error -> error
    end
  end

  defp fetch_event(event_id) do
    key = event_key(event_id)

    case Redix.command(:redix, ["GET", key]) do
      {:ok, nil} -> {:error, :not_found}
      {:ok, json} -> Event.deserialize(json)
      error -> error
    end
  end

  defp index_aggregate_event(event) do
    key = aggregate_events_key(event.aggregate_id)

    case Redix.command(:redix, ["RPUSH", key, event.event_id]) do
      {:ok, _} -> :ok
      error -> error
    end
  end

  defp event_key(event_id), do: "#{@event_prefix}#{event_id}"
  defp aggregate_events_key(aggregate_id), do: "#{@aggregate_events_prefix}#{aggregate_id}"
end
