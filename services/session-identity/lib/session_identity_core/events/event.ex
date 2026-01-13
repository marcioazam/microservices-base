defmodule SessionIdentityCore.Events.Event do
  @moduledoc """
  Event structure for event sourcing.
  
  Ensures:
  - Monotonic sequence numbers
  - Correlation ID on all events
  - ISO 8601 UTC timestamps
  - Schema versioning for migrations
  """

  alias SessionIdentityCore.Shared.DateTime, as: DT

  @current_schema_version 1

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

  @type t :: %__MODULE__{
          event_id: String.t(),
          event_type: String.t(),
          aggregate_id: String.t(),
          aggregate_type: String.t(),
          sequence_number: pos_integer(),
          timestamp: DateTime.t(),
          schema_version: pos_integer(),
          correlation_id: String.t(),
          causation_id: String.t() | nil,
          payload: map(),
          metadata: map()
        }

  @doc """
  Creates a new event with required fields.
  
  Generates event_id and correlation_id if not provided.
  """
  @spec new(map()) :: t()
  def new(attrs) do
    %__MODULE__{
      event_id: attrs[:event_id] || generate_id(),
      event_type: attrs[:event_type],
      aggregate_id: attrs[:aggregate_id],
      aggregate_type: attrs[:aggregate_type],
      sequence_number: attrs[:sequence_number],
      timestamp: attrs[:timestamp] || DateTime.utc_now(),
      schema_version: attrs[:schema_version] || @current_schema_version,
      correlation_id: attrs[:correlation_id] || generate_id(),
      causation_id: attrs[:causation_id],
      payload: attrs[:payload] || %{},
      metadata: attrs[:metadata] || %{}
    }
  end

  @doc """
  Serializes an event to a map for storage.
  """
  @spec to_map(t()) :: map()
  def to_map(%__MODULE__{} = event) do
    %{
      "event_id" => event.event_id,
      "event_type" => event.event_type,
      "aggregate_id" => event.aggregate_id,
      "aggregate_type" => event.aggregate_type,
      "sequence_number" => event.sequence_number,
      "timestamp" => DT.to_iso8601(event.timestamp),
      "schema_version" => event.schema_version,
      "correlation_id" => event.correlation_id,
      "causation_id" => event.causation_id,
      "payload" => event.payload,
      "metadata" => event.metadata
    }
  end

  @doc """
  Deserializes a map to an event struct.
  """
  @spec from_map(map()) :: {:ok, t()} | {:error, term()}
  def from_map(data) when is_map(data) do
    event = %__MODULE__{
      event_id: data["event_id"],
      event_type: data["event_type"],
      aggregate_id: data["aggregate_id"],
      aggregate_type: data["aggregate_type"],
      sequence_number: data["sequence_number"],
      timestamp: DT.from_iso8601(data["timestamp"]),
      schema_version: data["schema_version"] || 1,
      correlation_id: data["correlation_id"],
      causation_id: data["causation_id"],
      payload: data["payload"] || %{},
      metadata: data["metadata"] || %{}
    }

    {:ok, migrate_schema(event)}
  end

  @doc """
  Migrates an event to the current schema version.
  """
  @spec migrate_schema(t()) :: t()
  def migrate_schema(%__MODULE__{schema_version: @current_schema_version} = event), do: event

  def migrate_schema(%__MODULE__{schema_version: version} = event)
      when version < @current_schema_version do
    event
    |> apply_migrations(version)
    |> Map.put(:schema_version, @current_schema_version)
  end

  @doc """
  Returns the current schema version.
  """
  @spec current_schema_version() :: pos_integer()
  def current_schema_version, do: @current_schema_version

  @doc """
  Validates that an event has all required fields.
  """
  @spec valid?(t()) :: boolean()
  def valid?(%__MODULE__{} = event) do
    not is_nil(event.event_id) and
      not is_nil(event.event_type) and
      not is_nil(event.aggregate_id) and
      not is_nil(event.aggregate_type) and
      not is_nil(event.sequence_number) and
      not is_nil(event.timestamp) and
      not is_nil(event.correlation_id)
  end

  # Private functions

  defp generate_id do
    :crypto.strong_rand_bytes(16) |> Base.url_encode64(padding: false)
  end

  defp apply_migrations(event, _from_version) do
    # Add migration logic here as schema evolves
    event
  end
end
