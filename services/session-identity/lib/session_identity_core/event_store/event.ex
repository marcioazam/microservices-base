defmodule SessionIdentityCore.EventStore.Event do
  @moduledoc """
  Base event structure for event sourcing with versioning support.
  Implements append-only event storage with monotonically increasing sequence numbers.
  """

  @type t :: %__MODULE__{
          event_id: String.t(),
          event_type: String.t(),
          aggregate_id: String.t(),
          aggregate_type: String.t(),
          sequence_number: non_neg_integer(),
          timestamp: DateTime.t(),
          schema_version: non_neg_integer(),
          correlation_id: String.t(),
          causation_id: String.t() | nil,
          payload: map(),
          metadata: map()
        }

  @derive Jason.Encoder
  defstruct [
    :event_id,
    :event_type,
    :aggregate_id,
    :aggregate_type,
    :sequence_number,
    :timestamp,
    :schema_version,
    :correlation_id,
    :causation_id,
    :payload,
    :metadata
  ]

  @doc """
  Creates a new event with auto-generated ID and timestamp.
  """
  def new(attrs) do
    %__MODULE__{
      event_id: attrs[:event_id] || generate_event_id(),
      event_type: attrs[:event_type],
      aggregate_id: attrs[:aggregate_id],
      aggregate_type: attrs[:aggregate_type] || "Session",
      sequence_number: attrs[:sequence_number] || 0,
      timestamp: attrs[:timestamp] || DateTime.utc_now(),
      schema_version: attrs[:schema_version] || 1,
      correlation_id: attrs[:correlation_id] || generate_correlation_id(),
      causation_id: attrs[:causation_id],
      payload: attrs[:payload] || %{},
      metadata: attrs[:metadata] || %{}
    }
  end

  @doc """
  Serializes an event to JSON with schema version.
  """
  def serialize(%__MODULE__{} = event) do
    event
    |> Map.from_struct()
    |> Map.update!(:timestamp, &DateTime.to_iso8601/1)
    |> Jason.encode!()
  end

  @doc """
  Deserializes JSON to an event, handling schema migrations.
  """
  def deserialize(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, data} -> deserialize_map(data)
      {:error, reason} -> {:error, {:json_decode_error, reason}}
    end
  end

  defp deserialize_map(data) do
    schema_version = Map.get(data, "schema_version", 1)
    migrated_data = migrate_schema(data, schema_version)

    {:ok, timestamp, _} = DateTime.from_iso8601(migrated_data["timestamp"])

    {:ok,
     %__MODULE__{
       event_id: migrated_data["event_id"],
       event_type: migrated_data["event_type"],
       aggregate_id: migrated_data["aggregate_id"],
       aggregate_type: migrated_data["aggregate_type"],
       sequence_number: migrated_data["sequence_number"],
       timestamp: timestamp,
       schema_version: migrated_data["schema_version"],
       correlation_id: migrated_data["correlation_id"],
       causation_id: migrated_data["causation_id"],
       payload: migrated_data["payload"],
       metadata: migrated_data["metadata"] || %{}
     }}
  end

  @doc """
  Migrates event data from older schema versions to current.
  """
  def migrate_schema(data, 1), do: data

  def migrate_schema(data, version) when version < 1 do
    data
    |> Map.put("schema_version", 1)
    |> Map.put_new("metadata", %{})
    |> Map.put_new("causation_id", nil)
    |> migrate_schema(1)
  end

  defp generate_event_id, do: UUID.uuid4()
  defp generate_correlation_id, do: UUID.uuid4()
end
